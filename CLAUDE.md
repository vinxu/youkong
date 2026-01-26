# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

**有空 (YouKong)** - 低压力社交预约工具，让用户向特定圈子展示可约时间和地点。

核心理念：把「我想见你」变成「我有空，你看着办」

```
有空 = 时间窗口 × 地点区域 × 可见圈子
```

## 开发命令

### 后端 (Go + Gin)

所有命令在 `Backend/` 目录下执行：

```bash
cd Backend

# 开发模式运行 (推荐)
make dev

# 构建并运行
make run

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查 (需安装 golangci-lint)
make lint

# 下载依赖
make deps

# Docker 构建和运行
make docker-build
make docker-run
```

### 环境配置

复制 `Backend/.env.example` 到 `Backend/.env` 并配置相关参数。

### 数据库迁移

```bash
cd Backend && make migrate
# 或手动执行
mysql -u root -p youkong < Backend/migrations/001_init.sql
mysql -u root -p youkong < Backend/migrations/002_wechat_invite.sql
```

## 项目结构

```
youkong/
├── Backend/           # Go API 服务 (Gin)
│   ├── cmd/server/        # 应用入口 (main.go)
│   ├── internal/
│   │   ├── config/        # 配置加载 (Viper)
│   │   ├── handler/       # HTTP 请求处理器
│   │   ├── middleware/    # 认证、CORS、日志中间件
│   │   ├── model/         # 数据模型
│   │   ├── repository/    # 数据访问层 (sqlx)
│   │   ├── service/       # 业务逻辑层
│   │   └── pkg/           # 工具包 (jwt, response, tencent, validator)
│   └── migrations/        # SQL 迁移脚本
├── Android/           # (待开发)
├── iOS/               # (待开发)
└── Web/               # (待开发)
```

## 技术栈

| 平台 | 技术选型 |
|------|----------|
| 后端 | Go 1.21, Gin, sqlx, MySQL, Redis, Viper, Zap |
| iOS (计划) | Swift 5.9+, SwiftUI, MVVM + Clean Architecture |
| Android (计划) | Kotlin 1.9+, Jetpack Compose, MVVM + Clean Architecture |

## 后端架构

采用分层架构，各层通过构造函数注入依赖（见 `cmd/server/main.go`）：

```
Handler (HTTP 请求处理，参数校验)
    ↓
Service (业务逻辑，事务协调)
    ↓
Repository (数据访问，SQL 查询)
    ↓
MySQL + Redis
```

**初始化顺序**：Config → DB/Redis → Repository → Service → Handler → Router

## 核心功能

### 发布有空（3步流程）
```
Step 1: 选时间 → Step 2: 选地点 → Step 3: 选圈子 → 发布
```

### 可见性规则
```typescript
// 只显示用户有权限看到的状态
function getVisibleAvailabilities(userId: string) {
  return availabilities.filter(a =>
    a.visibleCircleIds.some(circleId =>
      circles[circleId].memberIds.includes(userId)
    )
  )
}
```

## 数据模型

核心模型定义在 `Backend/internal/model/`：

| 模型 | 说明 |
|------|------|
| `User` | 用户 (手机号/微信登录) |
| `WechatUser` | 微信用户绑定 |
| `Circle` | 圈子 (名称、emoji、颜色) |
| `Availability` | 有空状态 (时间、地点、可见圈子) |
| `Message` | 消息 (TEXT, AVAILABILITY_CARD, CONFIRM_REQUEST, CONFIRM_RESPONSE) |
| `Invitation` | 邀请链接 (邀请码、有效期、使用次数) |
| `InvitationRecord` | 邀请记录 (谁邀请谁、通过哪个圈子) |
| `Friendship` | 好友关系 (双向存储、来源追踪) |

**地点类型**: `PRESET` (预设) | `FLEXIBLE` (灵活) | `CUSTOM` (自定义)

**状态流转**: `ACTIVE` → `EXPIRED` | `CANCELLED` | `FULFILLED`

## 服务器部署

### 生产环境

| 项目 | 值 |
|------|-----|
| API 地址 | `http://49.232.13.41:8080/api/v1` |
| 健康检查 | `http://49.232.13.41:8080/health` |
| 服务器 | 腾讯云轻量服务器 2核2GB |
| 数据库 | MySQL 8.0 (本地) |
| 缓存 | Redis 7 (本地) |

### 测试账号

开发测试时使用以下固定账号，无需真实短信验证：

