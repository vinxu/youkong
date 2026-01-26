import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Avatar, Button, Loading, AvailabilityCard } from '../components'
import { useAuthStore } from '../stores/authStore'
import availabilityApi from '../api/availability'
import type { Availability } from '../types'

export function AvailabilityPage() {
  const { userId } = useParams<{ userId: string }>()
  const navigate = useNavigate()
  const { isAuthenticated, inviteContext, loadFromStorage } = useAuthStore()

  const [availabilities, setAvailabilities] = useState<Availability[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadFromStorage()
  }, [loadFromStorage])

  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/')
      return
    }

    fetchAvailabilities()
  }, [isAuthenticated, userId, navigate])

  const fetchAvailabilities = async () => {
    setLoading(true)
    setError(null)

    try {
      const response = await availabilityApi.getFriendsAvailabilities()
      if (response.code === 0 && response.data) {
        // 如果指定了 userId，过滤该用户的有空状态
        const filtered = userId
          ? response.data.filter((a) => a.user.id === userId)
          : response.data
        setAvailabilities(filtered)
      } else {
        setError(response.message || '获取数据失败')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '网络错误')
    } finally {
      setLoading(false)
    }
  }

  const handleDownloadApp = () => {
    window.open('https://youkong.app/download', '_blank')
  }

  // 获取要显示的用户信息
  const targetUser = inviteContext?.inviter || (availabilities.length > 0 ? availabilities[0].user : null)

  if (loading) {
    return <Loading text="正在获取有空状态..." />
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
        {targetUser && (
          <div className="flex items-center gap-2">
            <Avatar src={targetUser.avatar} name={targetUser.nickname} size="sm" />
            <span className="font-medium text-gray-900">{targetUser.nickname}</span>
          </div>
        )}
      </div>

      {/* 主内容 */}
      <div className="flex-1 p-4">
        {error ? (
          <div className="flex flex-col items-center justify-center py-12">
            <div className="text-4xl mb-4">😢</div>
            <p className="text-gray-500 mb-4">{error}</p>
            <Button variant="outline" onClick={fetchAvailabilities}>
              重试
            </Button>
          </div>
        ) : availabilities.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12">
            <div className="text-4xl mb-4">📭</div>
            <p className="text-gray-500 mb-2">暂无有空状态</p>
            <p className="text-sm text-gray-400 text-center">
              {targetUser ? `${targetUser.nickname} 还没有发布有空状态` : '暂时没有可见的有空状态'}
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-gray-900">
              {targetUser?.nickname || '好友'} 最近有空
            </h2>
            {availabilities.map((availability) => (
              <AvailabilityCard
                key={availability.id}
                availability={availability}
                showUser={!userId}
              />
            ))}
          </div>
        )}
      </div>

      {/* 底部引导 */}
      <div className="sticky bottom-0 bg-white border-t border-gray-100 p-4 safe-area-bottom">
        <div className="max-w-sm mx-auto">
          <p className="text-center text-sm text-gray-500 mb-3">
            想发起你的有空？下载 App 开始使用
          </p>
          <Button className="w-full" size="lg" onClick={handleDownloadApp}>
            下载「有空」App
          </Button>
        </div>
      </div>
    </div>
  )
}

export default AvailabilityPage
