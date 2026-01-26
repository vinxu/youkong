# 有空 - 产品需求文档

> 一眼看到谁可能有空

---

## 一、产品概述

### 1.1 一句话描述

**打开 APP，看到朋友按"有空概率"排列的列表，点击就能聊。**

### 1.2 核心理念

```
不是"我告诉你我有空"
而是"AI 告诉你谁可能有空"

不需要任何人主动发布状态
系统自动推测每个人此刻的有空概率
```

### 1.3 产品定位

| 维度 | 定位 |
|------|------|
| 类型 | 社交工具 |
| 场景 | 想找人但不知道谁有空 |
| 核心价值 | 降低约人的决策成本 |
| 目标用户 | 有社交需求但懒得主动的年轻人 |

### 1.4 与竞品的区别

| 产品 | 模式 | 问题 |
|------|------|------|
| 微信 | 我发消息问"有空吗" | 需要主动，可能被拒绝 |
| Calendly | 我发链接让你选时间 | 太正式，适合工作 |
| 其他社交APP | 我发布状态"我有空" | 太 E，I 人发不出去 |
| **有空** | AI 告诉我谁可能有空 | ✅ 不需要任何人主动 |

---

## 二、核心功能

### 2.1 产品结构

```
整个产品只有两个页面：

┌─────────────────┐
│                 │
│   朋友列表       │  ← 主页面，也是唯一页面
│  （按有空排序）   │
│                 │
└────────┬────────┘
         │ 点击某人
         ▼
┌─────────────────┐
│                 │
│   聊天页面       │  ← 点击朋友进入
│                 │
│                 │
└─────────────────┘
```

### 2.2 主页面 - 朋友列表

```
┌─────────────────────────────────────────┐
│ 有空                          周五 21:32 │
├─────────────────────────────────────────┤
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🟢 李四                    92%  │   │  ← 深绿：很可能有空
│  │    刚刷了40分钟手机              │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🟢 王五                    78%  │   │  ← 浅绿：可能有空
│  │    在家附近，周五晚上            │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🟡 张三                    45%  │   │  ← 黄色：不太确定
│  │    数据不足                     │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🔴 赵六                    12%  │   │  ← 红色：可能没空
│  │    在公司，可能还在加班          │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ ⚫ 钱七                     --  │   │  ← 灰色：无数据
│  │    未授权                       │   │
│  └─────────────────────────────────┘   │
│                                         │
└─────────────────────────────────────────┘
```

### 2.3 颜色系统

| 概率范围 | 颜色 | 含义 |
|----------|------|------|
| 80-100% | 🟢 深绿 `#22C55E` | 很可能有空 |
| 60-79% | 🟢 浅绿 `#86EFAC` | 可能有空 |
| 40-59% | 🟡 黄色 `#FACC15` | 不太确定 |
| 20-39% | 🟠 橙色 `#FB923C` | 可能没空 |
| 0-19% | 🔴 红色 `#EF4444` | 很可能没空 |
| 无数据 | ⚫ 灰色 `#9CA3AF` | 未授权/无法判断 |

### 2.4 聊天页面

点击朋友进入聊天，极简设计：

```
┌─────────────────────────────────────────┐
│ ← 李四                        🟢 92%   │
├─────────────────────────────────────────┤
│                                         │
│                                         │
│  ┌─────────────────────┐               │
│  │ 在干嘛               │  ← 我发的    │
│  └─────────────────────┘               │
│                                         │
│               ┌─────────────────────┐   │
│               │ 没事 刷手机         │   │  ← 对方回复
│               └─────────────────────┘   │
│                                         │
│  ┌─────────────────────┐               │
│  │ 出来吃个饭？         │               │
│  └─────────────────────┘               │
│                                         │
│                                         │
├─────────────────────────────────────────┤
│  输入消息...                    [发送]  │
└─────────────────────────────────────────┘
```

---

## 三、权限体系

### 3.1 三个核心权限

用户必须授权以下三个权限才能使用产品：

| 权限 | 用途 | 价值 |
|------|------|------|
| **屏幕使用时间** | 判断是否正在用手机 | 最核心的"有空"信号 |
| **地理位置** | 判断在哪里（家/公司/外面） | 辅助判断状态 |
| **通讯录** | 找到你的朋友 | 建立社交关系 |

### 3.2 权限申请流程

