# 聊天消息调试指南

## 问题现象

13800138000 和 13800000001 两个账户对话，双方都收不到对方的消息。

## 可能的原因

### 1. 会话ID不一致

两个用户可能各自创建了不同的会话，导致消息发送到不同的会话中。

### 2. 会话创建失败

后端创建会话时可能因为唯一性约束失败，但前端没有正确处理错误。

### 3. WebSocket 未连接

实时消息推送依赖 WebSocket，如果未连接会导致消息无法实时接收。

---

## 调试步骤

### 步骤 1: 检查数据库中的会话

登录数据库：
```bash
mysql -u root -p youkong
```

查询两个用户的会话：
```sql
-- 查询用户ID（通过手机号）
SELECT id, phone, nickname FROM users WHERE phone IN ('13800138000', '13800000001');

-- 假设得到的 user_id 分别是：user_id_1 和 user_id_2

-- 查询这两个用户之间的会话
SELECT * FROM conversations
WHERE (user1_id = 'user_id_1' AND user2_id = 'user_id_2')
   OR (user1_id = 'user_id_2' AND user2_id = 'user_id_1');

-- 查询所有会话
SELECT * FROM conversations ORDER BY created_at DESC LIMIT 10;
```

**期望结果**：应该只有 **1 个会话**，且 `user1_id < user2_id`（按字典序排列）

**如果有多个会话**：说明创建会话的逻辑有问题

**如果没有会话**：说明会话创建失败

### 步骤 2: 检查消息表

```sql
-- 查询最近的消息
SELECT m.*, c.user1_id, c.user2_id, u.nickname as sender_name
FROM messages m
JOIN conversations c ON m.conversation_id = c.id
JOIN users u ON m.sender_id = u.id
ORDER BY m.created_at DESC LIMIT 20;

-- 查询特定会话的消息
SELECT m.*, u.nickname as sender_name
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE m.conversation_id = 'conversation_id_here'
ORDER BY m.created_at ASC;
```

**期望结果**：所有消息应该属于同一个 `conversation_id`

### 步骤 3: 检查后端日志

启动后端时查看日志：
```bash
cd Backend
make dev
```

观察日志中是否有以下错误：
- `创建会话失败: Error 1062: Duplicate entry`（唯一性约束冲突）
- `会话不存在`
- `无权限访问此会话`

### 步骤 4: 使用 curl 测试完整流程

#### 4.1 登录两个账户

```bash
# 用户1: 13800138000
curl -X POST "http://localhost:8080/api/v1/auth/sms/verify" \
  -H "Content-Type: application/json" \
  -d '{"phone": "13800138000", "code": "123456"}' | jq

# 保存 token1

# 用户2: 13800000001
curl -X POST "http://localhost:8080/api/v1/auth/sms/verify" \
  -H "Content-Type: application/json" \
  -d '{"phone": "13800000001", "code": "111111"}' | jq

# 保存 token2
```

#### 4.2 获取用户信息

```bash
# 用户1 获取自己的信息
curl -X GET "http://localhost:8080/api/v1/users/me" \
  -H "Authorization: Bearer token1" | jq

# 用户2 获取自己的信息
curl -X GET "http://localhost:8080/api/v1/users/me" \
  -H "Authorization: Bearer token2" | jq

# 记录两个用户的 user_id
```

#### 4.3 用户1 创建与用户2 的会话

```bash
curl -X POST "http://localhost:8080/api/v1/conversations" \
  -H "Authorization: Bearer token1" \
  -H "Content-Type: application/json" \
  -d '{"partnerId": "user2_id"}' | jq

# 记录返回的 conversation_id
```

#### 4.4 用户2 也创建与用户1 的会话

```bash
curl -X POST "http://localhost:8080/api/v1/conversations" \
  -H "Authorization: Bearer token2" \
  -H "Content-Type: application/json" \
  -d '{"partnerId": "user1_id"}' | jq

# 查看返回的 conversation_id 是否与上一步相同
```

**关键检查**：两次创建会话返回的 `conversation_id` **必须相同**！

如果不同，说明后端的 `GetOrCreateConversation` 有问题。

#### 4.5 发送消息

```bash
# 用户1 发送消息
curl -X POST "http://localhost:8080/api/v1/conversations/conversation_id/messages" \
  -H "Authorization: Bearer token1" \
  -H "Content-Type: application/json" \
  -d '{"type": "TEXT", "content": "你好，我是用户1"}' | jq

# 用户2 发送消息
curl -X POST "http://localhost:8080/api/v1/conversations/conversation_id/messages" \
  -H "Authorization: Bearer token2" \
  -H "Content-Type: application/json" \
  -d '{"type": "TEXT", "content": "你好，我是用户2"}' | jq
```

