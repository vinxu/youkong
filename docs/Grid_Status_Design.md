# 有空 App - 实时状态宫格产品重构方案

## 产品定位

**核心理念**：通过 AI 推测你和你的朋友们**此刻**在做什么的工具

**关键特性**：
- 实时状态推测（不是未来空闲时段）
- 宫格布局（1-9 宫格，自适应）
- 状态 = 名字 + Emoji + 文字描述
- 隐私优先：只显示推测的状态和心情，不透露任何隐私信息
- 极简交互：首页只有两个按钮（更新状态、分享海报）

---

## 用户体验流程

### 1. 首次引导（Onboarding）

**不要马上请求权限**，先告诉用户这是什么：

```
第1屏：欢迎
━━━━━━━━━━━━━━━━
    有空 Logo

    用 AI 看透朋友们
    此刻在做什么

    [开始了解] 按钮
━━━━━━━━━━━━━━━━

第2屏：产品介绍
━━━━━━━━━━━━━━━━
    📱 实时状态
    AI 分析你的行为模式
    推测你此刻的状态和心情

    👥 朋友宫格
    一屏看清所有朋友
    谁在忙、谁有空、谁在路上

    [下一步] 按钮
━━━━━━━━━━━━━━━━

第3屏：隐私承诺
━━━━━━━━━━━━━━━━
    🔒 隐私至上

    ✅ 只显示推测的状态（如"工作中"）
    ✅ 不显示具体位置、日程内容
    ✅ 数据加密存储，仅你和好友可见
    ❌ 绝不出售数据，绝不广告追踪

    [我理解了] 按钮
━━━━━━━━━━━━━━━━

第4屏：权限请求（通过对话）
━━━━━━━━━━━━━━━━
    Agent 对话框：

    "为了更懂你，我需要一些帮助：

    1️⃣ 位置权限 - 判断你在家/公司/路上
    2️⃣ 日历权限 - 知道你现在有没有会议
    3️⃣ 计步权限 - 了解你是在运动还是静止

    这些都只用于推测状态，
    不会记录具体内容。

    现在授权吗？"

    [授权] [稍后] 按钮
━━━━━━━━━━━━━━━━
```

**关键设计原则**：
- 先讲清楚产品价值
- 再强调隐私保护
- 最后才请求权限（并解释为什么需要）
- 允许"稍后"（不强制）

### 2. 主界面（宫格布局）

**一屏展示所有好友**，不支持滚动：

```
━━━━━━━━━━━━━━━━━━━━━━━━
  有空            [设置图标]

┌─────────┬─────────┬─────────┐
│  🏃 我  │ 😴 小明  │ 💼 小红  │
│ 晨跑中  │  睡觉中  │  工作中  │
│ 2分钟前 │ 10分钟前 │ 刚刚    │
├─────────┼─────────┼─────────┤
│ 🍜 小李  │ 🚗 小张  │ 🎵 小王  │
│ 午饭中  │  通勤中  │  放松中  │
│ 5分钟前 │ 3分钟前  │ 15分钟前│
├─────────┼─────────┼─────────┤
│ 📚 小刘  │ 🏠 小陈  │ ❓ 小赵  │
│ 学习中  │  在家中  │  未知   │
│ 1小时前 │ 30分钟前 │ 3小时前 │
└─────────┴─────────┴─────────┘

    [🔄 更新状态]  [📤 分享]

━━━━━━━━━━━━━━━━━━━━━━━━
```

**宫格适配规则**：
- 1 个好友（包括自己）：1x1
- 2-4 个好友：2x2
- 5-9 个好友：3x3
- 10-16 个好友：4x4
- 超过 16 个：只显示最近更新的 16 个

**每个卡片包含**：
- Emoji（表情符号，由 AI 生成）
- 名字（昵称）
- 状态描述（≤10 字，如"工作中"、"运动中"）
- 更新时间（相对时间，如"刚刚"、"5分钟前"）

**点击卡片**：
- 弹出详情对话框
- 显示更详细的状态推理过程（可选）
- 可以发起聊天

### 3. 更新状态（Agent 分析）

**点击"更新状态"按钮**：

