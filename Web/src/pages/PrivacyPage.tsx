export function PrivacyPage() {
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
      <h1 style={{ fontSize: 28, fontWeight: 700, marginBottom: 8, color: '#fff' }}>隐私政策</h1>
      <p style={{ color: '#888', marginBottom: 32, fontSize: 14 }}>最后更新日期：2025年2月19日</p>

      <p>欢迎使用「有空」（以下简称"本应用"）。我们非常重视您的隐私保护，本隐私政策旨在向您说明我们如何收集、使用、存储和保护您的个人信息。请您在使用本应用前仔细阅读本政策。</p>

      <Section title="一、我们收集的信息">
        <SubSection title="1.1 您主动提供的信息">
          <ul>
            <li><b>手机号码</b>：用于账号注册和登录验证。</li>
            <li><b>昵称和头像</b>：用于个人资料展示。</li>
            <li><b>个人偏好信息</b>：如性别、角色类型、MBTI 等，用于 AI 更准确地分析您的日程状态。</li>
          </ul>
        </SubSection>
        <SubSection title="1.2 自动收集的信息">
          <p>经您明确授权后，本应用可能收集以下设备传感器数据，用于 AI 日程状态分析：</p>
          <ul>
            <li><b>位置信息</b>：用于判断您所在的位置类型（如家中、办公室、外出），不会记录精确地理坐标。</li>
            <li><b>日历信息</b>：用于判断您是否有日程安排，不会读取日程的具体内容。</li>
            <li><b>运动传感器数据</b>：用于判断您是否在移动，不会追踪运动轨迹。</li>
          </ul>
        </SubSection>
        <SubSection title="1.3 我们不收集的信息">
          <ul>
            <li>我们不会收集您的通讯录内容。</li>
            <li>我们不会收集您的短信、通话记录。</li>
            <li>我们不会收集您的照片、视频等媒体文件。</li>
            <li>我们不会在您未授权的情况下收集任何传感器数据。</li>
          </ul>
        </SubSection>
      </Section>

      <Section title="二、信息的使用">
        <p>我们收集的信息仅用于以下目的：</p>
        <ul>
          <li>提供 AI 日程状态分析服务，自动感知并展示您的当前状态。</li>
          <li>生成个性化时间表，帮助您管理每日日程。</li>
          <li>发送服务通知（如日程提醒）。</li>
          <li>改进产品功能和用户体验。</li>
        </ul>
      </Section>

      <Section title="三、信息的存储与保护">
        <ul>
          <li><b>本地处理</b>：传感器数据（位置、运动、日历）主要在您的设备本地进行处理和分析。</li>
          <li><b>加密传输</b>：所有网络传输的数据均采用加密方式保护。</li>
          <li><b>安全存储</b>：您的账号信息存储在受保护的服务器上，采用行业标准的安全措施。</li>
          <li><b>最小化原则</b>：我们仅上传 AI 分析所必需的摘要数据，不上传原始传感器数据。</li>
        </ul>
      </Section>

      <Section title="四、信息的共享">
        <p>我们不会将您的个人信息出售、出租或以其他方式提供给第三方，除非：</p>
        <ul>
          <li>获得您的明确同意。</li>
          <li>法律法规要求或政府部门依法要求。</li>
          <li>为保护本应用、用户或公众的权益所必需。</li>
        </ul>
      </Section>

      <Section title="五、您的权利">
        <p>您对自己的个人信息享有以下权利：</p>
        <ul>
          <li><b>查看和修改</b>：您可以随时在应用内查看和修改您的个人资料。</li>
          <li><b>权限管理</b>：您可以随时在系统设置中开启或关闭位置、日历、运动等权限。</li>
          <li><b>账号注销</b>：您可以联系我们申请注销账号，我们将在 15 个工作日内完成处理并删除您的所有数据。</li>
          <li><b>数据导出</b>：您可以联系我们申请导出您的个人数据。</li>
        </ul>
      </Section>

      <Section title="六、未成年人保护">
        <p>本应用面向年满 16 周岁的用户。我们不会故意收集未满 16 周岁未成年人的个人信息。如果我们发现无意中收集了未成年人的信息，将立即采取措施删除相关数据。</p>
      </Section>

      <Section title="七、隐私政策的更新">
        <p>我们可能会不时更新本隐私政策。更新后的政策将在本页面发布，重大变更会通过应用内通知告知您。建议您定期查看本政策以了解最新信息。</p>
      </Section>

      <Section title="八、联系我们">
        <p>如果您对本隐私政策有任何疑问、意见或建议，请通过以下方式联系我们：</p>
        <ul>
          <li>邮箱：support@playa.cn</li>
        </ul>
      </Section>

      <div style={{ marginTop: 48, paddingTop: 24, borderTop: '1px solid #222', color: '#666', fontSize: 13 }}>
        <p>© {new Date().getFullYear()} 有空 YouKong. All rights reserved.</p>
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

function SubSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ marginTop: 16 }}>
      <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 8, color: '#ddd' }}>{title}</h3>
      {children}
    </div>
  )
}
