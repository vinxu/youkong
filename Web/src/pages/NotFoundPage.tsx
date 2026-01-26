import { useNavigate } from 'react-router-dom'
import { Button } from '../components'

export function NotFoundPage() {
  const navigate = useNavigate()

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center p-6">
      <div className="text-6xl mb-6">🔍</div>

      <h1 className="text-2xl font-bold text-gray-900 mb-2">
        页面不存在
      </h1>

      <p className="text-gray-500 text-center mb-8">
        抱歉，您访问的页面不存在
      </p>

      <Button variant="outline" onClick={() => navigate('/')}>
        返回首页
      </Button>

      {/* 底部品牌 */}
      <div className="absolute bottom-8 flex items-center gap-2 text-gray-400">
        <span className="text-xl">⏰</span>
        <span className="font-medium">有空 YouKong</span>
      </div>
    </div>
  )
}

export default NotFoundPage
