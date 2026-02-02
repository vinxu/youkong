# Holmes 2.0 智能推断框架 - API 文档

## 概述

Holmes 2.0 是基于研究的智能推断框架，通过多层语义建模和 LLM 创意叙事生成，推测用户"此刻"正在做什么。

### 核心理念

```
❌ 旧方法：规则匹配 → 预设状态（工作中/休息中/通勤中）
✅ 新方法：多维特征 → LLM 自由推断 → 生动叙事
```

### 研究基础

1. **Digital Phenotyping（数字表型学）** - 从智能手机数据推断社交上下文
2. **Affective Computing（情感计算）** - 使用连续维度表示情绪（效价 × 唤醒度）
3. **Context-Aware Computing（上下文感知）** - 场所语义比 GPS 坐标更有意义

---

## 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                    Holmes 2.0 推断框架                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Layer 1: 信号收集 (Signal Collection)                          │
│  ┌───────────┬───────────┬───────────┬───────────┐              │
│  │ 位置信号  │ 运动信号  │ 设备信号  │ 时间信号  │              │
│  │ GPS/场所  │ 加速度计  │ 屏幕/电池 │ 日历/时段 │              │
│  └───────────┴───────────┴───────────┴───────────┘              │
│                           ↓                                     │
│  Layer 2: 语义上下文建模 (Semantic Context Modeling)             │
│  ┌────────────────────────────────────────────────┐              │
│  │  空间语义: 在什么样的地方（不是具体地址）      │              │
│  │  时间语义: 什么时间点、持续多久               │              │
│  │  活动语义: 身体在做什么、手机在做什么         │              │
│  │  能量状态: 此刻的能量水平                     │              │
│  └────────────────────────────────────────────────┘              │
│                           ↓                                     │
│  Layer 3: 记忆融合 + 异常检测 (Memory + Anomaly)                 │
│  ┌────────────────────────────────────────────────┐              │
│  │  CoreMemory: 这个人的行为模式、偏好、习惯       │              │
│  │  Anomaly Detection: 是否偏离常规模式            │              │
│  └────────────────────────────────────────────────┘              │
│                           ↓                                     │
│  Layer 4: 叙事生成 (Narrative Generation)                        │
│  ┌────────────────────────────────────────────────┐              │
│  │  LLM 自由推断，不限定预设状态                    │              │
│  │  输出: 生动的故事性描述 + Emoji + 心情维度       │              │
│  └────────────────────────────────────────────────┘              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## API 接口

### Holmes 2.0 流式状态分析

**Endpoint**: `POST /api/v1/agent/status/stream2`

**认证**: 需要 Bearer Token

**Content-Type**: `application/json`

**Accept**: `text/event-stream`

#### 请求参数

```json
{
  "screen": {
    "is_active": true,
    "activity_type": "entertainment",
    "session_duration_minutes": 30,
    "last_active_minutes_ago": 0
  },
  "location": {
    "place_type": "home",
    "at_place_since_minutes": 120
  },
  "extended_location": {
    "place_type": "home",
    "place_name": "家",
    "at_place_since_minutes": 120,
    "latitude": 39.9042,
    "longitude": 116.4074
  },
  "battery": {
    "battery_level": 80,
    "battery_state": "unplugged",
    "is_charging": false
  },
  "mode": {
    "is_low_power_mode": false,
    "is_focus_mode_on": false
  },
  "connection": {
    "is_headphones_connected": true,
    "network_type": "wifi"
  },
  "display": {
    "screen_brightness": 0.6
  },
  "calendar": {
    "has_current_event": false,
    "current_event_title": null,
    "event_end_minutes": 0,
    "next_event_in_minutes": -1,
    "today_remaining_count": 2
  },
  "movement": {
    "is_moving": false,
    "movement_type": "stationary",
    "steps_today": 3500,
    "steps_last_hour": 120,
    "stationary_minutes": 45
  }
}
```

#### SSE 事件类型

| 事件类型 | 说明 | 示例内容 |
|---------|------|---------|
| `phase` | 阶段标题 | `"📡 收集线索..."` |
| `clue` | 线索收集 | `"├─ 时间: 周日 下午3点42分"` |
| `context` | 语义上下文 | `"├─ 空间: 私密空间 (安静)"` |
| `anomaly` | 异常检测 | `"├─ ⚠️ 周末在公司，可能有紧急工作"` |
| `narrative` | 叙事推理（流式） | `"周日下午，典型的宅家时光..."` |
| `thinking` | 思考过程 | `"从静止状态和长时间娱乐来看..."` |
| `conclusion` | 结论输出 | `"├─ 🛋️ 窝在家里葛优躺，刷着手机享受周末"` |
| `done` | 完成 | 包含完整 result 对象 |
| `error` | 错误 | 错误信息 |

