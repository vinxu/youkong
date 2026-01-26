import { Button } from '../components'

export function ExpiredPage() {
  const handleDownloadApp = () => {
    window.open('https://youkong.app/download', '_blank')
  }

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center p-6">
      <div className="text-6xl mb-6">😢</div>

      <h1 className="text-2xl font-bold text-gray-900 mb-2">
        邀请链接已失效
      </h1>

      <p className="text-gray-500 text-center mb-8 max-w-xs">
        该邀请链接可能已过期、已被使用或已被取消
      </p>

      <div className="w-full max-w-xs space-y-3">
        <Button className="w-full" size="lg" onClick={handleDownloadApp}>
          下载「有空」App
        </Button>

        <p className="text-xs text-gray-400 text-center">
          下载 App 后可以创建自己的邀请链接
        </p>
      </div>

      {/* 底部品牌 */}
      <div className="absolute bottom-8 flex items-center gap-2 text-gray-400">
        <span className="text-xl">⏰</span>
        <span className="font-medium">有空 YouKong</span>
      </div>
    </div>
  )
}

export default ExpiredPage
