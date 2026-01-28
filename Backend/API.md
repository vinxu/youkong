# YouKong API 接口文档

## 基础信息

| 项目 | 值 |
|------|-----|
| Base URL | `https://api.youkong.app/api/v1` |
| 认证方式 | Bearer Token (JWT) |
| 内容类型 | `application/json` |

## 统一响应格式

```typescript
interface ApiResponse<T> {
  code: number      // 0=成功, 非0=失败
  message: string   // 响应消息
  data?: T          // 响应数据（成功时）
}
```

## 错误码

| 错误码 | 说明 | HTTP状态码 |
|--------|------|-----------|
| 0 | 成功 | 200 |
| 1001 | 参数错误 | 400 |
| 1002 | 未授权 | 401 |
| 1003 | Token已过期 | 401 |
| 1004 | 资源不存在 | 404 |
| 1005 | 无权限 | 403 |
| 5000 | 服务器内部错误 | 500 |

## 请求头

需要认证的接口必须携带：
```
Authorization: Bearer <token>
```

---

## 一、认证模块 `/auth`

### 1.1 发送验证码

**POST** `/auth/sms/send`

**请求参数**
```typescript
interface SendSMSRequest {
  phone: string  // 手机号，11位
}
```

**响应数据**
```typescript
interface SendSMSResponse {
  message: string  // "验证码已发送"
}
```

**示例**
```json
// 请求
{ "phone": "13800138000" }

// 响应
{ "code": 0, "message": "成功", "data": { "message": "验证码已发送" } }
```

---

### 1.2 验证码登录/注册

**POST** `/auth/sms/verify`

**请求参数**
```typescript
interface VerifySMSRequest {
  phone: string  // 手机号，11位
  code: string   // 验证码，6位数字
}
```

**响应数据**
```typescript
interface LoginResult {
  token: string      // JWT Token
  user: User         // 用户信息
  isNewUser: boolean // 是否新用户
}

interface User {
  id: string
  phone: string
  nickname: string
  avatar?: string
  createdAt: string   // ISO 8601 时间格式
  updatedAt: string
}
```

**示例**
```json
// 请求
{ "phone": "13800138000", "code": "123456" }

// 响应
{
  "code": 0,
  "message": "成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "phone": "13800138000",
      "nickname": "用户8000",
      "avatar": "",
      "createdAt": "2024-01-15T10:30:00Z",
      "updatedAt": "2024-01-15T10:30:00Z"
    },
    "isNewUser": true
  }
}
```

---

### 1.3 刷新Token

**POST** `/auth/refresh`

**请求参数**
```typescript
interface RefreshTokenRequest {
  token: string  // 旧的 Token
}
```

**响应数据**
```typescript
interface RefreshTokenResponse {
  token: string  // 新的 Token
}
```

---

## 二、用户模块 `/users`

> 以下接口需要认证

### 2.1 获取当前用户

**GET** `/users/me`

**响应数据**
```typescript
interface User {
  id: string
  phone: string
  nickname: string
  avatar?: string
  createdAt: string
  updatedAt: string
  lastActiveAt?: string
}
```

---

### 2.2 更新用户信息

**PUT** `/users/me`

**请求参数**
```typescript
interface UpdateUserRequest {
  nickname?: string  // 昵称，1-20字符
  avatar?: string    // 头像URL
}
```

**响应数据**
```typescript
// 返回更新后的 User 对象
```

---

### 2.3 获取我的邀请海报

**GET** `/users/me/poster`

> 返回当前用户的固定邀请海报图片（PNG），每个用户有唯一固定的邀请码

**响应**
| Header | 值 |
|--------|-----|
| Content-Type | image/png |
| Content-Disposition | inline; filename="my_invite_poster.png" |

**说明**
- 邀请码为用户ID前8位（去掉横杠），固定不变
- 海报包含：产品介绍、用户头像昵称、二维码、域名
- 图片尺寸: 750x1334

---

### 2.4 获取我的邀请信息

**GET** `/users/me/invite`

> 获取当前用户的固定邀请码和邀请链接

**响应数据**
```typescript
interface MyInviteInfo {
  code: string       // 邀请码（用户ID前8位）
  inviteUrl: string  // 完整邀请链接
}
```

**示例响应**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "code": "a1b2c3d4",
    "inviteUrl": "https://playa.cn/i/a1b2c3d4"
  }
}
```

---

### 2.5 获取用户信息

**GET** `/users/:id`

**路径参数**
| 参数 | 说明 |
|------|------|
| id | 用户ID |

**响应数据**
```typescript
interface UserProfile {
  id: string
  nickname: string
  avatar?: string
}
```

---

### 2.6 搜索用户

**GET** `/users/search?keyword=xxx`

**查询参数**
| 参数 | 必填 | 说明 |
|------|------|------|
| keyword | 是 | 搜索关键词 |

**响应数据**
```typescript
type SearchUsersResponse = UserProfile[]
```

---

## 三、圈子模块 `/circles`

> 以下接口需要认证

### 3.1 获取我的圈子

**GET** `/circles`

**响应数据**
```typescript
interface CircleDetail {
  id: string
  name: string         // 圈子名称，最多10字
  emoji: string        // 圈子图标
  color: string        // 颜色，如 "#EC4899"
  ownerId: string      // 创建者ID
  memberCount: number  // 成员数量
  createdAt: string
}

