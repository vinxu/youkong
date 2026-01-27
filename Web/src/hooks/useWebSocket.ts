import { useEffect, useRef, useCallback, useState } from 'react'
import type { WSMessage, Message } from '../types'

const WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL ||
  (window.location.protocol === 'https:' ? 'wss://' : 'ws://') +
  window.location.host + '/ws'

interface UseWebSocketOptions {
  onMessage?: (conversationId: string, message: Message) => void
  enabled?: boolean
}

export function useWebSocket({ onMessage, enabled = true }: UseWebSocketOptions = {}) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout>>()
  const [isConnected, setIsConnected] = useState(false)

  const connect = useCallback(() => {
    if (!enabled) return

    const token = localStorage.getItem('token')
    if (!token) return

    // 关闭现有连接
    if (wsRef.current) {
      wsRef.current.close()
    }

    const ws = new WebSocket(`${WS_BASE_URL}?token=${token}`)

    ws.onopen = () => {
      console.log('[WS] Connected')
      setIsConnected(true)
    }

    ws.onclose = () => {
      console.log('[WS] Disconnected')
      setIsConnected(false)
      // 5秒后重连
      reconnectTimeoutRef.current = setTimeout(connect, 5000)
    }

    ws.onerror = (error) => {
      console.error('[WS] Error:', error)
    }

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)

        if (msg.type === 'NEW_MESSAGE' && msg.data && onMessage) {
          onMessage(msg.data.conversation_id, msg.data.message)
        }
      } catch (e) {
        console.error('[WS] Parse error:', e)
      }
    }

    wsRef.current = ws
  }, [enabled, onMessage])

  // 发送心跳
  useEffect(() => {
    if (!isConnected || !wsRef.current) return

    const pingInterval = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'PING' }))
      }
    }, 30000)

    return () => clearInterval(pingInterval)
  }, [isConnected])

  // 连接和清理
  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [connect])

  return { isConnected }
}

export default useWebSocket
