#!/bin/bash

# YouKong API 测试脚本
# 用于测试所有 API 接口的完整用户流程

set -e

# 配置
BASE_URL="http://49.232.13.41:8080/api/v1"
TEST_PHONE="13800138000"
SMS_CODE="123456"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 变量存储
TOKEN=""
USER_ID=""
CIRCLE_ID=""
AVAILABILITY_ID=""

# 打印函数
print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
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

# API 调用函数
call_api() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    local auth="$4"

    local url="${BASE_URL}${endpoint}"
    local headers="-H 'Content-Type: application/json'"

    if [ -n "$auth" ]; then
        headers="$headers -H 'Authorization: Bearer $auth'"
    fi

    print_info "请求: $method $url"
    if [ -n "$data" ]; then
        print_info "数据: $data"
    fi

    local response
    if [ "$method" = "GET" ]; then
        if [ -n "$auth" ]; then
            response=$(curl -s -X GET "$url" \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $auth")
        else
            response=$(curl -s -X GET "$url" \
                -H "Content-Type: application/json")
        fi
    elif [ "$method" = "POST" ]; then
        if [ -n "$auth" ]; then
            response=$(curl -s -X POST "$url" \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $auth" \
                -d "$data")
        else
            response=$(curl -s -X POST "$url" \
                -H "Content-Type: application/json" \
                -d "$data")
        fi
    elif [ "$method" = "PUT" ]; then
        response=$(curl -s -X PUT "$url" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $auth" \
            -d "$data")
    elif [ "$method" = "DELETE" ]; then
        response=$(curl -s -X DELETE "$url" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $auth")
    fi

    echo -e "响应: $response"
    echo ""
    echo "$response"
}

# 1. 健康检查
test_health_check() {
    print_header "1. 健康检查"

    local response=$(curl -s "http://49.232.13.41:8080/health")
    echo "响应: $response"

    if echo "$response" | grep -q "ok\|healthy\|success"; then
        print_success "服务健康检查通过"
    else
        print_error "服务健康检查失败"
    fi
}

# 2. 发送短信验证码
test_send_sms() {
    print_header "2. 发送短信验证码"

    local data="{\"phone\":\"$TEST_PHONE\"}"
    local response=$(call_api "POST" "/auth/sms/send" "$data")

    local code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$code" = "0" ]; then
        print_success "发送验证码成功"
    else
        print_info "返回码: $code (测试环境可忽略，使用预设验证码)"
    fi
}

# 3. 验证登录
test_verify_login() {
    print_header "3. 验证码登录"

    local data="{\"phone\":\"$TEST_PHONE\",\"code\":\"$SMS_CODE\"}"
    local response=$(call_api "POST" "/auth/sms/verify" "$data")

    # 提取 token
    TOKEN=$(echo "$response" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if 'data' in data and data['data']:
        print(data['data'].get('token', ''))
    else:
        print('')
except:
    print('')
" 2>/dev/null || echo "")

    # 提取 user_id
    USER_ID=$(echo "$response" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if 'data' in data and data['data'] and 'user' in data['data']:
        print(data['data']['user'].get('id', ''))
    else:
        print('')
except:
    print('')
" 2>/dev/null || echo "")

    if [ -n "$TOKEN" ]; then
        print_success "登录成功"
        print_info "Token: ${TOKEN:0:50}..."
        print_info "User ID: $USER_ID"
    else
        print_error "登录失败，无法获取 Token"
        exit 1
    fi
}

# 4. 获取当前用户信息
test_get_current_user() {
    print_header "4. 获取当前用户信息"

    local response=$(call_api "GET" "/users/me" "" "$TOKEN")

    local code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$code" = "0" ]; then
        print_success "获取用户信息成功"
    else
        print_error "获取用户信息失败"
    fi
}

# 5. 更新用户信息
test_update_user() {
    print_header "5. 更新用户信息"

    local data="{\"nickname\":\"测试用户$(date +%s)\"}"
    local response=$(call_api "PUT" "/users/me" "$data" "$TOKEN")

    local code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$code" = "0" ]; then
        print_success "更新用户信息成功"
    else
        print_info "更新用户信息返回码: $code"
    fi
}

# 6. 创建圈子
test_create_circle() {
    print_header "6. 创建圈子"

    local data="{\"name\":\"测试圈子$(date +%s)\",\"emoji\":\"🎉\",\"color\":\"#EC4899\"}"
    local response=$(call_api "POST" "/circles" "$data" "$TOKEN")

    # 提取圈子 ID
    CIRCLE_ID=$(echo "$response" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if 'data' in data and data['data']:
        print(data['data'].get('id', ''))
    else:
        print('')
except:
    print('')
" 2>/dev/null || echo "")

    if [ -n "$CIRCLE_ID" ]; then
        print_success "创建圈子成功"
        print_info "Circle ID: $CIRCLE_ID"
    else
        print_error "创建圈子失败"
    fi
}

# 7. 获取圈子列表
test_get_circles() {
    print_header "7. 获取圈子列表"

    local response=$(call_api "GET" "/circles" "" "$TOKEN")

    local code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$code" = "0" ]; then
        print_success "获取圈子列表成功"

        # 如果之前没有创建圈子，尝试从列表获取一个
        if [ -z "$CIRCLE_ID" ]; then
            CIRCLE_ID=$(echo "$response" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if 'data' in data and data['data'] and len(data['data']) > 0:
        print(data['data'][0].get('id', ''))
    else:
        print('')
except:
    print('')
" 2>/dev/null || echo "")
            if [ -n "$CIRCLE_ID" ]; then
                print_info "从列表获取 Circle ID: $CIRCLE_ID"
            fi
        fi
    else
        print_error "获取圈子列表失败"
    fi
}

# 8. 获取圈子详情
test_get_circle_detail() {
    print_header "8. 获取圈子详情"

    if [ -z "$CIRCLE_ID" ]; then
        print_info "跳过：没有可用的圈子 ID"
        return
    fi

    local response=$(call_api "GET" "/circles/$CIRCLE_ID" "" "$TOKEN")

    local code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$code" = "0" ]; then
        print_success "获取圈子详情成功"
    else
        print_info "获取圈子详情返回码: $code"
    fi
}

# 9. 发布有空
test_create_availability() {
    print_header "9. 发布有空"

    if [ -z "$CIRCLE_ID" ]; then
        print_info "跳过：没有可用的圈子 ID"
        return
    fi

    # 计算时间（当前时间 + 1小时 到 + 3小时）
    local start_time=$(date -u -v+1H +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "+1 hour" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "2025-01-26T10:00:00Z")
    local end_time=$(date -u -v+3H +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "+3 hours" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "2025-01-26T12:00:00Z")

    local data="{
        \"startTime\":\"$start_time\",
        \"endTime\":\"$end_time\",
        \"locationType\":\"FLEXIBLE\",
        \"circleIds\":[\"$CIRCLE_ID\"]
    }"

    local response=$(call_api "POST" "/availabilities" "$data" "$TOKEN")

    # 提取有空 ID
    AVAILABILITY_ID=$(echo "$response" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if 'data' in data and data['data']:
        print(data['data'].get('id', ''))
    else:
        print('')
except:
    print('')
" 2>/dev/null || echo "")

    if [ -n "$AVAILABILITY_ID" ]; then
        print_success "发布有空成功"
        print_info "Availability ID: $AVAILABILITY_ID"
    else
        print_info "发布有空返回（可能需要检查接口格式）"
    fi
}

# 10. 获取好友有空列表
test_get_friends_availabilities() {
    print_header "10. 获取好友有空列表"

    local response=$(call_api "GET" "/availabilities/friends" "" "$TOKEN")

    local code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$code" = "0" ]; then
        print_success "获取好友有空列表成功"
    else
        print_info "获取好友有空列表返回码: $code"
    fi
}

# 11. 获取我的有空列表
test_get_my_availabilities() {
    print_header "11. 获取我的有空列表"

    local response=$(call_api "GET" "/availabilities/mine" "" "$TOKEN")

    local code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$code" = "0" ]; then
        print_success "获取我的有空列表成功"

        # 如果之前没有创建有空，尝试从列表获取一个
        if [ -z "$AVAILABILITY_ID" ]; then
            AVAILABILITY_ID=$(echo "$response" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if 'data' in data and data['data'] and len(data['data']) > 0:
        print(data['data'][0].get('id', ''))
    else:
        print('')
except:
    print('')
" 2>/dev/null || echo "")
            if [ -n "$AVAILABILITY_ID" ]; then
                print_info "从列表获取 Availability ID: $AVAILABILITY_ID"
            fi
        fi
    else
        print_info "获取我的有空列表返回码: $code"
    fi
}

# 12. 取消有空
test_cancel_availability() {
    print_header "12. 取消有空"

    if [ -z "$AVAILABILITY_ID" ]; then
        print_info "跳过：没有可用的有空 ID"
        return
    fi

    local response=$(call_api "DELETE" "/availabilities/$AVAILABILITY_ID" "" "$TOKEN")

    local code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$code" = "0" ]; then
        print_success "取消有空成功"
    else
        print_info "取消有空返回码: $code"
    fi
}

# 13. 获取会话列表
test_get_conversations() {
    print_header "13. 获取会话列表"

    local response=$(call_api "GET" "/conversations" "" "$TOKEN")

    local code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$code" = "0" ]; then
        print_success "获取会话列表成功"
    else
        print_info "获取会话列表返回码: $code"
    fi
}

# 14. 搜索用户
test_search_users() {
    print_header "14. 搜索用户"

    local response=$(call_api "GET" "/users/search?keyword=test" "" "$TOKEN")

    local code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$code" = "0" ]; then
        print_success "搜索用户成功"
    else
        print_info "搜索用户返回码: $code"
    fi
}

# 15. 刷新 Token
test_refresh_token() {
    print_header "15. 刷新 Token"

    local data="{\"token\":\"$TOKEN\"}"
    local response=$(call_api "POST" "/auth/refresh" "$data")

    local code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$code" = "0" ]; then
        print_success "刷新 Token 成功"
    else
        print_info "刷新 Token 返回码: $code"
    fi
}

# 主函数
main() {
    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║        YouKong API 测试脚本                            ║${NC}"
    echo -e "${GREEN}║        测试服务器: $BASE_URL          ║${NC}"
    echo -e "${GREEN}║        测试手机号: $TEST_PHONE                       ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════╝${NC}"
    echo ""

    # 执行所有测试
    test_health_check
    test_send_sms
    test_verify_login
    test_get_current_user
    test_update_user
    test_create_circle
    test_get_circles
    test_get_circle_detail
    test_create_availability
    test_get_friends_availabilities
    test_get_my_availabilities
    test_cancel_availability
    test_get_conversations
    test_search_users
    test_refresh_token

    # 测试总结
    print_header "测试完成"
    echo ""
    echo -e "${GREEN}所有 API 接口测试完成！${NC}"
    echo ""
    echo "存储的变量:"
    echo "  - Token: ${TOKEN:0:50}..."
    echo "  - User ID: $USER_ID"
    echo "  - Circle ID: $CIRCLE_ID"
    echo "  - Availability ID: $AVAILABILITY_ID"
    echo ""
}

# 运行主函数
main "$@"
