#!/bin/bash

# ============================================================
# 意图分发器测试脚本
# 测试 update_status vs create 的区分能力
# v2 架构改进：Tool Calling + CoT 混合机制
# ============================================================

API_BASE="http://49.232.13.41:8080/api/v1"
TOKEN=""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 获取测试 Token
get_token() {
    echo -e "${BLUE}正在获取测试 Token...${NC}"
    response=$(curl -s -X POST "$API_BASE/auth/sms/verify" \
        -H "Content-Type: application/json" \
        -d '{"phone": "13800000001", "code": "111111"}')

    TOKEN=$(echo "$response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

    if [ -z "$TOKEN" ]; then
        echo -e "${RED}获取 Token 失败！${NC}"
        exit 1
    fi
    echo -e "${GREEN}Token 获取成功${NC}"
}

# 发送消息并获取响应
send_message() {
    local text="$1"

    response=$(curl -s -X POST "$API_BASE/agent/voice-schedule/text" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"text\": \"$text\"}" \
        --max-time 120)

    echo "$response"
}

# 检查响应中的 action 类型
check_action() {
    local response="$1"
    local expected="$2"

    # 提取 event type
    event_type=$(echo "$response" | grep -o '"type":"[^"]*"' | tail -1 | cut -d'"' -f4)

    echo "$event_type"
}

# 测试用例
run_tests() {
    echo ""
    echo "=========================================="
    echo "  意图分发器测试（update_status vs create）"
    echo "=========================================="
    echo ""

    PASS=0
    FAIL=0

    # 测试 update_status 场景
    echo -e "${BLUE}1. 测试 update_status 场景${NC}"
    echo "-------------------------------------------"

    # 测试用例 1.1
    echo -n "  1.1 \"修改为睡不着\" => "
    resp=$(send_message "修改为睡不着")
    action=$(check_action "$resp" "update_status")
    if [[ "$action" == "current_status" ]]; then
        echo -e "${GREEN}PASS${NC} (返回 current_status)"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} (返回 $action，期望 current_status)"
        FAIL=$((FAIL + 1))
    fi

    # 测试用例 1.2
    echo -n "  1.2 \"我失眠了\" => "
    resp=$(send_message "我失眠了")
    action=$(check_action "$resp" "update_status")
    if [[ "$action" == "current_status" ]]; then
        echo -e "${GREEN}PASS${NC} (返回 current_status)"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} (返回 $action，期望 current_status)"
        FAIL=$((FAIL + 1))
    fi

    # 测试用例 1.3
    echo -n "  1.3 \"在工作\" => "
    resp=$(send_message "在工作")
    action=$(check_action "$resp" "update_status")
    if [[ "$action" == "current_status" ]]; then
        echo -e "${GREEN}PASS${NC} (返回 current_status)"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} (返回 $action，期望 current_status)"
        FAIL=$((FAIL + 1))
    fi

    # 测试用例 1.4
    echo -n "  1.4 \"帮我更新状态，我现在在嗑瓜子\" => "
    resp=$(send_message "帮我更新状态，我现在在嗑瓜子")
    action=$(check_action "$resp" "update_status")
    if [[ "$action" == "current_status" ]]; then
        echo -e "${GREEN}PASS${NC} (返回 current_status)"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} (返回 $action，期望 current_status)"
        FAIL=$((FAIL + 1))
    fi

    echo ""
    echo -e "${BLUE}2. 测试 create 场景${NC}"
    echo "-------------------------------------------"

    # 测试用例 2.1
    echo -n "  2.1 \"下午3点到5点开会\" => "
    resp=$(send_message "下午3点到5点开会")
    action=$(check_action "$resp" "create")
    if [[ "$action" == "schedule" ]]; then
        echo -e "${GREEN}PASS${NC} (返回 schedule)"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} (返回 $action，期望 schedule)"
        FAIL=$((FAIL + 1))
    fi

    # 测试用例 2.2
    echo -n "  2.2 \"明天上午健身\" => "
    resp=$(send_message "明天上午健身")
    action=$(check_action "$resp" "create")
    if [[ "$action" == "schedule" ]]; then
        echo -e "${GREEN}PASS${NC} (返回 schedule)"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} (返回 $action，期望 schedule)"
        FAIL=$((FAIL + 1))
    fi

    # 测试用例 2.3
    echo -n "  2.3 \"今晚8点到10点看电影\" => "
    resp=$(send_message "今晚8点到10点看电影")
    action=$(check_action "$resp" "create")
    if [[ "$action" == "schedule" ]]; then
        echo -e "${GREEN}PASS${NC} (返回 schedule)"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} (返回 $action，期望 schedule)"
        FAIL=$((FAIL + 1))
    fi

    echo ""
    echo -e "${BLUE}3. 测试 confirm/cancel 场景${NC}"
    echo "-------------------------------------------"

    # 测试用例 3.1
    echo -n "  3.1 \"好的，确认\" => "
    resp=$(send_message "好的，确认")
    action=$(check_action "$resp" "confirm")
    if [[ "$action" == "confirmed" ]]; then
        echo -e "${GREEN}PASS${NC} (返回 confirmed)"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} (返回 $action，期望 confirmed)"
        FAIL=$((FAIL + 1))
    fi

    # 测试用例 3.2
    echo -n "  3.2 \"取消\" => "
    resp=$(send_message "取消")
    action=$(check_action "$resp" "cancel")
    if [[ "$action" == "cancelled" ]]; then
        echo -e "${GREEN}PASS${NC} (返回 cancelled)"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} (返回 $action，期望 cancelled)"
        FAIL=$((FAIL + 1))
    fi

    echo ""
    echo "=========================================="
    echo "  测试结果汇总"
    echo "=========================================="
    echo -e "  通过: ${GREEN}$PASS${NC}"
    echo -e "  失败: ${RED}$FAIL${NC}"
    echo -e "  通过率: $(echo "scale=1; $PASS * 100 / ($PASS + $FAIL)" | bc)%"
    echo "=========================================="
}

# 主函数
main() {
    get_token
    run_tests
}

main
