# 有空 - 开发技术规格文档

> 给 Coding 团队的 Agent 架构实现指南

---

## 一、产品概述

### 1.1 核心功能

**一句话描述**：打开 APP → 看到朋友按"有空概率"排序的列表 → 点击进入聊天

### 1.2 产品界面

```
整个产品只有两个页面：

页面 1: 朋友列表（主页面）
┌─────────────────────────────────────────┐
│  🟢 李四                          92%  │ ← 深绿：很可能有空
│     刷了40分钟手机，在家                  │
├─────────────────────────────────────────┤
│  🟢 王五                          78%  │ ← 浅绿：可能有空
│     周五晚上，历史上他通常有空            │
├─────────────────────────────────────────┤
│  🟡 张三                          45%  │ ← 黄色：不确定
│     数据不足                            │
├─────────────────────────────────────────┤
│  🔴 赵六                          12%  │ ← 红色：可能没空
│     在公司，可能还在加班                  │
└─────────────────────────────────────────┘

页面 2: 聊天页面（点击朋友进入）
```

### 1.3 核心权限要求

用户必须授权以下三个权限才能使用：

| 权限 | 用途 | 技术实现 |
|------|------|----------|
| **屏幕使用时间** | 判断用户是否在用手机、用什么类型APP | Android: UsageStatsManager / iOS: DeviceActivity |
| **地理位置** | 判断用户在哪里（家/公司/外面） | Android: FusedLocationProvider / iOS: CoreLocation |
| **通讯录** | 找到用户的朋友 | 标准通讯录API |

---

## 二、Agent 架构核心概念

### 2.1 每人一个 Agent

```
核心理念：
- 每个用户都有一个专属 AI Agent
- Agent 负责收集、分析自己主人的数据
- 用户打开 APP 时，自己的 Agent 向所有朋友的 Agent 请求数据
- 自己的 Agent 用 LLM 综合分析，生成有空推荐列表
```

### 2.2 流程图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                         │
│  用户打开 APP                                                            │
│      │                                                                  │
│      ▼                                                                  │
│  我的 Agent 激活                                                         │
│      │                                                                  │
│      │  并行请求所有朋友的 Agent                                         │
│      │                                                                  │
│      ├───→ 李四的 Agent → 返回：屏幕状态、位置、历史规律                  │
│      ├───→ 王五的 Agent → 返回：屏幕状态、位置、历史规律                  │
│      ├───→ 张三的 Agent → 返回：屏幕状态、位置、历史规律                  │
│      │                                                                  │
│      ▼                                                                  │
│  我的 Agent 用 LLM 综合分析每个朋友的有空概率                             │
│      │                                                                  │
│      ▼                                                                  │
│  输出排序后的有空推荐列表                                                 │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 三、数据结构定义

### 3.1 Agent 暴露给其他 Agent 的数据

```typescript
// 当其他 Agent 请求时返回的数据
interface AgentExposedData {
  // 实时状态
  realtime: {
    screen: {
      is_active: boolean                // 当前是否在用手机
      activity_type: 'entertainment' | 'productivity' | 'communication' | 'idle'
      session_duration_minutes: number  // 本次使用了多久（四舍五入到5分钟）
      last_active_minutes_ago: number   // 上次活跃是多久前（四舍五入到5分钟）
    }
    location: {
      place_type: 'home' | 'work' | 'leisure' | 'transit' | 'unknown'
      at_place_since_minutes: number    // 在这个地方待了多久
    }
  }
  
  // 历史规律（Agent 学习得到的）
  patterns: {
    current_hour_free_rate: number      // 0-100，当前小时历史有空率
    current_weekday_free_rate: number   // 0-100，今天（周几）历史有空率
    at_home_free_rate: number           // 0-100，在家时有空率
    at_work_after_hours_free_rate: number  // 0-100，下班后在公司有空率
    avg_response_time_minutes: number   // 平均回复时间
    response_rate: number               // 0-100，回复率
  }
  
  // 数据质量
  data_quality: {
    screen_data_age_seconds: number     // 屏幕数据多久前更新的
    location_data_age_seconds: number   // 位置数据多久前更新的
    patterns_sample_size: number        // 历史规律基于多少样本
  }
}
```

### 3.2 用户数据模型

```typescript
interface User {
  id: string
  phone: string
  name: string
  avatar?: string
  created_at: timestamp
  
  // 权限状态
  permissions: {
    screen_time: boolean
    location: boolean
    contacts: boolean
  }
}
```

