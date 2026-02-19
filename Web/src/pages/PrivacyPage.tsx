const pageStyle: React.CSSProperties = {
  width: '100%',
  minHeight: '100vh',
  backgroundColor: '#0a0a0a',
  color: '#e0e0e0',
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
  lineHeight: 1.8,
}

const containerStyle: React.CSSProperties = {
  maxWidth: 800,
  margin: '0 auto',
  padding: '40px 20px',
}

export function PrivacyPage() {
  return (
    <div style={pageStyle}>
      <div style={containerStyle}>
        <h1 style={{ fontSize: 28, fontWeight: 700, marginBottom: 8, color: '#fff' }}>隐私政策</h1>
        <p style={{ color: '#888', marginBottom: 32, fontSize: 14 }}>最后更新日期：2025年2月19日 &nbsp;|&nbsp; 生效日期：2025年2月19日</p>

        <p>欢迎使用「有空」（以下简称"本应用"或"我们"）。我们深知个人信息对您的重要性，并会尽全力保护您的个人信息安全。本隐私政策适用于您通过本应用所提供的产品与服务。请您在使用本应用前，仔细阅读并充分理解本政策的全部内容。<b>一旦您开始使用本应用，即表示您已阅读并同意本隐私政策。</b></p>

        {/* 一、信息收集 */}
        <Section title="一、我们收集的信息">
          <SubSection title="1.1 您主动提供的信息">
            <p>在您使用本应用的过程中，我们可能需要您提供以下个人信息：</p>
            <ul>
              <li><b>手机号码</b>：用于账号注册、登录验证及账号安全保护。我们通过短信验证码方式验证您的身份，不会将您的手机号用于任何营销目的。</li>
              <li><b>昵称和头像</b>：用于您的个人资料展示和身份识别。您可以自由选择是否设置真实信息。</li>
              <li><b>个人偏好信息</b>：包括性别、职业角色类型、MBTI 性格类型等。这些信息帮助 AI 更准确地分析和预测您的日程状态，您可以选择跳过或随时修改。</li>
              <li><b>日程安排信息</b>：您通过 AI 对话助手主动告知的日程安排内容，用于生成您的个人时间表。</li>
            </ul>
          </SubSection>

          <SubSection title="1.2 设备权限与传感器数据">
            <p>为实现 AI 智能状态感知功能，本应用可能需要申请以下设备权限。<b>所有权限均需您明确授权后才会开启，且您可以随时在系统设置中关闭。</b></p>

            <PermissionBlock
              name="位置权限（前台与后台）"
              purpose="用于判断您所在的位置类型（如家中、办公室、外出等），从而推断您的当前状态。例如：周末在家通常表示比较有空。"
              scope="我们仅判断位置类型（家/公司/外出），不会记录或存储您的精确地理坐标。后台位置权限用于在应用未打开时也能持续感知您的状态变化。"
              refuse="如果您拒绝授权位置权限，AI 将无法基于位置进行状态判断，但不影响手动设置状态等其他功能的使用。"
            />
            <PermissionBlock
              name="日历权限"
              purpose="用于判断您是否有日程安排，从而判断您的忙闲状态。例如：没有会议的时候可能比较有空。"
              scope="我们仅读取日历事件的时间段信息（开始时间和结束时间），不会读取日程的标题、内容、参与者等具体信息。"
              refuse="如果您拒绝授权日历权限，AI 将无法参考日历信息进行分析，但不影响其他功能的使用。"
            />
            <PermissionBlock
              name="运动与健身权限"
              purpose="用于判断您当前的运动状态（静止、步行、跑步、驾车等），从而更准确地推断您的当前活动。"
              scope="我们仅采集运动类型数据，不会记录步数、距离、卡路里等健身数据，也不会追踪您的运动轨迹。"
              refuse="如果您拒绝授权运动权限，AI 将无法感知您的运动状态，但不影响其他功能的使用。"
            />
            <PermissionBlock
              name="通知权限"
              purpose="用于向您发送日程提醒和状态更新等服务通知。"
              scope="我们仅发送与应用功能相关的服务通知，不会发送广告或营销信息。"
              refuse="如果您拒绝授权通知权限，您将无法收到日程提醒，但不影响其他功能的使用。"
            />
          </SubSection>

          <SubSection title="1.3 自动收集的技术信息">
            <p>为保障服务的正常运行和安全，我们可能自动收集以下技术信息：</p>
            <ul>
              <li><b>设备信息</b>：设备型号、操作系统版本、屏幕分辨率等，用于适配不同设备的显示效果。</li>
              <li><b>网络信息</b>：网络连接类型（Wi-Fi/蜂窝），用于优化数据传输。</li>
              <li><b>应用运行信息</b>：崩溃日志、性能数据等，用于修复问题和改善应用稳定性。</li>
            </ul>
          </SubSection>

          <SubSection title="1.4 我们不会收集的信息">
            <p>我们承诺不会收集以下信息：</p>
            <ul>
              <li>您的通讯录/联系人数据</li>
              <li>您的短信、通话记录</li>
              <li>您的照片、视频等媒体文件</li>
              <li>您的浏览器历史记录</li>
              <li>您安装的其他应用列表</li>
              <li>任何未经您授权的传感器数据</li>
            </ul>
          </SubSection>
        </Section>

        {/* 二、信息使用 */}
        <Section title="二、信息的使用目的">
          <p>我们严格限制信息的使用范围，仅将收集的信息用于以下目的：</p>
          <ul>
            <li><b>核心功能</b>：提供 AI 日程状态分析服务，自动感知并展示您的当前状态；生成个性化时间表，帮助您管理每日日程。</li>
            <li><b>服务通知</b>：发送日程提醒、状态更新等与产品功能直接相关的通知。</li>
            <li><b>安全保障</b>：验证用户身份，防止欺诈和滥用行为，保障账号安全。</li>
            <li><b>产品改进</b>：分析匿名化的使用数据以改进产品功能和用户体验。我们不会将您的个人信息用于用户画像或精准营销。</li>
          </ul>
        </Section>

        {/* 三、数据处理与存储 */}
        <Section title="三、数据处理方式与存储">
          <SubSection title="3.1 本地处理优先原则">
            <p>本应用遵循「本地处理优先」的设计原则：</p>
            <ul>
              <li>传感器数据（位置类型、运动状态、日历忙闲）主要在您的设备本地进行采集和预处理。</li>
              <li>仅将 AI 分析所必需的<b>摘要数据</b>（如"在家"、"静止"、"有日程"等标签化信息）上传至服务器。</li>
              <li>原始传感器数据（如 GPS 坐标、加速度计原始数值）不会上传至服务器。</li>
            </ul>
          </SubSection>
          <SubSection title="3.2 数据存储">
            <ul>
              <li><b>存储位置</b>：服务器数据存储在中国大陆境内的云服务器上。</li>
              <li><b>存储期限</b>：账号信息在您的账号存续期间保留；传感器分析摘要数据保留不超过 90 天；您注销账号后，我们将在 15 个工作日内删除您的所有个人数据。</li>
              <li><b>加密保护</b>：所有网络传输均采用 HTTPS 加密；敏感信息（如认证令牌）在设备端使用 Keychain（iOS）安全存储。</li>
            </ul>
          </SubSection>
          <SubSection title="3.3 数据安全措施">
            <ul>
              <li>采用行业标准的加密技术保护数据传输和存储安全。</li>
              <li>实施严格的访问控制策略，仅授权人员可访问服务器数据。</li>
              <li>定期进行安全审计和漏洞检测。</li>
              <li>采用最小权限原则，每个系统组件仅可访问其所需的最少数据。</li>
            </ul>
          </SubSection>
        </Section>

        {/* 四、信息共享 */}
        <Section title="四、信息的共享与披露">
          <p><b>我们不会将您的个人信息出售、交易或出租给任何第三方。</b></p>
          <p style={{ marginTop: 12 }}>仅在以下情况下，我们可能会共享您的信息：</p>
          <ul>
            <li><b>获得您的明确同意</b>：在获得您的明确授权后，我们可能与第三方共享您指定的信息。</li>
            <li><b>法律要求</b>：根据法律法规的规定、诉讼或仲裁需要，或应行政、司法机关的合法要求。</li>
            <li><b>安全需要</b>：为保护本应用、其他用户或公众的人身财产安全所合理必需。</li>
          </ul>
          <p style={{ marginTop: 12 }}>我们目前不使用任何第三方数据分析或广告 SDK。如未来引入，将提前更新本隐私政策并告知您。</p>
        </Section>

        {/* 五、用户权利 */}
        <Section title="五、您的权利">
          <p>根据适用法律法规，您对自己的个人信息享有以下权利：</p>
          <ul>
            <li><b>知情权</b>：您有权了解我们如何收集、使用您的个人信息，本隐私政策即为此目的。</li>
            <li><b>访问权</b>：您可以随时在应用内查看您的个人资料和日程数据。</li>
            <li><b>更正权</b>：您可以随时在应用内修改您的昵称、头像、个人偏好等信息。</li>
            <li><b>删除权</b>：您可以申请删除您的账号及所有相关数据。</li>
            <li><b>撤回同意权</b>：您可以随时在手机「设置」中关闭位置、日历、运动传感器等权限授权。撤回同意不影响撤回前基于同意所进行的数据处理的合法性。</li>
            <li><b>账号注销</b>：您可以发送邮件至 <b>support@playa.cn</b> 申请注销账号。我们将在 15 个工作日内完成账号注销并彻底删除您的所有个人数据，该操作不可恢复。</li>
            <li><b>数据可携带权</b>：您可以联系我们申请导出您的个人数据副本。</li>
          </ul>
        </Section>

        {/* 六、第三方服务 */}
        <Section title="六、第三方服务">
          <p>本应用可能使用以下第三方服务：</p>
          <ul>
            <li><b>推送通知服务</b>：用于发送服务通知，可能收集设备标识符用于消息推送。</li>
            <li><b>云存储服务</b>：用于存储用户上传的头像等媒体文件。</li>
          </ul>
          <p style={{ marginTop: 12 }}>我们会审慎选择合作方，并要求其遵守严格的数据保护标准。第三方服务的隐私政策不受本政策约束，建议您查阅相关第三方的隐私政策。</p>
        </Section>

        {/* 七、未成年人保护 */}
        <Section title="七、未成年人保护">
          <p>本应用面向年满 16 周岁的用户。我们不会故意收集未满 16 周岁未成年人的个人信息。如果您是未成年人的监护人，发现您所监护的未成年人向我们提供了个人信息，请联系我们（support@playa.cn），我们将在确认后尽快删除相关信息。</p>
        </Section>

        {/* 八、政策更新 */}
        <Section title="八、隐私政策的变更">
          <p>我们可能会不时更新本隐私政策。更新后的政策将在本页面发布并标注新的生效日期。对于重大变更（如信息收集范围的扩大、使用目的的变化等），我们将通过应用内弹窗或推送通知的方式提前告知您，并在必要时重新征得您的同意。</p>
          <p style={{ marginTop: 8 }}>建议您定期查看本政策，以便及时了解我们对个人信息保护所采取的措施。</p>
        </Section>

        {/* 九、适用法律 */}
        <Section title="九、适用法律与争议解决">
          <p>本隐私政策的制定、执行和解释均适用中华人民共和国法律法规，包括但不限于《中华人民共和国个人信息保护法》《中华人民共和国网络安全法》《中华人民共和国数据安全法》等。因本政策产生的争议，双方应友好协商解决；协商不成的，任何一方可向本应用运营者所在地有管辖权的人民法院提起诉讼。</p>
        </Section>

        {/* 十、联系我们 */}
        <Section title="十、联系我们">
          <p>如果您对本隐私政策有任何疑问、意见或投诉，或希望行使您的个人信息权利，请通过以下方式联系我们：</p>
          <div style={{ backgroundColor: '#111', borderRadius: 8, padding: 20, marginTop: 12 }}>
            <p><b>邮箱</b>：support@playa.cn</p>
            <p><b>响应时间</b>：我们将在收到您的请求后 15 个工作日内予以回复。</p>
          </div>
        </Section>

        <div style={{ marginTop: 48, paddingTop: 24, borderTop: '1px solid #222', color: '#666', fontSize: 13 }}>
          <p>© {new Date().getFullYear()} 有空 YouKong. All rights reserved.</p>
        </div>
      </div>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ marginTop: 36 }}>
      <h2 style={{ fontSize: 20, fontWeight: 600, marginBottom: 12, color: '#fff' }}>{title}</h2>
      {children}
    </div>
  )
}

function SubSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ marginTop: 20 }}>
      <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 8, color: '#ddd' }}>{title}</h3>
      {children}
    </div>
  )
}

function PermissionBlock({ name, purpose, scope, refuse }: { name: string; purpose: string; scope: string; refuse: string }) {
  return (
    <div style={{ backgroundColor: '#111', borderRadius: 8, padding: 16, marginTop: 12, marginBottom: 12, borderLeft: '3px solid #10B981' }}>
      <p style={{ fontWeight: 600, color: '#10B981', marginBottom: 8 }}>{name}</p>
      <p style={{ marginBottom: 4 }}><b>使用目的</b>：{purpose}</p>
      <p style={{ marginBottom: 4 }}><b>收集范围</b>：{scope}</p>
      <p style={{ color: '#aaa' }}><b>拒绝授权的影响</b>：{refuse}</p>
    </div>
  )
}
