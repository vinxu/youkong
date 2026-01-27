# Agent 聊天接口文档

## 概述

本文档描述 Agent 自动聊天功能的 API 接口和 WebSocket 协议。

**Base URL**: `http://49.232.13.41:8080/api/v1`

**WebSocket URL**: `ws://49.232.13.41:8080/ws`

---

## HTTP 接口

### 1. 获取会话列表

获取当前用户的所有会话。

**请求**

```
GET /conversations
Authorization: Bearer <token>
```

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": [
    {
      "id": "conv-uuid-1",
      "partner": {
        "id": "user-uuid",
        "nickname": "小红",
        "avatar": "https://..."
      },
      "lastMessage": {
        "id": "msg-uuid",
        "sender": {
          "id": "user-uuid",
          "nickname": "小明",
          "avatar": "https://..."
        },
        "type": "TEXT",
        "content": "在干嘛呢？",
        "createdAt": "2024-01-27T10:30:00Z",
        "isRead": true
      },
      "unreadCount": 0,
      "createdAt": "2024-01-26T15:00:00Z"
    }
  ]
}
```

---

### 2. 创建会话

与指定好友创建或获取已存在的会话。

**请求**

```
POST /conversations
Authorization: Bearer <token>
Content-Type: application/json

{
  "partnerId": "user-uuid"
}
```

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "id": "conv-uuid",
    "partner": {
      "id": "user-uuid",
      "nickname": "小红",
      "avatar": "https://..."
    },
    "lastMessage": null,
    "unreadCount": 0,
    "createdAt": "2024-01-27T10:30:00Z"
  }
}
```

---

### 3. 获取消息列表

获取指定会话的消息历史。

**请求**

```
GET /conversations/:id/messages?limit=20&offset=0
Authorization: Bearer <token>
```

**参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 会话ID（路径参数） |
| limit | int | 否 | 每页数量，默认 20，最大 100 |
| offset | int | 否 | 偏移量，默认 0 |

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": [
    {
      "id": "msg-uuid-1",
      "sender": {
        "id": "user-uuid-1",
        "nickname": "小明",
        "avatar": "https://..."
      },
      "type": "TEXT",
      "content": "在干嘛呢？",
      "createdAt": "2024-01-27T10:30:00Z",
      "isRead": true
    },
    {
      "id": "msg-uuid-2",
      "sender": {
        "id": "user-uuid-2",
        "nickname": "小红",
        "avatar": "https://..."
      },
      "type": "TEXT",
      "content": "刚下班回家",
      "createdAt": "2024-01-27T10:31:00Z",
      "isRead": false
    }
  ]
}
```

**注意**: 返回的消息按时间**倒序**排列（最新的在前），前端需要 reverse 后显示。

---

### 4. 发送消息

手动发送一条消息（非 Agent 生成）。

**请求**

```
POST /conversations/:id/messages
Authorization: Bearer <token>
Content-Type: application/json

{
  "type": "TEXT",
  "content": "你好呀"
}
```

**参数**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 是 | 消息类型，目前支持 `TEXT` |
| content | string | 是 | 消息内容 |
| metadata | object | 否 | 扩展数据 |

**响应**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "id": "msg-uuid",
    "sender": {
      "id": "user-uuid",
      "nickname": "小明",
      "avatar": "https://..."
    },
    "type": "TEXT",
    "content": "你好呀",
    "createdAt": "2024-01-27T10:35:00Z",
    "isRead": false
  }
}
```

---

### 5. Agent 回复 ⭐

让 Agent（元婴）生成一条回复。这是核心接口。

**请求**

```
POST /conversations/:id/agent-reply
Authorization: Bearer <token>
```

**参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 会话ID（路径参数） |

无请求体。

**响应**

成功时返回生成的消息：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "id": "msg-uuid",
    "sender": {
      "id": "user-uuid",
      "nickname": "小明",
      "avatar": "https://..."
    },
    "type": "TEXT",
    "content": "今天累不累？要不周末一起吃个饭？",
    "createdAt": "2024-01-27T10:36:00Z"
  }
}
```

失败时：

```json
{
  "code": 5000,
  "message": "你的元婴罢工了"
}
```

**注意**:
1. 消息生成后会通过 WebSocket 同时推送给发送者和接收者
2. Agent 会自动同步对方的新消息到上下文
3. 响应时间可能较长（LLM 生成需要时间），建议前端设置 10 秒超时

---

## WebSocket 协议

### 连接

**URL**: `ws://49.232.13.41:8080/ws?token=<jwt_token>`