### 3.3 好友关系

```typescript
interface Friendship {
  id: string
  user_a: string
  user_b: string
  source: 'contacts' | 'phone' | 'invite'
  status: 'pending' | 'active' | 'blocked'
  created_at: timestamp
}
```

### 3.4 消息数据

```typescript
interface Message {
  id: string
  conversation_id: string
  sender_id: string
  content: string
  created_at: timestamp
  read_at?: timestamp
}

interface Conversation {
  id: string
  participants: [string, string]
  last_message_at: timestamp
  created_at: timestamp
}
```

### 3.5 有空推荐结果

```typescript
interface FriendRecommendation {
  friend_id: string
  name: string
  avatar?: string
  probability: number       // 0-100
  confidence: 'high' | 'medium' | 'low'
  reason: string            // "刷了40分钟手机，在家"
  color: string             // "#22C55E"
}
```

---

## 四、颜色系统

```typescript
const PROBABILITY_COLORS = {
  // 概率 80-100%: 深绿
  VERY_LIKELY: {
    dot: '#22C55E',
    background: '#F0FDF4'
  },
  // 概率 60-79%: 浅绿
  LIKELY: {
    dot: '#86EFAC',
    background: '#F0FDF4'
  },
  // 概率 40-59%: 黄色
  MAYBE: {
    dot: '#FACC15',
    background: '#FEFCE8'
  },
  // 概率 20-39%: 橙色
  UNLIKELY: {
    dot: '#FB923C',
    background: '#FFF7ED'
  },
  // 概率 0-19%: 红色
  BUSY: {
    dot: '#EF4444',
    background: '#FEF2F2'
  },
  // 无数据: 灰色
  UNKNOWN: {
    dot: '#9CA3AF',
    background: '#F9FAFB'
  }
}

function probabilityToColor(probability: number | null): ColorScheme {
  if (probability === null) return PROBABILITY_COLORS.UNKNOWN
  if (probability >= 80) return PROBABILITY_COLORS.VERY_LIKELY
  if (probability >= 60) return PROBABILITY_COLORS.LIKELY
  if (probability >= 40) return PROBABILITY_COLORS.MAYBE
  if (probability >= 20) return PROBABILITY_COLORS.UNLIKELY
  return PROBABILITY_COLORS.BUSY
}
```

---

## 五、Agent 核心实现

### 5.1 Agent 类结构

```typescript
class PersonalAgent {
  private ownerId: string
  private dataCollector: DataCollector
  private patternLearner: PatternLearner
  private llmClient: LLMClient
  
  // ========== 数据提供者角色 ==========
  
  // 被其他 Agent 请求时调用
  async getExposedData(): Promise<AgentExposedData> {
    const screen = await this.dataCollector.getScreenData()
    const location = await this.dataCollector.getLocationData()
    const patterns = await this.patternLearner.getPatterns()
    
    return {
      realtime: {
        screen: this.sanitizeScreenData(screen),
        location: this.sanitizeLocationData(location)
      },
      patterns: patterns,
      data_quality: {
        screen_data_age_seconds: this.getDataAge(screen.updated_at),
        location_data_age_seconds: this.getDataAge(location.updated_at),
        patterns_sample_size: patterns.sample_size
      }
    }
  }
  
  // ========== 分析者角色 ==========
  
  // 用户打开 APP 时调用
  async generateRecommendationList(): Promise<FriendRecommendation[]> {
    // 1. 获取所有朋友
    const friends = await this.getFriends()
    
    // 2. 并行请求所有朋友 Agent 的数据
    const friendDataList = await Promise.all(
      friends.map(friend => this.requestFriendAgentData(friend.agent_id))
    )
    
    // 3. 对每个朋友，用 LLM 计算有空概率
    const recommendations = await Promise.all(
      friends.map((friend, index) => 
        this.analyzeFriendAvailability(friend, friendDataList[index])
      )
    )
    
    // 4. 按概率排序
    recommendations.sort((a, b) => b.probability - a.probability)
    
    return recommendations
  }
  
  // 分析单个朋友的有空概率
  private async analyzeFriendAvailability(
    friend: Friend, 
    agentData: AgentExposedData
  ): Promise<FriendRecommendation> {
    
    // 获取我和这个朋友的关系数据
    const relationship = await this.getRelationship(friend.id)
    
    // 构建 LLM Prompt
    const prompt = this.buildAnalysisPrompt(friend, agentData, relationship)
    
    // 调用 LLM
    const result = await this.llmClient.analyze(prompt)
    
    return {
      friend_id: friend.id,
      name: friend.name,
      avatar: friend.avatar,
      probability: result.probability,
      confidence: result.confidence,
      reason: result.reason,
      color: probabilityToColor(result.probability).dot
    }
  }
}
```

