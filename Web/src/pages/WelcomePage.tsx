import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Avatar, Button, CircleTag } from '../components'
import { useAuthStore } from '../stores/authStore'

export function WelcomePage() {
  const navigate = useNavigate()
  const { inviteContext, user, isAuthenticated, loadFromStorage } = useAuthStore()

  useEffect(() => {
    loadFromStorage()
  }, [loadFromStorage])

  // 未登录则跳回首页
  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/')
    }
  }, [isAuthenticated, navigate])

  if (!inviteContext || !inviteContext.inviter) {
    return (
      <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center p-6">
        <div className="text-6xl mb-4">🎉</div>
        <h1 className="text-xl font-semibold text-gray-900 mb-2">
          欢迎使用有空
        </h1>
        <p className="text-gray-500 text-center mb-6">
          开始探索你的社交圈吧
        </p>
        <Button onClick={() => navigate('/')}>
          返回首页
        </Button>
      </div>
    )
  }

  const { inviter, circle, isNewUser } = inviteContext

  const handleViewAvailability = () => {
    if (inviter) {
      navigate(`/availability/${inviter.id}`)
    }
  }

  const handleDownloadApp = () => {
    // TODO: 根据设备类型跳转到对应的应用商店
    window.open('https://youkong.app/download', '_blank')
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-primary-50 to-white flex flex-col">
      {/* 顶部装饰 */}
      <div className="h-2 bg-primary-500" />

      {/* 主内容 */}
      <div className="flex-1 flex flex-col items-center justify-center px-6 py-12">
        {/* 成功图标 */}
        <div className="w-20 h-20 rounded-full bg-primary-100 flex items-center justify-center mb-6">
          <span className="text-4xl">🎉</span>
        </div>

        {/* 成功文案 */}
        <h1 className="text-2xl font-bold text-gray-900 mb-2">
          {isNewUser ? '注册成功！' : '加入成功！'}
        </h1>

        <div className="text-center space-y-2 mb-6">
          <p className="text-gray-600">
            你已成为 <span className="font-semibold text-gray-900">{inviter.nickname}</span> 的好友
          </p>
          {circle && (
            <p className="text-gray-600 flex items-center justify-center gap-2">
              并加入了
              <CircleTag
                name={circle.name}
                emoji={circle.emoji}
                color={circle.color}
                size="sm"
              />
              圈子
            </p>
          )}
        </div>

        {/* 邀请者信息卡片 */}
        <div className="w-full max-w-sm bg-white rounded-2xl p-6 shadow-sm border border-gray-100 mb-8">
          <div className="flex items-center gap-4">
            <Avatar src={inviter.avatar} name={inviter.nickname} size="lg" />
            <div className="flex-1">
              <h3 className="font-semibold text-gray-900">{inviter.nickname}</h3>
              <p className="text-sm text-gray-500">你的新朋友</p>
            </div>
          </div>
        </div>

        {/* 操作按钮 */}
        <div className="w-full max-w-sm space-y-3">
          <Button
            className="w-full"
            size="lg"
            onClick={handleViewAvailability}
          >
            查看 TA 的有空状态
          </Button>

          <Button
            className="w-full"
            size="lg"
            variant="outline"
            onClick={handleDownloadApp}
          >
            下载「有空」App
          </Button>
        </div>

        {/* 当前用户信息 */}
        {user && (
          <div className="mt-8 text-center">
            <p className="text-sm text-gray-400">
              当前账号：{user.nickname || user.phone}
            </p>
          </div>
        )}
      </div>
    </div>
  )
}

export default WelcomePage
