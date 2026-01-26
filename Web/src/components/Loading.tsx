interface LoadingProps {
  text?: string
  fullScreen?: boolean
}

export function Loading({ text = '加载中...', fullScreen = true }: LoadingProps) {
  const content = (
    <div className="flex flex-col items-center justify-center gap-3">
      <div className="relative w-10 h-10">
        <div className="absolute inset-0 rounded-full border-4 border-gray-200" />
        <div className="absolute inset-0 rounded-full border-4 border-primary-500 border-t-transparent animate-spin" />
      </div>
      {text && <p className="text-gray-500 text-sm">{text}</p>}
    </div>
  )

  if (fullScreen) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        {content}
      </div>
    )
  }

  return content
}

export default Loading