```
首次打开 APP：

┌─────────────────────────────────────────┐
│                                         │
│             欢迎使用「有空」             │
│                                         │
│    我们需要以下权限来帮你找到           │
│    可能有空的朋友                        │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 📱 屏幕使用时间                  │   │
│  │    用来判断你是否正在用手机       │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 📍 地理位置                      │   │
│  │    用来判断你在哪里              │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 👥 通讯录                        │   │
│  │    用来找到你的朋友              │   │
│  └─────────────────────────────────┘   │
│                                         │
│         [开始授权]                      │
│                                         │
│    所有数据仅用于计算有空概率            │
│    不会用于其他用途                      │
│                                         │
└─────────────────────────────────────────┘
```

### 3.3 权限缺失处理

如果用户拒绝任一权限：

```
┌─────────────────────────────────────────┐
│                                         │
│         需要完整权限才能使用             │
│                                         │
│    ❌ 屏幕使用时间 - 未授权              │
│    ✅ 地理位置 - 已授权                  │
│    ✅ 通讯录 - 已授权                    │
│                                         │
│    没有屏幕使用时间权限，                │
│    我们无法判断你和朋友是否有空          │
│                                         │
│         [去设置中开启]                   │
│                                         │
└─────────────────────────────────────────┘
```

### 3.4 隐私说明

```
数据使用原则：

1. 数据只用于计算有空概率
2. 不存储具体的 APP 使用记录
3. 不存储具体的位置坐标，只存储状态（家/公司/外面）
4. 通讯录只用于匹配好友，不上传原始数据
5. 所有计算尽可能在本地完成
```

---

## 四、有空概率算法

### 4.1 概率公式

```
有空概率 = f(屏幕使用状态, 地理位置, 当前时间)
```

### 4.2 三个核心维度

#### 维度一：屏幕使用状态

| 状态 | 判断依据 | 基础分 |
|------|----------|--------|
| 正在刷手机（娱乐类APP） | 使用时长 > 10分钟，APP类型为社交/视频/游戏 | +30 |
| 正在刷手机（其他APP） | 使用时长 > 10分钟，APP类型为工具/效率 | +10 |
| 刚用过手机 | 5分钟内有使用记录 | +15 |
| 手机闲置中 | 超过30分钟未使用 | -10 |
| 手机长时间闲置 | 超过2小时未使用 | -25 |

#### 维度二：地理位置

| 位置 | 判断依据 | 基础分 |
|------|----------|--------|
| 在家 | 在常驻位置（晚上最常出现的地方） | +15 |
| 在公司 | 在工作位置（工作日白天最常出现的地方） | -20（工作时间）/ +5（下班后） |
| 在外面（商圈/餐厅） | 在已知的休闲场所附近 | +10 |
| 在外面（其他） | 不在家也不在公司 | 0 |
| 位置未知 | 无位置数据 | 0 |

#### 维度三：当前时间

| 时间段 | 基础分 |
|--------|--------|
| 工作日 9:00-12:00 | -20 |
| 工作日 12:00-14:00（午休） | +5 |
| 工作日 14:00-18:00 | -15 |
| 工作日 18:00-22:00 | +20 |
| 工作日 22:00-24:00 | +10 |
| 工作日 0:00-9:00 | -30 |
| 周末 10:00-22:00 | +15 |
| 周末其他时间 | 0 |
| 节假日 | +20 |

### 4.3 计算逻辑

