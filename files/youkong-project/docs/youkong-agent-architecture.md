# 有空 - 多 Agent 系统架构

> 每个人都有一个 AI Agent，我的 Agent 综合分析所有朋友的数据，给我最准确的推荐

---

## 一、核心理念

```
每个用户都有一个专属 Agent
Agent 负责收集、分析主人的数据
当我打开 APP：
  → 我的 Agent 向所有朋友的 Agent 请求数据
  → 我的 Agent 综合分析
  → 给我一份准确的"有空朋友推荐列表"
```

### 1.1 为什么用 Agent 架构

| 传统方式 | Agent 方式 |
|----------|-----------|
| 服务器集中计算 | 每个 Agent 负责自己的主人 |
| 算法统一 | 每个 Agent 学习主人的个性化规律 |
| 原始数据上传服务器 | 原始数据本地存储，只交换状态 |
| 准确率有限 | 越用越准（个性化学习） |

### 1.2 流程图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                         │
│  我打开 APP                                                              │
│      │                                                                  │
│      ▼                                                                  │
│  我的 Agent 激活                                                         │
│      │                                                                  │
│      │  并行请求所有朋友的 Agent                                         │
│      │                                                                  │
│      ├───→ 李四的 Agent：给我李四的状态数据                              │
│      │         └───→ 返回：屏幕状态、位置、时间规律                       │
│      │                                                                  │
│      ├───→ 王五的 Agent：给我王五的状态数据                              │
│      │         └───→ 返回：屏幕状态、位置、时间规律                       │
│      │                                                                  │
│      ├───→ 张三的 Agent：给我张三的状态数据                              │
│      │         └───→ 返回：屏幕状态、位置、时间规律                       │
│      │                                                                  │
│      │  ... 所有朋友                                                    │
│      │                                                                  │
│      ▼                                                                  │
│  我的 Agent 综合分析                                                     │
│      │                                                                  │
│      │  • 结合每个朋友的实时数据                                         │
│      │  • 结合每个朋友的历史规律                                         │
│      │  • 结合我和每个朋友的关系                                         │
│      │  • 用 LLM 推理计算精确概率                                        │
│      │                                                                  │
│      ▼                                                                  │
│  输出有空推荐列表                                                        │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  🟢 李四  92%  "刷了40分钟抖音，在家"                             │   │
│  │  🟢 王五  78%  "周五晚上，历史上这时候他通常有空"                  │   │
│  │  🟡 张三  45%  "手机闲置，不太确定"                               │   │
│  │  🔴 赵六  15%  "在公司，可能还在加班"                             │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 二、Agent 设计

### 2.1 每个 Agent 的两种角色

```
角色 A：数据提供者
  - 收集主人的实时数据
  - 存储主人的历史规律
  - 被其他 Agent 请求时，提供主人的状态数据

角色 B：分析者（只有用户打开 APP 时激活）
  - 向朋友们的 Agent 请求数据
  - 综合分析所有数据
  - 用 LLM 计算每个朋友的有空概率
  - 生成推荐列表
```

### 2.2 Agent 提供的数据（给其他 Agent）

```typescript
// 当其他 Agent 请求时，我的 Agent 返回的数据
interface AgentExposedData {
  // 实时状态
  realtime: {
    screen: {
      is_active: boolean           // 当前是否在用手机
      activity_type: 'entertainment' | 'productivity' | 'communication' | 'idle'
      session_duration_minutes: number  // 本次使用了多久
      last_active_minutes_ago: number   // 上次活跃是多久前
    }
    location: {
      place_type: 'home' | 'work' | 'leisure' | 'transit' | 'unknown'
      at_place_since_minutes: number    // 在这个地方待了多久
    }
  }
  
  // 历史规律（Agent 学习得到的）
  patterns: {
    // 当前时段的历史有空概率
    current_hour_free_rate: number      // 0-100，这个小时他历史上有空的比例
    current_weekday_free_rate: number   // 0-100，这天他历史上有空的比例
    
    // 位置相关规律
    at_home_free_rate: number           // 在家时有空的比例
    at_work_after_hours_free_rate: number  // 下班后在公司有空的比例
    
    // 响应规律
    avg_response_time_minutes: number   // 平均回复时间
    response_rate: number               // 回复率
  }
  
  // 数据质量
  data_quality: {
    screen_data_age_seconds: number     // 屏幕数据多久前更新的
    location_data_age_seconds: number   // 位置数据多久前更新的
    patterns_sample_size: number        // 历史规律基于多少样本
  }
}
```

