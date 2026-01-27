#!/bin/bash

# Agent 聊天接口测试脚本
# 使用方法: ./test_agent_chat.sh [local|prod]

set -e

# 配置
ENV=${1:-local}
if [ "$ENV" = "prod" ]; then
    BASE_URL="http://49.232.13.41:8080"
else
    BASE_URL="http://localhost:8080"
fi

API_URL="$BASE_URL/api/v1"

# 测试账号
PHONE1="13800000001"
CODE1="111111"
PHONE2="13800000002"
CODE2="222222"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo_info() { echo -e "${YELLOW}[INFO]${NC} $1"; }
echo_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
echo_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查 jq 是否安装
if ! command -v jq &> /dev/null; then
    echo_error "需要安装 jq: brew install jq"
    exit 1
fi

echo "=========================================="
echo "Agent 聊天接口测试"
echo "环境: $ENV"
echo "API: $API_URL"
echo "=========================================="

# 1. 健康检查
echo ""
echo_info "1. 健康检查..."
HEALTH=$(curl -s "$BASE_URL/health")
if echo "$HEALTH" | jq -e '.status == "ok"' > /dev/null 2>&1; then
    echo_success "服务正常运行"
else
    echo_error "服务不可用: $HEALTH"
    exit 1
fi

# 2. 登录用户1
echo ""
echo_info "2. 登录用户1 ($PHONE1)..."
LOGIN1=$(curl -s -X POST "$API_URL/auth/sms/verify" \
    -H "Content-Type: application/json" \
    -d "{\"phone\": \"$PHONE1\", \"code\": \"$CODE1\"}")

if echo "$LOGIN1" | jq -e '.code == 0' > /dev/null 2>&1; then
    TOKEN1=$(echo "$LOGIN1" | jq -r '.data.token')
    USER1_ID=$(echo "$LOGIN1" | jq -r '.data.user.id')
    USER1_NAME=$(echo "$LOGIN1" | jq -r '.data.user.nickname')
    echo_success "用户1登录成功: $USER1_NAME ($USER1_ID)"
else
    echo_error "用户1登录失败: $(echo "$LOGIN1" | jq -r '.message')"
    exit 1
fi

# 3. 登录用户2
echo ""
echo_info "3. 登录用户2 ($PHONE2)..."
LOGIN2=$(curl -s -X POST "$API_URL/auth/sms/verify" \
    -H "Content-Type: application/json" \
    -d "{\"phone\": \"$PHONE2\", \"code\": \"$CODE2\"}")

if echo "$LOGIN2" | jq -e '.code == 0' > /dev/null 2>&1; then
    TOKEN2=$(echo "$LOGIN2" | jq -r '.data.token')
    USER2_ID=$(echo "$LOGIN2" | jq -r '.data.user.id')
    USER2_NAME=$(echo "$LOGIN2" | jq -r '.data.user.nickname')
    echo_success "用户2登录成功: $USER2_NAME ($USER2_ID)"
else
    echo_error "用户2登录失败: $(echo "$LOGIN2" | jq -r '.message')"
    exit 1
fi

# 4. 用户1获取会话列表
echo ""
echo_info "4. 用户1获取会话列表..."
CONVS=$(curl -s "$API_URL/conversations" \
    -H "Authorization: Bearer $TOKEN1")

if echo "$CONVS" | jq -e '.code == 0' > /dev/null 2>&1; then
    CONV_COUNT=$(echo "$CONVS" | jq '.data | length')
    echo_success "获取会话列表成功，共 $CONV_COUNT 个会话"
    echo "$CONVS" | jq '.data[:3]' 2>/dev/null || echo "$CONVS" | jq '.data'
else
    echo_error "获取会话列表失败: $(echo "$CONVS" | jq -r '.message')"
fi

# 5. 用户1创建与用户2的会话
echo ""
echo_info "5. 用户1创建与用户2的会话..."
CREATE_CONV=$(curl -s -X POST "$API_URL/conversations" \
    -H "Authorization: Bearer $TOKEN1" \
    -H "Content-Type: application/json" \
    -d "{\"partnerId\": \"$USER2_ID\"}")

if echo "$CREATE_CONV" | jq -e '.code == 0' > /dev/null 2>&1; then
    CONV_ID=$(echo "$CREATE_CONV" | jq -r '.data.id')
    PARTNER_NAME=$(echo "$CREATE_CONV" | jq -r '.data.partner.nickname')
    echo_success "创建/获取会话成功"
    echo "  会话ID: $CONV_ID"
    echo "  对方: $PARTNER_NAME"
else
    echo_error "创建会话失败: $(echo "$CREATE_CONV" | jq -r '.message')"
    exit 1
fi

# 6. 获取会话消息
echo ""
echo_info "6. 获取会话消息..."
MESSAGES=$(curl -s "$API_URL/conversations/$CONV_ID/messages?limit=10" \
    -H "Authorization: Bearer $TOKEN1")

if echo "$MESSAGES" | jq -e '.code == 0' > /dev/null 2>&1; then
    MSG_COUNT=$(echo "$MESSAGES" | jq '.data | length')
    echo_success "获取消息成功，共 $MSG_COUNT 条消息"
    if [ "$MSG_COUNT" -gt 0 ]; then
        echo "最近消息:"
        echo "$MESSAGES" | jq '.data[:3] | .[] | {sender: .sender.nickname, content: .content, time: .createdAt}'
    fi
