import { useEffect, useRef, useCallback, useState } from 'react'
import { getToken } from './auth'
import { loadConfig } from './config'

export interface Notification {
  id: number
  user_id: string
  sender_id: string | null
  type: string
  message: string
  reference_id: number | null
  is_read: boolean
  created_at: string
}

export function useNotificationSession(onNotification?: (noti: Notification) => void, onConnect?: () => void) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<number | null>(null)
  const onNotificationRef = useRef(onNotification)
  const onConnectRef = useRef(onConnect)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    onNotificationRef.current = onNotification
    onConnectRef.current = onConnect
  }, [onNotification, onConnect])

  const connect = useCallback(() => {
    const token = getToken()
    const config = loadConfig()
    if (!token || !config?.gatewayUrl) return

    if (wsRef.current?.readyState === WebSocket.OPEN) return

    setError(null)
    const gatewayUrl = config.gatewayUrl.replace(/^http/, 'ws')
    const ws = new WebSocket(`${gatewayUrl}/sessions/notification`)

    ws.onopen = () => {
      setConnected(true)
      setError(null)
      ws.send(JSON.stringify({ version: 1, type: 'auth', data: { token } }))
      onConnectRef.current?.()
    }
    ws.onclose = () => {
      setConnected(false)
      reconnectTimer.current = window.setTimeout(connect, 5000)
    }
    ws.onerror = () => {
      setError('Failed to connect to notification server')
      ws.close()
    }

    ws.onmessage = (event) => {
      try {
        const noti: Notification = JSON.parse(event.data)
        onNotificationRef.current?.(noti)
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
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [connect])

  return { connected, error }
}