| 手机号 | 验证码 | 说明 |
|--------|--------|------|
| `13800000001` | `111111` | 测试账号1 |
| `13800000002` | `222222` | 测试账号2 |
| `13800000003` | `333333` | 测试账号3 |
| `13800138000` | `123456` | 测试账号4 |

### 前端登录示例

```typescript
// 1. 发送验证码（测试账号可跳过）
await fetch('http://49.232.13.41:8080/api/v1/auth/sms/send', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ phone: '13800000001' })
});

// 2. 验证码登录
const res = await fetch('http://49.232.13.41:8080/api/v1/auth/sms/verify', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ phone: '13800000001', code: '111111' })
});
const { data } = await res.json();
const token = data.token;  // 保存用于后续请求

// 3. 调用需要认证的接口
await fetch('http://49.232.13.41:8080/api/v1/users/me', {
  headers: { 'Authorization': `Bearer ${token}` }
});
```

### 自动部署

项目使用 GitHub Actions + Webhook 实现全自动部署，push 代码后自动完成构建和部署。

**架构流程**:
```
开发者 push → GitHub Actions 构建 → 创建 Release → 调用 Webhook → 服务器下载并重启
```

**首次设置**（一次性）:
```bash
# 在服务器 VNC 终端执行
curl -fsSL https://raw.githubusercontent.com/vinxu/youkong/main/scripts/setup.sh | bash
```

脚本会：
1. 创建工作目录 `/opt/youkong`
2. 下载最新 release
3. 配置 systemd 服务
4. 生成 DEPLOY_TOKEN（需添加到 GitHub Secrets）

**环境变量配置**:
```bash
# /opt/youkong/.env 中添加以下配置
DEPLOY_TOKEN=<安装脚本生成的 token>
DEPLOY_GITHUB_REPO=vinxu/youkong
DEPLOY_WORK_DIR=/opt/youkong
DEPLOY_WEB_DIR=/opt/youkong/web
```

**日常使用**: 直接 `git push`，等待自动部署完成即可

**服务管理**:
```bash
systemctl status youkong    # 查看状态
systemctl restart youkong   # 重启服务
journalctl -u youkong -f    # 查看日志
```

**验证地址**:
- H5 首页: `http://49.232.13.41:8080/`
- 邀请页面: `http://49.232.13.41:8080/i/CODE`
- API 健康检查: `http://49.232.13.41:8080/health`

## API 接口

> 详细接口文档见 `Backend/API.md`

**Base URL**: `http://49.232.13.41:8080/api/v1`

### 接口列表

| 模块 | 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|------|
| 认证 | POST | `/auth/sms/send` | 发送验证码 | 否 |
| | POST | `/auth/sms/verify` | 验证码登录 | 否 |
| | POST | `/auth/refresh` | 刷新 Token | 否 |
| | POST | `/auth/wechat/login` | 微信授权登录 | 否 |
| 用户 | GET | `/users/me` | 获取当前用户 | 是 |
| | PUT | `/users/me` | 更新当前用户 | 是 |
| | GET | `/users/search?keyword=xx` | 搜索用户 | 是 |
| | GET | `/users/:id` | 获取用户信息 | 是 |
| 圈子 | GET | `/circles` | 获取我的圈子 | 是 |
| | POST | `/circles` | 创建圈子 | 是 |
| | GET | `/circles/:id` | 获取圈子详情 | 是 |
| | PUT | `/circles/:id` | 更新圈子 | 是 |
| | DELETE | `/circles/:id` | 删除圈子 | 是 |
| | POST | `/circles/:id/members` | 添加成员 | 是 |
| | DELETE | `/circles/:id/members/:userId` | 移除成员 | 是 |
| 有空 | GET | `/availabilities/friends` | 朋友的有空状态 | 是 |
| | GET | `/availabilities/mine` | 我的有空状态 | 是 |
| | POST | `/availabilities` | 发布有空 | 是 |
| | DELETE | `/availabilities/:id` | 取消有空 | 是 |
| 消息 | GET | `/conversations` | 会话列表 | 是 |
| | GET | `/conversations/:id/messages` | 获取消息 | 是 |
| | POST | `/conversations/:id/messages` | 发送消息 | 是 |
| 邀请 | POST | `/invitations` | 创建邀请链接 | 是 |
| | GET | `/invitations` | 获取我的邀请列表 | 是 |
| | GET | `/invitations/:code` | 获取邀请详情（公开） | 否 |
| | GET | `/invitations/:id/detail` | 获取邀请完整信息 | 是 |
| | DELETE | `/invitations/:id` | 禁用邀请链接 | 是 |
| | POST | `/invitations/:code/accept` | 接受邀请 | 是 |
| | GET | `/invitations/:id/poster` | 获取邀请海报图片 | 是 |
| | GET | `/invitations/:id/qrcode` | 获取邀请二维码 | 是 |
| 好友 | GET | `/friends` | 好友列表 | 是 |
| | DELETE | `/friends/:userId` | 删除好友 | 是 |
| | GET | `/friends/invited-by-me` | 我邀请的好友 | 是 |
| | GET | `/friends/invited-me` | 邀请我的好友 | 是 |
| | GET | `/friends/free-probability` | 好友有空概率列表 | 是 |
| Agent | POST | `/agent/status` | 上报用户状态 | 是 |
| | POST | `/agent/query` | Agent间数据请求 | 是 |