```
━━━━━━━━━━━━━━━━━━━━━━━━
  正在分析你的状态...

  [━━━━━━━━━━░░░░] 75%

  ✓ 位置数据已采集
  ✓ 日历数据已读取
  ✓ 运动数据已分析
  ⏳ AI 正在推理...

━━━━━━━━━━━━━━━━━━━━━━━━
```

**分析完成后**：

```
━━━━━━━━━━━━━━━━━━━━━━━━
  Agent: 我的推测

  🏃 你现在可能在晨跑

  推理过程：
  • 位置：户外移动中
  • 时间：早上 7:30
  • 步数：每分钟 120 步
  • 日历：无会议安排

  准确吗？

  [✓ 是的]  [✗ 不对，我在...]
━━━━━━━━━━━━━━━━━━━━━━━━
```

**用户可以修正**：
- 如果不准，允许手动输入/选择状态
- AI 会学习（保存到 MemoryService）

**发布成功**：
- 自动刷新首页宫格
- 显示新状态

### 4. 分享海报

**点击"分享"按钮**：

**生成海报**：
```
━━━━━━━━━━━━━━━━━━━━━━━━
┌────────────────────────┐
│                        │
│  当前时间：2026-02-01 18:30 │
│                        │
│  ┌──────┬──────┬──────┐ │
│  │ 🏃我 │😴小明│💼小红│ │
│  │晨跑中│睡觉中│工作中│ │
│  ├──────┼──────┼──────┤ │
│  │🍜小李│🚗小张│🎵小王│ │
│  │午饭中│通勤中│放松中│ │
│  └──────┴──────┴──────┘ │
│                        │
├────────────────────────┤
│ 有空 Logo  Slogan      QR│
│ "看透朋友此刻状态"     码│
└────────────────────────┘

   [保存到相册]  [分享到微信]
━━━━━━━━━━━━━━━━━━━━━━━━
```

**海报设计要求**：
- 上方：当前时间戳
- 中间：宫格截图（当前状态）
- 下方白色条：
  - 左侧：有空 Logo + Slogan
  - 右侧：邀请二维码

---

## 与现有功能的关系

### 保留的功能

1. **Agent 推理系统**（核心）
   - Holmes 分析引擎
   - 多维度数据收集（位置、日历、运动、屏幕使用）
   - LLM 生成状态描述
   - 保留完整的推理逻辑

2. **好友系统**
   - 邀请码机制
   - 圈子（可见性控制）
   - 好友列表

3. **消息系统**（简化）
   - 点击宫格卡片 → 进入聊天
   - 保留消息收发基础设施

4. **推送通知**
   - 好友状态变化时推送
   - 新消息推送

### 移除的功能

1. ❌ **Availability（有空发布）**
   - 不再需要"发布有空时段"
   - 不再需要时间选择、地点选择
   - 不再需要预约系统

2. ❌ **日历助手**
   - 不分析未来 14 天
   - 日历只用于判断"当前是否有会议"

3. ❌ **三步发布流程**
   - 删除 CreateAvailabilityViewModel
   - 删除时间/地点/圈子选择 UI

### 重构的功能

1. **首页**
   - 从"好友列表"改为"宫格布局"
   - 只显示最新状态，不显示有空时段

2. **Agent 数据**
   - 从"显示推理过程"改为"实时状态卡片"
   - 突出 Emoji + 状态描述

3. **分享**
   - 从"邀请海报"改为"状态宫格海报"

---

## 数据模型变更

### 1. 简化 Availability（或废弃）

**选项 A**：完全废弃 Availability
- 删除 `availabilities` 表
- 删除 `availability_circles` 表
- 用 `user_analysis_cache` 替代

**选项 B**：复用 Availability 存储当前状态
```sql
-- 简化为"单时刻快照"
ALTER TABLE availabilities
DROP COLUMN location_type,
DROP COLUMN location_name,
DROP COLUMN latitude,
DROP COLUMN longitude,
ADD COLUMN is_current_status BOOLEAN DEFAULT FALSE,
ADD INDEX idx_current_status (user_id, is_current_status);

-- 每个用户只有一条 is_current_status = TRUE 的记录
-- 更新时覆盖旧记录
```

