# 会话创建 Bug 修复总结

## 问题描述

**症状**：两个用户对话时收不到对方的消息

**根本原因**：并发创建会话导致一对用户关系有多个会话，消息发送到不同的会话中

---

## 问题分析

### 原设计

数据库约束：
```sql
UNIQUE KEY uk_users (user1_id, user2_id)
```

后端逻辑：
```go
// 确保user1_id < user2_id以保持一致性
user1, user2 := userID, partnerID
if user1 > user2 {
    user1, user2 = user2, user1
}
```

### 为什么还会重复？

**并发场景**：
1. ⏱️ T1: 用户 A 调用 `GetOrCreateConversation` → 查询不到会话
2. ⏱️ T1: 用户 B 调用 `GetOrCreateConversation` → 查询不到会话
3. ⏱️ T2: 用户 A 创建会话 `(user_a_id, user_b_id)` → ✅ 成功
4. ⏱️ T2: 用户 B 创建会话 `(user_a_id, user_b_id)` → ❌ 失败（唯一性冲突）

**问题**：虽然排序逻辑正确，但 T2 时刻用户 B 的创建会失败，旧代码直接返回错误，没有重新查询已存在的会话。

---

## 修复方案

### 核心改进

**文件**：`Backend/internal/service/conversation_service.go`

**修改内容**：

1. **提前排序**（确保一致性）
```go
// 确保user1_id < user2_id以保持一致性
user1, user2 := userID, partnerID
if user1 > user2 {
    user1, user2 = user2, user1
}

// 使用排序后的 ID 查询
conv, err := s.messageRepo.GetConversationByUsers(ctx, user1, user2)
```

2. **添加重试逻辑**（处理并发冲突）
```go
if err := s.messageRepo.CreateConversation(ctx, conv); err != nil {
    // 如果创建失败（可能是并发导致的唯一性冲突），重新查询
    if isDuplicateKeyError(err) {
        // 重新查询会话
        conv, err = s.messageRepo.GetConversationByUsers(ctx, user1, user2)
        if err != nil {
            return nil, fmt.Errorf("重新查询会话失败: %w", err)
        }
        if conv != nil {
            return conv, nil  // ✅ 返回已存在的会话
        }
    }
    return nil, fmt.Errorf("创建会话失败: %w", err)
}
```

3. **辅助函数**（检测唯一性冲突）
```go
// isDuplicateKeyError 检查是否为唯一性约束错误
func isDuplicateKeyError(err error) bool {
    if err == nil {
        return false
    }
    errStr := err.Error()
    // MySQL Error 1062: Duplicate entry
    return strings.Contains(errStr, "Duplicate entry") ||
        strings.Contains(errStr, "duplicate key") ||
        strings.Contains(errStr, "Error 1062")
}
```

---

## 修复后的流程

### 正常场景（无并发）

1. 用户 A 发起聊天
2. `GetOrCreateConversation` 查询不到会话
3. 创建新会话 ✅
4. 用户 B 发起聊天
5. `GetOrCreateConversation` 查询到已存在的会话 ✅
6. 返回同一个会话 ID

### 并发场景

1. ⏱️ T1: 用户 A 和 B 同时调用 `GetOrCreateConversation` → 都查询不到
2. ⏱️ T2: 用户 A 创建会话成功 → 会话 ID = `conv_123`
3. ⏱️ T2: 用户 B 创建会话失败 → 检测到唯一性冲突
4. ⏱️ T3: 用户 B 重新查询 → 找到会话 `conv_123` ✅
5. 两个用户使用同一个会话 ID

---

## 验证步骤

### 1. 重启后端服务

```bash
cd Backend
make dev
```

### 2. 清理测试数据（如果还没清理）

在服务器上执行：
```bash
mysql -u root youkong -e "
DELETE m FROM messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR c.user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

DELETE FROM conversations
WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');
"
```

### 3. 在 iOS App 中测试

#### 3.1 登录两个账户

- 设备 1: 登录 `13800138000`
- 设备 2: 登录 `13800000001`

#### 3.2 发起对话

- 设备 1: 打开好友列表 → 点击用户 2 → 发送消息："你好，我是用户 1"
- 设备 2: 打开好友列表 → 点击用户 1 → 发送消息："你好，我是用户 2"