#### SSE 事件格式

```
data: {"type":"phase","phase":"collecting","content":"📡 收集线索..."}

data: {"type":"clue","content":"├─ 时间: 周日 下午3点42分"}

data: {"type":"context","content":"├─ 空间: 私密空间 (安静)"}

data: {"type":"anomaly","content":"└─ 无异常，行为符合常规模式"}

data: {"type":"narrative","content":"周日下午，典型的宅家时光..."}

data: {"type":"done","result":{...}}
```

#### 最终结果结构

```json
{
  "type": "done",
  "result": {
    "raw_data": {
      "timestamp": "2026-02-02T15:42:00Z",
      "weekday": "周日",
      "time_period": "下午3点42分",
      "is_weekend": true,
      "place_type": "home",
      "at_place_since_minutes": 120,
      "screen_active": true,
      "activity_type": "entertainment",
      "screen_duration_mins": 30
    },
    "context": {
      "space": {
        "nature": "私密空间",
        "vibe": "安静",
        "social": "独处"
      },
      "time": {
        "phase": "放松期",
        "rhythm": "休闲节奏",
        "continuity": "进行中"
      },
      "activity": {
        "body_state": "静态",
        "mind_state": "消遣",
        "engagement": "深度投入"
      },
      "energy": {
        "physical": "正常",
        "mental": "放松",
        "social": "封闭"
      }
    },
    "anomalies": [],
    "creative": {
      "narrative": "周日下午，典型的宅家时光。从静止状态和长时间的娱乐内容来看，应该是窝在某个舒适的角落——可能是沙发或床上。45分钟的持续使用说明已经进入放松模式，不是随便刷刷，而是在认真"躺平"。心情应该是惬意的，享受这难得的无所事事。",
      "scene": "窝在家里葛优躺，刷着手机享受周末",
      "emoji": "🛋️",
      "mood": {
        "valence": 0.6,
        "arousal": 0.2,
        "openness": 0.3
      },
      "confidence": "high",
      "basis": ["周末在家符合习惯", "长时间娱乐内容", "静止状态"],
      "generated_at": 1738500120000
    },
    "result": {
      "available": true,
      "probability": 58,
      "confidence": "high",
      "summary": "窝在家里葛优躺，刷着手机享受周末",
      "emoji": "🛋️"
    },
    "generated_at": 1738500120000
  }
}
```

---

## 数据模型

### 语义上下文 (SemanticContext)

```go
type SemanticContext struct {
    Space    *SpaceSemantic    `json:"space"`
    Time     *TimeSemantic     `json:"time"`
    Activity *ActivitySemantic `json:"activity"`
    Energy   *EnergyLevel      `json:"energy"`
}
```

### 空间语义 (SpaceSemantic)

| 字段 | 说明 | 可能值 |
|-----|------|-------|
| `nature` | 空间性质 | `私密空间` / `公共空间` / `专业空间` / `移动中` |
| `vibe` | 氛围 | `安静` / `喧嚣` / `专业` / `休闲` / `户外` |
| `social` | 社交环境 | `独处` / `可能有他人` / `社交场合` / `陌生人群` |

### 时间语义 (TimeSemantic)

| 字段 | 说明 | 可能值 |
|-----|------|-------|
| `phase` | 时间阶段 | `苏醒期` / `高效期（上午）` / `休整期（午间）` / `高效期（下午）` / `放松期` / `入睡期` |
| `rhythm` | 生活节奏 | `工作节奏` / `休闲节奏` / `过渡期（通勤时段）` / `个人时间` |
| `continuity` | 状态持续性 | `刚开始` / `进行中` / `持续较久` |

### 活动语义 (ActivitySemantic)

| 字段 | 说明 | 可能值 |
|-----|------|-------|
| `body_state` | 身体状态 | `静态` / `长时间静态` / `轻度活动` / `运动中` / `移动中` |
| `mind_state` | 心智状态 | `专注` / `消遣` / `社交` / `休息` / `闲置` |
| `engagement` | 投入程度 | `深度投入` / `浅层互动` / `间歇使用` / `闲置` |

### 能量状态 (EnergyLevel)

| 字段 | 说明 | 可能值 |
|-----|------|-------|
| `physical` | 身体能量 | `充沛` / `活跃` / `正常` / `可能疲惫` |
| `mental` | 精神状态 | `专注` / `清醒` / `放松` / `平静` / `低迷` |
| `social` | 社交意愿 | `开放` / `中性` / `封闭` |

### 心情向量 (MoodVector)

