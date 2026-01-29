#!/bin/bash
# 检查测试账户的会话状态

echo "=========================================="
echo "检查会话和消息状态"
echo "=========================================="

mysql -u root youkong << 'EOF'

-- 查找测试账户的 user_id
SELECT '=== 测试账户 ID ===' as '';
SELECT id, phone, nickname FROM users WHERE phone IN ('13800138000', '13800000001');

-- 查找这两个用户之间的所有会话
SELECT '=== 会话列表 ===' as '';
SELECT
    c.id as conversation_id,
    c.user1_id,
    c.user2_id,
    c.created_at,
    (SELECT COUNT(*) FROM messages WHERE conversation_id = c.id) as message_count
FROM conversations c
WHERE (c.user1_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001'))
   AND c.user2_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001')))
ORDER BY c.created_at DESC;

-- 检查特定会话的消息
SELECT '=== 会话 5fe4730d 的消息 ===' as '';
SELECT
    m.id,
    m.sender_id,
    u.phone as sender_phone,
    m.type,
    LEFT(m.content, 50) as content_preview,
    m.created_at
FROM messages m
LEFT JOIN users u ON m.sender_id = u.id
WHERE m.conversation_id LIKE '5fe4730d%'
ORDER BY m.created_at DESC
LIMIT 20;

-- 检查是否有孤立的消息（conversation_id 不存在）
SELECT '=== 孤立消息检查 ===' as '';
SELECT
    m.id,
    m.conversation_id,
    m.sender_id,
    u.phone as sender_phone,
    m.created_at
FROM messages m
LEFT JOIN users u ON m.sender_id = u.id
LEFT JOIN conversations c ON m.conversation_id = c.id
WHERE c.id IS NULL
  AND m.sender_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001'))
ORDER BY m.created_at DESC
LIMIT 10;

EOF

echo ""
echo "=========================================="
echo "✅ 检查完成"
echo "=========================================="