### 统一响应格式

```typescript
interface ApiResponse<T> {
  code: number      // 0=成功, 非0=失败
  message: string   // 响应消息
  data?: T          // 响应数据
}
```

### 错误码

| 错误码 | 说明 | HTTP状态码 |
|--------|------|-----------|
| 0 | 成功 | 200 |
| 1001 | 参数错误 | 400 |
| 1002 | 未授权 | 401 |
| 1003 | Token已过期 | 401 |
| 1004 | 资源不存在 | 404 |
| 1005 | 无权限 | 403 |
| 5000 | 服务器内部错误 | 500 |

### 认证方式

```
Authorization: Bearer <token>
```

## 前端数据类型定义

### 用户

```typescript
interface User {
  id: string
  phone?: string        // 手机号（微信登录可能为空）
  nickname: string
  avatar?: string
  wechatBound: boolean  // 是否绑定微信
  createdAt: string     // ISO 8601
  updatedAt: string
}

interface UserProfile {
  id: string
  nickname: string
  avatar?: string
}
```

### 圈子

```typescript
interface Circle {
  id: string
  name: string         // 最多10字
  emoji: string
  color: string        // 如 "#EC4899"
  ownerId: string
  createdAt: string
}

interface CircleDetail extends Circle {
  memberCount: number
  members?: UserProfile[]
}

// 创建/更新圈子请求
interface CircleRequest {
  name: string
  emoji: string
  color: string
}
```

### 有空状态

```typescript
type LocationType = 'PRESET' | 'FLEXIBLE' | 'CUSTOM'
type AvailabilityStatus = 'ACTIVE' | 'EXPIRED' | 'CANCELLED' | 'FULFILLED'

interface Location {
  type: LocationType
  name?: string
  latitude?: number
  longitude?: number
}

interface Availability {
  id: string
  user: UserProfile
  startTime: string    // ISO 8601
  endTime: string
  location: Location
  status: AvailabilityStatus
  circleIds: string[]
  createdAt: string
}

// 发布有空请求
interface CreateAvailabilityRequest {
  startTime: string
  endTime: string
  locationType: LocationType
  locationName?: string
  latitude?: number
  longitude?: number
  circleIds: string[]   // 至少1个
}
```

### 消息

```typescript
type MessageType = 'TEXT' | 'AVAILABILITY_CARD' | 'CONFIRM_REQUEST' | 'CONFIRM_RESPONSE'

interface Message {
  id: string
  sender: UserProfile
  type: MessageType
  content?: string
  metadata?: Record<string, any>
  createdAt: string
  isRead: boolean
}

interface Conversation {
  id: string
  partner: UserProfile
  lastMessage?: Message
  unreadCount: number
  createdAt: string
}

// 发送消息请求
interface SendMessageRequest {
  type: MessageType
  content?: string
  metadata?: Record<string, any>
}
```

### 登录响应

```typescript
interface LoginResult {
  token: string
  user: User
  isNewUser: boolean
}

// 微信登录响应
interface WechatLoginResult {
  token: string
  user: User
  isNewUser: boolean
  joinedCircle?: CircleInfo  // 如果通过邀请链接注册
}
```

### 邀请

