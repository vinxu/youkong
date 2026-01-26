#!/bin/bash

# YouKong API 接口测试脚本
# 用法: ./test_api.sh <BASE_URL>
# 示例: ./test_api.sh http://localhost:8080

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 基础 URL
BASE_URL="${1:-http://localhost:8080}"
API_URL="$BASE_URL/api/v1"

# 测试用手机号
TEST_PHONE="13800138000"
TEST_CODE="123456"

# 存储变量
TOKEN=""
USER_ID=""
CIRCLE_ID=""
AVAILABILITY_ID=""
CONVERSATION_ID=""

echo -e "${BLUE}=============================="
echo "YouKong API 接口测试"
echo "Base URL: $BASE_URL"
echo -e "==============================${NC}"
echo ""

# 辅助函数
test_api() {
  local method=$1
  local endpoint=$2
  local data=$3
  local desc=$4
  local auth=$5

  echo -e "${YELLOW}测试: $desc${NC}"
  echo "  $method $endpoint"

  local headers="-H 'Content-Type: application/json'"
  if [ "$auth" = "yes" ] && [ -n "$TOKEN" ]; then
    headers="$headers -H 'Authorization: Bearer $TOKEN'"
  fi

  local cmd="curl -s -X $method"
  if [ "$auth" = "yes" ] && [ -n "$TOKEN" ]; then
    cmd="$cmd -H 'Authorization: Bearer $TOKEN'"
  fi
  cmd="$cmd -H 'Content-Type: application/json'"

  if [ -n "$data" ]; then
    cmd="$cmd -d '$data'"
  fi
  cmd="$cmd '$API_URL$endpoint'"

  local response=$(eval $cmd)
  local code=$(echo $response | jq -r '.code // "error"')

  if [ "$code" = "0" ]; then
    echo -e "  ${GREEN}✓ 成功${NC}"
    echo "  响应: $(echo $response | jq -c '.data // .')"
  else
    echo -e "  ${RED}✗ 失败${NC}"
    echo "  响应: $response"
  fi
  echo ""

  echo "$response"
}

# ==================== 健康检查 ====================
echo -e "${BLUE}[健康检查]${NC}"
echo -e "${YELLOW}测试: 健康检查${NC}"
echo "  GET /health"
HEALTH=$(curl -s "$BASE_URL/health")
echo "  响应: $HEALTH"
if echo "$HEALTH" | grep -q "ok"; then
  echo -e "  ${GREEN}✓ 服务正常${NC}"
else
  echo -e "  ${RED}✗ 服务异常${NC}"
  exit 1
fi
echo ""

# ==================== 认证模块 ====================
echo -e "${BLUE}[认证模块]${NC}"

# 1. 发送验证码
echo -e "${YELLOW}测试: 发送验证码${NC}"
echo "  POST /auth/sms/send"
RESP=$(curl -s -X POST "$API_URL/auth/sms/send" \
  -H 'Content-Type: application/json' \
  -d "{\"phone\":\"$TEST_PHONE\"}")
echo "  响应: $RESP"
CODE=$(echo $RESP | jq -r '.code')
if [ "$CODE" = "0" ]; then
  echo -e "  ${GREEN}✓ 成功${NC}"
else
  echo -e "  ${YELLOW}! 短信服务可能未配置，继续测试...${NC}"
fi
echo ""

# 2. 验证码登录 (开发模式下可能需要特殊处理)
echo -e "${YELLOW}测试: 验证码登录${NC}"
echo "  POST /auth/sms/verify"
echo -e "  ${YELLOW}注: 开发环境需要 Redis 中有验证码，跳过此测试...${NC}"
echo ""

# 模拟登录成功，设置测试 Token (实际部署时需要真实登录)
# TOKEN="your_test_token"
# USER_ID="test_user_id"

# ==================== 需要认证的接口测试 ====================
# 以下测试需要有效的 Token