```python
def calculate_free_probability(user_data: UserData, current_time: datetime) -> int:
    """
    计算有空概率
    返回 0-100 的整数
    """
    
    # 基础分 50
    score = 50
    reasons = []
    
    # ========== 屏幕使用状态 ==========
    screen_state = user_data.screen_usage
    
    if screen_state.is_active_now:
        if screen_state.current_app_category in ['社交', '视频', '游戏']:
            score += 30
            reasons.append(f"刷了{screen_state.session_duration}分钟手机")
        elif screen_state.current_app_category in ['工具', '效率']:
            score += 10
            reasons.append("在用手机")
    elif screen_state.minutes_since_last_use < 5:
        score += 15
        reasons.append("刚用过手机")
    elif screen_state.minutes_since_last_use > 120:
        score -= 25
        reasons.append("手机闲置很久")
    elif screen_state.minutes_since_last_use > 30:
        score -= 10
        reasons.append("有一会没用手机了")
    
    # ========== 地理位置 ==========
    location = user_data.location
    
    if location.place_type == 'home':
        score += 15
        reasons.append("在家")
    elif location.place_type == 'work':
        if is_work_hours(current_time):
            score -= 20
            reasons.append("在公司上班")
        else:
            score += 5
            reasons.append("在公司但已下班")
    elif location.place_type == 'leisure':
        score += 10
        reasons.append(f"在{location.place_name}附近")
    
    # ========== 当前时间 ==========
    time_score, time_reason = get_time_score(current_time)
    score += time_score
    if time_reason:
        reasons.append(time_reason)
    
    # ========== 边界处理 ==========
    score = max(0, min(100, score))
    
    # ========== 选择最佳理由 ==========
    # 只返回最重要的一个理由
    best_reason = select_best_reason(reasons, score)
    
    return score, best_reason


def get_time_score(current_time: datetime) -> tuple[int, str]:
    """根据当前时间返回分数和理由"""
    
    weekday = current_time.weekday()  # 0-6, 0是周一
    hour = current_time.hour
    
    is_weekend = weekday >= 5
    is_holiday = check_holiday(current_time)
    
    if is_holiday:
        return 20, "节假日"
    
    if is_weekend:
        if 10 <= hour <= 22:
            return 15, "周末"
        else:
            return 0, None
    
    # 工作日
    if 9 <= hour < 12:
        return -20, "上班时间"
    elif 12 <= hour < 14:
        return 5, "午休时间"
    elif 14 <= hour < 18:
        return -15, "上班时间"
    elif 18 <= hour < 22:
        return 20, "周五晚上" if weekday == 4 else "下班了"
    elif 22 <= hour < 24:
        return 10, "晚上"
    else:
        return -30, "深夜"
```

### 4.4 LLM 增强推理

对于复杂情况，使用 LLM 做更精细的判断：

```python
LLM_PROMPT = """
你是一个判断"某人此刻是否有空"的专家。

## 用户数据
- 姓名：{name}
- 当前时间：{current_time}（{weekday}）
- 屏幕状态：{screen_status}
- 位置：{location}
- 历史规律：{history_pattern}

## 任务
1. 综合分析这个人此刻有空的概率（0-100）
2. 用一句话（10字以内）解释为什么

## 输出格式（JSON）
{
  "probability": 82,
  "reason": "刚刷了40分钟手机"
}

## 注意
- reason 要口语化、简短
- 优先说最强的信号
- 如果数据不足，reason 写"数据不足"
"""
```

### 4.5 特殊情况处理

| 情况 | 处理方式 |
|------|----------|
| 用户未授权屏幕使用时间 | 概率显示"--"，颜色为灰色 |
| 用户未授权位置 | 只用屏幕+时间计算 |
| 对方不是APP用户 | 只用时间维度估算，置信度低 |
| 数据过期（超过5分钟） | 重新获取，期间显示上次数据 |

---

## 五、数据模型

### 5.1 用户数据

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
  
  // 学习到的位置
  known_places: {
    home?: GeoPoint
    work?: GeoPoint
  }
}
```

### 5.2 实时状态数据

```typescript
interface UserRealtimeStatus {
  user_id: string
  updated_at: timestamp
  
  // 屏幕使用状态
  screen: {
    is_active: boolean
    last_active_at: timestamp
    current_app_category?: '社交' | '视频' | '游戏' | '工具' | '效率' | '其他'
    session_duration_minutes?: number
  }
  
  // 位置状态
  location: {
    place_type: 'home' | 'work' | 'leisure' | 'other' | 'unknown'
    place_name?: string  // 如果是leisure，显示地点名
    updated_at: timestamp
  }
}
```

### 5.3 好友关系

```typescript
interface Friendship {
  id: string
  user_a: string
  user_b: string
  
  // 来源
  source: 'contacts' | 'phone' | 'invite'
  
  // 状态
  status: 'pending' | 'active' | 'blocked'
  
  created_at: timestamp
}
```

### 5.4 消息数据

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
  participants: [string, string]  // 两个用户ID
  last_message_at: timestamp
  created_at: timestamp
}
```

---

## 六、API 设计

### 6.1 获取朋友列表（带有空概率）

```
GET /api/friends/with-probability

Response:
{
  "friends": [
    {
      "id": "u_123",
      "name": "李四",
      "avatar": "https://...",
      "probability": 92,
      "reason": "刚刷了40分钟手机",
      "color": "#22C55E",
      "updated_at": 1706345520000
    },
    {
      "id": "u_456",
      "name": "王五",
      "avatar": "https://...",
      "probability": 78,
      "reason": "在家附近，周五晚上",
      "color": "#86EFAC",
      "updated_at": 1706345510000
    },
    // ...
  ],
  "updated_at": 1706345520000
}
```