**推荐**：选项 A - 完全废弃，简化架构

### 2. 扩展 user_analysis_cache

**现有结构**（已够用）：
```sql
user_analysis_cache (
    user_id VARCHAR(36) PRIMARY KEY,
    availability_status ENUM('available', 'busy', 'maybe'),
    availability_probability INT,
    availability_reason VARCHAR(255),
    life_status_emoji VARCHAR(10),
    life_status_label VARCHAR(100),
    life_status_description TEXT,
    updated_at TIMESTAMP
)
```

**无需修改**，已包含所需字段：
- `life_status_emoji` → 宫格 Emoji
- `life_status_label` → 宫格状态文字
- `updated_at` → "X分钟前"

### 3. 新增表：grid_layout_settings（可选）

```sql
CREATE TABLE grid_layout_settings (
    user_id VARCHAR(36) PRIMARY KEY,
    friend_order JSON,  -- ["user1", "user2", ...] 自定义排序
    max_displayed INT DEFAULT 16,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

**用途**：记住用户自定义的好友排序。

---

## 后端 API 变更

### 移除的接口

```
DELETE /api/v1/availabilities/*       # 所有 Availability 相关接口
DELETE /api/v1/appointments/*         # 预约系统（未实现，无需操作）
```

### 保留的接口

```
✅ POST   /api/v1/agent/status        # 上报状态
✅ POST   /api/v1/agent/status/stream # 流式推理（用于"更新状态"按钮）
✅ GET    /api/v1/agent/my-analysis   # 获取自己的状态
✅ GET    /api/v1/friends/free-probability  # 获取好友状态列表
✅ GET    /api/v1/friends              # 好友列表
✅ GET    /api/v1/conversations        # 消息列表
✅ POST   /api/v1/invitations          # 邀请码
```

### 新增的接口

#### 1. 获取宫格数据

```
GET /api/v1/home/grid
```

**响应**：
```json
{
  "code": 0,
  "data": {
    "grid_size": 9,  // 3x3
    "friends": [
      {
        "user_id": "me",
        "nickname": "我",
        "emoji": "🏃",
        "status": "晨跑中",
        "updated_at": "2026-02-01T07:30:00Z",
        "relative_time": "2分钟前"
      },
      {
        "user_id": "user2",
        "nickname": "小明",
        "emoji": "😴",
        "status": "睡觉中",
        "updated_at": "2026-02-01T07:20:00Z",
        "relative_time": "10分钟前"
      }
      // ... 最多 16 个
    ]
  }
}
```

**逻辑**：
1. 获取用户的所有好友
2. 获取每个好友的最新 `user_analysis_cache`
3. 按 `updated_at` 降序排序
4. 取前 16 个
5. 计算宫格大小（1, 4, 9, 16）

#### 2. 生成分享海报

```
POST /api/v1/home/poster
```

**请求**：
```json
{
  "user_ids": ["me", "user2", "user3"],  // 包含哪些好友
  "timestamp": "2026-02-01T18:30:00Z"
}
```

**响应**：
```json
{
  "code": 0,
  "data": {
    "poster_url": "https://cdn.youkong.app/posters/uuid.png"
  }
}
```

**生成逻辑**：
1. 获取指定用户的状态
2. 渲染宫格图片
3. 添加时间戳 + Logo + 二维码
4. 上传到 COS
5. 返回 URL

---

## 前端实现

### iOS 重构

#### 1. 删除的页面/组件

```
❌ iOS/YouKong/Presentation/Screens/Availability/*
❌ iOS/YouKong/Presentation/Screens/CalendarAssistant/*  (如果已创建)
```

#### 2. 新增 OnboardingView（引导页）

**文件**：`iOS/YouKong/Presentation/Screens/Onboarding/OnboardingView.swift`

```swift
struct OnboardingView: View {
    @State private var currentPage = 0

    var body: some View {
        TabView(selection: $currentPage) {
            WelcomeScreen().tag(0)
            ProductIntroScreen().tag(1)
            PrivacyPromiseScreen().tag(2)
            PermissionRequestScreen().tag(3)
        }
        .tabViewStyle(.page)
    }
}

struct PermissionRequestScreen: View {
    @ObservedObject var permissionManager: PermissionManager

    var body: some View {
        VStack {
            Text("Agent 对话框：")
            // ... Agent 样式的对话气泡

            Button("授权") {
                Task {
                    await permissionManager.requestAllPermissions()
                }
            }

            Button("稍后") {
                // 跳过，进入主界面
            }
        }
    }
}
```

#### 3. 重构 FriendsListView → GridHomeView

**文件**：`iOS/YouKong/Presentation/Screens/Home/GridHomeView.swift`（新建）

```swift
struct GridHomeView: View {
    @StateObject private var viewModel = GridHomeViewModel()

    var body: some View {
        VStack {
            // Header
            HStack {
                Text("有空")
                    .font(.largeTitle)
                Spacer()
                Button {
                    // 设置
                } label: {
                    Image(systemName: "gearshape")
                }
            }
            .padding()

            // Grid
            GridLayout(friends: viewModel.friends)

            Spacer()

            // Bottom Buttons
            HStack(spacing: 20) {
                Button {
                    Task {
                        await viewModel.updateStatus()
                    }
                } label: {
                    Label("更新状态", systemImage: "arrow.clockwise")
                }
                .buttonStyle(.borderedProminent)

                Button {
                    viewModel.showPosterSheet = true
                } label: {
                    Label("分享", systemImage: "square.and.arrow.up")
                }
                .buttonStyle(.bordered)
            }
            .padding()
        }
        .sheet(isPresented: $viewModel.showPosterSheet) {
            PosterShareView(friends: viewModel.friends)
        }
    }
}

struct GridLayout: View {
    let friends: [FriendStatus]

    var body: some View {
        let gridSize = calculateGridSize(count: friends.count)

        LazyVGrid(columns: Array(repeating: GridItem(.flexible()), count: gridSize)) {
            ForEach(friends) { friend in
                FriendCard(friend: friend)
            }
        }
        .padding()
    }

    func calculateGridSize(count: Int) -> Int {
        if count <= 1 { return 1 }
        if count <= 4 { return 2 }
        if count <= 9 { return 3 }
        return 4
    }
}

struct FriendCard: View {
    let friend: FriendStatus

    var body: some View {
        VStack(spacing: 4) {
            Text(friend.emoji)
                .font(.system(size: 40))

            Text(friend.nickname)
                .font(.caption)
                .fontWeight(.bold)

            Text(friend.status)
                .font(.caption2)
                .foregroundColor(.secondary)

            Text(friend.relativeTime)
                .font(.caption2)
                .foregroundColor(.gray)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(.systemGray6))
        .cornerRadius(12)
    }
}
```

#### 4. GridHomeViewModel

**文件**：`iOS/YouKong/Presentation/Screens/Home/GridHomeViewModel.swift`

```swift
@MainActor
class GridHomeViewModel: ObservableObject {
    @Published var friends: [FriendStatus] = []
    @Published var isLoading = false
    @Published var showPosterSheet = false

    @Injected(\.agentRepository) private var agentRepository

    func loadGrid() async {
        isLoading = true
        do {
            friends = try await agentRepository.getGridData()
        } catch {
            // 错误处理
        }
        isLoading = false
    }

    func updateStatus() async {
        // 触发 Agent 分析
        // 使用现有的 reportStatus 逻辑
    }
}

struct FriendStatus: Identifiable {
    let id: String  // user_id
    let nickname: String
    let emoji: String
    let status: String
    let updatedAt: Date
    var relativeTime: String {
        updatedAt.relativeTimeText()
    }
}
```

#### 5. 海报分享

**文件**：`iOS/YouKong/Presentation/Screens/Home/PosterShareView.swift`

```swift
struct PosterShareView: View {
    let friends: [FriendStatus]
    @State private var posterImage: UIImage?

    var body: some View {
        VStack {
            if let image = posterImage {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFit()

                HStack {
                    ShareLink(item: image, preview: SharePreview("有空状态", image: image)) {
                        Label("分享", systemImage: "square.and.arrow.up")
                    }

                    Button("保存到相册") {
                        UIImageWriteToSavedPhotosAlbum(image, nil, nil, nil)
                    }
                }
            } else {
                ProgressView("生成海报中...")
            }
        }
        .task {
            posterImage = await generatePoster()
        }
    }

    func generatePoster() async -> UIImage? {
        // 调用后端 API 生成海报
        // 或本地渲染（使用 SwiftUI snapshot）
    }
}
```

### Android 重构

#### 1. 删除的模块

```
❌ Android/feature/feature-availability/
❌ Android/feature/feature-calendar-assistant/  (如果已创建)
```

#### 2. 新增 OnboardingScreen

**文件**：`Android/feature/feature-onboarding/src/main/java/.../OnboardingScreen.kt`

```kotlin
@Composable
fun OnboardingScreen(
    onComplete: () -> Unit
) {
    val pagerState = rememberPagerState { 4 }

    HorizontalPager(state = pagerState) { page ->
        when (page) {
            0 -> WelcomeScreen()
            1 -> ProductIntroScreen()
            2 -> PrivacyPromiseScreen()
            3 -> PermissionRequestScreen(onComplete)
        }
    }
}

@Composable
fun PermissionRequestScreen(onComplete: () -> Unit) {
    val permissionManager = hiltViewModel<PermissionViewModel>()

    Column {
        Text("Agent 对话框：")
        // ... Agent 样式对话

        Button(onClick = {
            permissionManager.requestAllPermissions()
            onComplete()
        }) {
            Text("授权")
        }

        TextButton(onClick = onComplete) {
            Text("稍后")
        }
    }
}
```

#### 3. 重构 FriendsListScreen → GridHomeScreen

**文件**：`Android/feature/feature-home/src/main/java/.../GridHomeScreen.kt`

```kotlin
@Composable
fun GridHomeScreen(
    viewModel: GridHomeViewModel = hiltViewModel()
) {
    val friends by viewModel.friends.collectAsState()
    val isLoading by viewModel.isLoading.collectAsState()

    Column(modifier = Modifier.fillMaxSize()) {
        // Header
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Text(
                text = "有空",
                style = MaterialTheme.typography.headlineLarge
            )
            IconButton(onClick = { /* 设置 */ }) {
                Icon(Icons.Default.Settings, "设置")
            }
        }

        // Grid
        FriendGrid(
            friends = friends,
            modifier = Modifier
                .weight(1f)
                .padding(16.dp)
        )

        // Bottom Buttons
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Button(
                onClick = { viewModel.updateStatus() },
                modifier = Modifier.weight(1f)
            ) {
                Icon(Icons.Default.Refresh, null)
                Spacer(Modifier.width(8.dp))
                Text("更新状态")
            }

            OutlinedButton(
                onClick = { viewModel.showPosterDialog = true },
                modifier = Modifier.weight(1f)
            ) {
                Icon(Icons.Default.Share, null)
                Spacer(Modifier.width(8.dp))
                Text("分享")
            }
        }
    }
}

