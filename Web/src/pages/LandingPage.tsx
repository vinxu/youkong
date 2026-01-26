import { Button } from '../components'

export function LandingPage() {
  const handleDownloadiOS = () => {
    window.open('https://apps.apple.com/app/youkong', '_blank')
  }

  const handleDownloadAndroid = () => {
    window.open('https://play.google.com/store/apps/details?id=app.youkong', '_blank')
  }

  return (
    <div className="min-h-screen bg-white">
      {/* Hero Section */}
      <section className="relative overflow-hidden bg-gradient-to-br from-emerald-50 via-white to-teal-50">
        <div className="absolute inset-0 overflow-hidden">
          <div className="absolute -top-40 -right-40 w-80 h-80 bg-emerald-200 rounded-full opacity-20 blur-3xl" />
          <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-teal-200 rounded-full opacity-20 blur-3xl" />
        </div>

        <div className="relative max-w-6xl mx-auto px-6 py-20 md:py-32">
          <div className="text-center">
            {/* Logo */}
            <div className="inline-flex items-center gap-3 mb-8">
              <span className="text-5xl">⏰</span>
              <span className="text-3xl font-bold text-gray-900">有空</span>
            </div>

            {/* 主标语 */}
            <h1 className="text-4xl md:text-6xl font-bold text-gray-900 mb-6 leading-tight">
              一眼看到
              <br />
              <span className="text-emerald-500">谁可能有空</span>
            </h1>

            {/* 副标语 */}
            <p className="text-xl md:text-2xl text-gray-600 mb-4 max-w-2xl mx-auto">
              不用问「在吗」，AI 帮你判断
            </p>
            <p className="text-lg text-gray-500 mb-10 max-w-xl mx-auto">
              打开 APP，看到朋友按「有空概率」排序，点击就能聊
            </p>

            {/* 下载按钮 */}
            <div className="flex flex-col sm:flex-row gap-4 justify-center mb-16">
              <Button
                size="lg"
                onClick={handleDownloadiOS}
                className="inline-flex items-center gap-2 px-8 bg-emerald-500 hover:bg-emerald-600"
              >
                <AppleIcon />
                App Store 下载
              </Button>
              <Button
                size="lg"
                variant="outline"
                onClick={handleDownloadAndroid}
                className="inline-flex items-center gap-2 px-8 border-emerald-500 text-emerald-600"
              >
                <AndroidIcon />
                Google Play 下载
              </Button>
            </div>

            {/* 模拟手机界面 */}
            <div className="relative max-w-xs mx-auto">
              <div className="bg-gray-900 rounded-[2.5rem] p-2 shadow-2xl">
                <div className="bg-white rounded-[2rem] overflow-hidden">
                  {/* 状态栏 */}
                  <div className="bg-gray-50 px-4 py-2 flex justify-between text-xs text-gray-500">
                    <span>有空</span>
                    <span>周五 21:32</span>
                  </div>
                  {/* 朋友列表 */}
                  <div className="p-3 space-y-2">
                    <FriendCard name="李四" probability={92} reason="刷了40分钟手机，在家" color="very" />
                    <FriendCard name="王五" probability={78} reason="周五晚上，历史上通常有空" color="likely" />
                    <FriendCard name="张三" probability={45} reason="数据不足" color="maybe" />
                    <FriendCard name="赵六" probability={12} reason="在公司，可能还在加班" color="busy" />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* 核心理念 Section */}
      <section className="py-20 bg-gray-50">
        <div className="max-w-6xl mx-auto px-6">
          <h2 className="text-3xl md:text-4xl font-bold text-center text-gray-900 mb-4">
            不是「我告诉你我有空」
          </h2>
          <p className="text-xl text-emerald-600 text-center mb-12 font-medium">
            而是「AI 告诉你谁可能有空」
          </p>

          <div className="grid md:grid-cols-2 gap-8 max-w-4xl mx-auto">
            <CompareCard
              type="old"
              title="传统方式"
              items={[
                '发消息问「在吗」— 可能被拒绝',
                '发状态说「我有空」— 太 E 人了',
                '来回问时间 — 最后不了了之',
              ]}
            />
            <CompareCard
              type="new"
              title="有空方式"
              items={[
                '打开 APP 直接看到谁有空',
                '不需要任何人主动发布',
                'AI 自动分析，越用越准',
              ]}
            />
          </div>
        </div>
      </section>

      {/* 工作原理 Section */}
      <section className="py-20 bg-white">
        <div className="max-w-6xl mx-auto px-6">
          <h2 className="text-3xl md:text-4xl font-bold text-center text-gray-900 mb-4">
            AI 如何判断有空？
          </h2>
          <p className="text-lg text-gray-500 text-center mb-12 max-w-2xl mx-auto">
            通过三个维度综合分析，给出精准的有空概率
          </p>

          <div className="grid md:grid-cols-3 gap-8">
            <SignalCard
              emoji="📱"
              title="屏幕使用"
              description="正在刷手机 40 分钟？大概率有空。手机闲置 2 小时？可能在忙别的。"
              examples={[
                { signal: '刷娱乐 APP 30分钟', score: '+30' },
                { signal: '手机闲置很久', score: '-25' },
              ]}
            />
            <SignalCard
              emoji="📍"
              title="地理位置"
              description="在家通常更放松，在公司可能在忙。AI 会学习你的常去地点。"
              examples={[
                { signal: '在家', score: '+15' },
                { signal: '在公司上班时间', score: '-20' },
              ]}
            />
            <SignalCard
              emoji="🕐"
              title="时间规律"
              description="周五晚上 vs 周二早上？每个人有自己的时间规律，AI 会学习。"
              examples={[
                { signal: '周五晚上', score: '+20' },
                { signal: '工作日上午', score: '-20' },
              ]}
            />
          </div>
        </div>
      </section>

      {/* 颜色系统 Section */}
      <section className="py-20 bg-gray-50">
        <div className="max-w-6xl mx-auto px-6">
          <h2 className="text-3xl md:text-4xl font-bold text-center text-gray-900 mb-4">
            一眼就懂的颜色
          </h2>
          <p className="text-lg text-gray-500 text-center mb-12">
            绿色约，红色等，就这么简单
          </p>

          <div className="max-w-2xl mx-auto space-y-4">
            <ColorRow color="#22C55E" range="80-100%" label="很可能有空" example="刷了40分钟手机，在家" />
            <ColorRow color="#86EFAC" range="60-79%" label="可能有空" example="周五晚上，历史上通常有空" />
            <ColorRow color="#FACC15" range="40-59%" label="不太确定" example="数据不足" />
            <ColorRow color="#FB923C" range="20-39%" label="可能没空" example="在公司，快下班了" />
            <ColorRow color="#EF4444" range="0-19%" label="很可能没空" example="在公司，可能还在加班" />
            <ColorRow color="#9CA3AF" range="--" label="无法判断" example="未授权" />
          </div>
        </div>
      </section>

      {/* 权限说明 Section */}
      <section className="py-20 bg-white">
        <div className="max-w-6xl mx-auto px-6">
          <h2 className="text-3xl md:text-4xl font-bold text-center text-gray-900 mb-4">
            需要三个权限
          </h2>
          <p className="text-lg text-gray-500 text-center mb-12 max-w-2xl mx-auto">
            为了准确判断有空概率，我们需要以下权限
          </p>

          <div className="grid md:grid-cols-3 gap-8 max-w-4xl mx-auto">
            <PermissionCard
              emoji="📱"
              title="屏幕使用时间"
              description="判断是否正在用手机，这是最核心的「有空」信号"
            />
            <PermissionCard
              emoji="📍"
              title="地理位置"
              description="判断在家还是在公司，辅助分析有空状态"
            />
            <PermissionCard
              emoji="👥"
              title="通讯录"
              description="找到你的朋友，建立社交关系"
            />
          </div>

          <div className="mt-12 max-w-2xl mx-auto bg-emerald-50 rounded-2xl p-6">
            <h3 className="font-semibold text-gray-900 mb-3 flex items-center gap-2">
              <span>🔒</span> 隐私承诺
            </h3>
            <ul className="space-y-2 text-gray-600 text-sm">
              <li>• 不存储具体使用了什么 APP，只记录「娱乐/工作/通讯」类别</li>
              <li>• 不存储精确坐标，只记录「家/公司/外面」状态</li>
              <li>• 通讯录只用于匹配好友，不上传原始数据</li>
              <li>• 所有计算优先在本地完成</li>
            </ul>
          </div>
        </div>
      </section>

      {/* Agent 架构简介 */}
      <section className="py-20 bg-gray-900 text-white">
        <div className="max-w-6xl mx-auto px-6">
          <h2 className="text-3xl md:text-4xl font-bold text-center mb-4">
            每个人都有专属 AI Agent
          </h2>
          <p className="text-lg text-gray-400 text-center mb-12 max-w-2xl mx-auto">
            你的 Agent 负责收集你的数据，朋友的 Agent 负责他们的数据
            <br />
            打开 APP 时，你的 Agent 会综合分析所有朋友的状态
          </p>

          <div className="max-w-3xl mx-auto bg-gray-800 rounded-2xl p-6 font-mono text-sm">
            <div className="text-gray-400 mb-4">// 你打开 APP 时发生的事</div>
            <div className="space-y-2 text-gray-300">
              <div><span className="text-emerald-400">你的 Agent</span>.激活()</div>
              <div className="pl-4">
                <div><span className="text-blue-400">→</span> 向 李四.Agent 请求状态数据</div>
                <div><span className="text-blue-400">→</span> 向 王五.Agent 请求状态数据</div>
                <div><span className="text-blue-400">→</span> 向 张三.Agent 请求状态数据</div>
              </div>
              <div><span className="text-emerald-400">你的 Agent</span>.用 LLM 综合分析()</div>
              <div className="pl-4">
                <div><span className="text-yellow-400">→</span> 李四 92% <span className="text-gray-500">"刷了40分钟手机，在家"</span></div>
                <div><span className="text-yellow-400">→</span> 王五 78% <span className="text-gray-500">"周五晚上"</span></div>
                <div><span className="text-yellow-400">→</span> 张三 45% <span className="text-gray-500">"数据不足"</span></div>
              </div>
              <div><span className="text-emerald-400">输出</span>: 按概率排序的有空列表</div>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 bg-gradient-to-br from-emerald-500 to-teal-500">
        <div className="max-w-4xl mx-auto px-6 text-center">
          <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">
            不用再问「在吗」
          </h2>
          <p className="text-xl text-emerald-100 mb-10">
            下载「有空」，一眼看到谁可能有空
          </p>

          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <button
              onClick={handleDownloadiOS}
              className="inline-flex items-center justify-center gap-2 px-8 py-4 bg-white text-gray-900 rounded-xl font-medium hover:bg-gray-100 transition-colors"
            >
              <AppleIcon />
              App Store 下载
            </button>
            <button
              onClick={handleDownloadAndroid}
              className="inline-flex items-center justify-center gap-2 px-8 py-4 bg-white/10 text-white border-2 border-white/30 rounded-xl font-medium hover:bg-white/20 transition-colors"
            >
              <AndroidIcon />
              Google Play 下载
            </button>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 bg-gray-900 text-gray-400">
        <div className="max-w-6xl mx-auto px-6">
          <div className="flex flex-col md:flex-row justify-between items-center gap-6">
            <div className="flex items-center gap-2">
              <span className="text-2xl">⏰</span>
              <span className="text-lg font-medium text-white">有空 YouKong</span>
            </div>

            <div className="flex gap-8 text-sm">
              <a href="/privacy" className="hover:text-white transition-colors">隐私政策</a>
              <a href="/terms" className="hover:text-white transition-colors">服务条款</a>
              <a href="mailto:support@youkong.app" className="hover:text-white transition-colors">联系我们</a>
            </div>

            <p className="text-sm">
              © 2024 YouKong. All rights reserved.
            </p>
          </div>
        </div>
      </footer>
    </div>
  )
}

