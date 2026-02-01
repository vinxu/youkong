#!/bin/bash

# 有空 App - Grid API 测试脚本

BASE_URL="http://localhost:8080/api/v1"
PHONE="13800000001"
CODE="111111"

echo "========================================="
echo "有空 App - Grid API 测试"
echo "========================================="
echo ""

# 1. 登录获取 Token
echo "1. 登录测试账号: $PHONE"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/sms/verify" \
  -H "Content-Type: application/json" \
  -d "{\"phone\":\"$PHONE\",\"code\":\"$CODE\"}")

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.token')

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
  echo "❌ 登录失败"
  echo $LOGIN_RESPONSE | jq .
  exit 1
fi

echo "✅ 登录成功"
echo "Token: ${TOKEN:0:20}..."
echo ""

# 2. 测试宫格接口
echo "2. 测试宫格接口 GET /api/v1/home/grid"
GRID_RESPONSE=$(curl -s -X GET "$BASE_URL/home/grid" \
  -H "Authorization: Bearer $TOKEN")

echo "响应:"
echo $GRID_RESPONSE | jq .

GRID_SIZE=$(echo $GRID_RESPONSE | jq -r '.data.grid_size')
FRIENDS_COUNT=$(echo $GRID_RESPONSE | jq -r '.data.friends | length')

if [ "$GRID_SIZE" != "null" ]; then
  echo ""
  echo "✅ 宫格接口正常"
  echo "   宫格大小: ${GRID_SIZE}x${GRID_SIZE}"
  echo "   好友数量: $FRIENDS_COUNT"
else
  echo "❌ 宫格接口返回异常"
fi
echo ""

# 3. 测试海报接口(预期返回未实现)
echo "3. 测试海报接口 POST /api/v1/home/poster"
POSTER_RESPONSE=$(curl -s -X POST "$BASE_URL/home/poster" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_ids":["test"]}')

echo "响应:"
echo $POSTER_RESPONSE | jq .
echo ""

# 4. 验证旧的 availability 接口已删除
echo "4. 验证旧接口已删除 GET /api/v1/availabilities/mine"
OLD_RESPONSE=$(curl -s -X GET "$BASE_URL/availabilities/mine" \
  -H "Authorization: Bearer $TOKEN")

echo "响应:"
echo $OLD_RESPONSE | jq .

if echo $OLD_RESPONSE | grep -q "404"; then
  echo "✅ 旧接口已正确删除"
else
  echo "⚠️  旧接口仍然存在"
fi
echo ""

echo "========================================="
echo "测试完成"
echo "========================================="