type GetCirclesResponse = CircleDetail[]
```

**示例响应**
```json
{
  "code": 0,
  "message": "成功",
  "data": [
    {
      "id": "circle-001",
      "name": "闺蜜",
      "emoji": "💕",
      "color": "#EC4899",
      "ownerId": "user-001",
      "memberCount": 5,
      "createdAt": "2024-01-10T08:00:00Z"
    },
    {
      "id": "circle-002",
      "name": "同事",
      "emoji": "💼",
      "color": "#3B82F6",
      "ownerId": "user-001",
      "memberCount": 12,
      "createdAt": "2024-01-12T09:00:00Z"
    }
  ]
}
```

---

### 3.2 创建圈子

**POST** `/circles`

**请求参数**
```typescript
interface CreateCircleRequest {
  name: string   // 圈子名称，必填，最多20字符
  emoji: string  // 圈子图标，必填
  color: string  // 颜色代码，必填，如 "#EC4899"
}
```

**响应数据**
```typescript
interface Circle {
  id: string
  name: string
  emoji: string
  color: string
  ownerId: string
  createdAt: string
  updatedAt: string
}
```

**预设颜色**
```typescript
const circleColors = [
  { name: 'Pink', value: '#EC4899' },
  { name: 'Orange', value: '#F97316' },
  { name: 'Blue', value: '#3B82F6' },
  { name: 'Green', value: '#22C55E' },
  { name: 'Purple', value: '#8B5CF6' },
  { name: 'Yellow', value: '#EAB308' },
]
```

---

### 3.3 获取圈子详情

**GET** `/circles/:id`

**路径参数**
| 参数 | 说明 |
|------|------|
| id | 圈子ID |

**响应数据**
```typescript
interface CircleDetail {
  id: string
  name: string
  emoji: string
  color: string
  ownerId: string
  memberCount: number
  members: UserProfile[]  // 成员列表
  createdAt: string
}
```

---

### 3.4 更新圈子

**PUT** `/circles/:id`

> 仅圈主可操作

**请求参数**
```typescript
interface UpdateCircleRequest {
  name: string
  emoji: string
  color: string
}
```

**响应数据**
```typescript
// 返回更新后的 Circle 对象
```

---

### 3.5 删除圈子

**DELETE** `/circles/:id`

> 仅圈主可操作

**响应数据**
```typescript
interface DeleteResponse {
  message: string  // "删除成功"
}
```

---

### 3.6 添加成员

**POST** `/circles/:id/members`

> 仅圈主可操作

**请求参数**
```typescript
interface AddMemberRequest {
  userId: string  // 要添加的用户ID
}
```

**响应数据**
```typescript
interface AddMemberResponse {
  message: string  // "添加成功"
}
```

---

### 3.7 移除成员

**DELETE** `/circles/:id/members/:userId`

> 仅圈主可操作，不能移除圈主自己

**响应数据**
```typescript
interface RemoveMemberResponse {
  message: string  // "移除成功"
}
```

---

## 四、有空状态模块 `/availabilities`

> 以下接口需要认证

### 4.1 获取朋友的有空状态

**GET** `/availabilities/friends`

> 返回当前用户所在圈子中，其他用户发布的有空状态（已按可见性过滤）

**响应数据**
```typescript
interface AvailabilityResponse {
  id: string
  user: UserProfile        // 发布者信息
  startTime: string        // 开始时间 ISO 8601
  endTime: string          // 结束时间 ISO 8601
  location: Location       // 地点信息
  status: AvailabilityStatus
  circleIds: string[]      // 可见圈子ID列表
  createdAt: string
}

interface Location {
  type: 'PRESET' | 'FLEXIBLE' | 'CUSTOM'
  name?: string            // 地点名称
  latitude?: number        // 纬度
  longitude?: number       // 经度
}

type AvailabilityStatus = 'ACTIVE' | 'EXPIRED' | 'CANCELLED' | 'FULFILLED'