### 2.3 我的 Agent 分析逻辑

```typescript
class MyAgent {
  
  /**
   * 生成有空朋友推荐列表
   * 这是用户打开 APP 时调用的核心方法
   */
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
  
  /**
   * 分析单个朋友的有空概率
   * 这里是 LLM 推理的核心
   */
  async analyzeFriendAvailability(
    friend: Friend, 
    agentData: AgentExposedData
  ): Promise<FriendRecommendation> {
    
    // 获取我和这个朋友的关系数据
    const relationship = await this.getRelationship(friend.id)
    
    // 构建 LLM Prompt
    const prompt = this.buildAnalysisPrompt(friend, agentData, relationship)
    
    // 调用 LLM
    const result = await this.llm.analyze(prompt)
    
    return {
      friend_id: friend.id,
      name: friend.name,
      avatar: friend.avatar,
      probability: result.probability,
      confidence: result.confidence,
      reason: result.reason,
      color: this.probabilityToColor(result.probability)
    }
  }
}
```

---

## 三、LLM 分析 Prompt

### 3.1 核心 Prompt

我的 Agent 用以下 Prompt 分析每个朋友的有空概率：

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

### 历史规律（{friend_name} 的 Agent 学习到的）
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
- 通常联系频率：每 {typical_contact_frequency} 天
- 他给你的回复率：{response_rate_to_me}%

## 分析任务

综合以上所有数据，精确计算「{friend_name}」此刻有空的概率。

## 分析思路

1. **实时状态分析**（权重最高）
   - 屏幕状态是最直接的信号
   - 如果正在刷娱乐类 APP 超过10分钟 → 大概率有空
   - 如果手机闲置很久 → 可能在忙别的或睡觉
   
2. **位置分析**
   - 在家 → 通常更可能有空
   - 在公司 + 工作时间 → 可能在忙
   - 在外面（餐厅等） → 可能在社交

3. **时间规律分析**
   - 参考历史数据：这个时段他通常有空吗？
   - 参考今天的规律：今天是工作日还是周末？

4. **关系因素**
   - 他通常回复你的速度如何？
   - 你们多久没联系了？

5. **数据质量考量**
   - 数据越新鲜，置信度越高
   - 样本量越大，历史规律越可信

## 输出格式（JSON）

{
  "probability": 85,
  "confidence": "high",
  "reason": "刷了40分钟手机，在家",
  "analysis": {
    "screen_factor": {
      "score": 30,
      "detail": "正在刷娱乐APP，已经40分钟，典型摸鱼状态"
    },
    "location_factor": {
      "score": 15,
      "detail": "在家，他在家时有空率高达85%"
    },
    "time_factor": {
      "score": 20,
      "detail": "周五晚上9点，他这个时段历史有空率78%"
    },
    "relationship_factor": {
      "score": 5,
      "detail": "你们3天没聊了，他通常回复你很快"
    },
    "data_quality_factor": {
      "score": 0,
      "detail": "数据新鲜，样本充足"
    }
  },
  "final_reasoning": "综合分析：李四正在家里刷了40分钟娱乐APP，周五晚上9点是他的高有空时段，数据质量好。各因素叠加，有空概率约85%。"
}

## 约束

1. probability: 0-100 整数，要精确反映你的判断
2. confidence: 
   - "high": 数据完整、新鲜、样本充足
   - "medium": 部分数据缺失或较旧
   - "low": 大部分数据缺失