**参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| token | string | 是 | JWT Token（Query 参数） |

**连接流程**

```javascript
const token = localStorage.getItem('token')
const ws = new WebSocket(`ws://49.232.13.41:8080/ws?token=${token}`)

ws.onopen = () => {
  console.log('WebSocket 连接成功')
}

ws.onclose = () => {
  console.log('WebSocket 断开，5秒后重连')
  setTimeout(connect, 5000)
}

ws.onerror = (error) => {
  console.error('WebSocket 错误:', error)
}
```

**认证失败**

如果 token 无效或过期，连接会被拒绝，返回 HTTP 401。

---

### 消息格式

所有 WebSocket 消息都是 JSON 格式：

```typescript
interface WSMessage {
  type: 'NEW_MESSAGE' | 'PING' | 'PONG'
  data?: any
}
```

---

### 消息类型

#### 1. NEW_MESSAGE - 新消息推送

当有新消息时（包括自己发的和对方发的），服务端会推送：

```json
{
  "type": "NEW_MESSAGE",
  "data": {
    "conversation_id": "conv-uuid",
    "message": {
      "id": "msg-uuid",
      "sender": {
        "id": "user-uuid",
        "nickname": "小明",
        "avatar": "https://..."
      },
      "type": "TEXT",
      "content": "今天累不累？",
      "createdAt": "2024-01-27T10:36:00Z",
      "isRead": false
    }
  }
}
```

**处理示例**

```javascript
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data)

  if (msg.type === 'NEW_MESSAGE') {
    const { conversation_id, message } = msg.data

    // 如果是当前会话，添加消息到列表
    if (conversation_id === currentConversationId) {
      setMessages(prev => {
        // 避免重复
        if (prev.some(m => m.id === message.id)) {
          return prev
        }
        return [...prev, message]
      })
    }

    // 可选：更新会话列表的未读数
    updateConversationUnread(conversation_id)
  }
}
```

#### 2. PING/PONG - 心跳

客户端应定期发送心跳保持连接：

**客户端发送**

```json
{
  "type": "PING"
}
```

**服务端响应**

```json
{
  "type": "PONG"
}
```

**建议**: 每 30 秒发送一次心跳

```javascript
setInterval(() => {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'PING' }))
  }
}, 30000)
```

---

### 完整示例

```typescript
// hooks/useWebSocket.ts
import { useEffect, useRef, useCallback, useState } from 'react'
import type { Message } from '../types'

const WS_URL = 'ws://49.232.13.41:8080/ws'

interface UseWebSocketOptions {
  onMessage?: (conversationId: string, message: Message) => void
  enabled?: boolean
}

export function useWebSocket({ onMessage, enabled = true }: UseWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null)
  const [isConnected, setIsConnected] = useState(false)

  const connect = useCallback(() => {
    if (!enabled) return

    const token = localStorage.getItem('token')
    if (!token) return

    const ws = new WebSocket(`${WS_URL}?token=${token}`)

    ws.onopen = () => {
      console.log('[WS] Connected')
      setIsConnected(true)
    }

    ws.onclose = () => {
      console.log('[WS] Disconnected')
      setIsConnected(false)
      // 自动重连
      setTimeout(connect, 5000)
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        if (msg.type === 'NEW_MESSAGE' && msg.data && onMessage) {
          onMessage(msg.data.conversation_id, msg.data.message)
        }
      } catch (e) {
        console.error('[WS] Parse error:', e)
      }
    }

    wsRef.current = ws
  }, [enabled, onMessage])

  // 心跳
  useEffect(() => {
    if (!isConnected) return

    const interval = setInterval(() => {
      wsRef.current?.send(JSON.stringify({ type: 'PING' }))
    }, 30000)

    return () => clearInterval(interval)
  }, [isConnected])

  useEffect(() => {
    connect()
    return () => wsRef.current?.close()
  }, [connect])

  return { isConnected }
}
```

---

## 数据类型定义

### TypeScript 类型

```typescript
// 用户简介
interface UserProfile {
  id: string
  nickname: string
  avatar?: string
}