### 5.2 数据收集器

```typescript
class DataCollector {
  
  // ========== 屏幕使用数据 ==========
  
  async getScreenData(): Promise<ScreenData> {
    // Android: 使用 UsageStatsManager
    // iOS: 使用 DeviceActivity 阈值回调
    
    return {
      is_active: boolean,
      activity_type: 'entertainment' | 'productivity' | 'communication' | 'idle',
      session_duration_minutes: number,
      last_active_minutes_ago: number,
      updated_at: timestamp
    }
  }
  
  // APP 类别分类
  categorizeApp(packageName: string): ActivityType {
    const ENTERTAINMENT_APPS = ['抖音', '快手', '微博', 'B站', '小红书', '游戏类']
    const PRODUCTIVITY_APPS = ['钉钉', '飞书', '企业微信', 'Office类', '笔记类']
    const COMMUNICATION_APPS = ['微信', 'QQ', '电话', '短信']
    
    // 根据 package name 或 app category 判断
  }
  
  // ========== 位置数据 ==========
  
  async getLocationData(): Promise<LocationData> {
    const currentLocation = await this.getCurrentLocation()
    const placeType = await this.classifyLocation(currentLocation)
    
    return {
      place_type: placeType,
      at_place_since_minutes: this.getAtPlaceDuration(),
      updated_at: timestamp
    }
  }
  
  // 位置分类
  async classifyLocation(location: GeoPoint): Promise<PlaceType> {
    const knownPlaces = await this.getKnownPlaces()
    
    if (this.isNear(location, knownPlaces.home)) return 'home'
    if (this.isNear(location, knownPlaces.work)) return 'work'
    // ... 其他判断
    return 'unknown'
  }
  
  // 学习家和公司位置
  async learnPlaces(locationHistory: LocationRecord[]): Promise<void> {
    // 家：晚上10点-早上7点最常待的地方
    // 公司：工作日9点-18点最常待的地方
  }
}
```

### 5.3 模式学习器

```typescript
class PatternLearner {
  
  // 学习时间模式
  async learnTimePatterns(): Promise<TimePatterns> {
    const responseHistory = await this.storage.getResponseHistory()
    
    // 统计每小时的有空率
    const hourlyRates = new Array(24).fill(0)
    const hourlyCounts = new Array(24).fill(0)
    
    for (const record of responseHistory) {
      const hour = new Date(record.timestamp).getHours()
      hourlyCounts[hour]++
      if (record.responded) {
        hourlyRates[hour]++
      }
    }
    
    return {
      hourly_free_rate: hourlyRates.map((rate, i) => 
        hourlyCounts[i] > 0 ? Math.round(rate / hourlyCounts[i] * 100) : 50
      ),
      // ... 类似计算 weekday_free_rate
    }
  }
  
  // 反馈学习
  async updateFromFeedback(feedback: {
    friend_id: string
    predicted_probability: number
    actual_responded: boolean
    response_time_minutes?: number
  }): Promise<void> {
    // 记录预测结果，用于持续提高准确率
  }
}
```

---

## 六、LLM 分析 Prompt

### 6.1 核心 Prompt 模板

```python
FRIEND_ANALYSIS_PROMPT = """
你是用户的专属 AI Agent。你的任务是分析「{friend_name}」此刻有空的概率。

## 「{friend_name}」的 Agent 提供的数据

### 实时状态
屏幕使用：
- 当前是否在用手机：{is_active}
- 使用类型：{activity_type}
- 本次使用时长：{session_duration} 分钟
- 上次活跃：{last_active_minutes_ago} 分钟前

位置：
- 当前位置类型：{place_type}
- 在此位置已待：{at_place_since} 分钟

### 历史规律
- 现在是 {current_time}（{weekday}）
- 这个时段他历史上有空的概率：{current_hour_free_rate}%
- 今天（{weekday}）他历史上有空的概率：{current_weekday_free_rate}%
- 他在家时有空的概率：{at_home_free_rate}%
- 他的平均回复时间：{avg_response_time} 分钟
- 他的回复率：{response_rate}%

### 数据质量
- 屏幕数据更新于：{screen_data_age} 秒前
- 位置数据更新于：{location_data_age} 秒前
- 历史规律基于：{patterns_sample_size} 个样本

## 你和「{friend_name}」的关系
- 亲密度：{intimacy_score}/100
- 上次联系：{days_since_last_contact} 天前
- 他给你的回复率：{response_rate_to_me}%

## 分析任务

综合以上数据，精确计算「{friend_name}」此刻有空的概率。

## 输出格式（JSON）

{
  "probability": 85,
  "confidence": "high",
  "reason": "刷了40分钟手机，在家"
}

## 约束

1. probability: 0-100 整数
2. confidence: "high"(数据充足) / "medium"(部分缺失) / "low"(大部分缺失)
3. reason: ≤15个汉字，口语化，突出最重要的1-2个因素
"""
```

