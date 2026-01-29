-- 重置测试数据脚本
-- 清理所有测试账户的数据，避免遗留问题

-- ============================================
-- 第一步：查看要删除的数据（可选，用于确认）
-- ============================================

-- 查看测试账户
SELECT id, phone, nickname FROM users WHERE phone LIKE '138000%';

-- 查看测试账户的会话
SELECT c.*,
       u1.phone as user1_phone,
       u2.phone as user2_phone
FROM conversations c
JOIN users u1 ON c.user1_id = u1.id
JOIN users u2 ON c.user2_id = u2.id
WHERE u1.phone LIKE '138000%' OR u2.phone LIKE '138000%'
ORDER BY c.created_at DESC;

-- 查看测试账户的消息
SELECT m.id,
       u.phone as sender_phone,
       m.content,
       m.created_at
FROM messages m
JOIN conversations c ON m.conversation_id = c.id
JOIN users u ON m.sender_id = u.id
WHERE c.id IN (
    SELECT c2.id FROM conversations c2
    JOIN users u1 ON c2.user1_id = u1.id OR c2.user2_id = u1.id
    WHERE u1.phone LIKE '138000%'
)
ORDER BY m.created_at DESC;

-- ============================================
-- 第二步：删除测试数据
-- ============================================

-- 删除测试账户的所有消息
DELETE m FROM messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR c.user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

-- 删除测试账户的所有会话
DELETE FROM conversations
WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

-- 可选：如果要完全重置用户数据（圈子、好友等）
-- DELETE FROM circle_members WHERE user_id IN (SELECT id FROM users WHERE phone LIKE '138000%');
-- DELETE FROM circles WHERE owner_id IN (SELECT id FROM users WHERE phone LIKE '138000%');
-- DELETE FROM friendships WHERE user_id IN (SELECT id FROM users WHERE phone LIKE '138000%') OR friend_id IN (SELECT id FROM users WHERE phone LIKE '138000%');
-- DELETE FROM friend_requests WHERE from_user_id IN (SELECT id FROM users WHERE phone LIKE '138000%') OR to_user_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

-- 可选：如果要彻底删除测试账户
-- DELETE FROM users WHERE phone LIKE '138000%';

-- ============================================
-- 第三步：验证清理结果
-- ============================================

-- 应该返回 0 行
SELECT COUNT(*) as conversation_count FROM conversations
WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

-- 应该返回 0 行
SELECT COUNT(*) as message_count FROM messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR c.user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

-- 测试账户应该还在（除非你执行了删除用户的可选步骤）
SELECT id, phone, nickname FROM users WHERE phone LIKE '138000%';

-- ============================================
-- 完成！现在可以在 App 中重新测试
-- ============================================