### 6.2 上报自己的状态

```
POST /api/status/report

Request:
{
  "screen": {
    "is_active": true,
    "current_app_category": "视频",
    "session_duration_minutes": 40
  },
  "location": {
    "latitude": 39.9042,
    "longitude": 116.4074
  }
}

Response:
{
  "success": true
}
```

### 6.3 获取聊天消息

```
GET /api/conversations/{conversation_id}/messages?limit=50&before={message_id}

Response:
{
  "messages": [
    {
      "id": "m_789",
      "sender_id": "u_123",
      "content": "在干嘛",
      "created_at": 1706345000000,
      "read_at": 1706345010000
    },
    // ...
  ],
  "has_more": false
}
```

### 6.4 发送消息

```
POST /api/conversations/{conversation_id}/messages

Request:
{
  "content": "出来吃个饭？"
}

Response:
{
  "message": {
    "id": "m_790",
    "sender_id": "u_me",
    "content": "出来吃个饭？",
    "created_at": 1706345520000
  }
}
```

---

## 七、技术架构

### 7.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                         客户端                               │
├──────────────────────────┬──────────────────────────────────┤
│       iOS App            │         Android App              │
│  ┌─────────────────┐    │    ┌─────────────────┐           │
│  │ Screen Time API │    │    │ UsageStatsManager│           │
│  │ (Extension)     │    │    │                 │           │
│  └────────┬────────┘    │    └────────┬────────┘           │
│           │              │             │                    │
│  ┌────────┴────────┐    │    ┌────────┴────────┐           │
│  │ Core Location   │    │    │ FusedLocation   │           │
│  └────────┬────────┘    │    └────────┬────────┘           │
│           │              │             │                    │
│           └──────────────┴─────────────┘                    │
│                          │                                  │
│                     状态上报                                 │
│                          ▼                                  │
├─────────────────────────────────────────────────────────────┤
│                        服务端                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    API Gateway                       │   │
│  └──────────────────────────┬──────────────────────────┘   │
│                             │                               │
│  ┌──────────────┬───────────┴───────────┬──────────────┐   │
│  │              │                       │              │   │
│  ▼              ▼                       ▼              ▼   │
│ ┌────┐      ┌────────┐           ┌──────────┐    ┌────┐   │
│ │用户│      │状态服务 │           │概率计算   │    │消息│   │
│ │服务│      │        │           │服务(LLM) │    │服务│   │
│ └─┬──┘      └───┬────┘           └────┬─────┘    └─┬──┘   │
│   │             │                     │            │       │
│   └─────────────┴─────────────────────┴────────────┘       │
│                             │                               │
│                             ▼                               │
│                    ┌────────────────┐                      │
│                    │    数据库       │                      │
│                    │ (PostgreSQL)   │                      │
│                    └────────────────┘                      │
│                             │                               │
│                             ▼                               │
│                    ┌────────────────┐                      │
│                    │   Redis        │                      │
│                    │ (实时状态缓存)  │                      │
│                    └────────────────┘                      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 7.2 数据流

```
1. 状态上报（每分钟）
   客户端 → POST /api/status/report → 状态服务 → Redis

2. 获取朋友列表
   客户端 → GET /api/friends/with-probability
         → 用户服务（获取好友列表）
         → 状态服务（获取每个好友的实时状态）
         → 概率计算服务（计算有空概率）
         → 返回排序后的列表

3. 消息发送
   客户端 → POST /api/messages → 消息服务 → 数据库 + 推送
```

### 7.3 状态同步策略

| 场景 | 策略 |
|------|------|
| APP 在前台 | 每 30 秒上报一次状态 |
| APP 在后台 | 每 5 分钟上报一次状态（需要后台权限） |
| 获取好友状态 | 首次加载拉取，之后每 30 秒增量更新 |
| 状态过期 | 超过 5 分钟未更新，标记为"数据过期" |

---

## 八、iOS 实现要点

### 8.1 Screen Time 集成

