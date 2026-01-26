# CLAUDE.md - 后端服务

> ⚠️ **先阅读** `../CLAUDE.md` **了解共享定义**

---

## 我的职责

- 实现 REST API（见 `../shared/api-contracts.yaml`）
- 数据库设计与存储
- AI 建圈服务（调用 Claude API）
- 推送通知服务（FCM/APNs）
- WebSocket 实时消息

---

## 技术栈

```
Runtime:    Node.js 20 LTS
Framework:  Fastify 4.x
Database:   PostgreSQL 15
Cache:      Redis 7
ORM:        Prisma
AI:         Anthropic Claude API
Push:       Firebase Admin SDK
```

---

## 项目结构

```
backend/
├── src/
│   ├── app.ts                 # 应用入口
│   ├── config/
│   │   ├── index.ts           # 配置管理
│   │   └── env.ts             # 环境变量
│   ├── routes/
│   │   ├── auth.ts            # 认证路由
│   │   ├── availabilities.ts  # 有空状态路由
│   │   ├── circles.ts         # 圈子路由
│   │   └── ai.ts              # AI服务路由
│   ├── services/
│   │   ├── auth.service.ts
│   │   ├── availability.service.ts
│   │   ├── circle.service.ts
│   │   ├── ai-circle.service.ts
│   │   └── push.service.ts
│   ├── repositories/
│   │   ├── user.repo.ts
│   │   ├── availability.repo.ts
│   │   └── circle.repo.ts
│   ├── middlewares/
│   │   ├── auth.ts
│   │   └── error-handler.ts
│   └── utils/
│       ├── jwt.ts
│       └── response.ts
├── prisma/
│   └── schema.prisma
├── tests/
├── package.json
└── tsconfig.json
```

---

## 数据库 Schema (Prisma)

```prisma
model User {
  id           String   @id @default(uuid())
  nickname     String
  avatar       String?
  phone        String?  @unique
  wechatOpenId String?  @unique
  createdAt    DateTime @default(now())
  lastActiveAt DateTime @default(now())
  
  availabilities Availability[]
  ownedCircles   Circle[]       @relation("CircleOwner")
  circles        CircleMember[]
  messages       Message[]
}

model Circle {
  id        String   @id @default(uuid())
  name      String
  emoji     String
  color     String
  ownerId   String
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
  
  owner          User                  @relation("CircleOwner", fields: [ownerId], references: [id])
  members        CircleMember[]
  availabilities AvailabilityCircle[]
}

model CircleMember {
  circleId  String
  userId    String
  joinedAt  DateTime @default(now())
  
  circle Circle @relation(fields: [circleId], references: [id], onDelete: Cascade)
  user   User   @relation(fields: [userId], references: [id], onDelete: Cascade)
  
  @@id([circleId, userId])
}

model Availability {
  id              String   @id @default(uuid())
  userId          String
  startTime       DateTime
  endTime         DateTime
  locationType    String   // PRESET, FLEXIBLE, CUSTOM
  locationName    String?
  locationLat     Float?
  locationLng     Float?
  locationRadius  Int?
  status          String   @default("ACTIVE") // ACTIVE, EXPIRED, CANCELLED, FULFILLED
  createdAt       DateTime @default(now())
  updatedAt       DateTime @updatedAt
  
  user    User                 @relation(fields: [userId], references: [id])
  circles AvailabilityCircle[]
}

model AvailabilityCircle {
  availabilityId String
  circleId       String
  
  availability Availability @relation(fields: [availabilityId], references: [id], onDelete: Cascade)
  circle       Circle       @relation(fields: [circleId], references: [id], onDelete: Cascade)
  
  @@id([availabilityId, circleId])
}

model Message {
  id             String   @id @default(uuid())
  conversationId String
  senderId       String
  type           String   // TEXT, AVAILABILITY_CARD, CONFIRM_REQUEST, CONFIRM_RESPONSE
  content        String
  metadata       Json?
  createdAt      DateTime @default(now())
  readAt         DateTime?
  
  sender User @relation(fields: [senderId], references: [id])
  
  @@index([conversationId])
}
```

---

## AI 建圈服务

### 调用 Claude API

```typescript
// services/ai-circle.service.ts
import Anthropic from '@anthropic-ai/sdk'

export class AICircleService {
  private client = new Anthropic()

  async analyzeContacts(contacts: ContactInfo[], groupSources: GroupSource[]) {
    const response = await this.client.messages.create({
      model: 'claude-sonnet-4-20250514',
      max_tokens: 2000,
      messages: [{
        role: 'user',
        content: `
分析以下用户的社交数据，生成圈子建议：

## 联系人列表
${JSON.stringify(contacts)}

## 微信群来源
${JSON.stringify(groupSources)}

## 要求
1. 生成3-5个圈子建议
2. 每个圈子包含：name, emoji, members, reason
3. 优先推荐高频使用的圈子

## 输出格式 (JSON)
{
  "circles": [
    { "name": "...", "emoji": "...", "members": [...], "reason": "..." }
  ]
}
        `
      }]
    })
    
    return JSON.parse(response.content[0].text)
  }
}
```

---

## 推送通知

```typescript
// services/push.service.ts
import admin from 'firebase-admin'

export class PushService {
  async notifyFriendAvailable(userIds: string[], availability: Availability) {
    const tokens = await this.getTokens(userIds)
    
    await admin.messaging().sendEachForMulticast({
      tokens,
      notification: {
        title: '有朋友有空啦',
        body: `${availability.user.nickname} ${this.formatTime(availability)} 有空`
      },
      data: {
        type: 'FRIEND_AVAILABLE',
        availabilityId: availability.id
      }
    })
  }
}
```

---

## 环境变量

```env
# .env
DATABASE_URL=postgresql://user:pass@localhost:5432/youkong
REDIS_URL=redis://localhost:6379

JWT_SECRET=your-jwt-secret
JWT_EXPIRES_IN=7d

ANTHROPIC_API_KEY=sk-ant-xxx

FIREBASE_PROJECT_ID=youkong
FIREBASE_PRIVATE_KEY=xxx
FIREBASE_CLIENT_EMAIL=xxx

WECHAT_APP_ID=xxx
WECHAT_APP_SECRET=xxx
```

---

## 启动命令

```bash
# 开发
npm run dev

# 生成 Prisma Client
npx prisma generate

# 数据库迁移
npx prisma migrate dev

# 生产构建
npm run build
npm start
```

---

## 与前端的接口

- REST API: 标准 HTTP
- 实时消息: WebSocket (`/ws`)
- 推送: FCM (Android) / APNs (iOS)

---

## 待处理事件

检查 `../shared/events/` 目录获取待实现的功能。

---

*版本: 1.0.0*
