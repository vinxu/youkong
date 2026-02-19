export function SupportPage() {
  return (
    <div style={{
      maxWidth: 800,
      margin: '0 auto',
      padding: '40px 20px',
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
      color: '#e0e0e0',
      backgroundColor: '#0a0a0a',
      minHeight: '100vh',
      lineHeight: 1.8,
    }}>
      <h1 style={{ fontSize: 28, fontWeight: 700, marginBottom: 8, color: '#fff' }}>技术支持</h1>
      <p style={{ color: '#888', marginBottom: 32, fontSize: 14 }}>有空 YouKong - 智能日程状态管理工具</p>

      <Section title="常见问题">
        <QA
          q="如何设置我的日程状态？"
          a="打开应用后，AI 会根据您的设备传感器数据自动判断您的当前状态。您也可以点击状态卡片手动修改，或通过 AI 对话助手告诉它您的日程安排。"
        />
        <QA
          q="如何管理权限设置？"
          a="进入手机「设置」→ 找到「有空」→ 可以分别管理位置、日历、运动传感器等权限。开启更多权限可以让 AI 的状态判断更加准确。"
        />
        <QA
          q="我的数据安全吗？"
          a="我们非常重视数据安全。传感器数据主要在您的设备本地处理，仅上传必要的分析摘要。所有网络传输均采用加密保护。详细信息请查阅我们的隐私政策。"
        />
        <QA
          q="如何注销我的账号？"
          a="如需注销账号，请发送邮件至 support@playa.cn，标题注明「账号注销申请」并提供您的注册手机号。我们将在 15 个工作日内完成处理并删除您的所有数据。"
        />
        <QA
          q="应用支持哪些设备？"
          a="本应用支持 iOS 16.0 及以上版本的 iPhone 设备。"
        />
        <QA
          q="为什么 AI 的状态判断不准确？"
          a="AI 状态分析的准确性取决于您授权的权限。建议开启位置、日历和运动传感器权限，让 AI 获得更多维度的数据来判断您的状态。同时，AI 会随着使用时间的增长不断学习您的习惯，变得越来越准确。"
        />
      </Section>

      <Section title="联系我们">
        <div style={{
          backgroundColor: '#111',
          borderRadius: 8,
          padding: 24,
          marginTop: 12,
        }}>
          <p style={{ marginBottom: 16 }}>如果您遇到任何问题或有任何建议，欢迎通过以下方式联系我们：</p>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <ContactItem label="客服邮箱" value="support@playa.cn" />
            <ContactItem label="工作时间" value="周一至周五 9:00 - 18:00" />
            <ContactItem label="响应时间" value="通常在 1-2 个工作日内回复" />
          </div>
        </div>
      </Section>

      <Section title="反馈与建议">
        <p>我们持续改进产品体验，您的反馈对我们非常重要。如有功能建议或使用体验反馈，请发送邮件至 support@playa.cn，标题注明「产品反馈」。</p>
      </Section>

      <div style={{ marginTop: 48, paddingTop: 24, borderTop: '1px solid #222', color: '#666', fontSize: 13 }}>
        <p>© {new Date().getFullYear()} 有空 YouKong. All rights reserved.</p>
        <p style={{ marginTop: 4 }}>
          <a href="/privacy" style={{ color: '#10B981', textDecoration: 'none' }}>隐私政策</a>
        </p>
      </div>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ marginTop: 32 }}>
      <h2 style={{ fontSize: 20, fontWeight: 600, marginBottom: 12, color: '#fff' }}>{title}</h2>
      {children}
    </div>
  )
}

function QA({ q, a }: { q: string; a: string }) {
  return (
    <div style={{ marginBottom: 20 }}>
      <p style={{ fontWeight: 600, color: '#10B981', marginBottom: 4 }}>Q: {q}</p>
      <p style={{ color: '#ccc', paddingLeft: 20 }}>A: {a}</p>
    </div>
  )
}

function ContactItem({ label, value }: { label: string; value: string }) {
  return (
    <div style={{ display: 'flex', gap: 12 }}>
      <span style={{ color: '#888', minWidth: 80 }}>{label}</span>
      <span style={{ color: '#fff' }}>{value}</span>
    </div>
  )
}