@Composable
fun FriendGrid(
    friends: List<FriendStatus>,
    modifier: Modifier = Modifier
) {
    val gridSize = when {
        friends.size <= 1 -> 1
        friends.size <= 4 -> 2
        friends.size <= 9 -> 3
        else -> 4
    }

    LazyVerticalGrid(
        columns = GridCells.Fixed(gridSize),
        modifier = modifier
    ) {
        items(friends) { friend ->
            FriendCard(friend = friend)
        }
    }
}

@Composable
fun FriendCard(friend: FriendStatus) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(1f)
            .padding(4.dp)
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(8.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center
        ) {
            Text(
                text = friend.emoji,
                style = MaterialTheme.typography.displayMedium
            )
            Text(
                text = friend.nickname,
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = friend.status,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.secondary
            )
            Text(
                text = friend.relativeTime,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.outline
            )
        }
    }
}
```

#### 4. GridHomeViewModel

**文件**：`Android/feature/feature-home/src/main/java/.../GridHomeViewModel.kt`

```kotlin
@HiltViewModel
class GridHomeViewModel @Inject constructor(
    private val agentRepository: AgentRepository,
    private val friendRepository: FriendRepository
) : ViewModel() {

    private val _friends = MutableStateFlow<List<FriendStatus>>(emptyList())
    val friends: StateFlow<List<FriendStatus>> = _friends

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading

    var showPosterDialog by mutableStateOf(false)

    init {
        loadGrid()
    }

    fun loadGrid() {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                _friends.value = agentRepository.getGridData()
            } catch (e: Exception) {
                // 错误处理
            } finally {
                _isLoading.value = false
            }
        }
    }

    fun updateStatus() {
        viewModelScope.launch {
            // 触发 Agent 分析（复用现有逻辑）
            agentRepository.reportStatus()
            loadGrid()  // 刷新
        }
    }
}