else
    echo_error "获取消息失败: $(echo "$MESSAGES" | jq -r '.message')"
fi

# 7. 用户1发送普通消息
echo ""
echo_info "7. 用户1发送普通消息..."
SEND_MSG=$(curl -s -X POST "$API_URL/conversations/$CONV_ID/messages" \
    -H "Authorization: Bearer $TOKEN1" \
    -H "Content-Type: application/json" \
    -d '{"type": "TEXT", "content": "测试消息 - '"$(date +%H:%M:%S)"'"}')

if echo "$SEND_MSG" | jq -e '.code == 0' > /dev/null 2>&1; then
    MSG_ID=$(echo "$SEND_MSG" | jq -r '.data.id')
    MSG_CONTENT=$(echo "$SEND_MSG" | jq -r '.data.content')
    echo_success "发送消息成功"
    echo "  消息ID: $MSG_ID"
    echo "  内容: $MSG_CONTENT"
else
    echo_error "发送消息失败: $(echo "$SEND_MSG" | jq -r '.message')"
fi

# 8. 用户1让 Agent 回复 (核心测试)
echo ""
echo_info "8. 用户1让 Agent 回复 (核心测试)..."
echo_info "   这可能需要几秒钟..."

AGENT_REPLY=$(curl -s -X POST "$API_URL/conversations/$CONV_ID/agent-reply" \
    -H "Authorization: Bearer $TOKEN1" \
    --max-time 30)

if echo "$AGENT_REPLY" | jq -e '.code == 0' > /dev/null 2>&1; then
    REPLY_ID=$(echo "$AGENT_REPLY" | jq -r '.data.id')
    REPLY_CONTENT=$(echo "$AGENT_REPLY" | jq -r '.data.content')
    REPLY_SENDER=$(echo "$AGENT_REPLY" | jq -r '.data.sender.nickname')
    echo_success "Agent 回复成功!"
    echo "  消息ID: $REPLY_ID"
    echo "  发送者: $REPLY_SENDER"
    echo "  内容: $REPLY_CONTENT"

    # 验证响应字段完整性
    echo ""
    echo_info "验证响应字段..."
    FIELDS_OK=true

    if [ "$(echo "$AGENT_REPLY" | jq -r '.data.id')" = "null" ]; then
        echo_error "缺少字段: id"
        FIELDS_OK=false
    fi
    if [ "$(echo "$AGENT_REPLY" | jq -r '.data.sender')" = "null" ]; then
        echo_error "缺少字段: sender"
        FIELDS_OK=false
    fi
    if [ "$(echo "$AGENT_REPLY" | jq -r '.data.sender.id')" = "null" ]; then
        echo_error "缺少字段: sender.id"
        FIELDS_OK=false
    fi
    if [ "$(echo "$AGENT_REPLY" | jq -r '.data.sender.nickname')" = "null" ]; then
        echo_error "缺少字段: sender.nickname"
        FIELDS_OK=false
    fi
    if [ "$(echo "$AGENT_REPLY" | jq -r '.data.type')" = "null" ]; then
        echo_error "缺少字段: type"
        FIELDS_OK=false
    fi
    if [ "$(echo "$AGENT_REPLY" | jq -r '.data.content')" = "null" ]; then
        echo_error "缺少字段: content"
        FIELDS_OK=false
    fi
    if [ "$(echo "$AGENT_REPLY" | jq -r '.data.createdAt')" = "null" ]; then
        echo_error "缺少字段: createdAt"
        FIELDS_OK=false
    fi

    if [ "$FIELDS_OK" = true ]; then
        echo_success "所有字段完整"
    fi
else
    echo_error "Agent 回复失败: $(echo "$AGENT_REPLY" | jq -r '.message')"
    echo "完整响应:"
    echo "$AGENT_REPLY" | jq .
fi

# 9. 再次获取消息验证
echo ""
echo_info "9. 再次获取消息验证..."
MESSAGES2=$(curl -s "$API_URL/conversations/$CONV_ID/messages?limit=10" \
    -H "Authorization: Bearer $TOKEN1")

if echo "$MESSAGES2" | jq -e '.code == 0' > /dev/null 2>&1; then
    MSG_COUNT2=$(echo "$MESSAGES2" | jq '.data | length')
    echo_success "获取消息成功，现在共 $MSG_COUNT2 条消息"
    echo "最新消息:"
    echo "$MESSAGES2" | jq '.data[0] | {sender: .sender.nickname, content: .content, type: .type}'
else
    echo_error "获取消息失败"
fi

# 10. 用户2视角验证
echo ""
echo_info "10. 用户2视角获取消息..."
MESSAGES_U2=$(curl -s "$API_URL/conversations/$CONV_ID/messages?limit=5" \
    -H "Authorization: Bearer $TOKEN2")

if echo "$MESSAGES_U2" | jq -e '.code == 0' > /dev/null 2>&1; then
    echo_success "用户2可以看到消息"
    echo "$MESSAGES_U2" | jq '.data[:2] | .[] | {sender: .sender.nickname, content: .content}'
else
    echo_error "用户2获取消息失败: $(echo "$MESSAGES_U2" | jq -r '.message')"
fi

# 总结
echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
echo ""
echo "会话ID: $CONV_ID"
echo "可以在浏览器访问: /chat/$CONV_ID"
echo ""
echo "WebSocket 测试命令:"
echo "  wscat -c \"ws://${BASE_URL#http://}/ws?token=\$TOKEN1\""