### 6.2 分析逻辑说明

```
因素权重：

1. 屏幕状态（权重最高）
   - 刷娱乐APP超过10分钟 → +30分
   - 刷工作APP → -10分
   - 手机闲置超过1小时 → -20分

2. 位置
   - 在家 → +15分
   - 在公司+工作时间 → -20分
   - 在公司+非工作时间 → +5分

3. 时间规律
   - 当前时段历史有空率高 → +15分
   - 周五晚上/周末 → +10分
   - 工作日白天 → -15分

4. 关系因素
   - 好久没联系 → 可在 reason 中提及
   - 回复率高 → 增加置信度

基础分：50分
最终概率：基础分 + 各因素得分（限制在0-100范围内）
```

---

## 七、API 设计

### 7.1 Agent 状态上报

```
POST /api/agent/status

Request:
{
  "agent_id": "agent_123",
  "timestamp": 1706345520000,
  "screen": {
    "is_active": true,
    "activity_type": "entertainment",
    "session_duration_minutes": 42
  },
  "location": {
    "place_type": "home"
  }
}

Response:
{
  "success": true,
  "next_report_in": 60  // 下次上报时间（秒）
}
```

### 7.2 获取好友有空列表

```
GET /api/friends/free-probability

Response:
{
  "friends": [
    {
      "id": "user_456",
      "name": "李四",
      "avatar": "https://...",
      "probability": 92,
      "confidence": "high",
      "reason": "刷了40分钟手机，在家",
      "color": "#22C55E"
    },
    // ...
  ],
  "generated_at": 1706345520000
}
```

### 7.3 Agent 数据请求（Agent 间通信）

```
POST /api/agent/query

Request:
{
  "from_agent": "agent_123",
  "to_agent": "agent_456",
  "timestamp": 1706345520000
}

Response:
{
  "realtime": {
    "screen": {
      "is_active": true,
      "activity_type": "entertainment",
      "session_duration_minutes": 40,
      "last_active_minutes_ago": 0
    },
    "location": {
      "place_type": "home",
      "at_place_since_minutes": 120
    }
  },
  "patterns": {
    "current_hour_free_rate": 78,
    "current_weekday_free_rate": 72,
    "at_home_free_rate": 85,
    "avg_response_time_minutes": 5,
    "response_rate": 82
  },
  "data_quality": {
    "screen_data_age_seconds": 30,
    "location_data_age_seconds": 120,
    "patterns_sample_size": 150
  }
}
```

### 7.4 聊天相关 API

```
GET /api/conversations/{conversation_id}/messages?limit=50

POST /api/conversations/{conversation_id}/messages
Request: { "content": "在干嘛" }
```

---

## 八、平台实现要点

### 8.1 Android 屏幕使用数据

```kotlin
// 1. 检查权限
fun hasUsageStatsPermission(context: Context): Boolean {
    val appOps = context.getSystemService(Context.APP_OPS_SERVICE) as AppOpsManager
    val mode = appOps.checkOpNoThrow(
        AppOpsManager.OPSTR_GET_USAGE_STATS,
        android.os.Process.myUid(),
        context.packageName
    )
    return mode == AppOpsManager.MODE_ALLOWED
}

// 2. 请求权限
fun requestUsageStatsPermission(context: Context) {
    val intent = Intent(Settings.ACTION_USAGE_ACCESS_SETTINGS)
    context.startActivity(intent)
}

// 3. 获取使用状态
fun getCurrentUsageStatus(context: Context): ScreenData {
    val usageStatsManager = context.getSystemService(Context.USAGE_STATS_SERVICE) as UsageStatsManager
    val endTime = System.currentTimeMillis()
    val startTime = endTime - 60 * 60 * 1000  // 过去1小时
    
    val events = usageStatsManager.queryEvents(startTime, endTime)
    // ... 解析事件，获取当前前台 APP
}

// 4. 获取 APP 类别
fun getAppCategory(context: Context, packageName: String): String {
    val pm = context.packageManager
    val appInfo = pm.getApplicationInfo(packageName, 0)
    return when (appInfo.category) {
        ApplicationInfo.CATEGORY_SOCIAL -> "entertainment"
        ApplicationInfo.CATEGORY_VIDEO -> "entertainment"
        ApplicationInfo.CATEGORY_GAME -> "entertainment"
        ApplicationInfo.CATEGORY_PRODUCTIVITY -> "productivity"
        else -> "other"
    }
}
```