data class FriendStatus(
    val userId: String,
    val nickname: String,
    val emoji: String,
    val status: String,
    val updatedAt: String,
    val relativeTime: String
)
```

---

## 实施步骤

### Phase 1: 数据清理与 API 调整（第 1 周）

**优先级**：🔥 最高

**后端**：
- [ ] 确认废弃 Availability 功能
  - [ ] 备份 `availabilities` 表数据（如果需要）
  - [ ] 删除相关代码：
    - [ ] `handler/availability.go`
    - [ ] `service/availability_service.go`
    - [ ] `repository/availability_repo.go`
  - [ ] 删除路由注册
- [ ] 新增 `/api/v1/home/grid` 接口
  - [ ] 在 `handler/home_handler.go`（新建）
  - [ ] 查询好友 + user_analysis_cache
  - [ ] 返回宫格数据
- [ ] 新增 `/api/v1/home/poster` 接口
  - [ ] 生成海报（调用 poster.Generator 或新实现）

**验证**：
```bash
curl http://localhost:8080/api/v1/home/grid \
  -H "Authorization: Bearer $TOKEN" | jq

# 应返回宫格数据
```

### Phase 2: iOS 重构（第 2-3 周）

**优先级**：🔥 高

- [ ] 创建 OnboardingView（4 屏引导）
- [ ] 删除 Availability 相关页面
- [ ] 创建 GridHomeView（宫格布局）
- [ ] 创建 GridHomeViewModel
- [ ] 创建 FriendCard 组件
- [ ] 创建 PosterShareView
- [ ] 修改 RootView（首页改为 GridHomeView）
- [ ] 修改 MainTabView（移除 Availability tab）

**验证**：
- 打开 App，看到 4 屏引导
- 授权后进入宫格首页
- 点击"更新状态"能触发分析
- 点击"分享"能生成海报

### Phase 3: Android 重构（第 3-4 周）

**优先级**：🔥 高

- [ ] 创建 feature-onboarding 模块
- [ ] 删除 feature-availability 模块
- [ ] 重构 feature-home（改为宫格）
- [ ] 创建 GridHomeScreen
- [ ] 创建 GridHomeViewModel
- [ ] 创建 FriendCard Composable
- [ ] 创建 PosterDialog
- [ ] 修改导航（首页改为 GridHome）

**验证**：
- 同 iOS

### Phase 4: 海报生成优化（第 5 周）

**优先级**：中

**后端**：
- [ ] 优化海报渲染
  - [ ] 使用 Go 图形库（如 `fogleman/gg`）
  - [ ] 或调用第三方服务
- [ ] 添加 Logo 水印
- [ ] 生成邀请二维码
- [ ] 上传到 COS

**iOS/Android**：
- [ ] 海报预览优化
- [ ] 分享到微信/Instagram 等

### Phase 5: 细节打磨（第 6 周）

**优先级**：低

- [ ] 动画效果（宫格刷新动画）
- [ ] 骨架屏（加载状态）
- [ ] 错误提示优化
- [ ] 无好友状态的空态
- [ ] 自定义好友排序
- [ ] A/B 测试引导文案

---

## 关键文件清单

### 后端

| 优先级 | 操作 | 文件路径 | 说明 |
|--------|------|---------|------|
| 🔥 | 删除 | `Backend/internal/handler/availability.go` | 废弃 |
| 🔥 | 删除 | `Backend/internal/service/availability_service.go` | 废弃 |
| 🔥 | 删除 | `Backend/internal/repository/availability_repo.go` | 废弃 |
| 🔥 | 新建 | `Backend/internal/handler/home_handler.go` | 宫格 + 海报接口 |
| 🔥 | 修改 | `Backend/cmd/server/main.go` | 删除 availability 路由，新增 home 路由 |

### iOS

| 优先级 | 操作 | 文件路径 | 说明 |
|--------|------|---------|------|
| 🔥 | 新建 | `iOS/YouKong/Presentation/Screens/Onboarding/OnboardingView.swift` | 引导页 |
| 🔥 | 新建 | `iOS/YouKong/Presentation/Screens/Home/GridHomeView.swift` | 宫格首页 |
| 🔥 | 新建 | `iOS/YouKong/Presentation/Screens/Home/GridHomeViewModel.swift` | ViewModel |
| 🔥 | 新建 | `iOS/YouKong/Presentation/Screens/Home/PosterShareView.swift` | 海报分享 |
| 🔥 | 修改 | `iOS/YouKong/App/RootView.swift` | 首页改为 GridHomeView |
| 🔥 | 删除 | `iOS/YouKong/Presentation/Screens/Availability/` | 废弃整个目录 |

### Android

| 优先级 | 操作 | 文件路径 | 说明 |
|--------|------|---------|------|
| 🔥 | 新建 | `Android/feature/feature-onboarding/` | 引导页模块 |
| 🔥 | 修改 | `Android/feature/feature-home/.../GridHomeScreen.kt` | 宫格首页 |
| 🔥 | 修改 | `Android/feature/feature-home/.../GridHomeViewModel.kt` | ViewModel |
| 🔥 | 删除 | `Android/feature/feature-availability/` | 废弃整个模块 |
| 🔥 | 修改 | `Android/app/.../YouKongNavHost.kt` | 修改导航 |

---

## UI/UX 规范

### 宫格尺寸

```
屏幕宽度 = W
Padding = 16dp
Card 间距 = 8dp

