#!/bin/bash
# 在服务器上执行此脚本诊断消息问题

echo "=========================================="
echo "诊断消息系统问题"
echo "=========================================="
echo ""

sudo mysql youkong << 'EOF'

-- 1. 查找测试账户
SELECT '=== 1. 测试账户信息 ===' as '';
SELECT id, phone, nickname, created_at
FROM users
WHERE phone IN ('13800138000', '13800000001')
ORDER BY phone;

-- 2. 查找这两个用户之间的所有会话
SELECT '' as '';
SELECT '=== 2. 两个账户之间的所有会话 ===' as '';
SELECT
    c.id as conversation_id,
    c.user1_id,
    u1.phone as user1_phone,
    c.user2_id,
    u2.phone as user2_phone,
    c.created_at as conversation_created,
    (SELECT COUNT(*) FROM messages WHERE conversation_id = c.id) as message_count,
    (SELECT MAX(created_at) FROM messages WHERE conversation_id = c.id) as last_message_time
FROM conversations c
LEFT JOIN users u1 ON c.user1_id = u1.id
LEFT JOIN users u2 ON c.user2_id = u2.id
WHERE (c.user1_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001'))
   AND c.user2_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001')))
ORDER BY c.created_at DESC;

-- 3. 检查特定会话的消息
SELECT '' as '';
SELECT '=== 3. 会话 5fe4730d 的详细信息 ===' as '';
SELECT
    c.id as conversation_id,
    c.user1_id,
    u1.phone as user1_phone,
    c.user2_id,
    u2.phone as user2_phone,
    c.created_at
FROM conversations c
LEFT JOIN users u1 ON c.user1_id = u1.id
LEFT JOIN users u2 ON c.user2_id = u2.id
WHERE c.id LIKE '5fe4730d%';

SELECT '' as '';
SELECT '=== 4. 会话 5fe4730d 的所有消息 ===' as '';
SELECT
    m.id as message_id,
    m.sender_id,
    u.phone as sender_phone,
    u.nickname as sender_name,
    m.type,
    m.content,
    m.created_at as message_time
FROM messages m
LEFT JOIN users u ON m.sender_id = u.id
WHERE m.conversation_id LIKE '5fe4730d%'
ORDER BY m.created_at ASC;

-- 5. 检查测试账户的所有消息（任何会话）
SELECT '' as '';
SELECT '=== 5. 测试账户发送的所有消息（最近20条）===' as '';
SELECT
    m.id as message_id,
    m.conversation_id,
    m.sender_id,
    u.phone as sender_phone,
    m.type,
    LEFT(m.content, 50) as content_preview,
    m.created_at as message_time
FROM messages m
LEFT JOIN users u ON m.sender_id = u.id
WHERE m.sender_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001'))
ORDER BY m.created_at DESC
LIMIT 20;

-- 6. 检查是否有孤立的消息（conversation_id 不存在）
SELECT '' as '';
SELECT '=== 6. 孤立消息检查（conversation_id 不存在）===' as '';
SELECT
    m.id as message_id,
    m.conversation_id,
    m.sender_id,
    u.phone as sender_phone,
    LEFT(m.content, 30) as content,
    m.created_at
FROM messages m
LEFT JOIN users u ON m.sender_id = u.id
LEFT JOIN conversations c ON m.conversation_id = c.id
WHERE c.id IS NULL
  AND m.sender_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001'))
ORDER BY m.created_at DESC
LIMIT 10;

-- 7. 统计信息
SELECT '' as '';
SELECT '=== 7. 统计信息 ===' as '';
SELECT
    '测试账户总会话数' as metric,
    COUNT(*) as value
FROM conversations c
WHERE (c.user1_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001'))
   AND c.user2_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001')))
UNION ALL
SELECT
    '测试账户总消息数' as metric,
    COUNT(*) as value
FROM messages m
WHERE m.sender_id IN (SELECT id FROM users WHERE phone IN ('13800138000', '13800000001'));

EOF

echo ""
echo "=========================================="
echo "✅ 诊断完成"
echo "=========================================="
echo ""
echo "请将上面的输出截图发给我，我会帮你分析问题所在。"
echo ""