基于情感计算研究，使用连续维度而非离散类别：

| 字段 | 说明 | 范围 |
|-----|------|-----|
| `valence` | 效价（积极-消极） | -1.0 ~ 1.0 |
| `arousal` | 唤醒度（激动-平静） | 0.0 ~ 1.0 |
| `openness` | 社交开放度 | 0.0 ~ 1.0 |

**心情维度模型**：

```
        激动 (arousal=1)
           ↑
    兴奋   |   焦虑
    开心   |   紧张
           |
消极 ←────●────→ 积极
(valence=-1) (valence=1)
    悲伤   |   平和
    疲惫   |   满足
           ↓
        平静 (arousal=0)
```

### 异常标记 (Anomaly)

| 字段 | 说明 |
|-----|------|
| `type` | 异常类型：`unusual_location` / `unusual_time` / `behavior_change` |
| `detail` | 异常描述，如 `"通常这时候已经休息了，今晚还在活跃"` |

### 创意叙事结果 (HolmesCreativeResult)

| 字段 | 说明 |
|-----|------|
| `narrative` | 叙事推理过程（100-200字） |
| `scene` | 场景描述（15-25字，用于展示） |
| `emoji` | 最传神的 emoji |
| `mood` | 心情向量 |
| `confidence` | 置信度：`high` / `medium` / `low` |
| `basis` | 推断依据列表 |
| `generated_at` | 生成时间戳（毫秒） |

---

## 有空概率计算

Holmes 2.0 根据心情向量自动计算有空概率：

```go
// 基础分数 = 社交开放度 * 60 + 效价 * 20
probability := int(openness*60 + (valence+1)*10)

// 唤醒度过高或过低都会降低概率
if arousal > 0.7 || arousal < 0.2 {
    probability -= 10
}
```

---

## 兼容性

Holmes 2.0 完全兼容 Holmes 1.0 的响应格式：

- 旧版接口 `/api/v1/agent/status/stream` 继续可用
- `result` 字段保持相同结构（`available`, `probability`, `confidence`, `summary`, `emoji`）
- 新增 `context`, `anomalies`, `creative` 字段为可选

---

## 示例

### 示例 1：周末在家刷手机

**输入**:
- 周日下午 3:30
- 在家，静止 2 小时
- 屏幕活跃，娱乐内容 45 分钟
- 戴着耳机

**输出**:
```json
{
  "scene": "窝在家里葛优躺，刷着手机享受周末",
  "emoji": "🛋️",
  "mood": {
    "valence": 0.6,
    "arousal": 0.2,
    "openness": 0.3
  },
  "confidence": "high"
}
```

### 示例 2：工作日深夜还在活跃

**输入**:
- 周三凌晨 1:00
- 在家，屏幕活跃 90 分钟
- 记忆显示"通常早睡"

**异常检测**:
```json
{
  "type": "unusual_time",
  "detail": "通常这时候已经休息了，今晚还在活跃"
}
```

**输出**:
```json
{
  "scene": "深夜还没睡，可能有心事或在追剧",
  "emoji": "🌙",
  "mood": {
    "valence": 0.1,
    "arousal": 0.3,
    "openness": 0.2
  },
  "confidence": "medium"
}
```

### 示例 3：周末在公司

**输入**:
- 周六上午 10:00
- 在公司，静止 1 小时
- 屏幕活跃，工作内容

**异常检测**:
```json
{
  "type": "unusual_location",
  "detail": "周末在公司，可能有紧急工作或加班"
}
```

**输出**:
```json
{
  "scene": "周末加班中，专注处理工作事务",
  "emoji": "💼",
  "mood": {
    "valence": -0.1,
    "arousal": 0.5,
    "openness": 0.2
  },
  "confidence": "high"
}
```

---

## 错误处理

### SSE 错误事件

```
data: {"type":"error","content":"LLM 调用失败，使用规则引擎降级"}
```

### HTTP 错误码

| 状态码 | 说明 |
|-------|------|
| 401 | 未授权，Token 无效或过期 |
| 400 | 请求参数错误 |
| 500 | 服务器内部错误 |

### 降级策略

当 LLM 不可用时，自动降级到规则引擎：
1. 发送 `thinking` 事件通知前端
2. 使用 Holmes 1.0 规则引擎计算结果
3. 返回基础的状态描述（如"在家"、"工作中"）

---

## 版本历史

| 版本 | 日期 | 变更 |
|-----|------|------|
| 2.0.0 | 2026-02-02 | 新增语义上下文建模、异常检测、心情向量、创意叙事生成 |
| 1.0.0 | 2025-12-01 | 初始版本，基于规则的福尔摩斯推理 |
