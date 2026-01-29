#!/bin/bash
# 在线上服务器执行此脚本清理测试数据

echo "=================================="
echo "清理测试账户的会话和消息"
echo "=================================="

mysql -u root youkong << 'EOF'

-- 查看要清理的数据
SELECT '=== 清理前统计 ===' as '';
SELECT COUNT(*) as '测试账户的会话数' FROM conversations
WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

SELECT COUNT(*) as '测试账户的消息数' FROM messages m
WHERE m.conversation_id IN (
    SELECT id FROM conversations
    WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
       OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
);

-- 开始清理
SELECT '=== 开始清理 ===' as '';

-- 删除消息
DELETE m FROM messages m
WHERE m.conversation_id IN (
    SELECT id FROM conversations
    WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
       OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
);

-- 删除会话
DELETE FROM conversations
WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

-- 验证清理结果
SELECT '=== 清理后验证 ===' as '';
SELECT COUNT(*) as '剩余会话数（应该为0）' FROM conversations
WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
   OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%');

SELECT COUNT(*) as '剩余消息数（应该为0）' FROM messages m
WHERE m.conversation_id IN (
    SELECT id FROM conversations
    WHERE user1_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
       OR user2_id IN (SELECT id FROM users WHERE phone LIKE '138000%')
);

SELECT '✅ 清理完成！' as '';

EOF

echo ""
echo "=================================="
echo "✅ 服务器数据已清理"
echo "=================================="
echo ""
echo "接下来在 App 中操作："
echo "1. 打开消息列表，下拉刷新（会话列表会变空）"
echo "2. 重新打开好友列表，点击好友发送消息"
echo "3. 测试双方能否正常收发消息"
echo ""