1x1: Card = W - 32dp
2x2: Card = (W - 32dp - 8dp) / 2
3x3: Card = (W - 32dp - 16dp) / 3
4x4: Card = (W - 32dp - 24dp) / 4
```

### 颜色

```
背景：#FFFFFF
卡片：#F5F5F5
文字主色：#000000
文字次要：#666666
强调色：#007AFF（iOS Blue）
```

### 字体

```
Emoji: 40sp/pt
昵称: 14sp/pt, Bold
状态: 12sp/pt, Regular
时间: 10sp/pt, Regular
```

### 动画

```
刷新动画: 淡入淡出 0.3s
卡片点击: 缩放 0.1s
```

---

## 隐私保护说明文档

**在引导页第 3 屏展示**：

```
🔒 隐私至上

我们如何保护你的隐私：

1. 本地分析
   ✅ 位置、日历、计步数据在你的设备上分析
   ✅ 只上传推测结果，不上传原始数据

2. 加密传输
   ✅ 所有数据使用 TLS 1.3 加密
   ✅ 服务器不保存原始传感器数据

3. 好友可见
   ✅ 只有你的好友能看到你的状态
   ✅ 你可以随时删除好友

4. 绝不滥用
   ❌ 不出售数据
   ❌ 不投放广告
   ❌ 不追踪你的行为