### 8.2 iOS 屏幕使用数据（取巧方案）

```swift
import DeviceActivity
import FamilyControls

// 1. 请求授权
func requestScreenTimePermission() async throws {
    try await AuthorizationCenter.shared.requestAuthorization(for: .individual)
}

// 2. 设置阈值监控
class UsageMonitor: DeviceActivityMonitor {
    
    func startMonitoring() throws {
        let schedule = DeviceActivitySchedule(
            intervalStart: DateComponents(hour: 0, minute: 0),
            intervalEnd: DateComponents(hour: 23, minute: 59),
            repeats: true
        )
        
        // 每5分钟设置一个阈值
        var events: [DeviceActivityEvent.Name: DeviceActivityEvent] = [:]
        for minutes in stride(from: 5, through: 180, by: 5) {
            let eventName = DeviceActivityEvent.Name("usage_\(minutes)")
            events[eventName] = DeviceActivityEvent(
                threshold: DateComponents(minute: minutes)
            )
        }
        
        try DeviceActivityCenter().startMonitoring(.daily, during: schedule, events: events)
    }
    
    // 当达到阈值时触发
    override func eventDidReachThreshold(_ event: DeviceActivityEvent.Name, activity: DeviceActivityName) {
        let minutes = extractMinutes(from: event)
        StatusReporter.shared.report(screenTimeMinutes: minutes)
    }
}
```

**iOS 限制说明**：
- iOS Screen Time API 数据被严格沙盒隔离
- 无法直接读取具体 APP 使用数据
- 只能通过阈值回调间接获知使用时长
- 精度约为 5-10 分钟级别

### 8.3 位置监控

```swift
// iOS
import CoreLocation

class LocationMonitor: NSObject, CLLocationManagerDelegate {
    private let locationManager = CLLocationManager()
    
    func startMonitoring() {
        locationManager.delegate = self
        locationManager.requestAlwaysAuthorization()
        locationManager.allowsBackgroundLocationUpdates = true
        locationManager.startMonitoringSignificantLocationChanges()  // 省电
    }
}
```

```kotlin
// Android
import com.google.android.gms.location.FusedLocationProviderClient

class LocationMonitor(context: Context) {
    private val fusedLocationClient = LocationServices.getFusedLocationProviderClient(context)
    
    fun startMonitoring() {
        val locationRequest = LocationRequest.create().apply {
            interval = 5 * 60 * 1000  // 5分钟
            fastestInterval = 2 * 60 * 1000
            priority = LocationRequest.PRIORITY_BALANCED_POWER_ACCURACY
        }
        fusedLocationClient.requestLocationUpdates(locationRequest, locationCallback, null)
    }
}
```

---

## 九、系统架构

### 9.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                         │
│  用户设备                                                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                          Agent                                  │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │   │
│  │  │ 数据收集器   │  │  模式学习器  │  │  分析引擎   │             │   │
│  │  │ (本地)      │  │  (本地)     │  │  (LLM)     │             │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘             │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                    │
│                                    ▼                                    │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                         Agent Hub (云端)                         │   │
│  │  • Agent 间消息路由                                              │   │
│  │  • 状态缓存                                                      │   │
│  │  • LLM API 代理                                                  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                    │
│                                    ▼                                    │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                           数据层                                 │   │
│  │  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐           │   │
│  │  │ PostgreSQL  │   │   Redis     │   │    S3       │           │   │
│  │  │ (用户/好友) │   │ (状态缓存)  │   │ (头像等)    │           │   │
│  │  └─────────────┘   └─────────────┘   └─────────────┘           │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 9.2 数据同步策略

