import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Avatar, Loading, Button } from '../components'
import { useAuthStore } from '../stores/authStore'
import { useWebSocket } from '../hooks/useWebSocket'
import conversationApi from '../api/conversation'
import type { Message, Conversation } from '../types'

export function ChatPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { isAuthenticated, user, loadFromStorage } = useAuthStore()

  const [messages, setMessages] = useState<Message[]>([])
  const [conversation, setConversation] = useState<Conversation | null>(null)
  const [loading, setLoading] = useState(true)
  const [agentLoading, setAgentLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const messagesEndRef = useRef<HTMLDivElement>(null)

  // WebSocket 消息处理
  const handleNewMessage = useCallback((conversationId: string, message: Message) => {
    if (conversationId === id) {
      setMessages(prev => {
        // 避免重复添加
        if (prev.some(m => m.id === message.id)) {
          return prev
        }
        return [...prev, message]
      })
    }
  }, [id])

  // 连接 WebSocket
  useWebSocket({
    onMessage: handleNewMessage,
    enabled: isAuthenticated && !!id,
  })

  useEffect(() => {
    loadFromStorage()
  }, [loadFromStorage])

  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/')
      return
    }

    if (id) {
      fetchConversationAndMessages()
    }
  }, [isAuthenticated, id, navigate])

  // 滚动到底部
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const fetchConversationAndMessages = async () => {
    if (!id) return

    setLoading(true)
    setError(null)

    try {
      // 获取消息
      const response = await conversationApi.getMessages(id, 50, 0)
      if (response.code === 0 && response.data) {
        // 消息按时间正序显示
        setMessages([...response.data].reverse())

        // 从消息中推断对方信息
        if (response.data.length > 0) {
          const partner = response.data.find((m: Message) => m.sender.id !== user?.id)?.sender
          if (partner) {
            setConversation({
              id,
              partner,
              unreadCount: 0,
              createdAt: response.data[0].createdAt,
            })
          }
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '网络错误')
    } finally {
      setLoading(false)
    }
  }

  const handleAgentSpeak = async () => {
    if (!id || agentLoading) return

    setAgentLoading(true)
    setError(null)

    try {
      const response = await conversationApi.agentReply(id)
      if (response.code === 0 && response.data) {
        // WebSocket 会推送消息，这里不需要手动添加
        // 但如果 WS 未连接，手动添加
        const newMsg = response.data
        setMessages(prev => {
          if (prev.some(m => m.id === newMsg.id)) {
            return prev
          }
          return [...prev, newMsg]
        })
      } else {
        setError(response.message || '生成回复失败')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '你的元婴罢工了')
    } finally {
      setAgentLoading(false)
    }
  }

  if (loading) {
    return <Loading text="加载中..." />
  }

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col">
      {/* 顶部导航 */}
      <div className="bg-white border-b border-gray-100 px-4 py-3 flex items-center gap-3">
        <button
          onClick={() => navigate(-1)}
          className="p-2 -ml-2 hover:bg-gray-100 rounded-lg transition-colors"
        >
          <svg
            className="w-5 h-5 text-gray-600"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M15 19l-7-7 7-7"
            />
          </svg>
        </button>
        {conversation?.partner && (
          <div className="flex items-center gap-2">
            <Avatar
              src={conversation.partner.avatar}
              name={conversation.partner.nickname}
              size="sm"
            />
            <span className="font-medium text-gray-900">
              {conversation.partner.nickname}
            </span>
          </div>
        )}
      </div>

      {/* 消息列表 */}
      <div className="flex-1 overflow-auto p-4 space-y-4">
        {messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12">
            <div className="text-4xl mb-4">💬</div>
            <p className="text-gray-500">还没有消息</p>
            <p className="text-sm text-gray-400 mt-1">点击下方按钮让元婴开始聊天</p>
          </div>
        ) : (
          messages.map((message) => (
            <MessageBubble
              key={message.id}
              message={message}
              isOwn={message.sender.id === user?.id}
            />
          ))
        )}

        {error && (
          <div className="text-center py-2">
            <span className="text-red-500 text-sm">{error}</span>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* 底部操作栏 */}
      <div className="sticky bottom-0 bg-white border-t border-gray-100 p-4 safe-area-bottom">
        <Button
          className="w-full"
          size="lg"
          onClick={handleAgentSpeak}
          disabled={agentLoading}
        >
          {agentLoading ? (
            <span className="flex items-center justify-center gap-2">
              <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24">
                <circle
                  className="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  strokeWidth="4"
                  fill="none"
                />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
              元婴思考中...
            </span>
          ) : (
            '让我的元婴说话'
          )}
        </Button>
      </div>
    </div>
  )
}

// 消息气泡组件
interface MessageBubbleProps {
  message: Message
  isOwn: boolean
}

function MessageBubble({ message, isOwn }: MessageBubbleProps) {
  return (
    <div className={`flex ${isOwn ? 'justify-end' : 'justify-start'}`}>
      <div className={`flex gap-2 max-w-[80%] ${isOwn ? 'flex-row-reverse' : ''}`}>
        <Avatar
          src={message.sender.avatar}
          name={message.sender.nickname}
          size="sm"
          className="flex-shrink-0"
        />
        <div
          className={`px-4 py-2 rounded-2xl ${
            isOwn
              ? 'bg-emerald-500 text-white rounded-tr-sm'
              : 'bg-white text-gray-900 rounded-tl-sm shadow-sm'
          }`}
        >
          {message.content || ''}
        </div>
      </div>
    </div>
  )
}

export default ChatPage