type GetFriendsAvailabilitiesResponse = AvailabilityResponse[]
```

**示例响应**
```json
{
  "code": 0,
  "message": "成功",
  "data": [
    {
      "id": "avail-001",
      "user": {
        "id": "user-002",
        "nickname": "小明",
        "avatar": "https://..."
      },
      "startTime": "2024-01-15T18:00:00Z",
      "endTime": "2024-01-15T22:00:00Z",
      "location": {
        "type": "PRESET",
        "name": "三里屯",
        "latitude": 39.9334,
        "longitude": 116.4551
      },
      "status": "ACTIVE",
      "circleIds": ["circle-001", "circle-002"],
      "createdAt": "2024-01-15T10:00:00Z"
    }
  ]
}
```

---

### 4.2 获取我的有空状态

**GET** `/availabilities/mine`

**响应数据**
```typescript
// 同上 AvailabilityResponse[]
```

---

### 4.3 发布有空

**POST** `/availabilities`

**请求参数**
```typescript
interface CreateAvailabilityRequest {
  startTime: string           // 开始时间 ISO 8601，必填
  endTime: string             // 结束时间 ISO 8601，必填
  locationType: 'PRESET' | 'FLEXIBLE' | 'CUSTOM'  // 必填
  locationName?: string       // 地点名称
  latitude?: number           // 纬度
  longitude?: number          // 经度
  circleIds: string[]         // 可见圈子ID列表，必填，至少1个
}
```

**响应数据**
```typescript
// 返回创建的 AvailabilityResponse 对象
```

**预设地点**
```typescript
const presetLocations = [
  { name: '三里屯', emoji: '🛍️', lat: 39.9334, lng: 116.4551 },
  { name: '国贸', emoji: '🏢', lat: 39.9087, lng: 116.4605 },
  { name: '望京', emoji: '🌆', lat: 39.9982, lng: 116.4744 },
  { name: '中关村', emoji: '💻', lat: 39.9836, lng: 116.3164 },
  { name: '五道口', emoji: '🎓', lat: 39.9927, lng: 116.3377 },
]
```

**时间预设**
```typescript
const timePresets = {
  tonight: { start: 18, end: 22 },    // 今晚
  tomorrow: { start: 10, end: 22 },   // 明天
  weekend: { start: 0, end: 24 },     // 周末全天
}
```

---

### 4.4 取消有空

**DELETE** `/availabilities/:id`

> 仅发布者可操作

**响应数据**
```typescript
interface CancelResponse {
  message: string  // "已取消"
}
```

---

## 五、会话消息模块 `/conversations`

> 以下接口需要认证

### 5.1 获取会话列表

**GET** `/conversations`

**响应数据**
```typescript
interface ConversationResponse {
  id: string
  partner: UserProfile      // 对方用户信息
  lastMessage?: MessageResponse  // 最后一条消息
  unreadCount: number       // 未读消息数
  createdAt: string
}

type GetConversationsResponse = ConversationResponse[]
```

---

### 5.2 获取消息历史

**GET** `/conversations/:id/messages`

**查询参数**
| 参数 | 默认值 | 说明 |
|------|--------|------|
| limit | 20 | 每页数量，最大100 |
| offset | 0 | 偏移量 |

**响应数据**
```typescript
interface MessageResponse {
  id: string
  sender: UserProfile      // 发送者信息
  type: MessageType
  content?: string         // 文本内容
  metadata?: object        // 元数据（卡片类消息使用）
  createdAt: string
  isRead: boolean
}

type MessageType =
  | 'TEXT'              // 普通文本
  | 'AVAILABILITY_CARD' // 有空卡片
  | 'CONFIRM_REQUEST'   // 确认请求
  | 'CONFIRM_RESPONSE'  // 确认响应

type GetMessagesResponse = MessageResponse[]
```

---

### 5.3 发送消息

**POST** `/conversations/:id/messages`

**请求参数**
```typescript
interface SendMessageRequest {
  type: MessageType        // 消息类型，必填
  content?: string         // 文本内容
  metadata?: object        // 元数据
}
```

**消息类型元数据格式**
```typescript
// TEXT 类型
{ type: 'TEXT', content: '你好！' }

// AVAILABILITY_CARD 类型
{
  type: 'AVAILABILITY_CARD',
  metadata: {
    availabilityId: 'avail-001'
  }
}

// CONFIRM_REQUEST 类型（约见请求）
{
  type: 'CONFIRM_REQUEST',
  metadata: {
    availabilityId: 'avail-001',
    message: '想一起吃个饭？'
  }
}

// CONFIRM_RESPONSE 类型（约见响应）
{
  type: 'CONFIRM_RESPONSE',
  metadata: {
    requestMessageId: 'msg-001',
    accepted: true,
    message: '好的，到时见！'
  }
}
```

**响应数据**
```typescript
// 返回创建的 MessageResponse 对象
```

---

## 六、邀请模块 `/invitations`

### 6.1 创建邀请链接

**POST** `/invitations`

> 需要认证。创建一个邀请链接，可关联到圈子。

**请求参数**
```typescript
interface CreateInvitationRequest {
  circleId?: string    // 关联圈子ID（可选）
  maxUses?: number     // 最大使用次数，默认100
  expiresDays?: number // 有效天数，默认7，最长30
}
```

**响应数据**
```typescript
interface InvitationDetail {
  id: string
  code: string                    // 邀请码
  inviteUrl: string               // 完整邀请链接
  qrcodeUrl?: string              // 二维码链接
  inviter?: UserProfile           // 邀请人信息
  circle?: CircleInfo             // 关联圈子信息
  maxUses: number                 // 最大使用次数
  useCount: number                // 已使用次数
  expiresAt?: string              // 过期时间 ISO 8601
  status: 'ACTIVE' | 'DISABLED' | 'EXPIRED'
  isValid: boolean                // 是否有效
  createdAt: string
}

interface CircleInfo {
  id: string
  name: string
  emoji: string
  color?: string
  memberCount?: number
}
```

**示例**
```json
// 请求
{ "circleId": "circle-001", "expiresDays": 7 }

