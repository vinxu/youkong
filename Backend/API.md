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

### 2.3 获取用户信息

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

### 2.4 搜索用户

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

## 六、数据类型汇总

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

## 七、前端本地存储建议

```typescript
// Token 存储
const TOKEN_KEY = 'youkong_token'

// 用户信息缓存
const USER_KEY = 'youkong_user'

// 圈子列表缓存
const CIRCLES_KEY = 'youkong_circles'
```

---

## 八、WebSocket 连接（预留）

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