```typescript
type InvitationStatus = 'ACTIVE' | 'DISABLED' | 'EXPIRED'

interface Invitation {
  id: string
  code: string
  inviteUrl: string
  qrcodeUrl?: string
  inviter?: UserProfile
  circle?: CircleInfo
  maxUses: number
  useCount: number
  expiresAt?: string
  status: InvitationStatus
  isValid: boolean
  createdAt: string
}

interface CircleInfo {
  id: string
  name: string
  emoji: string
  color?: string
  memberCount?: number
}

// 创建邀请请求
interface CreateInvitationRequest {
  circleId?: string     // 关联圈子（可选）
  maxUses?: number      // 默认100
  expiresDays?: number  // 默认7天，最长30天
}

// 公开邀请信息（落地页展示）
interface InvitationPublicInfo {
  inviter: UserProfile
  circle?: CircleInfo
  isValid: boolean
}
```

### 好友

```typescript
type FriendshipSource = 'INVITATION' | 'SEARCH' | 'MANUAL'

interface FriendInfo {
  user: UserProfile
  source: FriendshipSource
  createdAt: string
}

interface FriendWithInvitation {
  user: UserProfile
  source: FriendshipSource
  circleName?: string    // 通过哪个圈子邀请的
  createdAt: string
}
```

### Agent 数据模型

```typescript
// 屏幕使用类型
type ActivityType = 'entertainment' | 'productivity' | 'communication' | 'idle'

// 位置类型
type PlaceType = 'home' | 'work' | 'leisure' | 'transit' | 'unknown'

// 屏幕使用数据
interface ScreenData {
  is_active: boolean                // 当前是否在用手机
  activity_type: ActivityType       // 使用类型
  session_duration_minutes: number  // 本次使用时长(分钟)
  last_active_minutes_ago: number   // 上次活跃是多久前(分钟)
}

// 位置数据
interface LocationData {
  place_type: PlaceType             // 位置类型
  at_place_since_minutes: number    // 在此位置待了多久(分钟)
}

// 状态上报请求
interface StatusReportRequest {
  screen?: ScreenData
  location?: LocationData
}

// 好友有空概率推荐
interface FriendRecommendation {
  friend_id: string
  name: string
  avatar?: string
  probability: number      // 0-100, -1表示无数据
  confidence: 'high' | 'medium' | 'low'
  reason: string           // 口语化理由，≤15字
  color: string            // 颜色代码
  updated_at: number       // 毫秒时间戳
}

// 有空概率列表响应
interface FreeProbabilityResponse {
  friends: FriendRecommendation[]
  generated_at: number     // 毫秒时间戳
}

// 概率颜色映射
// 80-100%: #22C55E (绿色-很可能有空)
// 60-79%:  #86EFAC (浅绿-可能有空)
// 40-59%:  #FACC15 (黄色-可能)
// 20-39%:  #FB923C (橙色-可能没空)
// 0-19%:   #EF4444 (红色-忙碌)
// 无数据:   #9CA3AF (灰色)
```

## 预设数据

### 预设地点

```typescript
const presetLocations = [
  { name: '三里屯', emoji: '🛍️', lat: 39.9334, lng: 116.4551 },
  { name: '国贸', emoji: '🏢', lat: 39.9087, lng: 116.4605 },
  { name: '望京', emoji: '🌆', lat: 39.9982, lng: 116.4744 },
  { name: '中关村', emoji: '💻', lat: 39.9836, lng: 116.3164 },
  { name: '五道口', emoji: '🎓', lat: 39.9927, lng: 116.3377 },
]
```

### 时间预设

```typescript
const timePresets = {
  tonight: { start: 18, end: 22 },    // 今晚
  tomorrow: { start: 10, end: 22 },   // 明天
  weekend: { start: 0, end: 24 },     // 周末全天
}
```

### 圈子颜色

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

## UI 规范

### 主题色
```
Primary: #10B981 (Emerald-500)
Secondary: #14B8A6 (Teal-500)
```

### 间距
```
xs: 4, sm: 8, md: 12, lg: 16, xl: 20, xxl: 24, xxxl: 32
```

### 圆角
```
sm: 8, md: 12, lg: 16, xl: 20, xxl: 24
```

### 圈子颜色
```
Pink: #EC4899, Orange: #F97316, Blue: #3B82F6
Green: #22C55E, Purple: #8B5CF6, Yellow: #EAB308
```

## 开发规范

### 命名
- Go: PascalCase (导出), camelCase (私有)
- 客户端类/结构体: PascalCase
- 客户端函数/变量: camelCase
- Android常量: SCREAMING_SNAKE_CASE

### Git
```
分支: main | develop | feature/xxx | bugfix/xxx | hotfix/xxx
提交: feat: | fix: | docs: | refactor:
```

## 兼容性要求

- iOS: 16.0+
- Android: 8.0+ (API 26)
- 性能: 首屏<1.5s，列表滚动60fps