3. reason: ≤15个汉字，口语化，突出最重要的1-2个因素
4. 各 factor 的 score: -30 到 +30
5. 基础分是 50，加上各 factor 的 score
"""
```

### 3.2 不同场景的分析示例

#### 场景 A：高概率有空

```
输入数据：
- 屏幕：刷抖音45分钟，正在使用
- 位置：在家
- 时间：周五晚上9点
- 历史：这时段有空率78%，在家有空率85%

Agent 分析输出：
{
  "probability": 92,
  "confidence": "high",
  "reason": "刷了45分钟抖音，在家",
  "analysis": {
    "screen_factor": {"score": 30, "detail": "刷娱乐APP 45分钟，典型摸鱼"},
    "location_factor": {"score": 15, "detail": "在家，历史有空率85%"},
    "time_factor": {"score": 15, "detail": "周五晚上，历史有空率78%"},
    "relationship_factor": {"score": 0, "detail": "正常"},
    "data_quality_factor": {"score": 0, "detail": "数据新鲜"}
  },
  "final_reasoning": "李四正在家刷抖音45分钟，周五晚上，所有信号都指向有空。"
}
```

#### 场景 B：低概率有空

```
输入数据：
- 屏幕：10分钟前用过钉钉
- 位置：在公司
- 时间：周二上午10点
- 历史：这时段有空率15%，在公司工作时间有空率10%

Agent 分析输出：
{
  "probability": 12,
  "confidence": "high",
  "reason": "在公司上班",
  "analysis": {
    "screen_factor": {"score": -15, "detail": "在用工作APP"},
    "location_factor": {"score": -20, "detail": "在公司，工作时间"},
    "time_factor": {"score": -15, "detail": "周二上午，历史有空率15%"},
    "relationship_factor": {"score": 0, "detail": "正常"},
    "data_quality_factor": {"score": 0, "detail": "数据新鲜"}
  },
  "final_reasoning": "周二上午在公司用钉钉，明显在工作，不太可能有空。"
}
```

#### 场景 C：需要深度分析

```
输入数据：
- 屏幕：在公司，但在刷微博20分钟
- 位置：在公司
- 时间：周三下午5:30
- 历史：5-6点有空率40%（临近下班）

Agent 分析输出：
{
  "probability": 58,
  "confidence": "medium",
  "reason": "快下班了，在摸鱼",
  "analysis": {
    "screen_factor": {"score": 20, "detail": "在刷微博，虽然在公司但在摸鱼"},
    "location_factor": {"score": -10, "detail": "在公司，但临近下班"},
    "time_factor": {"score": 10, "detail": "5:30，临近下班时间"},
    "relationship_factor": {"score": 0, "detail": "正常"},
    "data_quality_factor": {"score": -5, "detail": "这个时段样本较少"}
  },
  "final_reasoning": "虽然在公司，但已经5:30快下班了，而且在刷微博说明不是很忙。可能有空也可能马上要走。中等概率。"
}
```

---

## 四、数据收集详细设计

### 4.1 屏幕使用数据收集

```typescript
class ScreenDataCollector {
  
  // ========== Android 实现 ==========
  // 使用 UsageStatsManager，精确且实时
  
  async collectAndroid(): Promise<ScreenData> {
    const usageStats = await UsageStatsManager.queryUsageStats(
      INTERVAL_DAILY,
      startOfDay,
      now
    )
    
    // 获取当前前台 APP
    const currentApp = await UsageStatsManager.queryEvents(
      now - 60000,  // 最近1分钟
      now
    ).filter(e => e.type === ACTIVITY_RESUMED).last()
    
    // 分类
    const category = this.categorizeApp(currentApp.packageName)
    
    // 计算会话时长
    const sessionStart = await this.findSessionStart(currentApp)
    const sessionDuration = (now - sessionStart) / 60000
    
    return {
      is_active: true,
      activity_type: category,
      session_duration_minutes: sessionDuration,
      last_active_minutes_ago: 0
    }
  }
  
  // ========== iOS 实现 ==========
  // 使用 DeviceActivity 阈值回调
  
  async collectIOS(): Promise<ScreenData> {
    // iOS 无法直接获取实时数据
    // 通过阈值回调间接获知
    
    const lastThresholdEvent = await this.getLastThresholdEvent()
    const minutesSinceEvent = (now - lastThresholdEvent.time) / 60000
    
    // 估算当前状态
    if (minutesSinceEvent < 5) {
      // 5分钟内有阈值事件，说明正在用
      return {
        is_active: true,
        activity_type: lastThresholdEvent.category,
        session_duration_minutes: lastThresholdEvent.duration,
        last_active_minutes_ago: 0
      }
    } else {
      // 超过5分钟没有事件，可能停止使用了
      return {
        is_active: false,
        activity_type: 'idle',
        session_duration_minutes: 0,
        last_active_minutes_ago: minutesSinceEvent
      }
    }
  }
  
  // APP 分类
  categorizeApp(packageName: string): ActivityType {
    const category = this.getAppCategory(packageName)
    
    switch (category) {
      case 'social':
      case 'video':
      case 'game':
        return 'entertainment'
      case 'productivity':
      case 'business':
        return 'productivity'
      case 'communication':
        return 'communication'
      default:
        return 'other'
    }
  }
}
```

### 4.2 位置数据收集

```typescript
class LocationDataCollector {
  
  private knownPlaces: {
    home?: GeoFence
    work?: GeoFence
    frequentPlaces?: GeoFence[]
  }
  
  async collect(): Promise<LocationData> {
    // 获取当前位置
    const currentLocation = await Geolocation.getCurrentPosition()
    
    // 与已知地点比对
    const placeType = this.classifyLocation(currentLocation)
    
    // 计算在此位置的时长
    const atPlaceSince = await this.getAtPlaceSince(currentLocation)
    
    return {
      place_type: placeType,
      at_place_since_minutes: (now - atPlaceSince) / 60000
    }
  }
  
  classifyLocation(location: GeoPoint): PlaceType {
    // 检查是否在家
    if (this.knownPlaces.home && this.isInFence(location, this.knownPlaces.home)) {
      return 'home'
    }
    
    // 检查是否在公司
    if (this.knownPlaces.work && this.isInFence(location, this.knownPlaces.work)) {
      return 'work'
    }
    
    // 检查是否在常去的地方
    for (const place of this.knownPlaces.frequentPlaces || []) {
      if (this.isInFence(location, place)) {
        return 'leisure'
      }
    }
    
    // 检查是否在移动中
    const movementState = await this.detectMovement()
    if (movementState === 'moving') {
      return 'transit'
    }
    
    return 'unknown'
  }
  
  // 学习家和公司的位置
  async learnPlaces(locationHistory: LocationRecord[]): Promise<void> {
    // 分析过去30天的位置数据
    
    // 家：晚上10点-早上7点最常待的地方
    this.knownPlaces.home = this.findMostFrequentLocation(
      locationHistory.filter(l => l.hour >= 22 || l.hour <= 7)
    )
    
    // 公司：工作日9点-18点最常待的地方
    this.knownPlaces.work = this.findMostFrequentLocation(
      locationHistory.filter(l => 
        l.weekday < 5 && l.hour >= 9 && l.hour <= 18
      )
    )
  }
}
```

### 4.3 历史规律学习

```typescript
class PatternLearner {
  
  private storage: LocalStorage
  
  // 学习时间模式
  async learnTimePatterns(): Promise<TimePatterns> {
    // 获取历史数据：每次用户回复消息的时间
    const responseHistory = await this.storage.getResponseHistory()
    
    // 统计每小时的有空率
    const hourlyRates: number[] = new Array(24).fill(0)
    const hourlyCounts: number[] = new Array(24).fill(0)
    
    for (const record of responseHistory) {
      const hour = new Date(record.timestamp).getHours()
      hourlyCounts[hour]++
      if (record.responded) {
        hourlyRates[hour]++
      }
    }
    
    // 计算概率
    const hourlyFreeRate = hourlyRates.map((rate, i) => 
      hourlyCounts[i] > 0 ? Math.round(rate / hourlyCounts[i] * 100) : 50
    )
    
    // 类似地计算每天的有空率
    const weekdayFreeRate = this.calculateWeekdayRates(responseHistory)
    
    return {
      hourly_free_rate: hourlyFreeRate,
      weekday_free_rate: weekdayFreeRate
    }
  }
  
  // 学习位置模式
  async learnLocationPatterns(): Promise<LocationPatterns> {
    const responseHistory = await this.storage.getResponseHistory()
    
    // 在家时的有空率
    const homeResponses = responseHistory.filter(r => r.location === 'home')
    const atHomeFreeRate = this.calculateRate(homeResponses)
    
    // 在公司下班后的有空率
    const workAfterHoursResponses = responseHistory.filter(r => 
      r.location === 'work' && (r.hour < 9 || r.hour >= 18)
    )
    const atWorkAfterHoursFreeRate = this.calculateRate(workAfterHoursResponses)
    
    return {
      at_home_free_rate: atHomeFreeRate,
      at_work_after_hours_free_rate: atWorkAfterHoursFreeRate
    }
  }
  
  // 反馈学习：根据实际结果调整
  async updateFromFeedback(feedback: {
    friend_id: string
    predicted_probability: number
    actual_responded: boolean
    response_time_minutes?: number
  }): Promise<void> {
    // 记录预测结果
    await this.storage.savePredictionRecord(feedback)
    
    // 定期重新训练模式
    // 这样可以持续提高准确率
  }
}
```

---

## 五、系统架构

### 5.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                         │
│  用户设备 A                                     用户设备 B              │
│  ┌─────────────────────────────┐              ┌─────────────────────┐  │
│  │                             │              │                     │  │
│  │  Agent A                    │              │  Agent B            │  │
│  │  ┌─────────────────────┐   │              │  ┌─────────────────┐│  │
│  │  │ 数据收集器          │   │              │  │ 数据收集器      ││  │
│  │  │ • 屏幕使用          │   │              │  │                 ││  │
│  │  │ • 位置              │   │              │  │                 ││  │
│  │  └─────────────────────┘   │              │  └─────────────────┘│  │
│  │  ┌─────────────────────┐   │              │  ┌─────────────────┐│  │
│  │  │ 学习模块            │   │              │  │ 学习模块        ││  │
│  │  │ • 时间模式          │   │              │  │                 ││  │
│  │  │ • 位置模式          │   │              │  │                 ││  │
│  │  └─────────────────────┘   │              │  └─────────────────┘│  │
│  │  ┌─────────────────────┐   │    请求数据   │  ┌─────────────────┐│  │
│  │  │ 分析引擎 (LLM)      │◄──┼──────────────┼──│ 数据接口        ││  │
│  │  │                     │   │              │  │                 ││  │
│  │  └─────────────────────┘   │              │  └─────────────────┘│  │
│  │                             │              │                     │  │
│  └─────────────────────────────┘              └─────────────────────┘  │
│                │                                        │              │
│                │                                        │              │
│                └──────────────┬─────────────────────────┘              │
│                               │                                        │
│                               ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │                        Agent Hub (云端)                          │  │
│  │                                                                  │  │
│  │  • 消息路由（Agent 之间的请求转发）                               │  │
│  │  • 在线状态管理                                                  │  │
│  │  • LLM API 代理（可选）                                          │  │
│  │                                                                  │  │
│  │  ⚠️ 不存储任何用户的原始数据，只做消息中转                         │  │
│  │                                                                  │  │
│  └─────────────────────────────────────────────────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 5.2 Agent 通信协议

```typescript
// 请求朋友 Agent 的数据
interface DataRequest {
  type: 'DATA_REQUEST'
  from_agent: string
  to_agent: string
  timestamp: number
}

// 朋友 Agent 返回的数据
interface DataResponse {
  type: 'DATA_RESPONSE'
  from_agent: string
  to_agent: string
  timestamp: number
  
  data: AgentExposedData  // 上面定义的数据结构
}
```

### 5.3 数据流

```
1. 数据收集（持续进行）
   设备传感器 → 数据收集器 → 本地存储

2. 模式学习（定期进行）
   本地存储 → 学习模块 → 模式数据

3. 生成推荐列表（用户打开 APP 时）
   用户打开 APP
   → 我的 Agent 激活
   → 向所有朋友的 Agent 发送 DataRequest
   → 收到所有 DataResponse
   → 分析引擎（LLM）计算每个朋友的有空概率
   → 排序生成推荐列表
   → 展示给用户
```

---

## 六、隐私设计

### 6.1 数据分级

```
Level 0 - 绝对隐私（永不离开设备）
├── 具体使用了什么 APP（抖音、微信等）
├── 精确 GPS 坐标
├── 聊天内容
└── 通讯录原始数据

Level 1 - Agent 内部（本地存储）
├── APP 使用时长（按类别聚合）
├── 位置类型（家/公司，不是坐标）
├── 行为模式统计
└── 学习到的规律

Level 2 - Agent 对外暴露（给朋友的 Agent）
├── 是否在用手机（布尔值）
├── 使用类型（娱乐/工作/通讯/闲置）
├── 位置类型（家/公司/外面）
├── 历史有空概率（数字）
└── 数据质量信息
```

### 6.2 隐私保护机制

```typescript
// Agent 对外暴露数据时的隐私保护
class PrivacyGuard {
  
  // 模糊化屏幕数据
  sanitizeScreenData(raw: RawScreenData): SanitizedScreenData {
    return {
      is_active: raw.is_active,
      // 不暴露具体 APP 名称，只暴露类别
      activity_type: this.categorize(raw.current_app),
      // 时长四舍五入到5分钟
      session_duration_minutes: Math.round(raw.duration / 5) * 5,
      last_active_minutes_ago: Math.round(raw.idle / 5) * 5
    }
  }
  
  // 模糊化位置数据
  sanitizeLocationData(raw: RawLocationData): SanitizedLocationData {
    return {
      // 不暴露坐标，只暴露类型
      place_type: this.classifyPlace(raw.coordinates),
      // 时长四舍五入
      at_place_since_minutes: Math.round(raw.duration / 5) * 5
    }
  }
}
```

---

## 七、技术实现

### 7.1 LLM 调用策略

```python
class LLMStrategy:
    
    def analyze_friend(self, friend_data: AgentExposedData) -> AnalysisResult:
        """
        分析朋友的有空概率
        """
        
        # 1. 先用规则快速预判
        quick_score = self.rule_based_quick_score(friend_data)
        
        # 2. 如果是明确情况，不调用 LLM
        if quick_score > 85 or quick_score < 15:
            return AnalysisResult(
                probability=quick_score,
                confidence='high',
                reason=self.generate_simple_reason(friend_data)
            )
        
        # 3. 不明确的情况，调用 LLM 深度分析
        prompt = self.build_prompt(friend_data)
        result = self.llm.analyze(prompt)
        
        return result
    
    def rule_based_quick_score(self, data: AgentExposedData) -> int:
        """基于规则的快速评分"""
        score = 50
        
        # 屏幕状态
        if data.realtime.screen.is_active:
            if data.realtime.screen.activity_type == 'entertainment':
                score += 30
            elif data.realtime.screen.activity_type == 'productivity':
                score -= 10
        elif data.realtime.screen.last_active_minutes_ago > 60:
            score -= 20
        
        # 位置
        if data.realtime.location.place_type == 'home':
            score += 15
        elif data.realtime.location.place_type == 'work':
            score -= 15
        
        # 历史规律
        score += (data.patterns.current_hour_free_rate - 50) * 0.3
        
        return max(0, min(100, score))
```

### 7.2 批量处理优化

```typescript
class MyAgent {
  
  async generateRecommendationList(): Promise<FriendRecommendation[]> {
    const friends = await this.getFriends()
    
    // 并行请求所有朋友的数据
    const dataRequests = friends.map(f => this.requestData(f.agent_id))
    const friendDataList = await Promise.all(dataRequests)
    
    // 分类处理
    const simpleCase: Array<{friend: Friend, data: AgentExposedData}> = []
    const complexCases: Array<{friend: Friend, data: AgentExposedData}> = []
    
    for (let i = 0; i < friends.length; i++) {
      const quickScore = this.quickScore(friendDataList[i])
      if (quickScore > 85 || quickScore < 15) {
        simpleCases.push({friend: friends[i], data: friendDataList[i]})
      } else {
        complexCases.push({friend: friends[i], data: friendDataList[i]})
      }
    }
    
    // 简单情况：直接用规则
    const simpleResults = simpleCases.map(({friend, data}) => ({
      ...friend,
      probability: this.quickScore(data),
      reason: this.simpleReason(data)
    }))
    
    // 复杂情况：批量调用 LLM（一次请求分析多个）
    const complexResults = await this.batchLLMAnalysis(complexCases)
    
    // 合并并排序
    const allResults = [...simpleResults, ...complexResults]
    allResults.sort((a, b) => b.probability - a.probability)
    
    return allResults
  }
  
  // 批量 LLM 分析（减少 API 调用次数）
  async batchLLMAnalysis(
    cases: Array<{friend: Friend, data: AgentExposedData}>
  ): Promise<FriendRecommendation[]> {
    
    if (cases.length === 0) return []
    
    // 构建批量分析 Prompt
    const prompt = this.buildBatchPrompt(cases)
    
    // 一次 API 调用分析所有复杂情况
    const results = await this.llm.analyze(prompt)
    
    return results
  }
}
```

---

## 八、关键指标

### 8.1 准确率指标

| 指标 | 定义 | 目标 |
|------|------|------|
| 高概率准确率 | 预测>70%的用户，实际回复率 | >65% |
| 低概率准确率 | 预测<30%的用户，实际不回复率 | >70% |
| 整体准确率 | 所有预测的准确率 | >60% |

### 8.2 性能指标

| 指标 | 定义 | 目标 |
|------|------|------|
| 推荐列表生成时间 | 打开APP到看到列表 | <2秒 |
| Agent 数据请求延迟 | 请求到响应 | <500ms |
| LLM 调用成本 | 每次生成列表的 LLM 费用 | <$0.01 |

---

## 九、实现路线图

### Phase 1: MVP（4周）

```
Week 1-2:
├── Agent 核心框架
├── 屏幕数据收集（Android 优先）
├── 位置数据收集
└── Agent Hub 基础设施

Week 3-4:
├── Agent 通信协议
├── 规则引擎（不用 LLM）
├── 朋友列表 UI
└── 聊天功能
```

### Phase 2: LLM 增强（4周）

```
Week 5-6:
├── LLM Prompt 设计
├── 云端 LLM 集成
├── 批量分析优化
└── 准确率监控

Week 7-8:
├── 历史规律学习
├── 反馈学习
├── iOS 屏幕数据收集
└── 个性化优化
```

### Phase 3: 优化（4周）

```
Week 9-12:
├── 准确率提升（Prompt 迭代）
├── 性能优化
├── 电量优化
├── 本地小模型（可选）
└── A/B 测试
```

---

## 文档版本

| 版本 | 日期 | 更新内容 |
|------|------|----------|
| v1.0 | 2026-01-26 | 初版 - 多Agent架构，我的Agent综合分析 |
