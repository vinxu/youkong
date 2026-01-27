#!/bin/bash

# Agent 多轮对话测试脚本
# 测试上下文记忆和关系学习功能

set -e

BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api/v1"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo_info() { echo -e "${YELLOW}[INFO]${NC} $1"; }
echo_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
echo_error() { echo -e "${RED}[ERROR]${NC} $1"; }
echo_msg() { echo -e "${CYAN}[消息]${NC} $1"; }

echo "=========================================="
echo "Agent 多轮对话测试"
echo "=========================================="

# 登录两个用户
echo ""
echo_info "登录用户..."
LOGIN1=$(curl -s -X POST "$API_URL/auth/sms/verify" \
    -H "Content-Type: application/json" \
    -d '{"phone": "13800000001", "code": "111111"}')
TOKEN1=$(echo "$LOGIN1" | jq -r '.data.token')
USER1_ID=$(echo "$LOGIN1" | jq -r '.data.user.id')

LOGIN2=$(curl -s -X POST "$API_URL/auth/sms/verify" \
    -H "Content-Type: application/json" \
    -d '{"phone": "13800000002", "code": "222222"}')
TOKEN2=$(echo "$LOGIN2" | jq -r '.data.token')
USER2_ID=$(echo "$LOGIN2" | jq -r '.data.user.id')

echo_success "用户1 ID: $USER1_ID"
echo_success "用户2 ID: $USER2_ID"

# 创建会话
echo ""
echo_info "创建会话..."
CONV=$(curl -s -X POST "$API_URL/conversations" \
    -H "Authorization: Bearer $TOKEN1" \
    -H "Content-Type: application/json" \
    -d "{\"partnerId\": \"$USER2_ID\"}")
CONV_ID=$(echo "$CONV" | jq -r '.data.id')
echo_success "会话ID: $CONV_ID"

# 模拟多轮对话
echo ""
echo "=========================================="
echo "开始模拟多轮对话"
echo "=========================================="

# 第1轮：用户1的Agent说话
echo ""
echo_info "第1轮：用户1的Agent打招呼..."
REPLY1=$(curl -s -X POST "$API_URL/conversations/$CONV_ID/agent-reply" \
    -H "Authorization: Bearer $TOKEN1" --max-time 30)
CONTENT1=$(echo "$REPLY1" | jq -r '.data.content')
echo_msg "用户1: $CONTENT1"

# 模拟用户2回复
echo ""
echo_info "用户2手动回复..."
curl -s -X POST "$API_URL/conversations/$CONV_ID/messages" \
    -H "Authorization: Bearer $TOKEN2" \
    -H "Content-Type: application/json" \
    -d '{"type": "TEXT", "content": "在呢，刚吃完饭，你呢？"}' > /dev/null
echo_msg "用户2: 在呢，刚吃完饭，你呢？"

sleep 1

# 第2轮：用户1的Agent继续
echo ""
echo_info "第2轮：用户1的Agent继续对话..."
REPLY2=$(curl -s -X POST "$API_URL/conversations/$CONV_ID/agent-reply" \
    -H "Authorization: Bearer $TOKEN1" --max-time 30)
CONTENT2=$(echo "$REPLY2" | jq -r '.data.content')
echo_msg "用户1: $CONTENT2"

# 模拟用户2回复
echo ""
echo_info "用户2手动回复..."
curl -s -X POST "$API_URL/conversations/$CONV_ID/messages" \
    -H "Authorization: Bearer $TOKEN2" \
    -H "Content-Type: application/json" \
    -d '{"type": "TEXT", "content": "还行吧，周末有空吗？想约你吃个饭"}' > /dev/null
echo_msg "用户2: 还行吧，周末有空吗？想约你吃个饭"

sleep 1

# 第3轮：用户1的Agent回应约饭
echo ""
echo_info "第3轮：用户1的Agent回应约饭邀请..."
REPLY3=$(curl -s -X POST "$API_URL/conversations/$CONV_ID/agent-reply" \
    -H "Authorization: Bearer $TOKEN1" --max-time 30)
CONTENT3=$(echo "$REPLY3" | jq -r '.data.content')
echo_msg "用户1: $CONTENT3"

# 查看所有消息
echo ""
echo "=========================================="
echo "完整对话记录"
echo "=========================================="
MESSAGES=$(curl -s "$API_URL/conversations/$CONV_ID/messages?limit=20" \
    -H "Authorization: Bearer $TOKEN1")

echo "$MESSAGES" | jq -r '.data | reverse | .[] | "\(.sender.nickname): \(.content)"'

# 检查数据库中的人设和关系
echo ""
echo "=========================================="
echo "检查学习数据"
echo "=========================================="

# 检查人设表
echo ""
echo_info "用户人设数据:"
mysql -u root -pxuxuheng43 youkong -e "SELECT user_id, personality, speaking_style, sample_count FROM user_personas LIMIT 2;" 2>/dev/null || echo "  (无数据或无法访问)"

# 检查关系表
echo ""
echo_info "关系画像数据:"
mysql -u root -pxuxuheng43 youkong -e "SELECT user_id, friend_id, closeness, message_count FROM relationships LIMIT 5;" 2>/dev/null || echo "  (无数据或无法访问)"

# 检查上下文表
echo ""
echo_info "对话上下文数据:"
mysql -u root -pxuxuheng43 youkong -e "SELECT conversation_id, token_count, LENGTH(messages) as msg_len FROM conversation_contexts WHERE conversation_id='$CONV_ID';" 2>/dev/null || echo "  (无数据或无法访问)"

echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