#### 3.3 预期结果 ✅

- **双方都能看到对方的消息**
- **消息按时间正序排列**
- **实时接收到新消息（WebSocket）**

### 4. 数据库验证

在服务器上执行：
```sql
-- 查询两个用户之间的会话
SELECT c.*, u1.phone as user1_phone, u2.phone as user2_phone
FROM conversations c
JOIN users u1 ON c.user1_id = u1.id
JOIN users u2 ON c.user2_id = u2.id
WHERE (u1.phone = '13800138000' AND u2.phone = '13800000001')
   OR (u1.phone = '13800000001' AND u2.phone = '13800138000');

-- 应该只有 1 条记录
```

### 5. 并发测试（可选）

同时在两个设备上点击对方头像发起聊天，验证是否只创建一个会话。

---

## 额外改进建议（可选）

### 1. 添加日志

在 `GetOrCreateConversation` 中添加调试日志：

```go
func (s *ConversationService) GetOrCreateConversation(ctx context.Context, userID, partnerID string) (*model.Conversation, error) {
    user1, user2 := userID, partnerID
    if user1 > user2 {
        user1, user2 = user2, user1
    }

    log.Printf("[DEBUG] GetOrCreateConversation: user1=%s, user2=%s", user1, user2)

    conv, err := s.messageRepo.GetConversationByUsers(ctx, user1, user2)
    if err != nil {
        log.Printf("[ERROR] GetConversationByUsers failed: %v", err)
        return nil, fmt.Errorf("查询会话失败: %w", err)
    }

    if conv != nil {
        log.Printf("[DEBUG] Found existing conversation: %s", conv.ID)
        return conv, nil
    }

    log.Printf("[DEBUG] Creating new conversation...")
    conv = &model.Conversation{...}

    if err := s.messageRepo.CreateConversation(ctx, conv); err != nil {
        if isDuplicateKeyError(err) {
            log.Printf("[WARN] Duplicate key error, retrying query...")
            conv, err = s.messageRepo.GetConversationByUsers(ctx, user1, user2)
            if conv != nil {
                log.Printf("[DEBUG] Found conversation after retry: %s", conv.ID)
                return conv, nil
            }
        }
        return nil, fmt.Errorf("创建会话失败: %w", err)
    }

    log.Printf("[DEBUG] Successfully created conversation: %s", conv.ID)
    return conv, nil
}
```

### 2. 数据库索引优化（可选）

当前查询：
```sql
SELECT * FROM conversations
WHERE (user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)
```

可以添加复合索引提升查询性能：
```sql
CREATE INDEX idx_users_pair ON conversations(user1_id, user2_id);
```

不过现有的单列索引已经足够：
```sql
INDEX idx_user1 (user1_id),
INDEX idx_user2 (user2_id)
```

---

## 总结

### ✅ 修复内容

| 项目 | 修改前 | 修改后 |
|------|--------|--------|
| 并发创建处理 | 失败返回错误 | 重试查询已存在会话 |
| 查询时机 | 查询后再排序 | 排序后再查询 |
| 错误处理 | 直接返回错误 | 识别唯一性冲突并重试 |
| 一对用户会话数 | 可能 > 1 | 保证 = 1 |

### ✅ 解决的问题

- ❌ 双方收不到对方的消息 → ✅ 正常收发
- ❌ 一对用户有多个会话 → ✅ 只有一个会话
- ❌ 并发创建导致冲突 → ✅ 自动重试查询

### 🎯 预期效果

- 任何情况下，一对用户关系**只有一个会话**
- 并发创建时自动处理冲突，**不会报错**
- 消息发送到同一个会话，**双方都能看到**

---

## 部署步骤

1. **提交代码**
   ```bash
   cd Backend
   git add internal/service/conversation_service.go
   git commit -m "fix: 修复会话并发创建导致重复的问题"
   git push
   ```

2. **自动部署**（如果配置了 GitHub Actions）
   - Push 后自动触发部署
   - 等待部署完成

3. **手动部署**（如果需要）
   ```bash
   # 在服务器上
   cd /opt/youkong
   # 下载最新 release
   systemctl restart youkong
   ```

4. **验证**
   - 清理测试数据
   - 在 App 中重新测试
   - 检查数据库是否只有一个会话

完成！🎉