5. 你的控制权
   ✅ 随时撤销权限
   ✅ 随时删除账号
   ✅ 随时导出数据

详细隐私政策：https://youkong.app/privacy
```

---

## 端到端测试场景

### 场景 1：首次使用

```
1. 打开 App
2. 看到欢迎屏
3. 滑动到产品介绍
4. 滑动到隐私承诺
5. 滑动到权限请求
6. 点击"授权"
7. 系统弹出位置权限 → 允许
8. 系统弹出日历权限 → 允许
9. 系统弹出运动权限 → 允许
10. 进入宫格首页（只有自己 1x1）
11. 验证：显示"暂无好友"提示
```

### 场景 2：更新状态

```
1. 进入宫格首页
2. 点击"更新状态"按钮
3. 显示"正在分析..."进度条
4. Agent 分析完成，弹出推测结果
5. 用户确认"✓ 是的"
6. 宫格刷新，"我"的卡片更新
7. 验证：
   - Emoji 已变化
   - 状态文字已更新
   - 时间显示"刚刚"
```

### 场景 3：分享海报

```
1. 宫格首页有 4 个好友（2x2）
2. 点击"分享"按钮
3. 显示海报预览
4. 验证海报内容：
   - 顶部有时间戳
   - 中间是 2x2 宫格截图
   - 底部白条有 Logo + 二维码
