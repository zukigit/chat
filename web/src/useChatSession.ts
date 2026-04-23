import { useEffect, useRef, useCallback, useState } from 'react'
import { getToken } from './auth'
import { loadConfig } from './config'
import { addMessage, type StoredMessage } from './messageStore'

interface ChatEnvelope {
  version: number
  type: string
  data: StoredMessage | { conversation_id: number; message_id: number }
}

export function useChatSession(onMessage?: (msg: StoredMessage) => void) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<number | null>(null)
  const [connected, setConnected] = useState(false)

  const connect = useCallback(() => {
    const token = getToken()
    const config = loadConfig()
    if (!token || !config?.gatewayUrl) return

    if (wsRef.current?.readyState === WebSocket.OPEN) return

    const gatewayUrl = config.gatewayUrl.replace(/^http/, 'ws')
    const ws = new WebSocket(`${gatewayUrl}/sessions/chat?token=${encodeURIComponent(token)}`)

    ws.onopen = () => setConnected(true)
    ws.onclose = () => {
      setConnected(false)
      reconnectTimer.current = window.setTimeout(connect, 3000)
    }
    ws.onerror = () => ws.close()

    ws.onmessage = (event) => {
      try {
        const envelope: ChatEnvelope = JSON.parse(event.data)
        if (envelope.type === 'message' && envelope.data) {
          const msg = envelope.data as StoredMessage
          addMessage(msg)
          onMessage?.(msg)
        }
      } catch {
        // ignore unparseable frames
      }
    }

    wsRef.current = ws
  }, [onMessage])

  useEffect(() => {
    connect()
    return () => {
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current)
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

  return { connected, send, connect }
}
