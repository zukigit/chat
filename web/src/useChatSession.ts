import { useEffect, useRef, useCallback, useState } from 'react'
import { getToken } from './auth'
import { loadConfig } from './config'
import { addMessage, addRemoteSentMessage, hasPendingSent, markSentDelivered, markSentSent, markSentSeen, markSentFailed, type StoredMessage } from './messageStore'
import { encrypt, decrypt, hasPrivateKey, AGE_ARMOR_BEGIN } from './crypto'
import { getPublicKeys } from './api/keysApi'

interface ChatResponseEnvelope {
  version: number
  type: string
  data: StoredMessage | { conversation_id: number; message_id: string } | { code: number; message: string }
}

function looksEncrypted(content: string): boolean {
  return content.startsWith(AGE_ARMOR_BEGIN)
}

export function useChatSession(activeConversationId: number | null, onMessage?: (msg: StoredMessage) => void, onSent?: (conversationId: number, messageId: string) => void, onDelivered?: (conversationId: number, messageId: string) => void, onError?: (code: number, message: string, conversationId?: number) => void, onConnect?: () => void) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<number | null>(null)
  const countdownTimer = useRef<number | null>(null)
  const activeConvIdRef = useRef(activeConversationId)
  const onMessageRef = useRef(onMessage)
  const onSentRef = useRef(onSent)
  const onDeliveredRef = useRef(onDelivered)
  const onErrorRef = useRef(onError)
  const onConnectRef = useRef(onConnect)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [retryCountdown, setRetryCountdown] = useState(0)

  useEffect(() => {
    activeConvIdRef.current = activeConversationId
    onMessageRef.current = onMessage
    onSentRef.current = onSent
    onDeliveredRef.current = onDelivered
    onErrorRef.current = onError
    onConnectRef.current = onConnect
  }, [activeConversationId, onMessage, onSent, onDelivered, onError, onConnect])

  const connect = useCallback(() => {
    const token = getToken()
    const config = loadConfig()
    if (!token || !config?.gatewayUrl) return

    if (wsRef.current?.readyState === WebSocket.OPEN) return

    setError(null)
    setRetryCountdown(0)
    const gatewayUrl = config.gatewayUrl.replace(/^http/, 'ws')
    const ws = new WebSocket(`${gatewayUrl}/sessions/chat`)

    ws.onopen = () => {
      setConnected(true)
      setError(null)
      setRetryCountdown(0)
      ws.send(JSON.stringify({ version: 1, type: 'auth', data: { token } }))
      onConnectRef.current?.()
    }
    ws.onclose = () => {
      setConnected(false)
      if (countdownTimer.current) clearInterval(countdownTimer.current)
      let countdown = 5
      setRetryCountdown(countdown)
      countdownTimer.current = window.setInterval(() => {
        countdown--
        setRetryCountdown(countdown)
        if (countdown <= 0 && countdownTimer.current) clearInterval(countdownTimer.current)
      }, 1000)
      reconnectTimer.current = window.setTimeout(connect, 5000)
    }
    ws.onerror = () => {
      setError('Failed to connect to server')
      ws.close()
    }

    ws.onmessage = async (event) => {
      try {
        const envelope: ChatResponseEnvelope = JSON.parse(event.data)
        if (envelope.type === 'message' && envelope.data) {
          const msg = envelope.data as StoredMessage
          const token = getToken()
          let isOwnMessage = false
          if (token) {
            try {
              const payload = JSON.parse(atob(token.split('.')[1]))
              const currentUserId = payload.sub || payload.user_id
              isOwnMessage = msg.sender_id === currentUserId
            } catch { /* ignore */ }
          }

          // Decrypt for display only — msg.content stays encrypted for storage
          let displayContent = msg.content
          const hasKey = hasPrivateKey()
          if (hasKey && looksEncrypted(msg.content)) {
            try {
              displayContent = await decrypt(msg.content)
            } catch {
              console.error('E2EE: decryption failed for message', msg.id)
            }
          }

          if (isOwnMessage) {
            // Only add a remote-sent placeholder if this message did NOT
            // originate locally (i.e. no pending local sent entry exists).
            if (!hasPendingSent(msg.conversation_id, msg.id)) {
              addRemoteSentMessage(msg.conversation_id, msg.id, displayContent, msg.created_at)
            }
            onDeliveredRef.current?.(msg.conversation_id, msg.id)
          }

          addMessage(msg)

          // Pass decrypted content to React for display
          const displayMsg: StoredMessage = { ...msg, content: displayContent }
          onMessageRef.current?.(displayMsg)
          if (msg.conversation_id === activeConvIdRef.current) {
            if (token && !isOwnMessage) {
              try {
                const payload = JSON.parse(atob(token.split('.')[1]))
                const currentUserId = payload.sub || payload.user_id
                if (msg.sender_id !== currentUserId) {
                  sendRead(wsRef.current, msg.conversation_id, msg.id, msg.sender_id)
                }
              } catch { /* ignore */ }
            }
          }
        } else if (envelope.type === 'sent' && envelope.data) {
          const s = envelope.data as { conversation_id: number; message_id: string }
          markSentSent(s.conversation_id, s.message_id)
          onSentRef.current?.(s.conversation_id, s.message_id)
        } else if (envelope.type === 'delivered' && envelope.data) {
          const d = envelope.data as { conversation_id: number; message_id: string }
          markSentDelivered(d.conversation_id, d.message_id)
          onDeliveredRef.current?.(d.conversation_id, d.message_id)
        } else if (envelope.type === 'read' && envelope.data) {
          const d = envelope.data as { conversation_id: number; message_id: string }
          markSentSeen(d.conversation_id, d.message_id)
          onDeliveredRef.current?.(d.conversation_id, d.message_id)
        } else if (envelope.type === 'error' && envelope.data) {
          const e = envelope.data as { code: number; message: string; conversation_id?: number; message_id?: string }
          if (e.conversation_id !== undefined && e.message_id) {
            markSentFailed(e.conversation_id, e.message_id)
            onErrorRef.current?.(e.code, e.message, e.conversation_id)
          } else {
            onErrorRef.current?.(e.code, e.message)
          }
        }
      } catch {
        // ignore unparseable frames
      }
    }

    wsRef.current = ws
  }, [])

  useEffect(() => {
    connect()
    return () => {
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current)
      if (countdownTimer.current) clearInterval(countdownTimer.current)
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [connect])

  const send = useCallback(async (conversationId: number, messageId: string, content: string, messageType = 'text', replyTo = '', memberUserIds?: string[]) => {
    if (wsRef.current?.readyState !== WebSocket.OPEN) {
      markSentFailed(conversationId, messageId)
      return
    }

    let encryptedContent = content
    const ready = hasPrivateKey()
    const hasRecipients = memberUserIds && memberUserIds.length > 0
    if (ready && hasRecipients) {
      try {
        const pubKeys = await getPublicKeys(memberUserIds)
        const recipients = Object.values(pubKeys)
        if (recipients.length > 0) {
          encryptedContent = await encrypt(content, recipients)
        }
      } catch (err) {
        markSentFailed(conversationId, messageId)
        onErrorRef.current?.(-1, `Encryption failed: ${err instanceof Error ? err.message : 'unknown error'}`, conversationId)
        return
      }
    }

    const config = loadConfig()
    const payload: Record<string, unknown> = {
      version: config?.chatRequestVersion ?? 1,
      type: 'send',
      data: {
        conversation_id: conversationId,
        message_id: messageId,
        content: encryptedContent,
        message_type: messageType,
      },
    }
    if (replyTo) {
      ;(payload.data as Record<string, unknown>).reply_to_message_id = replyTo
    }
    wsRef.current.send(JSON.stringify(payload))
  }, [])

  const markRead = useCallback((conversationId: number, messageId: string, senderId: string) => {
    if (wsRef.current?.readyState !== WebSocket.OPEN) return
    const config = loadConfig()
    wsRef.current.send(JSON.stringify({
      version: config?.chatRequestVersion ?? 1,
      type: 'read',
      data: {
        conversation_id: conversationId,
        message_id: messageId,
        sender_id: senderId,
      },
    }))
  }, [])

  const markAllRead = useCallback((conversationId: number, messages: StoredMessage[], currentUserId: string) => {
    for (const m of messages) {
      if (m.sender_id !== currentUserId) {
        markRead(conversationId, m.id, m.sender_id)
      }
    }
  }, [markRead])

  const retrySend = useCallback(async (_tempId: string, conversationId: number, messageId: string, content: string, messageType = 'text', replyTo = '', memberUserIds?: string[]) => {
    if (wsRef.current?.readyState !== WebSocket.OPEN) {
      markSentFailed(conversationId, messageId)
      return
    }

    let encryptedContent = content
    const retryReady = hasPrivateKey()
    const retryHasRecipients = memberUserIds && memberUserIds.length > 0
    if (retryReady && retryHasRecipients) {
      try {
        const pubKeys = await getPublicKeys(memberUserIds)
        const recipients = Object.values(pubKeys)
        if (recipients.length > 0) {
          encryptedContent = await encrypt(content, recipients)
        }
      } catch (err) {
        markSentFailed(conversationId, messageId)
        onErrorRef.current?.(-1, `Encryption failed: ${err instanceof Error ? err.message : 'unknown error'}`, conversationId)
        return
      }
    }

    const config = loadConfig()
    const payload: Record<string, unknown> = {
      version: config?.chatRequestVersion ?? 1,
      type: 'send',
      data: {
        conversation_id: conversationId,
        message_id: messageId,
        content: encryptedContent,
        message_type: messageType,
      },
    }
    if (replyTo) {
      ;(payload.data as Record<string, unknown>).reply_to_message_id = replyTo
    }
    wsRef.current.send(JSON.stringify(payload))
  }, [])

  return { connected, error, retryCountdown, send, markRead, markAllRead, connect, retrySend }
}

function sendRead(ws: WebSocket | null, conversationId: number, messageId: string, senderId: string) {
  if (ws?.readyState !== WebSocket.OPEN) return
  const config = loadConfig()
  ws.send(JSON.stringify({
    version: config?.chatRequestVersion ?? 1,
    type: 'read',
    data: {
      conversation_id: conversationId,
      message_id: messageId,
      sender_id: senderId,
    },
  }))
}
