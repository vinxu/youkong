#!/bin/bash

# YouKong API 简化测试脚本
# 测试前请先在服务器执行: redis-cli SET sms:code:13800138000 123456 EX 300

BASE_URL="http://49.232.13.41:8080/api/v1"
PHONE="13800138000"
CODE="123456"

echo "========================================"
echo "YouKong API 测试"
echo "服务器: $BASE_URL"
echo "========================================"
echo ""

# 1. 健康检查
echo "1. 健康检查"
curl -s "http://49.232.13.41:8080/health"
echo -e "\n"

# 2. 验证码登录
echo "2. 验证码登录 (phone=$PHONE, code=$CODE)"
LOGIN_RESULT=$(curl -s -X POST "$BASE_URL/auth/sms/verify" \
  -H "Content-Type: application/json" \
  -d "{\"phone\":\"$PHONE\",\"code\":\"$CODE\"}")
echo "$LOGIN_RESULT"
echo ""

# 提取 Token
TOKEN=$(echo "$LOGIN_RESULT" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if 'data' in data and data['data']:
        print(data['data'].get('token', ''))
except:
    pass
" 2>/dev/null)

if [ -z "$TOKEN" ]; then
    echo "登录失败，无法获取 Token"
    echo "请确保已在服务器设置验证码: redis-cli SET sms:code:$PHONE $CODE EX 300"
    exit 1
fi

echo "Token 获取成功: ${TOKEN:0:50}..."
echo ""

# 3. 获取当前用户
echo "3. 获取当前用户"
curl -s "$BASE_URL/users/me" \
  -H "Authorization: Bearer $TOKEN"
echo -e "\n"

# 4. 获取圈子列表
echo "4. 获取圈子列表"
curl -s "$BASE_URL/circles" \
  -H "Authorization: Bearer $TOKEN"
echo -e "\n"

# 5. 创建圈子
echo "5. 创建圈子"
CIRCLE_RESULT=$(curl -s -X POST "$BASE_URL/circles" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"测试圈子","emoji":"🎉","color":"#EC4899"}')
echo "$CIRCLE_RESULT"
echo ""

# 提取圈子 ID
CIRCLE_ID=$(echo "$CIRCLE_RESULT" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if 'data' in data and data['data']:
        print(data['data'].get('id', ''))
except:
    pass
" 2>/dev/null)

if [ -n "$CIRCLE_ID" ]; then
    echo "圈子创建成功: $CIRCLE_ID"
fi
echo ""

# 6. 获取好友有空列表
echo "6. 获取好友有空列表"
curl -s "$BASE_URL/availabilities/friends" \
  -H "Authorization: Bearer $TOKEN"
echo -e "\n"

# 7. 获取我的有空列表
echo "7. 获取我的有空列表"
curl -s "$BASE_URL/availabilities/mine" \
  -H "Authorization: Bearer $TOKEN"
echo -e "\n"

# 8. 获取会话列表
echo "8. 获取会话列表"
curl -s "$BASE_URL/conversations" \
  -H "Authorization: Bearer $TOKEN"
echo -e "\n"

echo "========================================"
echo "测试完成"
echo "========================================"
echo ""
echo "保存的变量:"
echo "  TOKEN=$TOKEN"
echo "  CIRCLE_ID=$CIRCLE_ID"