// 响应
{
  "code": 0,
  "message": "成功",
  "data": {
    "id": "inv-001",
    "code": "ABC123",
    "inviteUrl": "http://49.232.13.41:8080/i/ABC123",
    "inviter": {
      "id": "user-001",
      "nickname": "小明",
      "avatar": "https://..."
    },
    "circle": {
      "id": "circle-001",
      "name": "闺蜜",
      "emoji": "💕",
      "color": "#EC4899",
      "memberCount": 5
    },
    "maxUses": 100,
    "useCount": 0,
    "expiresAt": "2026-02-03T00:00:00Z",
    "status": "ACTIVE",
    "isValid": true,
    "createdAt": "2026-01-27T00:00:00Z"
  }
}
```

---

### 6.2 获取邀请信息（公开）

**GET** `/invite/:code`

> ❌ 无需认证。用于邀请落地页展示邀请人和圈子信息。

**响应数据**
```typescript
interface InvitationPublicInfo {
  inviter: UserProfile    // 邀请人信息
  circle?: CircleInfo     // 圈子信息（可选）
  isValid: boolean        // 是否有效
}
```

**示例**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "inviter": {
      "id": "user-001",
      "nickname": "小明",
      "avatar": "https://..."
    },
    "circle": {
      "id": "circle-001",
      "name": "闺蜜",
      "emoji": "💕",
      "memberCount": 5
    },
    "isValid": true
  }
}
```

---

### 6.3 接受邀请

**POST** `/invite/:code/accept`

> 需要认证。接受邀请后自动与邀请人成为好友，并加入关联圈子（如果有）。

**响应数据**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "message": "已接受邀请",
    "joinedCircle": {
      "id": "circle-001",
      "name": "闺蜜",
      "emoji": "💕"
    }
  }
}
```

**错误情况**
| 错误 | message |
|------|---------|
| 邀请不存在 | `邀请不存在或已失效` |
| 邀请已过期 | `邀请已过期` |
| 使用次数已满 | `邀请使用次数已达上限` |
| 不能接受自己的邀请 | `不能接受自己的邀请` |

---

### 6.4 获取我的邀请列表

**GET** `/invitations`

> 需要认证。获取当前用户创建的所有邀请链接。

**响应数据**
```typescript
type GetMyInvitationsResponse = InvitationDetail[]
```

---

### 6.5 获取邀请详情

**GET** `/invitations/:id`

> 需要认证。获取指定邀请的完整信息。

**响应数据**
```typescript
// InvitationDetail（同 6.1）
```

---

### 6.6 禁用邀请链接

**DELETE** `/invitations/:id`

> 需要认证。仅邀请创建者可操作。

**响应数据**
```json
{ "code": 0, "message": "成功", "data": { "message": "邀请已禁用" } }
```

---

### 6.7 获取邀请海报

**GET** `/invitations/:id/poster`

> 需要认证。返回 PNG 图片，包含邀请信息和二维码。

**响应**
- Content-Type: `image/png`
- 图片尺寸: 750x1334

---

### 6.8 获取邀请二维码

**GET** `/invitations/:id/qrcode`

> 需要认证。返回单独的二维码 PNG 图片。

**响应**
- Content-Type: `image/png`
- 图片尺寸: 300x300

---

## 七、数据类型汇总

### 用户相关

```typescript
interface User {
  id: string
  phone: string
  nickname: string
  avatar?: string
  createdAt: string
  updatedAt: string
  lastActiveAt?: string
}

interface UserProfile {
  id: string
  nickname: string
  avatar?: string
}
```

### 圈子相关

```typescript
interface Circle {
  id: string
  name: string
  emoji: string
  color: string
  ownerId: string
  createdAt: string
  updatedAt: string
}

interface CircleDetail extends Omit<Circle, 'updatedAt'> {
  memberCount: number
  members?: UserProfile[]
}
```

### 有空状态相关

```typescript
type LocationType = 'PRESET' | 'FLEXIBLE' | 'CUSTOM'
type AvailabilityStatus = 'ACTIVE' | 'EXPIRED' | 'CANCELLED' | 'FULFILLED'

interface Location {
  type: LocationType
  name?: string
  latitude?: number
  longitude?: number
}

interface AvailabilityResponse {
  id: string
  user: UserProfile
  startTime: string
  endTime: string
  location: Location
  status: AvailabilityStatus
  circleIds: string[]
  createdAt: string
}
```

### 消息相关

```typescript
type MessageType = 'TEXT' | 'AVAILABILITY_CARD' | 'CONFIRM_REQUEST' | 'CONFIRM_RESPONSE'

interface MessageResponse {
  id: string
  sender: UserProfile
  type: MessageType
  content?: string
  metadata?: Record<string, any>
  createdAt: string
  isRead: boolean
}

interface ConversationResponse {
  id: string
  partner: UserProfile
  lastMessage?: MessageResponse
  unreadCount: number
  createdAt: string
}
```

---

## 八、前端本地存储建议

```typescript
// Token 存储
const TOKEN_KEY = 'youkong_token'

// 用户信息缓存
const USER_KEY = 'youkong_user'

// 圈子列表缓存
const CIRCLES_KEY = 'youkong_circles'
```

---

## 九、Agent 智能记忆模块 `/agent`

> 以下接口需要认证

### 9.1 状态上报（智能分析）

**POST** `/agent/status`

> 上报用户当前状态，返回 AI 分析的有空概率和生活状态 emoji

**请求参数**
```typescript
interface StatusReportRequest {
  screen?: {
    is_active: boolean                    // 当前是否在用手机
    activity_type: ActivityType           // 使用类型
    session_duration_minutes: number      // 本次使用时长(分钟)
    last_active_minutes_ago: number       // 上次活跃是多久前(分钟)
  }
  location?: {
    place_type: PlaceType                 // 位置类型
    at_place_since_minutes: number        // 在此位置待了多久(分钟)
  }
  battery?: {
    battery_level: number                 // 电量 0-100
    battery_state: string                 // charging/unplugged/full
    is_charging: boolean                  // 是否充电中
  }
  mode?: {
    is_low_power_mode: boolean            // 低电量模式
    is_focus_mode_on: boolean             // 专注模式(勿扰)
  }
  connection?: {
    is_headphones_connected: boolean      // 耳机已连接
    network_type: NetworkType             // 网络类型
  }
  display?: {
    screen_brightness: number             // 屏幕亮度 0.0-1.0
  }
}

