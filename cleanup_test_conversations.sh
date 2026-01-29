#!/bin/bash
# 清理线上测试账户的会话和消息
# 在服务器上执行: bash cleanup_test_conversations.sh

echo "==================================="
echo "清理测试账户的会话和消息"
echo "==================================="
echo ""

# 执行 SQL 清理
mysql -u root youkong << 'EOF'

-- 查看要清理的数据
SELECT '清理前统计:' as '';

SELECT COUNT(*) as test_conversations FROM conversations
WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

SELECT COUNT(*) as test_messages FROM messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR c.user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

-- 开始清理
SELECT '开始清理...' as '';

-- 删除消息
DELETE m FROM messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR c.user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

SELECT ROW_COUNT() as deleted_messages;

-- 删除会话
DELETE FROM conversations
WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

SELECT ROW_COUNT() as deleted_conversations;

-- 验证清理结果
SELECT '清理后验证:' as '';

SELECT COUNT(*) as remaining_conversations FROM conversations
WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

SELECT COUNT(*) as remaining_messages FROM messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR c.user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

SELECT '清理完成！' as '';

EOF

echo ""
echo "==================================="
echo "清理完成！现在可以在 App 中重新测试"
echo "==================================="
