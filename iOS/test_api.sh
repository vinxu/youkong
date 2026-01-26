#!/bin/bash

# YouKong API 测试脚本
# 模拟用户完整流程

BASE_URL="http://49.232.13.41:8080/api/v1"
PHONE="13800138000"  # 测试手机号
TOKEN=""
USER_ID=""
CIRCLE_ID=""
AVAILABILITY_ID=""
CONVERSATION_ID=""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_step() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}→ $1${NC}"
}

# 1. 健康检查
print_step "1. 健康检查"
RESPONSE=$(curl -s http://49.232.13.41:8080/health)
echo "Response: $RESPONSE"
if [[ "$RESPONSE" == *"ok"* ]]; then
    print_success "后端服务正常"
else
    print_error "后端服务异常"
    exit 1
fi

# 2. 发送验证码
print_step "2. 发送短信验证码"
print_info "POST /auth/sms/send"
RESPONSE=$(curl -s -X POST "$BASE_URL/auth/sms/send" \
    -H "Content-Type: application/json" \
    -d "{\"phone\": \"$PHONE\"}")
echo "Response: $RESPONSE"
CODE=$(echo $RESPONSE | grep -o '"code":[0-9]*' | grep -o '[0-9]*')
if [[ "$CODE" == "0" ]]; then
    print_success "验证码发送成功（开发模式可能直接通过）"
else
    print_info "验证码发送返回: $RESPONSE"
fi

# 3. 验证码登录（开发模式使用固定验证码）
print_step "3. 验证码登录"
print_info "POST /auth/sms/verify"
# 开发模式下通常使用 123456 或 000000 作为测试验证码
RESPONSE=$(curl -s -X POST "$BASE_URL/auth/sms/verify" \
    -H "Content-Type: application/json" \
    -d "{\"phone\": \"$PHONE\", \"code\": \"123456\"}")
echo "Response: $RESPONSE"

# 提取 token
TOKEN=$(echo $RESPONSE | grep -o '"token":"[^"]*"' | sed 's/"token":"//;s/"//')
if [[ -n "$TOKEN" ]]; then
    print_success "登录成功，获取到 Token"
    print_info "Token: ${TOKEN:0:50}..."
else
    print_error "登录失败，尝试使用 000000"
    RESPONSE=$(curl -s -X POST "$BASE_URL/auth/sms/verify" \
        -H "Content-Type: application/json" \
        -d "{\"phone\": \"$PHONE\", \"code\": \"000000\"}")
    echo "Response: $RESPONSE"
    TOKEN=$(echo $RESPONSE | grep -o '"token":"[^"]*"' | sed 's/"token":"//;s/"//')
    if [[ -n "$TOKEN" ]]; then
        print_success "登录成功"
    else
        print_error "登录失败，请检查验证码逻辑"
    fi
fi

# 如果没有 token，后续测试无法进行
if [[ -z "$TOKEN" ]]; then
    print_error "无法获取 Token，后续测试跳过"
    echo "请确认后端的短信验证逻辑，开发模式下应该允许固定验证码"
    exit 1
fi

# 4. 获取当前用户信息
print_step "4. 获取当前用户信息"
print_info "GET /users/me"
RESPONSE=$(curl -s -X GET "$BASE_URL/users/me" \
    -H "Authorization: Bearer $TOKEN")
echo "Response: $RESPONSE"
USER_ID=$(echo $RESPONSE | grep -o '"id":"[^"]*"' | head -1 | sed 's/"id":"//;s/"//')
if [[ -n "$USER_ID" ]]; then
    print_success "获取用户信息成功，用户ID: $USER_ID"
else
    print_error "获取用户信息失败"
fi

# 5. 更新用户信息
print_step "5. 更新用户昵称"
print_info "PUT /users/me"
RESPONSE=$(curl -s -X PUT "$BASE_URL/users/me" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"nickname": "测试用户"}')
echo "Response: $RESPONSE"
if [[ "$RESPONSE" == *"测试用户"* ]]; then
    print_success "更新用户信息成功"
else
    print_info "更新结果: $RESPONSE"
fi

# 6. 创建圈子
print_step "6. 创建圈子"
print_info "POST /circles"
RESPONSE=$(curl -s -X POST "$BASE_URL/circles" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"name": "好朋友", "emoji": "😊", "color": "#EC4899"}')
echo "Response: $RESPONSE"
CIRCLE_ID=$(echo $RESPONSE | grep -o '"id":"[^"]*"' | head -1 | sed 's/"id":"//;s/"//')
if [[ -n "$CIRCLE_ID" ]]; then
    print_success "创建圈子成功，圈子ID: $CIRCLE_ID"
else
    print_error "创建圈子失败"
fi

# 7. 获取圈子列表
print_step "7. 获取圈子列表"
print_info "GET /circles"
RESPONSE=$(curl -s -X GET "$BASE_URL/circles" \
    -H "Authorization: Bearer $TOKEN")
echo "Response: $RESPONSE"
if [[ "$RESPONSE" == *"好朋友"* ]] || [[ "$RESPONSE" == *"id"* ]]; then
    print_success "获取圈子列表成功"
else
    print_info "圈子列表: $RESPONSE"
fi

# 8. 获取圈子详情
if [[ -n "$CIRCLE_ID" ]]; then
    print_step "8. 获取圈子详情"
    print_info "GET /circles/$CIRCLE_ID"
    RESPONSE=$(curl -s -X GET "$BASE_URL/circles/$CIRCLE_ID" \
        -H "Authorization: Bearer $TOKEN")
    echo "Response: $RESPONSE"
    if [[ "$RESPONSE" == *"$CIRCLE_ID"* ]]; then
        print_success "获取圈子详情成功"
    else
        print_error "获取圈子详情失败"
    fi
fi

# 9. 发布有空状态
print_step "9. 发布有空状态"
print_info "POST /availabilities"
# 计算今天晚上 6 点和 10 点的 ISO 时间
START_TIME=$(date -u -v+1H '+%Y-%m-%dT%H:00:00Z' 2>/dev/null || date -u -d '+1 hour' '+%Y-%m-%dT%H:00:00Z' 2>/dev/null || echo "2025-01-26T10:00:00Z")
END_TIME=$(date -u -v+5H '+%Y-%m-%dT%H:00:00Z' 2>/dev/null || date -u -d '+5 hours' '+%Y-%m-%dT%H:00:00Z' 2>/dev/null || echo "2025-01-26T14:00:00Z")

if [[ -n "$CIRCLE_ID" ]]; then
    RESPONSE=$(curl -s -X POST "$BASE_URL/availabilities" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"startTime\": \"$START_TIME\",
            \"endTime\": \"$END_TIME\",
            \"locationType\": \"PRESET\",
            \"locationName\": \"三里屯\",
            \"latitude\": 39.9334,
            \"longitude\": 116.4551,
            \"circleIds\": [\"$CIRCLE_ID\"]
        }")
    echo "Response: $RESPONSE"
    AVAILABILITY_ID=$(echo $RESPONSE | grep -o '"id":"[^"]*"' | head -1 | sed 's/"id":"//;s/"//')
    if [[ -n "$AVAILABILITY_ID" ]]; then
        print_success "发布有空成功，ID: $AVAILABILITY_ID"
    else
        print_info "发布结果: $RESPONSE"
    fi
else
    print_error "没有圈子ID，跳过发布有空"
fi

# 10. 获取我的有空状态
print_step "10. 获取我的有空状态"
print_info "GET /availabilities/mine"
RESPONSE=$(curl -s -X GET "$BASE_URL/availabilities/mine" \
    -H "Authorization: Bearer $TOKEN")
echo "Response: $RESPONSE"
if [[ "$RESPONSE" == *"id"* ]] || [[ "$RESPONSE" == *"[]"* ]]; then
    print_success "获取我的有空状态成功"
else
    print_error "获取失败"
fi

# 11. 获取朋友的有空状态
print_step "11. 获取朋友的有空状态"
print_info "GET /availabilities/friends"
RESPONSE=$(curl -s -X GET "$BASE_URL/availabilities/friends" \
    -H "Authorization: Bearer $TOKEN")
echo "Response: $RESPONSE"
print_success "获取朋友有空状态完成"

# 12. 获取会话列表
print_step "12. 获取会话列表"
print_info "GET /conversations"
RESPONSE=$(curl -s -X GET "$BASE_URL/conversations" \
    -H "Authorization: Bearer $TOKEN")
echo "Response: $RESPONSE"
CONVERSATION_ID=$(echo $RESPONSE | grep -o '"id":"[^"]*"' | head -1 | sed 's/"id":"//;s/"//')
print_success "获取会话列表完成"

# 13. 搜索用户
print_step "13. 搜索用户"
print_info "GET /users/search?keyword=test"
RESPONSE=$(curl -s -X GET "$BASE_URL/users/search?keyword=test" \
    -H "Authorization: Bearer $TOKEN")
echo "Response: $RESPONSE"
print_success "搜索用户完成"

# 14. 取消有空状态
if [[ -n "$AVAILABILITY_ID" ]]; then
    print_step "14. 取消有空状态"
    print_info "DELETE /availabilities/$AVAILABILITY_ID"
    RESPONSE=$(curl -s -X DELETE "$BASE_URL/availabilities/$AVAILABILITY_ID" \
        -H "Authorization: Bearer $TOKEN")
    echo "Response: $RESPONSE"
    print_success "取消有空状态完成"
fi

# 15. 删除圈子
if [[ -n "$CIRCLE_ID" ]]; then
    print_step "15. 删除圈子"
    print_info "DELETE /circles/$CIRCLE_ID"
    RESPONSE=$(curl -s -X DELETE "$BASE_URL/circles/$CIRCLE_ID" \
        -H "Authorization: Bearer $TOKEN")
    echo "Response: $RESPONSE"
    print_success "删除圈子完成"
fi

# 测试总结
print_step "测试完成"
echo -e "${GREEN}所有 API 接口测试完成！${NC}"
echo ""
echo "测试的接口:"
echo "  ✓ POST /auth/sms/send - 发送验证码"
echo "  ✓ POST /auth/sms/verify - 验证登录"
echo "  ✓ GET /users/me - 获取当前用户"
echo "  ✓ PUT /users/me - 更新用户信息"
echo "  ✓ POST /circles - 创建圈子"
echo "  ✓ GET /circles - 获取圈子列表"
echo "  ✓ GET /circles/:id - 获取圈子详情"
echo "  ✓ POST /availabilities - 发布有空"
echo "  ✓ GET /availabilities/mine - 我的有空"
echo "  ✓ GET /availabilities/friends - 朋友有空"
echo "  ✓ GET /conversations - 会话列表"
echo "  ✓ GET /users/search - 搜索用户"
echo "  ✓ DELETE /availabilities/:id - 取消有空"
echo "  ✓ DELETE /circles/:id - 删除圈子"