type ActivityType = 'entertainment' | 'productivity' | 'communication' | 'idle'
type PlaceType = 'home' | 'work' | 'leisure' | 'transit' | 'unknown'
type NetworkType = 'wifi' | 'cellular' | 'none'
```

**响应数据**
```typescript
interface StatusReportResponse {
  success: boolean                        // 上报是否成功
  next_report_in: number                  // 建议下次上报间隔(秒)
  analysis?: {
    availability: {
      status: '有空' | '忙碌' | '可能有空'  // 有空状态
      probability: number                 // 有空概率 0-100
      reason: string                      // 理由(≤15字)
      confidence: 'high' | 'medium' | 'low'  // 置信度
    }
    life_status: {
      emoji: string                       // 生活状态 Emoji
      label: string                       // 状态标签
      description?: string                // 状态描述(可选)
    }
  }
}
```

**示例请求**
```json
{
  "screen": {
    "is_active": true,
    "activity_type": "entertainment",
    "session_duration_minutes": 25,
    "last_active_minutes_ago": 0
  },
  "location": {
    "place_type": "home",
    "at_place_since_minutes": 120
  },
  "battery": {
    "battery_level": 75,
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
  }
}
```

**示例响应**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "success": true,
    "next_report_in": 60,
    "analysis": {
      "availability": {
        "status": "可能有空",
        "probability": 85,
        "reason": "正在专心娱乐，不便打扰",
        "confidence": "high"
      },
      "life_status": {
        "emoji": "📺",
        "label": "在家追剧看片",
        "description": "戴着耳机，看起来很投入"
      }
    }
  }
}
```

---

### 9.2 获取用户记忆

**GET** `/agent/memory`

> 获取当前用户的核心记忆（AI 学习到的行为规律）

**响应数据**
```typescript
interface CoreMemoryResponse {
  behavior_insights: string      // 行为模式洞察
  time_patterns: string          // 时间规律
  location_preferences: string   // 地点偏好
  social_tendency: string        // 社交倾向
  confidence_score: number       // 置信度 0-100
  sample_count: number           // 样本数量
  updated_at: string             // 最后更新时间 ISO 8601
}
```

**示例响应**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "behavior_insights": "周一晚上在家长时间进行娱乐活动",
    "time_patterns": "22点后是个人娱乐时间",
    "location_preferences": "晚上主要在家",
    "social_tendency": "晚间娱乐时倾向于独处",
    "confidence_score": 14,
    "sample_count": 7,
    "updated_at": "2026-01-26T22:19:50+08:00"
  }
}
```

---

### 9.3 好友有空概率列表（带生活状态）

**GET** `/friends/free-probability`

> 获取好友的有空概率列表，包含生活状态 emoji

**响应数据**
```typescript
interface FreeProbabilityResponse {
  friends: FriendRecommendation[]
  generated_at: number           // 毫秒时间戳
}

interface FriendRecommendation {
  friend_id: string
  name: string
  avatar?: string
  probability: number            // 有空概率 0-100，-1表示无数据
  confidence: 'high' | 'medium' | 'low'
  reason: string                 // 理由(≤15字)
  color: string                  // 概率颜色代码
  life_status?: {                // 生活状态(可选)
    emoji: string
    label: string
  }
  updated_at: number             // 毫秒时间戳
}
```

**概率颜色映射**
```typescript
// 80-100%: #22C55E (绿色-很可能有空)
// 60-79%:  #86EFAC (浅绿-可能有空)
// 40-59%:  #FACC15 (黄色-可能)
// 20-39%:  #FB923C (橙色-可能没空)
// 0-19%:   #EF4444 (红色-忙碌)
// 无数据:   #9CA3AF (灰色)
```

**示例响应**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "friends": [
      {
        "friend_id": "user456",
        "name": "小王",
        "avatar": "https://...",
        "probability": 85,
        "confidence": "high",
        "reason": "周末晚上通常有空",
        "color": "#22C55E",
        "life_status": {
          "emoji": "🛋️",
          "label": "在家躺着"
        },
        "updated_at": 1706266200000
      }
    ],
    "generated_at": 1706266200000
  }
}
```

---

### 9.4 生活状态 Emoji 参考表

