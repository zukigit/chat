import { useEffect, useRef, useCallback, useState } from 'react'
import { getToken } from './auth'
import { loadConfig } from './config'
import { addMessage, addRemoteSentMessage, markSentDelivered, markSentSeen, type StoredMessage } from './messageStore'

interface ChatResponseEnvelope {
  version: number
  type: string
  data: StoredMessage | { conversation_id: number; message_id: string } | { code: number; message: string }
}

export function useChatSession(activeConversationId: number | null, onMessage?: (msg: StoredMessage) => void, onDelivered?: (conversationId: number, messageId: string) => void, onError?: (code: number, message: string) => void, onConnect?: () => void) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<number | null>(null)
  const countdownTimer = useRef<number | null>(null)
  const activeConvIdRef = useRef(activeConversationId)
  const onMessageRef = useRef(onMessage)
  const onDeliveredRef = useRef(onDelivered)
  const onErrorRef = useRef(onError)
  const onConnectRef = useRef(onConnect)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [retryCountdown, setRetryCountdown] = useState(0)

  useEffect(() => {
    activeConvIdRef.current = activeConversationId
    onMessageRef.current = onMessage
    onDeliveredRef.current = onDelivered
    onErrorRef.current = onError
    onConnectRef.current = onConnect
  }, [activeConversationId, onMessage, onDelivered, onError, onConnect])

  const connect = useCallback(() => {
    const token = getToken()
    const config = loadConfig()
    if (!token || !config?.gatewayUrl) return

    if (wsRef.current?.readyState === WebSocket.OPEN) return

    setError(null)
    setRetryCountdown(0)
    const gatewayUrl = config.gatewayUrl.replace(/^http/, 'ws')
    const ws = new WebSocket(`${gatewayUrl}/sessions/chat?token=${encodeURIComponent(token)}`)

    ws.onopen = () => {
      setConnected(true)
      setError(null)
      setRetryCountdown(0)
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

    ws.onmessage = (event) => {
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
          if (isOwnMessage) {
            addRemoteSentMessage(msg.conversation_id, msg.content)
            onDeliveredRef.current?.(msg.conversation_id, msg.id)
          }
          addMessage(msg)
          onMessageRef.current?.(msg)
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
        } else if (envelope.type === 'delivered' && envelope.data) {
          const d = envelope.data as { conversation_id: number; message_id: string }
          markSentDelivered(d.conversation_id, d.message_id)
          onDeliveredRef.current?.(d.conversation_id, d.message_id)
        } else if (envelope.type === 'read' && envelope.data) {
          const d = envelope.data as { conversation_id: number; message_id: string }
          markSentSeen(d.conversation_id, d.message_id)
          onDeliveredRef.current?.(d.conversation_id, d.message_id)
        } else if (envelope.type === 'error' && envelope.data) {
          const e = envelope.data as { code: number; message: string }
          onErrorRef.current?.(e.code, e.message)
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

  const send = useCallback((conversationId: number, content: string, messageType = 'text', replyTo = '') => {
    if (wsRef.current?.readyState !== WebSocket.OPEN) return
    const config = loadConfig()
    const payload: Record<string, unknown> = {
      version: config?.chatRequestVersion ?? 1,
      type: 'send',
      data: {
        conversation_id: conversationId,
        content,
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

  return { connected, error, retryCountdown, send, markRead, markAllRead, connect }
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
