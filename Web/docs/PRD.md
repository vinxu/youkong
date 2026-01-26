# YouKong Web 落地页 - 产品需求文档

## 1. 功能概述

### 1.1 产品定位

为「有空」App 提供 H5 落地页，支持邀请链接的 Web 端展示和登录。

### 1.2 核心目标

- 被邀请者通过扫码/链接进入落地页
- 完成手机号登录并自动加入好友/圈子
- 查看邀请者的有空状态
- 引导下载 App

---

## 2. 用户流程

```
用户扫描二维码
      │
      ▼
┌─────────────────────────────┐
│  邀请落地页 (InvitePage)      │
│                             │
│  [邀请者头像]                │
│  小明 邀请你加入「有空」      │
│  加入「闺蜜」圈子             │
│                             │
│  ┌─────────────────────┐    │
│  │ 📱 输入手机号        │    │
│  └─────────────────────┘    │
│  ┌─────────────────────┐    │
│  │ 🔢 输入验证码        │    │
│  └─────────────────────┘    │
│                             │
│  [登录并加入]                │
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│  欢迎页 (WelcomePage)        │
│                             │
│  🎉 加入成功！               │
│                             │
│  你已成为 小明 的好友         │
│  并加入了「闺蜜」圈子         │
│                             │
│  [查看 TA 的有空状态]         │
│  [下载「有空」App]            │
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│  有空状态页 (AvailabilityPage)│
│                             │
│  小明 最近有空：             │
│                             │
│  ┌───────────────────────┐  │
│  │ 📅 今晚 18:00-22:00   │  │
│  │ 📍 三里屯             │  │
│  └───────────────────────┘  │
│                             │
│  想发起你的有空？            │
│  [下载「有空」App]           │
└─────────────────────────────┘
```

---

## 3. 页面详情

### 3.1 邀请落地页 (InvitePage)

**路由**: `/i/:code`

**功能说明**:
1. 根据 URL 中的邀请码获取邀请详情
2. 显示邀请者头像、昵称
3. 显示关联圈子信息（如有）
4. 手机号 + 验证码登录
5. 登录成功后自动接受邀请

**UI 规范**:
- 邀请者头像：圆形，80px，居中
- 邀请文案：24px 粗体
- 圈子标签：带 emoji 和颜色
- 输入框：圆角 12px，高度 48px
- 登录按钮：主色背景，圆角 12px

**异常处理**:
- 邀请码不存在 → 显示失效页
- 邀请已过期 → 显示失效页
- 邀请已用完 → 显示失效页
- 网络错误 → 显示重试按钮

### 3.2 欢迎页 (WelcomePage)

**路由**: `/welcome`

**功能说明**:
1. 显示加入成功信息
2. 展示邀请者信息
3. 展示加入的圈子（如有）
4. 提供查看有空状态入口
5. 提供下载 App 入口

**数据来源**: 从登录响应或 sessionStorage 获取

### 3.3 有空状态页 (AvailabilityPage)

**路由**: `/availability/:userId`

**功能说明**:
1. 显示指定用户的有空状态列表
2. 时间格式化（今天/明天/日期）
3. 地点类型展示
4. 底部固定下载引导

**API 调用**:
```
GET /api/v1/availabilities/friends
Authorization: Bearer <token>
```

### 3.4 邀请失效页 (ExpiredPage)

**路由**: `/expired`

**功能说明**:
- 显示邀请已失效提示
- 引导下载 App

### 3.5 404 页面 (NotFoundPage)

**路由**: `*`

**功能说明**:
- 显示页面不存在提示
- 提供返回首页按钮

---

## 4. API 接口

### 4.1 获取邀请详情（公开）

```
GET /api/v1/invitations/:code

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "inviter": {
      "id": "xxx",
      "nickname": "小明",
      "avatar": "https://..."
    },
    "circle": {
      "id": "xxx",
      "name": "闺蜜",
      "emoji": "👯",
      "color": "#EC4899",
      "memberCount": 5
    },
    "isValid": true
  }
}
```

### 4.2 发送验证码

```
POST /api/v1/auth/sms/send
Content-Type: application/json

{
  "phone": "13800000001"
}

Response:
{
  "code": 0,
  "message": "验证码已发送"
}
```

### 4.3 验证码登录

```
POST /api/v1/auth/sms/verify
Content-Type: application/json

{
  "phone": "13800000001",
  "code": "123456"
}

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": "xxx",
      "nickname": "用户123",
      "phone": "13800000001"
    },
    "isNewUser": true
  }
}
```

### 4.4 接受邀请

```
POST /api/v1/invitations/:code/accept
Authorization: Bearer <token>

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "joinedCircle": {
      "id": "xxx",
      "name": "闺蜜",
      "emoji": "👯",
      "memberCount": 6
    }
  }
}
```

### 4.5 获取好友有空状态

```
GET /api/v1/availabilities/friends
Authorization: Bearer <token>

Response:
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "xxx",
      "user": {
        "id": "xxx",
        "nickname": "小明",
        "avatar": "https://..."
      },
      "startTime": "2024-01-20T18:00:00+08:00",
      "endTime": "2024-01-20T22:00:00+08:00",
      "location": {
        "type": "PRESET",
        "name": "三里屯"
      },
      "status": "ACTIVE",
      "createdAt": "2024-01-20T10:00:00+08:00"
    }
  ]
}
```

---

## 5. 测试账号

| 手机号 | 验证码 | 说明 |
|--------|--------|------|
| 13800000001 | 111111 | 测试账号1 |
| 13800000002 | 222222 | 测试账号2 |
| 13800000003 | 333333 | 测试账号3 |

---

## 6. 验收标准

### 6.1 功能验收

- [ ] 访问有效邀请链接，正确显示邀请信息
- [ ] 访问无效邀请链接，显示失效页面
- [ ] 手机号格式验证正确
- [ ] 验证码 60 秒倒计时正常
- [ ] 登录成功后自动接受邀请
- [ ] 欢迎页正确显示邀请者和圈子信息
- [ ] 有空状态页正确显示列表
- [ ] 下载按钮可点击

### 6.2 UI 验收

- [ ] 移动端适配正常（375px - 428px）
- [ ] 页面加载有 loading 状态
- [ ] 错误提示清晰明确
- [ ] 按钮点击有反馈
- [ ] 颜色符合设计规范

### 6.3 性能验收

- [ ] 首屏加载 < 2s
- [ ] 页面切换流畅
- [ ] 接口超时有提示

---

## 7. 技术实现

### 7.1 技术栈

| 项目 | 选型 |
|------|------|
| 框架 | React 18 |
| 语言 | TypeScript |
| 构建 | Vite |
| 样式 | Tailwind CSS |
| 路由 | React Router v6 |
| 状态 | Zustand |
| HTTP | Axios |

### 7.2 项目结构

```
Web/
├── src/
│   ├── api/           # API 请求
│   ├── components/    # 通用组件
│   ├── pages/         # 页面组件
│   ├── stores/        # 状态管理
│   ├── hooks/         # 自定义 hooks
│   ├── types/         # TypeScript 类型
│   └── utils/         # 工具函数
├── public/            # 静态资源
├── docs/              # 文档
└── ...
```

### 7.3 开发命令

```bash
# 安装依赖
npm install

# 开发模式
npm run dev

# 构建生产版本
npm run build

# 预览生产版本
npm run preview
```