| 场景 | 策略 |
|------|------|
| APP 在前台 | 每 30 秒上报状态 |
| APP 在后台 | 每 5 分钟上报状态 |
| 获取好友状态 | 首次全量拉取，之后每 30 秒增量更新 |
| 状态过期 | 超过 5 分钟未更新，显示"数据可能过期" |

---

## 十、隐私设计

### 10.1 数据分级

```
Level 0 - 绝对隐私（永不离开设备）
├── 具体使用了什么 APP（抖音、微信等）
├── 精确 GPS 坐标
├── 聊天内容
└── 通讯录原始数据

Level 1 - Agent 内部（本地存储）
├── APP 使用时长（按类别聚合）
├── 位置类型（家/公司，不是坐标）
└── 行为模式统计

Level 2 - Agent 对外暴露（给朋友的 Agent）
├── 是否在用手机（布尔值）
├── 使用类型（娱乐/工作/通讯/闲置）
├── 位置类型（家/公司/外面）
└── 历史有空概率（数字）
```

### 10.2 隐私保护实现

```typescript
// 模糊化屏幕数据
function sanitizeScreenData(raw: RawScreenData): SanitizedScreenData {
  return {
    is_active: raw.is_active,
    // 不暴露具体 APP 名称，只暴露类别
    activity_type: categorize(raw.current_app),
    // 时长四舍五入到5分钟
    session_duration_minutes: Math.round(raw.duration / 5) * 5,
    last_active_minutes_ago: Math.round(raw.idle / 5) * 5
  }
}

// 模糊化位置数据
function sanitizeLocationData(raw: RawLocationData): SanitizedLocationData {
  return {
    // 不暴露坐标，只暴露类型
    place_type: classifyPlace(raw.coordinates),
    at_place_since_minutes: Math.round(raw.duration / 5) * 5
  }
}
```

---

## 十一、开发路线图

### Phase 1: MVP（4周）

**Week 1-2: 基础架构**
- [ ] Agent 核心框架
- [ ] 数据收集器（屏幕时间、位置）- Android 优先
- [ ] 简单规则引擎（不用 LLM）
- [ ] Agent Hub 基础设施

**Week 3-4: 客户端**
- [ ] iOS/Android APP
- [ ] 权限申请流程
- [ ] 朋友列表 UI
- [ ] 聊天功能

### Phase 2: LLM 增强（4周）

**Week 5-6: LLM 集成**
- [ ] Prompt 设计与测试
- [ ] 云端 LLM 调用
- [ ] 批量分析优化
- [ ] 准确率监控

**Week 7-8: 学习模块**
- [ ] 历史规律学习
- [ ] 反馈学习
- [ ] iOS 屏幕数据收集
- [ ] 个性化优化

### Phase 3: 优化（4周）

**Week 9-12**
- [ ] 准确率提升（Prompt 迭代）
- [ ] 性能优化
- [ ] 电量优化
- [ ] A/B 测试

---

## 十二、关键技术决策

### 12.1 LLM 选择

| 选项 | 优点 | 缺点 | 推荐 |
|------|------|------|------|
| Claude API | 推理能力强 | 成本较高 | ✅ 推荐用于复杂分析 |
| GPT-4 | 生态完善 | 成本较高 | 备选 |
| 本地小模型 | 成本低、延迟低 | 能力有限 | 用于简单场景 |

### 12.2 状态存储

| 选项 | 用途 |
|------|------|
| Redis | 实时状态缓存（TTL 60s） |
| PostgreSQL | 用户数据、好友关系、消息记录 |
| 本地 SQLite | Agent 学习到的模式 |

### 12.3 通信协议

- Agent 间通信：WebSocket（实时性高）
- 状态上报：HTTP POST（可批量）
- 消息推送：Firebase/APNs

---

## 十三、文档版本

| 版本 | 日期 | 更新内容 |
|------|------|----------|
| v1.0 | 2026-01-26 | 初版 - Agent架构开发规格 |

---

## 附录：快速开始检查清单

```
□ 权限申请流程实现
  □ Android: UsageStats、Location、Contacts
  □ iOS: ScreenTime、Location、Contacts

□ 数据收集器实现
  □ 屏幕使用数据收集
  □ 位置数据收集
  □ APP 类别分类

□ Agent 核心逻辑
  □ Agent 数据暴露接口
  □ Agent 间通信协议
  □ LLM 分析集成

□ UI 实现
  □ 朋友列表页面
  □ 颜色系统
  □ 聊天页面

□ 后端服务
  □ Agent Hub 消息路由
  □ 状态缓存
  □ 用户/好友 API
  □ 消息 API
```