#### 4.6 获取消息列表

```bash
# 用户1 获取消息
curl -X GET "http://localhost:8080/api/v1/conversations/conversation_id/messages" \
  -H "Authorization: Bearer token1" | jq

# 用户2 获取消息
curl -X GET "http://localhost:8080/api/v1/conversations/conversation_id/messages" \
  -H "Authorization: Bearer token2" | jq
```

**期望结果**：两个用户都应该看到 **两条消息**

---

## 已知问题修复

### 问题 1: GetConversationByUsers 可能的 Bug

**文件**: `Backend/internal/repository/message_repo.go` (行 39-50)

**当前代码**：
```go
func (r *MessageRepository) GetConversationByUsers(ctx context.Context, user1ID, user2ID string) (*model.Conversation, error) {
	var conv model.Conversation
	query := `SELECT * FROM conversations WHERE (user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)`
	err := r.db.GetContext(ctx, &conv, query, user1ID, user2ID, user2ID, user1ID)
	...
}
```

这个查询是正确的，因为它会检查两种顺序的组合。

### 问题 2: iOS 前端没有正确处理会话创建

**文件**: `iOS/YouKong/Presentation/Screens/Messages/ChatViewModel.swift` (行 136-147)

**潜在问题**：`createConversation()` 成功后没有立即更新 `conversationId`

**验证方法**：在 iOS 中添加日志
```swift
private func createConversation() async {
    guard let partnerId = partner?.id else {
        print("[ChatViewModel] Missing partner information")
        return
    }
    do {
        let conversation = try await repository.createConversation(partnerId: partnerId)
        self.conversationId = conversation.id
        print("[ChatViewModel] ✅ Created conversation: \(conversation.id)")  // 添加日志
        print("[ChatViewModel] Partner: \(partnerId)")  // 添加日志
    } catch {
        print("[ChatViewModel] ❌ Create conversation error: \(error)")
    }
}
```

---

## 快速修复方案

### 方案 1: 清理数据库重新测试

```sql
-- 删除所有测试账户的会话和消息
DELETE m FROM messages m
JOIN conversations c ON m.conversation_id = c.id
JOIN users u1 ON c.user1_id = u1.id
JOIN users u2 ON c.user2_id = u2.id
WHERE u1.phone IN ('13800138000', '13800000001')
   OR u2.phone IN ('13800138000', '13800000001');

DELETE FROM conversations
WHERE user1_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001'))
   OR user2_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001'));
```

然后重新在 App 中测试发送消息。

### 方案 2: 检查是否需要添加调试日志

如果问题依然存在，需要在后端添加更详细的日志：

**文件**: `Backend/internal/service/conversation_service.go`

在关键位置添加日志：
```go
func (s *ConversationService) GetOrCreateConversation(ctx context.Context, userID, partnerID string) (*model.Conversation, error) {
	log.Printf("[DEBUG] GetOrCreateConversation: userID=%s, partnerID=%s", userID, partnerID)

	conv, err := s.messageRepo.GetConversationByUsers(ctx, userID, partnerID)
	if err != nil {
		log.Printf("[ERROR] GetConversationByUsers failed: %v", err)
		return nil, fmt.Errorf("查询会话失败: %w", err)
	}

	if conv != nil {
		log.Printf("[DEBUG] Found existing conversation: %s", conv.ID)
		return conv, nil
	}

	// 确保user1_id < user2_id以保持一致性
	user1, user2 := userID, partnerID
	if user1 > user2 {
		user1, user2 = user2, user1
	}

	log.Printf("[DEBUG] Creating new conversation: user1=%s, user2=%s", user1, user2)

	conv = &model.Conversation{
		ID:        uuid.New().String(),
		User1ID:   user1,
		User2ID:   user2,
		CreatedAt: time.Now(),
	}

	if err := s.messageRepo.CreateConversation(ctx, conv); err != nil {
		log.Printf("[ERROR] CreateConversation failed: %v", err)
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	log.Printf("[DEBUG] Successfully created conversation: %s", conv.ID)
	return conv, nil
}
```

---

## 总结

请按照以下顺序排查：

1. **数据库检查** - 查看是否有重复会话或消息丢失
2. **后端日志** - 观察创建会话和发送消息的日志
3. **curl 测试** - 绕过前端直接测试后端 API
4. **iOS 日志** - 查看前端是否正确获取了 conversationId

最可能的原因是：
- ✅ 两个用户各自创建了不同的会话（conversationId 不一致）
- ✅ 后端 `GetConversationByUsers` 没有正确找到已存在的会话
- ✅ iOS 前端在创建会话后没有正确保存 conversationId

请先执行**步骤 1 和步骤 4**，然后告诉我结果，我可以帮你进一步定位问题！