| Emoji | Label | 触发条件 |
|-------|-------|----------|
| 🎮 | 在玩游戏 | 娱乐类 + 长时间会话 |
| 📺 | 在追剧 | 娱乐类 + 在家 |
| 💼 | 在工作 | 工作类 + 公司 |
| ☕ | 在摸鱼 | 娱乐类 + 公司 |
| 🍜 | 在吃饭 | 餐点时间 + 休闲场所 |
| 🛋️ | 在家躺着 | 闲置/娱乐 + 在家 |
| 🚶 | 在外面逛 | 移动中/休闲场所 |
| 😴 | 可能在睡觉 | 不活跃 + 深夜 |
| 📱 | 在刷手机 | 娱乐类 + 短会话 |
| 💬 | 在聊天 | 通讯类活跃 |
| 🎧 | 在听音乐 | 耳机连接 + 娱乐 |
| 🏃 | 在运动 | 移动中 + 不活跃 |
| 🍻 | 可能在聚会 | 周末晚 + 外出 |
| 🔕 | 不想被打扰 | 专注模式开启 |
| 🪫 | 电量告急 | 低电量模式 |
| 🤔 | 状态未知 | 数据不足 |

---

### 9.5 数据采集说明

**屏幕数据采集建议**

| 字段 | iOS 获取方式 | Android 获取方式 |
|------|-------------|-----------------|
| `is_active` | `UIApplication.shared.applicationState` | `PowerManager.isInteractive()` |
| `activity_type` | 使用 Screen Time API 或自行分类 | `UsageStatsManager` |
| `session_duration_minutes` | 计算 App 前台时间 | `UsageStatsManager` |

**位置数据采集建议**

| 字段 | 获取方式 |
|------|----------|
| `place_type` | 地理围栏判断（家/公司坐标） |
| `at_place_since_minutes` | 记录进入围栏时间 |

**设备数据采集**

| 字段 | iOS | Android |
|------|-----|---------|
| `battery_level` | `UIDevice.current.batteryLevel` | `BatteryManager` |
| `is_charging` | `UIDevice.current.batteryState` | `BatteryManager` |
| `is_low_power_mode` | `ProcessInfo.processInfo.isLowPowerModeEnabled` | `PowerManager` |
| `is_focus_mode_on` | 需要授权 Focus Status | `NotificationManager` |
| `is_headphones_connected` | `AVAudioSession` | `AudioManager` |
| `network_type` | `NWPathMonitor` | `ConnectivityManager` |
| `screen_brightness` | `UIScreen.main.brightness` | `Settings.System` |

---

## 十、好友模块 `/friends`

> 以下接口需要认证

### 10.1 获取好友列表

**GET** `/friends`

**响应数据**
```typescript
interface FriendInfo {
  user: UserProfile
  source: 'INVITATION' | 'SEARCH' | 'CONTACTS' | 'MANUAL'
  createdAt: string
}

type GetFriendsResponse = FriendInfo[]
```

---

### 10.2 通过手机号加好友

**POST** `/friends/add-by-phone`

> 直接通过手机号添加好友，无需对方确认

**请求参数**
```typescript
interface AddFriendByPhoneRequest {
  phone: string   // 手机号，11位数字
}
```

**响应数据**
```typescript
interface AddFriendByPhoneResponse {
  user: UserProfile   // 好友信息
  added: boolean      // true=新添加, false=已是好友
}
```

**示例**
```json
// 请求
{ "phone": "13800000002" }

// 响应（成功添加）
{
  "code": 0,
  "message": "成功",
  "data": {
    "user": {
      "id": "user-002",
      "nickname": "小红",
      "avatar": "https://..."
    },
    "added": true
  }
}

// 响应（已是好友）
{
  "code": 0,
  "message": "成功",
  "data": {
    "user": {
      "id": "user-002",
      "nickname": "小红",
      "avatar": "https://..."
    },
    "added": false
  }
}
```

**错误情况**
| 错误 | message |
|------|---------|
| 手机号未注册 | `该手机号未注册` |
| 添加自己 | `不能添加自己为好友` |
| 格式错误 | `手机号格式错误，需要11位数字` |

---

### 10.3 删除好友

**DELETE** `/friends/:userId`

**响应数据**
```json
{ "code": 0, "message": "成功", "data": { "message": "已删除好友" } }
```

---

### 10.4 我邀请的好友

**GET** `/friends/invited-by-me`

> 获取通过我的邀请链接添加的好友

**响应数据**
```typescript
interface FriendWithInvitation {
  user: UserProfile
  source: string
  circleName?: string   // 通过哪个圈子邀请的
  createdAt: string
}

type GetInvitedByMeResponse = FriendWithInvitation[]
```

---

### 10.5 邀请我的好友

**GET** `/friends/invited-me`

> 获取邀请我的好友列表

**响应数据**
```typescript
// 同上 FriendWithInvitation[]
```

---

### 10.6 发送好友请求

**POST** `/friends/request`

> 通过手机号发送好友请求，需要对方确认

**请求参数**
```typescript
interface SendFriendRequestRequest {
  phone: string     // 手机号，11位数字
  message?: string  // 附言（可选，最多200字）
}
```

**响应数据**
```typescript
interface SendFriendRequestResponse {
  requestId: string       // 请求ID
  user: UserProfile       // 目标用户信息
  status: 'PENDING' | 'ALREADY_FRIENDS' | 'ALREADY_REQUESTED'
}
```

**示例**
```json
// 请求
{ "phone": "13800000002", "message": "我是小明，认识一下" }

// 响应（发送成功）
{
  "code": 0,
  "message": "成功",
  "data": {
    "requestId": "req-001",
    "user": {
      "id": "user-002",
      "nickname": "小红",
      "avatar": "https://..."
    },
    "status": "PENDING"
  }
}

// 响应（已是好友）
{
  "code": 0,
  "message": "成功",
  "data": {
    "requestId": "",
    "user": { ... },
    "status": "ALREADY_FRIENDS"
  }
}
```