5. 点击"保存到相册"
6. 验证：相册中有海报图片
7. 点击"分享到微信"
8. 跳转到微信分享
```

### 场景 4：添加好友

```
1. 点击设置 → 邀请好友
2. 生成邀请码
3. 好友扫码/输入邀请码
4. 好友加入，自动成为好友
5. 宫格刷新，从 1x1 变为 2x2
6. 验证：显示好友的状态卡片
```

---

## 数据库清理脚本（可选）

如果决定完全废弃 Availability：

```sql
-- 备份（可选）
CREATE TABLE availabilities_backup AS SELECT * FROM availabilities;
CREATE TABLE availability_circles_backup AS SELECT * FROM availability_circles;

-- 删除表
DROP TABLE IF EXISTS availability_circles;
DROP TABLE IF EXISTS availabilities;

-- 清理相关记录
DELETE FROM messages WHERE type IN ('AVAILABILITY_CARD', 'CONFIRM_REQUEST', 'CONFIRM_RESPONSE');
```

**注意**：执行前务必备份！

---

## 监控指标

### 用户行为

```
- 引导页完成率（4 屏全部看完）
- 权限授权率（位置、日历、计步）
- 首次更新状态成功率
- 日活跃用户数（DAU）
- 周活跃用户数（WAU）
```

### 功能使用

```
- 平均每用户每天更新状态次数
- 分享海报次数
- 邀请好友成功率
- 消息发送量
```

### 性能

```
- 宫格加载耗时（p50, p95）
- Agent 分析耗时
- 海报生成耗时
- API 响应时间
```

### 质量

```
- 状态推测准确率（用户确认 vs 修正）
- Crash 率
- API 错误率
```

---

## 验证方式

### 后端验证

```bash
# 1. 确认 availability 接口已删除
curl http://localhost:8080/api/v1/availabilities
# 应返回 404

# 2. 测试宫格接口
curl http://localhost:8080/api/v1/home/grid \
  -H "Authorization: Bearer $TOKEN" | jq
# 应返回好友列表

# 3. 测试海报接口
curl -X POST http://localhost:8080/api/v1/home/poster \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"user_ids": ["me", "user2"]}' | jq
# 应返回海报 URL
```

### iOS 验证

```
1. 删除 App 重装
2. 确认看到 4 屏引导
3. 授权所有权限
4. 进入宫格首页
5. 点击"更新状态"成功
6. 点击"分享"生成海报
7. 添加好友后宫格自动扩展
```

### Android 验证

```
同 iOS
```

---

## 总结

**核心变更**：
- 从"未来空闲时段"改为"实时状态推测"
- 从"三步发布流程"改为"一键更新"
- 从"好友列表"改为"宫格布局"
- 新增"引导流程"和"分享海报"

**删除的功能**：
- ❌ Availability 发布系统
- ❌ 时间/地点/圈子选择
- ❌ 预约系统

**保留的功能**：
- ✅ Agent 推理引擎
- ✅ 好友系统
- ✅ 消息系统
- ✅ 邀请码

**新增的功能**：
- ✅ 宫格首页
- ✅ 引导流程
- ✅ 海报分享

**实施优先级**：
1. 🔥 后端 API 调整（Week 1）
2. 🔥 iOS 重构（Week 2-3）
3. 🔥 Android 重构（Week 3-4）
4. 中 海报优化（Week 5）
5. 低 细节打磨（Week 6）
