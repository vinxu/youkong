# 事件：新增多时段发布功能

> 创建时间: 2025-01-25  
> 状态: 进行中

---

## 变更类型

API 新增 + UI 新增

---

## 需求描述

用户可以一次发布多个时间段的有空状态，而不是只能发布单个时段。

**示例**：
- 今晚 19:00-21:00
- 明天下午 14:00-17:00

---

## 契约变更

### API 变更

```yaml
# 新增字段: timeSlots (替代单一的 startTime/endTime)
CreateAvailabilityRequest:
  type: object
  properties:
    timeSlots:           # 新增
      type: array
      items:
        type: object
        properties:
          startTime:
            type: integer
          endTime:
            type: integer
      minItems: 1
      maxItems: 5        # 最多5个时段
    location:
      $ref: '#/components/schemas/AvailabilityLocation'
    visibleCircleIds:
      type: array
      items:
        type: string
```

### 数据模型变更

```typescript
// 新增 TimeSlot 类型
interface TimeSlot {
  startTime: number
  endTime: number
}

// 修改 CreateAvailabilityRequest
interface CreateAvailabilityRequest {
  timeSlots: TimeSlot[]  // 替代 startTime/endTime
  location: AvailabilityLocation
  visibleCircleIds: string[]
}
```

---

## 影响范围

- [x] 后端：修改 API 接口，支持多时段存储
- [x] iOS：发布流程 UI 支持添加/删除时段
- [x] Android：发布流程 UI 支持添加/删除时段
- [x] 小程序：发布流程 UI 支持添加/删除时段

---

## 各端任务

| 端 | 任务 | 负责人 | 状态 |
|---|---|---|---|
| 后端 | 修改 /availabilities POST 接口，支持 timeSlots | - | ⏳ 进行中 |
| 后端 | 数据库 schema 支持多时段 | - | ⏳ 进行中 |
| iOS | TimeSelectionView 支持多时段选择 | - | 🔲 待开始 |
| Android | TimeSelectionStep 支持多时段选择 | - | 🔲 待开始 |
| 小程序 | publish/time 页面支持多时段选择 | - | 🔲 待开始 |

---

## UI 设计参考

```
┌─────────────────────────────────────────┐
│  选择时间                               │
├─────────────────────────────────────────┤
│                                         │
│  时段 1                          [删除]  │
│  ┌─────────────────────────────────┐   │
│  │ 今晚 19:00 - 21:00              │   │
│  └─────────────────────────────────┘   │
│                                         │
│  时段 2                          [删除]  │
│  ┌─────────────────────────────────┐   │
│  │ 明天 14:00 - 17:00              │   │
│  └─────────────────────────────────┘   │
│                                         │
│  [ + 添加时段 ] (最多5个)               │
│                                         │
│  ────────────────────────────────────  │
│                                         │
│  [ 下一步：选择地点 ]                    │
│                                         │
└─────────────────────────────────────────┘
```

---

## 完成标准

- [ ] 后端 API 支持 timeSlots 数组
- [ ] 前端可以添加/删除多个时段
- [ ] 最多支持5个时段
- [ ] 首页 Feed 正确显示多时段
- [ ] 单元测试覆盖

---

## 备注

后端先行，完成后通知前端开始联调。