```swift
// 1. 请求授权
import FamilyControls

func requestScreenTimePermission() async throws {
    try await AuthorizationCenter.shared.requestAuthorization(for: .individual)
}

// 2. 监控使用（通过阈值回调）
import DeviceActivity

class UsageMonitor: DeviceActivityMonitor {
    
    // 设置每5分钟一个阈值
    func startMonitoring() throws {
        let schedule = DeviceActivitySchedule(
            intervalStart: DateComponents(hour: 0, minute: 0),
            intervalEnd: DateComponents(hour: 23, minute: 59),
            repeats: true
        )
        
        // 为每5分钟设置一个事件
        var events: [DeviceActivityEvent.Name: DeviceActivityEvent] = [:]
        for minutes in stride(from: 5, through: 180, by: 5) {
            let eventName = DeviceActivityEvent.Name("usage_\(minutes)")
            events[eventName] = DeviceActivityEvent(
                threshold: DateComponents(minute: minutes)
            )
        }
        
        try DeviceActivityCenter().startMonitoring(
            .daily,
            during: schedule,
            events: events
        )
    }
    
    // 当达到阈值时触发
    override func eventDidReachThreshold(_ event: DeviceActivityEvent.Name, activity: DeviceActivityName) {
        // 提取分钟数
        let minutes = extractMinutes(from: event)
        
        // 上报到服务器
        StatusReporter.shared.report(screenTimeMinutes: minutes)
    }
}
```

### 8.2 位置监控

```swift
import CoreLocation

class LocationMonitor: NSObject, CLLocationManagerDelegate {
    
    private let locationManager = CLLocationManager()
    
    func startMonitoring() {
        locationManager.delegate = self
        locationManager.requestAlwaysAuthorization()
        locationManager.allowsBackgroundLocationUpdates = true
        locationManager.startMonitoringSignificantLocationChanges()
    }
    
    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let location = locations.last else { return }
        
        // 判断是家/公司/其他
        let placeType = classifyLocation(location)
        
        // 上报
        StatusReporter.shared.report(location: placeType)
    }
    
    private func classifyLocation(_ location: CLLocation) -> PlaceType {
        // 与已知的家/公司位置比较
        if isNear(location, knownPlace: UserData.homeLocation) {
            return .home
        } else if isNear(location, knownPlace: UserData.workLocation) {
            return .work
        } else {
            return .other
        }
    }
}
```

---

## 九、Android 实现要点

### 9.1 UsageStats 集成

```kotlin
// 1. 请求授权
fun requestUsageStatsPermission(context: Context) {
    if (!hasUsageStatsPermission(context)) {
        val intent = Intent(Settings.ACTION_USAGE_ACCESS_SETTINGS)
        context.startActivity(intent)
    }
}

fun hasUsageStatsPermission(context: Context): Boolean {
    val appOps = context.getSystemService(Context.APP_OPS_SERVICE) as AppOpsManager
    val mode = appOps.checkOpNoThrow(
        AppOpsManager.OPSTR_GET_USAGE_STATS,
        android.os.Process.myUid(),
        context.packageName
    )
    return mode == AppOpsManager.MODE_ALLOWED
}

// 2. 获取使用状态
fun getCurrentUsageStatus(context: Context): ScreenUsageStatus {
    val usageStatsManager = context.getSystemService(Context.USAGE_STATS_SERVICE) as UsageStatsManager
    
    val endTime = System.currentTimeMillis()
    val startTime = endTime - 60 * 60 * 1000  // 过去1小时
    
    val events = usageStatsManager.queryEvents(startTime, endTime)
    val event = UsageEvents.Event()
    
    var lastForegroundApp: String? = null
    var lastForegroundTime: Long = 0
    
    while (events.hasNextEvent()) {
        events.getNextEvent(event)
        if (event.eventType == UsageEvents.Event.ACTIVITY_RESUMED) {
            lastForegroundApp = event.packageName
            lastForegroundTime = event.timeStamp
        }
    }
    
    val isActiveNow = (endTime - lastForegroundTime) < 5 * 60 * 1000  // 5分钟内
    val appCategory = getAppCategory(context, lastForegroundApp)
    
    return ScreenUsageStatus(
        isActive = isActiveNow,
        lastActiveAt = lastForegroundTime,
        currentAppCategory = appCategory
    )
}

// 3. 获取APP类别
fun getAppCategory(context: Context, packageName: String?): String {
    if (packageName == null) return "unknown"
    
    return try {
        val pm = context.packageManager
        val appInfo = pm.getApplicationInfo(packageName, 0)
        val category = appInfo.category
        
        when (category) {
            ApplicationInfo.CATEGORY_SOCIAL -> "社交"
            ApplicationInfo.CATEGORY_VIDEO -> "视频"
            ApplicationInfo.CATEGORY_GAME -> "游戏"
            ApplicationInfo.CATEGORY_PRODUCTIVITY -> "效率"
            else -> "其他"
        }
    } catch (e: Exception) {
        "其他"
    }
}
```