if [ -z "$TOKEN" ]; then
  echo -e "${YELLOW}==============================${NC}"
  echo -e "${YELLOW}提示: 未获取到 Token，以下测试将跳过${NC}"
  echo -e "${YELLOW}请手动测试登录接口获取 Token 后重新运行${NC}"
  echo -e "${YELLOW}==============================${NC}"
  echo ""

  # 输出测试命令模板
  echo -e "${BLUE}[手动测试命令模板]${NC}"
  echo ""

  echo "# 1. 获取当前用户"
  echo "curl -X GET '$API_URL/users/me' \\"
  echo "  -H 'Authorization: Bearer <TOKEN>'"
  echo ""

  echo "# 2. 更新用户信息"
  echo "curl -X PUT '$API_URL/users/me' \\"
  echo "  -H 'Authorization: Bearer <TOKEN>' \\"
  echo "  -H 'Content-Type: application/json' \\"
  echo "  -d '{\"nickname\":\"测试用户\",\"avatar\":\"https://example.com/avatar.jpg\"}'"
  echo ""

  echo "# 3. 搜索用户"
  echo "curl -X GET '$API_URL/users/search?keyword=测试' \\"
  echo "  -H 'Authorization: Bearer <TOKEN>'"
  echo ""

  echo "# 4. 创建圈子"
  echo "curl -X POST '$API_URL/circles' \\"
  echo "  -H 'Authorization: Bearer <TOKEN>' \\"
  echo "  -H 'Content-Type: application/json' \\"
  echo "  -d '{\"name\":\"闺蜜\",\"emoji\":\"💕\",\"color\":\"#EC4899\"}'"
  echo ""

  echo "# 5. 获取我的圈子"
  echo "curl -X GET '$API_URL/circles' \\"
  echo "  -H 'Authorization: Bearer <TOKEN>'"
  echo ""

  echo "# 6. 发布有空"
  echo "curl -X POST '$API_URL/availabilities' \\"
  echo "  -H 'Authorization: Bearer <TOKEN>' \\"
  echo "  -H 'Content-Type: application/json' \\"
  echo "  -d '{\"startTime\":\"2024-01-20T18:00:00Z\",\"endTime\":\"2024-01-20T22:00:00Z\",\"locationType\":\"PRESET\",\"locationName\":\"三里屯\",\"latitude\":39.9334,\"longitude\":116.4551,\"circleIds\":[\"<CIRCLE_ID>\"]}'"
  echo ""

  echo "# 7. 获取朋友的有空状态"
  echo "curl -X GET '$API_URL/availabilities/friends' \\"
  echo "  -H 'Authorization: Bearer <TOKEN>'"
  echo ""

  echo "# 8. 获取我的有空状态"
  echo "curl -X GET '$API_URL/availabilities/mine' \\"
  echo "  -H 'Authorization: Bearer <TOKEN>'"
  echo ""

  echo "# 9. 获取会话列表"
  echo "curl -X GET '$API_URL/conversations' \\"
  echo "  -H 'Authorization: Bearer <TOKEN>'"
  echo ""

  exit 0
fi

# ==================== 用户模块 ====================
echo -e "${BLUE}[用户模块]${NC}"

# 获取当前用户
RESP=$(curl -s -X GET "$API_URL/users/me" \
  -H "Authorization: Bearer $TOKEN")
echo -e "${YELLOW}测试: 获取当前用户${NC}"
echo "  GET /users/me"
echo "  响应: $RESP"
USER_ID=$(echo $RESP | jq -r '.data.id')
echo ""

# 更新用户信息
RESP=$(curl -s -X PUT "$API_URL/users/me" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"nickname":"测试用户"}')
echo -e "${YELLOW}测试: 更新用户信息${NC}"
echo "  PUT /users/me"
echo "  响应: $RESP"
echo ""

# 搜索用户
RESP=$(curl -s -X GET "$API_URL/users/search?keyword=测试" \
  -H "Authorization: Bearer $TOKEN")
echo -e "${YELLOW}测试: 搜索用户${NC}"
echo "  GET /users/search?keyword=测试"
echo "  响应: $RESP"
echo ""

# ==================== 圈子模块 ====================
echo -e "${BLUE}[圈子模块]${NC}"

# 创建圈子
RESP=$(curl -s -X POST "$API_URL/circles" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"闺蜜","emoji":"💕","color":"#EC4899"}')
echo -e "${YELLOW}测试: 创建圈子${NC}"
echo "  POST /circles"
echo "  响应: $RESP"
CIRCLE_ID=$(echo $RESP | jq -r '.data.id')
echo ""

