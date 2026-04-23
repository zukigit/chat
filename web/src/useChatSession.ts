import { useEffect, useRef, useCallback, useState } from 'react'
import { getToken } from './auth'
import { loadConfig } from './config'
import { addMessage, type StoredMessage } from './messageStore'

interface ChatEnvelope {
  version: number
  type: string
  data: StoredMessage | { conversation_id: number; message_id: number }
}

export function useChatSession(onMessage?: (msg: StoredMessage) => void, onConnect?: () => void) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<number | null>(null)
  const countdownTimer = useRef<number | null>(null)
  const onMessageRef = useRef(onMessage)
  const onConnectRef = useRef(onConnect)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [retryCountdown, setRetryCountdown] = useState(0)

  useEffect(() => {
    onMessageRef.current = onMessage
    onConnectRef.current = onConnect
  }, [onMessage, onConnect])

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
        const envelope: ChatEnvelope = JSON.parse(event.data)
        if (envelope.type === 'message' && envelope.data) {
          const msg = envelope.data as StoredMessage
          addMessage(msg)
          onMessageRef.current?.(msg)
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
    wsRef.current.send(JSON.stringify({
      conversation_id: conversationId,
      content,
      message_type: messageType,
      reply_to_message_id: replyTo,
    }))
  }, [])

  return { connected, error, retryCountdown, send, connect }
}