**错误情况**
| 错误 | message |
|------|---------|
| 手机号未注册 | `该手机号未注册` |
| 添加自己 | `不能添加自己为好友` |
| 格式错误 | `手机号格式错误，需要11位数字` |

---

### 10.7 获取收到的好友请求

**GET** `/friends/requests/received`

> 获取他人发给我的好友请求（待处理）

**响应数据**
```typescript
interface FriendRequestInfo {
  id: string              // 请求ID
  user: UserProfile       // 对方用户信息
  message?: string        // 附言
  status: 'PENDING' | 'ACCEPTED' | 'REJECTED' | 'CANCELLED'
  createdAt: string       // ISO 8601
}

type GetReceivedRequestsResponse = FriendRequestInfo[]
```

**示例响应**
```json
{
  "code": 0,
  "message": "成功",
  "data": [
    {
      "id": "req-001",
      "user": {
        "id": "user-003",
        "nickname": "小李",
        "avatar": "https://..."
      },
      "message": "我是小李，加个好友",
      "status": "PENDING",
      "createdAt": "2026-01-27T10:00:00Z"
    }
  ]
}
```

---

### 10.8 获取发出的好友请求

**GET** `/friends/requests/sent`

> 获取我发出的好友请求

**响应数据**
```typescript
// 同上 FriendRequestInfo[]
```

---

### 10.9 处理好友请求

**POST** `/friends/requests/:id/handle`

> 同意或拒绝好友请求

**路径参数**
| 参数 | 说明 |
|------|------|
| id | 请求ID |

**请求参数**
```typescript
interface HandleFriendRequestRequest {
  accept: boolean   // true=同意, false=拒绝
}
```

**响应数据**
```json
// 同意
{ "code": 0, "message": "成功", "data": { "message": "已同意，你们已成为好友" } }

// 拒绝
{ "code": 0, "message": "成功", "data": { "message": "已拒绝" } }
```

**错误情况**
| 错误 | message |
|------|---------|
| 请求不存在 | `好友请求不存在` |
| 不是发给自己的 | `无权处理此请求` |
| 已处理 | `该请求已处理` |

---

### 10.10 获取待处理请求数量

**GET** `/friends/requests/count`

> 获取收到的待处理好友请求数量（用于显示红点）

**响应数据**
```json
{ "code": 0, "message": "成功", "data": { "count": 3 } }
```

---

## 十一、福尔摩斯推理框架 API

> 福尔摩斯推理框架通过多维度数据（时间、位置、屏幕、日历、移动等）像侦探一样推断用户是否有空。

### 11.1 好友有空概率列表（福尔摩斯版）

**GET** `/friends/holmes-probability`

> 获取好友的有空概率列表，包含完整的推理过程

**响应数据**
```typescript
interface HolmesFriendListResponse {
  friends: HolmesAPIResponse[]
  generated_at: number           // 毫秒时间戳
}

interface HolmesAPIResponse {
  friend_id: string
  name: string
  avatar?: string

  // 原始线索（供前端展示推理过程）
  raw_data?: HolmesClue

  // 特征层
  features?: HolmesFeatures

  // 推理过程
  reasoning?: HolmesReasoning

  // 最终结果
  result: {
    available: boolean
    probability: number          // 0-100
    confidence: 'high' | 'medium' | 'low'
    summary: string              // 简短总结（15字以内）
    emoji?: string
    color: string
  }

  updated_at: number
}
```

**示例响应**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "friends": [
      {
        "friend_id": "user456",
        "name": "小王",
        "avatar": "https://...",
        "raw_data": {
          "timestamp": "2026-01-28T15:42:00+08:00",
          "weekday": "周六",
          "time_period": "下午3点42分",
          "is_weekend": true,
          "place_name": "星巴克(三里屯店)",
          "place_type": "leisure",
          "at_place_since_minutes": 23,
          "steps_today": 2847,
          "steps_last_hour": 127,
          "screen_active": true,
          "screen_duration_mins": 18,
          "activity_type": "entertainment",
          "has_calendar_event": false
        },
        "features": {
          "location_type": "星巴克(三里屯店)",
          "movement_state": "静止（已23分钟）",
          "time_period": "周末下午",
          "activity": "娱乐内容（已18分钟）",
          "schedule": "无日程",
          "device_state": "正常"
        },
        "reasoning": {
          "model": "qwen3-max-2026-01-23",
          "thinking": "周六下午在星巴克坐了20多分钟，一直在看娱乐内容，没有日程安排。这种场景通常表示用户在休闲放松，社交意愿应该比较高...",
          "conclusion": "很可能有空约朋友"
        },
        "result": {
          "available": true,
          "probability": 88,
          "confidence": "high",
          "summary": "在咖啡厅休闲",
          "emoji": "☕",
          "color": "#22C55E"
        },
        "updated_at": 1706430120000
      }
    ],
    "generated_at": 1706430120000
  }
}
```

---

### 11.2 单个好友福尔摩斯分析详情

**GET** `/friends/:id/holmes`

> 获取单个好友的完整福尔摩斯分析结果

**响应数据**
```typescript
interface HolmesResult {
  raw_data: HolmesClue
  features: HolmesFeatures
  reasoning: HolmesReasoning
  result: {
    available: boolean
    probability: number
    confidence: string
    summary: string
  }
  generated_at: string
}
```

---

### 11.3 数据结构定义

**HolmesClue - 原始线索**
```typescript
interface HolmesClue {
  // 时间线索
  timestamp: string              // ISO 8601
  weekday: string                // 星期几（中文）
  time_period: string            // 时间段描述
  is_weekend: boolean