# 获取我的圈子
RESP=$(curl -s -X GET "$API_URL/circles" \
  -H "Authorization: Bearer $TOKEN")
echo -e "${YELLOW}测试: 获取我的圈子${NC}"
echo "  GET /circles"
echo "  响应: $RESP"
echo ""

# 获取圈子详情
if [ -n "$CIRCLE_ID" ] && [ "$CIRCLE_ID" != "null" ]; then
  RESP=$(curl -s -X GET "$API_URL/circles/$CIRCLE_ID" \
    -H "Authorization: Bearer $TOKEN")
  echo -e "${YELLOW}测试: 获取圈子详情${NC}"
  echo "  GET /circles/$CIRCLE_ID"
  echo "  响应: $RESP"
  echo ""
fi

# ==================== 有空状态模块 ====================
echo -e "${BLUE}[有空状态模块]${NC}"

# 发布有空
if [ -n "$CIRCLE_ID" ] && [ "$CIRCLE_ID" != "null" ]; then
  START_TIME=$(date -u -v+1d +"%Y-%m-%dT18:00:00Z" 2>/dev/null || date -u -d "+1 day" +"%Y-%m-%dT18:00:00Z")
  END_TIME=$(date -u -v+1d +"%Y-%m-%dT22:00:00Z" 2>/dev/null || date -u -d "+1 day" +"%Y-%m-%dT22:00:00Z")

  RESP=$(curl -s -X POST "$API_URL/availabilities" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"startTime\":\"$START_TIME\",\"endTime\":\"$END_TIME\",\"locationType\":\"PRESET\",\"locationName\":\"三里屯\",\"latitude\":39.9334,\"longitude\":116.4551,\"circleIds\":[\"$CIRCLE_ID\"]}")
  echo -e "${YELLOW}测试: 发布有空${NC}"
  echo "  POST /availabilities"
  echo "  响应: $RESP"
  AVAILABILITY_ID=$(echo $RESP | jq -r '.data.id')
  echo ""
fi

# 获取我的有空状态
RESP=$(curl -s -X GET "$API_URL/availabilities/mine" \
  -H "Authorization: Bearer $TOKEN")
echo -e "${YELLOW}测试: 获取我的有空状态${NC}"
echo "  GET /availabilities/mine"
echo "  响应: $RESP"
echo ""

# 获取朋友的有空状态
RESP=$(curl -s -X GET "$API_URL/availabilities/friends" \
  -H "Authorization: Bearer $TOKEN")
echo -e "${YELLOW}测试: 获取朋友的有空状态${NC}"
echo "  GET /availabilities/friends"
echo "  响应: $RESP"
echo ""

# 取消有空
if [ -n "$AVAILABILITY_ID" ] && [ "$AVAILABILITY_ID" != "null" ]; then
  RESP=$(curl -s -X DELETE "$API_URL/availabilities/$AVAILABILITY_ID" \
    -H "Authorization: Bearer $TOKEN")
  echo -e "${YELLOW}测试: 取消有空${NC}"
  echo "  DELETE /availabilities/$AVAILABILITY_ID"
  echo "  响应: $RESP"
  echo ""
fi

# ==================== 会话消息模块 ====================
echo -e "${BLUE}[会话消息模块]${NC}"

# 获取会话列表
RESP=$(curl -s -X GET "$API_URL/conversations" \
  -H "Authorization: Bearer $TOKEN")
echo -e "${YELLOW}测试: 获取会话列表${NC}"
echo "  GET /conversations"
echo "  响应: $RESP"
echo ""

# ==================== 清理测试数据 ====================
echo -e "${BLUE}[清理测试数据]${NC}"

# 删除测试圈子
if [ -n "$CIRCLE_ID" ] && [ "$CIRCLE_ID" != "null" ]; then
  RESP=$(curl -s -X DELETE "$API_URL/circles/$CIRCLE_ID" \
    -H "Authorization: Bearer $TOKEN")
  echo -e "${YELLOW}清理: 删除测试圈子${NC}"
  echo "  DELETE /circles/$CIRCLE_ID"
  echo "  响应: $RESP"
  echo ""
fi

echo -e "${GREEN}=============================="
echo "测试完成!"
echo -e "==============================${NC}"
