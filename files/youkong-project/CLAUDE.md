# CLAUDE.md - 有空 项目指南

> 这是给 AI 代码助手（如 Claude Code）的项目说明文件

---

## 项目概述

**有空** 是一个帮助用户找到可能有空的朋友的社交应用。

**核心功能**：打开 APP → 看到朋友按"有空概率"排序的列表 → 点击进入聊天

**技术特点**：每个用户有一个 AI Agent，Agent 之间交换数据，用 LLM 计算朋友的有空概率

---

## 核心架构

### Agent 系统

```
每个用户都有一个专属 Agent：
- Agent 收集主人的屏幕使用数据、位置数据
- Agent 学习主人的行为规律
- 当用户打开 APP，自己的 Agent 向所有朋友的 Agent 请求数据
- 自己的 Agent 用 LLM 综合分析，生成有空推荐列表
```

### 三个核心权限

| 权限 | 用途 | 实现 |
|------|------|------|
| 屏幕使用时间 | 判断用户是否在用手机 | Android: UsageStatsManager / iOS: DeviceActivity |
| 地理位置 | 判断用户在哪里 | FusedLocation / CoreLocation |
| 通讯录 | 找到用户的朋友 | 标准通讯录 API |

---

## 项目结构

```
youkong-project/
├── CLAUDE.md                        # 本文件 - AI 代码助手指南
├── docs/
│   ├── youkong-product-doc.md       # 产品需求文档
│   ├── youkong-agent-architecture.md # Agent 架构详细设计
│   └── youkong-dev-spec.md          # 开发技术规格（核心参考文档）
├── mobile/
│   ├── android/                     # Android 客户端
│   └── ios/                         # iOS 客户端
├── backend/
│   ├── api/                         # API 服务
│   ├── agent-hub/                   # Agent 通信中心
│   └── llm-service/                 # LLM 分析服务
└── shared/
    └── types/                       # 共享类型定义
```

---

## 关键类型定义

### Agent 暴露数据

```typescript
interface AgentExposedData {
  realtime: {
    screen: {
      is_active: boolean
      activity_type: 'entertainment' | 'productivity' | 'communication' | 'idle'
      session_duration_minutes: number
      last_active_minutes_ago: number
    }
    location: {
      place_type: 'home' | 'work' | 'leisure' | 'transit' | 'unknown'
      at_place_since_minutes: number
    }
  }
  patterns: {
    current_hour_free_rate: number      // 0-100
    current_weekday_free_rate: number   // 0-100
    at_home_free_rate: number           // 0-100
    avg_response_time_minutes: number
    response_rate: number               // 0-100
  }
  data_quality: {
    screen_data_age_seconds: number
    location_data_age_seconds: number
    patterns_sample_size: number
  }
}
```

### 推荐结果

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

## 颜色系统

```typescript
// 概率 → 颜色映射
80-100% → 深绿 #22C55E
60-79%  → 浅绿 #86EFAC
40-59%  → 黄色 #FACC15
20-39%  → 橙色 #FB923C
0-19%   → 红色 #EF4444
无数据   → 灰色 #9CA3AF
```

---

## LLM Prompt 核心逻辑

分析朋友有空概率时，考虑以下因素：

1. **屏幕状态**（权重最高）
   - 刷娱乐APP超过10分钟 → +30分
   - 刷工作APP → -10分
   - 手机闲置超过1小时 → -20分

2. **位置**
   - 在家 → +15分
   - 在公司+工作时间 → -20分

3. **时间规律**
   - 当前时段历史有空率高 → +15分
   - 周五晚上/周末 → +10分

**基础分 50 分，加上各因素得分，限制在 0-100 范围内**

---

## API 端点

```
POST /api/agent/status              # Agent 状态上报
GET  /api/friends/free-probability  # 获取好友有空列表
POST /api/agent/query               # Agent 间数据请求
GET  /api/conversations/{id}/messages   # 获取聊天消息
POST /api/conversations/{id}/messages   # 发送消息
```

---

## 开发注意事项

### Android 屏幕使用数据
- 需要 `PACKAGE_USAGE_STATS` 权限
- 用户需要在设置中手动授权
- 使用 `UsageStatsManager` 获取数据

### iOS 屏幕使用数据
- 使用 `DeviceActivity` API（iOS 16+）
- 数据被严格沙盒隔离，无法直接读取
- 需要通过阈值回调间接获知使用时长
- 精度约 5-10 分钟

### 隐私保护
- 不暴露具体 APP 名称，只暴露类别
- 不暴露精确坐标，只暴露位置类型（家/公司/外面）
- 时长数据四舍五入到 5 分钟

---

## 常见开发任务

### 添加新的数据收集器
参考 `docs/youkong-dev-spec.md` 中的 DataCollector 实现

### 修改 LLM Prompt
参考 `docs/youkong-dev-spec.md` 第六节 "LLM 分析 Prompt"

### 添加新 API
遵循现有 API 格式，参考 `docs/youkong-dev-spec.md` 第七节

### UI 颜色调整
修改颜色常量，参考 `docs/youkong-dev-spec.md` 第四节 "颜色系统"

---

## 参考文档

- `docs/youkong-product-doc.md` - 完整产品需求
- `docs/youkong-agent-architecture.md` - Agent 架构详细设计
- `docs/youkong-dev-spec.md` - **开发技术规格（最重要）**