  // 位置线索
  place_name?: string            // 地点名称
  place_type: string             // home/work/leisure/transit/unknown
  at_place_since_minutes: number // 在此地多久
  altitude?: number              // 海拔（米）
  floor?: number                 // 推测楼层

  // 移动线索
  is_moving: boolean
  movement_type?: string         // stationary/walking/running/driving
  steps_today?: number
  steps_last_hour?: number
  stationary_minutes?: number

  // 屏幕线索
  screen_active: boolean
  screen_duration_mins?: number
  activity_type?: string         // entertainment/productivity/communication/idle
  last_active_minutes_ago?: number

  // 日历线索
  has_calendar_event: boolean
  calendar_event_title?: string  // 脱敏后的标题
  event_end_minutes?: number
  today_remaining_events: number

  // 设备线索
  headphones_connected?: boolean
  focus_mode_on?: boolean
  low_battery_mode?: boolean
  battery_level?: number
}
```

**HolmesFeatures - 特征层**
```typescript
interface HolmesFeatures {
  location_type: string          // 咖啡厅/办公室/家/商场/...
  movement_state: string         // 静止/步行/乘车/跑步
  time_period: string            // 工作日上午/周末下午/...
  activity: string               // 娱乐内容/工作内容/通讯/闲置
  schedule: string               // 有会议/无日程/会议即将结束
  device_state: string           // 专注模式/低电量/戴耳机/...
}
```

**HolmesReasoning - 推理过程**
```typescript
interface HolmesReasoning {
  model: string                  // 使用的模型
  thinking: string               // 思考过程
  conclusion: string             // 推理结论
}
```

---

### 11.4 扩展状态上报请求

**POST** `/agent/status`

> 上报用户状态，支持新增的日历、移动、海拔数据

**请求参数**
```typescript
interface ExtendedStatusReportRequest {
  screen?: ScreenData
  location?: LocationData
  extended_location?: ExtendedLocationData  // 扩展位置数据
  battery?: BatteryData
  mode?: ModeData
  connection?: ConnectionData
  display?: DisplayData
  calendar?: CalendarData         // 新增：日历数据
  movement?: MovementData         // 新增：移动状态
  altitude?: AltitudeData         // 新增：海拔数据
}

// 新增数据结构
interface CalendarData {
  has_current_event: boolean      // 当前是否有日程
  current_event_title?: string    // 当前日程标题（脱敏后）
  event_end_minutes?: number      // 当前日程还有多久结束
  next_event_in_minutes?: number  // 下个日程还有多久开始
  today_remaining_count: number   // 今天剩余日程数量
}

interface MovementData {
  is_moving: boolean              // 是否在移动
  movement_type?: string          // stationary/walking/running/driving/cycling
  steps_today?: number            // 今日步数
  steps_last_hour?: number        // 最近1小时步数
  stationary_minutes?: number     // 静止了多久（分钟）
  moving_direction?: string       // to_work/to_home/unknown
  current_speed_kmh?: number      // 当前速度（km/h）
}

interface AltitudeData {
  altitude: number                // 当前海拔（米）
  relative_altitude?: number      // 相对海拔变化
  floor?: number                  // 推测楼层
}

interface ExtendedLocationData {
  place_type: string              // 位置类型
  place_name?: string             // 地点名称（如"星巴克(三里屯店)"）
  at_place_since_minutes: number  // 在此位置待了多久
  latitude?: number               // 纬度
  longitude?: number              // 经度
}
```

**响应数据**
```typescript
interface StatusReportResponse {
  success: boolean
  next_report_in: number          // 下次上报间隔（秒）
  analysis?: AnalysisResult       // 旧版分析结果
  holmes?: HolmesResult           // 福尔摩斯分析结果
}
```

---

### 11.5 福尔摩斯推理示例

| 线索组合 | LLM 推理 | 结论 |
|----------|----------|------|
| 早8点 + 步行 + 向公司方向 + 步频快 | 正在赶去上班，步频快说明可能要迟到 | 没空 (95%) |
| 周六 + 咖啡厅 + 静止 + 看娱乐内容 | 周末在咖啡厅休闲，很放松 | 有空 (88%) |
| 工作日 + 公司 + 14楼 + 日历有会议 | 在办公室楼层，正在开会 | 没空 (98%) |
| 晚8点 + 在家 + 躺着 + 屏幕闲置 | 可能在休息或看电视 | 可能有空 (65%) |
| 海拔快速上升 + 在商场 | 在坐电梯，可能在逛街 | 有空 (70%) |

---

## 十二、WebSocket 连接（预留）

```
ws://api.youkong.app/ws?token=<jwt_token>
```

**消息格式**
```typescript
interface WSMessage {
  type: 'NEW_MESSAGE' | 'NEW_AVAILABILITY' | 'AVAILABILITY_CANCELLED'
  payload: any
}
```
