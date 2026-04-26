import { useEffect, useRef, useCallback, useState } from 'react'
import { getToken } from './auth'
import { loadConfig } from './config'
import { addMessage, markSentDelivered, markSentSeen, type StoredMessage } from './messageStore'

interface ChatResponseEnvelope {
  version: number
  type: string
  data: StoredMessage | { conversation_id: number; message_id: number }
}

export function useChatSession(onMessage?: (msg: StoredMessage) => void, onDelivered?: (conversationId: number, messageId: number) => void, onConnect?: () => void) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<number | null>(null)
  const countdownTimer = useRef<number | null>(null)
  const onMessageRef = useRef(onMessage)
  const onDeliveredRef = useRef(onDelivered)
  const onConnectRef = useRef(onConnect)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [retryCountdown, setRetryCountdown] = useState(0)

  useEffect(() => {
    onMessageRef.current = onMessage
    onDeliveredRef.current = onDelivered
    onConnectRef.current = onConnect
  }, [onMessage, onDelivered, onConnect])

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
          addMessage(msg)
          onMessageRef.current?.(msg)
          // If this message is from someone other than the current user, mark it as read
          const token = getToken()
          if (token) {
            try {
              const payload = JSON.parse(atob(token.split('.')[1]))
              const currentUserId = payload.sub || payload.user_id
              if (msg.sender_id !== currentUserId) {
                markRead(msg.conversation_id, msg.id, msg.sender_id)
              }
            } catch { /* ignore */ }
          }
        } else if (envelope.type === 'delivered' && envelope.data) {
          const d = envelope.data as { conversation_id: number; message_id: number }
          markSentDelivered(d.conversation_id, d.message_id)
          onDeliveredRef.current?.(d.conversation_id, d.message_id)
        } else if (envelope.type === 'read' && envelope.data) {
          const d = envelope.data as { conversation_id: number; message_id: number }
          markSentSeen(d.conversation_id, d.message_id)
          onDeliveredRef.current?.(d.conversation_id, d.message_id)
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

  const send = useCallback((conversationId: number, content: string, messageType = 'text', replyTo = 0) => {
    if (wsRef.current?.readyState !== WebSocket.OPEN) return
    const config = loadConfig()
    wsRef.current.send(JSON.stringify({
      version: config?.chatRequestVersion ?? 1,
      type: 'send',
      data: {
        conversation_id: conversationId,
        content,
        message_type: messageType,
        reply_to_message_id: replyTo,
      },
    }))
  }, [])

  const markRead = useCallback((conversationId: number, messageId: number, senderId: string) => {
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