// 模拟朋友卡片
function FriendCard({ name, probability, reason, color }: {
  name: string
  probability: number
  reason: string
  color: 'very' | 'likely' | 'maybe' | 'busy'
}) {
  const colorMap = {
    very: { dot: '#22C55E', bg: '#F0FDF4' },
    likely: { dot: '#86EFAC', bg: '#F0FDF4' },
    maybe: { dot: '#FACC15', bg: '#FEFCE8' },
    busy: { dot: '#EF4444', bg: '#FEF2F2' },
  }
  const colors = colorMap[color]

  return (
    <div
      className="rounded-xl p-3 flex items-center gap-3"
      style={{ backgroundColor: colors.bg }}
    >
      <div
        className="w-3 h-3 rounded-full flex-shrink-0"
        style={{ backgroundColor: colors.dot }}
      />
      <div className="flex-1 min-w-0">
        <div className="flex justify-between items-center">
          <span className="font-medium text-gray-900 text-sm">{name}</span>
          <span className="text-sm font-medium" style={{ color: colors.dot }}>{probability}%</span>
        </div>
        <p className="text-xs text-gray-500 truncate">{reason}</p>
      </div>
    </div>
  )
}

// 对比卡片
function CompareCard({ type, title, items }: {
  type: 'old' | 'new'
  title: string
  items: string[]
}) {
  const isNew = type === 'new'
  return (
    <div className={`rounded-2xl p-6 ${isNew ? 'bg-emerald-50 border-2 border-emerald-200' : 'bg-white border border-gray-200'}`}>
      <h3 className={`text-lg font-semibold mb-4 ${isNew ? 'text-emerald-700' : 'text-gray-500'}`}>
        {isNew ? '✓ ' : '✗ '}{title}
      </h3>
      <ul className="space-y-3">
        {items.map((item, i) => (
          <li key={i} className={`flex items-start gap-2 ${isNew ? 'text-gray-700' : 'text-gray-500'}`}>
            <span className={isNew ? 'text-emerald-500' : 'text-gray-400'}>{isNew ? '✓' : '✗'}</span>
            <span className="text-sm">{item}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

// 信号卡片
function SignalCard({ emoji, title, description, examples }: {
  emoji: string
  title: string
  description: string
  examples: { signal: string; score: string }[]
}) {
  return (
    <div className="bg-gray-50 rounded-2xl p-6">
      <span className="text-4xl mb-4 block">{emoji}</span>
      <h3 className="text-xl font-semibold text-gray-900 mb-2">{title}</h3>
      <p className="text-gray-600 text-sm mb-4">{description}</p>
      <div className="space-y-2">
        {examples.map((ex, i) => (
          <div key={i} className="flex justify-between text-sm">
            <span className="text-gray-600">{ex.signal}</span>
            <span className={ex.score.startsWith('+') ? 'text-emerald-600 font-medium' : 'text-red-500 font-medium'}>
              {ex.score}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

// 颜色行
function ColorRow({ color, range, label, example }: {
  color: string
  range: string
  label: string
  example: string
}) {
  return (
    <div className="flex items-center gap-4 bg-white rounded-xl p-4">
      <div
        className="w-4 h-4 rounded-full flex-shrink-0"
        style={{ backgroundColor: color }}
      />
      <div className="w-20 text-sm font-medium text-gray-500">{range}</div>
      <div className="w-28 text-sm font-medium text-gray-900">{label}</div>
      <div className="flex-1 text-sm text-gray-500 truncate">{example}</div>
    </div>
  )
}

// 权限卡片
function PermissionCard({ emoji, title, description }: {
  emoji: string
  title: string
  description: string
}) {
  return (
    <div className="bg-gray-50 rounded-2xl p-6 text-center">
      <span className="text-4xl mb-4 block">{emoji}</span>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">{title}</h3>
      <p className="text-gray-600 text-sm">{description}</p>
    </div>
  )
}

// Apple 图标
function AppleIcon() {
  return (
    <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
      <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/>
    </svg>
  )
}

// Android 图标
function AndroidIcon() {
  return (
    <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
      <path d="M17.6 9.48l1.84-3.18c.16-.31.04-.69-.26-.85-.29-.15-.65-.06-.83.22l-1.88 3.24c-1.4-.59-2.98-.92-4.66-.92-1.68 0-3.26.33-4.67.92L5.27 5.67c-.18-.29-.54-.38-.83-.22-.3.16-.42.54-.26.85l1.84 3.18C2.92 11.07.82 14.02.82 17.5h22.36c0-3.48-2.1-6.43-5.58-8.02zM7 15.25a1.25 1.25 0 110-2.5 1.25 1.25 0 010 2.5zm10 0a1.25 1.25 0 110-2.5 1.25 1.25 0 010 2.5z"/>
    </svg>
  )
}

export default LandingPage