### 9.2 后台服务

```kotlin
class StatusReportService : Service() {
    
    private val handler = Handler(Looper.getMainLooper())
    private val reportInterval = 60 * 1000L  // 每分钟上报
    
    private val reportRunnable = object : Runnable {
        override fun run() {
            reportStatus()
            handler.postDelayed(this, reportInterval)
        }
    }
    
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startForeground(NOTIFICATION_ID, createNotification())
        handler.post(reportRunnable)
        return START_STICKY
    }
    
    private fun reportStatus() {
        val screenStatus = getCurrentUsageStatus(this)
        val location = LocationMonitor.getCurrentLocation()
        
        ApiClient.reportStatus(
            screen = screenStatus,
            location = location
        )
    }
}
```

---

## 十、产品迭代计划

### 10.1 MVP（2周）

| 功能 | 优先级 |
|------|--------|
| 手机号注册/登录 | P0 |
| 通讯录好友匹配 | P0 |
| 屏幕使用时间监控（Android） | P0 |
| 有空概率计算（简化版） | P0 |
| 朋友列表展示 | P0 |
| 基础聊天 | P0 |

### 10.2 V1.1（+2周）

| 功能 | 优先级 |
|------|--------|
| 地理位置监控 | P1 |
| 位置学习（家/公司） | P1 |
| iOS Screen Time 集成 | P1 |
| 概率算法优化 | P1 |

### 10.3 V1.2（+2周）

| 功能 | 优先级 |
|------|--------|
| LLM 增强推理 | P2 |
| 历史规律学习 | P2 |
| 推送通知 | P2 |
| 消息已读状态 | P2 |

### 10.4 未来可能

- 好友分组
- 有空时段预测（未来几小时）
- 群聊
- 约见记录
- 社交数据分析

---

## 十一、关键指标

### 11.1 核心指标

| 指标 | 定义 | 目标 |
|------|------|------|
| DAU | 日活跃用户 | - |
| 概率准确率 | 高概率(>70%)用户实际回复率 | >60% |
| 消息发送率 | 打开APP后发送消息的比例 | >30% |
| 回复率 | 收到消息后回复的比例 | >50% |

### 11.2 监控指标

| 指标 | 定义 | 告警阈值 |
|------|------|----------|
| 状态上报成功率 | 上报请求成功比例 | <95% |
| 概率计算延迟 | 计算有空概率的耗时 | >500ms |
| API 错误率 | API 请求失败比例 | >1% |

---

## 十二、附录

### 12.1 APP 类别映射表

| Android Category | iOS Category | 映射类别 |
|-----------------|--------------|----------|
| CATEGORY_SOCIAL | SNS | 社交 |
| CATEGORY_VIDEO | Entertainment | 视频 |
| CATEGORY_GAME | Games | 游戏 |
| CATEGORY_PRODUCTIVITY | Productivity | 效率 |
| CATEGORY_NEWS | News | 资讯 |
| 其他 | 其他 | 其他 |

### 12.2 颜色代码

```css
/* 有空概率颜色 */
--color-free-very:    #22C55E;  /* 80-100% */
--color-free-likely:  #86EFAC;  /* 60-79% */
--color-free-maybe:   #FACC15;  /* 40-59% */
--color-free-unlikely:#FB923C;  /* 20-39% */
--color-free-busy:    #EF4444;  /* 0-19% */
--color-free-unknown: #9CA3AF;  /* 无数据 */

/* 背景色（卡片） */
--color-card-very:    #F0FDF4;  /* 80-100% */
--color-card-likely:  #F0FDF4;  /* 60-79% */
--color-card-maybe:   #FEFCE8;  /* 40-59% */
--color-card-unlikely:#FFF7ED;  /* 20-39% */
--color-card-busy:    #FEF2F2;  /* 0-19% */
--color-card-unknown: #F9FAFB;  /* 无数据 */
```

### 12.3 文案规范

**有空理由**（10字以内，口语化）：

```
✅ 好的例子：
- "刚刷了40分钟手机"
- "在家附近"
- "周五晚上"
- "在公司，可能加班"
- "手机闲置很久"
- "数据不足"

❌ 不好的例子：
- "用户当前正在使用社交类应用程序"（太正式）
- "位于家庭常驻位置附近区域"（太啰嗦）
- "有空"（没有信息量）
```

---

## 文档版本

| 版本 | 日期 | 更新内容 |
|------|------|----------|
| v1.0 | 2026-01-26 | 初版 |