// 消息类型
type MessageType = 'TEXT' | 'AVAILABILITY_CARD' | 'CONFIRM_REQUEST' | 'CONFIRM_RESPONSE'

// 消息
interface Message {
  id: string
  sender: UserProfile
  type: MessageType
  content?: string
  metadata?: Record<string, unknown>
  createdAt: string  // ISO 8601
  isRead: boolean
}

// 会话
interface Conversation {
  id: string
  partner: UserProfile
  lastMessage?: Message
  unreadCount: number
  createdAt: string  // ISO 8601
}

// WebSocket 消息
type WSMessageType = 'NEW_MESSAGE' | 'PING' | 'PONG'

interface WSMessage {
  type: WSMessageType
  data?: {
    conversation_id: string
    message: Message
  }
}

// API 响应
interface ApiResponse<T> {
  code: number      // 0=成功, 非0=失败
  message: string
  data?: T
}
```

---

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

**常见错误消息**

| 消息 | 说明 |
|------|------|
| "你的元婴罢工了" | LLM 调用失败 |
| "会话不存在" | 会话ID无效 |
| "无权限访问此会话" | 用户不是会话参与者 |

---

## 调试指南

### 使用 cURL 测试

**1. 登录获取 Token**

```bash
# 发送验证码（测试账号可跳过）
curl -X POST http://49.232.13.41:8080/api/v1/auth/sms/send \
  -H "Content-Type: application/json" \
  -d '{"phone": "13800000001"}'

# 验证码登录
curl -X POST http://49.232.13.41:8080/api/v1/auth/sms/verify \
  -H "Content-Type: application/json" \
  -d '{"phone": "13800000001", "code": "111111"}'

# 保存返回的 token
export TOKEN="eyJhbGci..."
```

**2. 获取会话列表**

```bash
curl http://49.232.13.41:8080/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN"
```

**3. 创建会话**

```bash
curl -X POST http://49.232.13.41:8080/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"partnerId": "对方用户ID"}'
```

**4. Agent 回复**

```bash
curl -X POST http://49.232.13.41:8080/api/v1/conversations/{会话ID}/agent-reply \
  -H "Authorization: Bearer $TOKEN"
```

**5. 获取消息**

```bash
curl "http://49.232.13.41:8080/api/v1/conversations/{会话ID}/messages?limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

### 使用 wscat 测试 WebSocket

```bash
# 安装 wscat
npm install -g wscat

# 连接 WebSocket
wscat -c "ws://49.232.13.41:8080/ws?token=$TOKEN"

# 连接后发送心跳
> {"type": "PING"}
< {"type": "PONG"}

# 等待接收新消息推送
< {"type":"NEW_MESSAGE","data":{"conversation_id":"...","message":{...}}}
```

---

## 数据库迁移

部署前需要执行数据库迁移：

```bash
# SSH 登录服务器
ssh root@49.232.13.41

# 登录 MySQL
sudo mysql youkong

# 执行迁移 SQL
source /path/to/005_agent_chat.sql

# 或者直接粘贴以下 SQL
```

```sql
-- 用户人设表
CREATE TABLE IF NOT EXISTS user_personas (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL UNIQUE,
    personality TEXT,
    speaking_style TEXT,
    interests TEXT,
    social_habits TEXT,
    common_phrases TEXT,
    emoji_usage VARCHAR(50),
    message_length VARCHAR(20),
    confidence_score INT DEFAULT 0,
    sample_count INT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 关系画像表
CREATE TABLE IF NOT EXISTS relationships (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    friend_id VARCHAR(36) NOT NULL,
    closeness VARCHAR(20) DEFAULT '还不太熟',
    interact_style VARCHAR(20) DEFAULT '未知',
    common_topics TEXT,
    tone_with_this TEXT,
    joke_level VARCHAR(20) DEFAULT '低',
    shared_memory TEXT,
    message_count INT DEFAULT 0,
    confidence_score INT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_friend (user_id, friend_id),
    INDEX idx_user_id (user_id),
    INDEX idx_friend_id (friend_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 对话上下文表
CREATE TABLE IF NOT EXISTS conversation_contexts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    conversation_id VARCHAR(36) NOT NULL UNIQUE,
    messages JSON NOT NULL,
    token_count INT DEFAULT 0,
    summary TEXT,
    last_sync_msg_id VARCHAR(36),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_conversation (conversation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```
