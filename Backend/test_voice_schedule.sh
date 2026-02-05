#!/bin/bash

# 语音时刻表多轮对话测试脚本
# 用于评估 LLM 输出是否符合人类预期

API_BASE="http://49.232.13.41:8080/api/v1"
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYmFlYzEyNTMtNThhNy00NzE2LThjOGEtNDg3MTBhNjNjNjc0IiwicGhvbmUiOiIxMzgwMDAwMDAwMSIsImlzcyI6InlvdWtvbmciLCJleHAiOjE3NzA4MDA2NTgsIm5iZiI6MTc3MDE5NTg1OCwiaWF0IjoxNzcwMTk1ODU4fQ.re9GqMUfq24Eo92yj3vMBg7QAFR4JWajrZuytxXDYoA"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试函数
test_schedule() {
    local text="$1"
    local session_id="$2"
    local expected="$3"

    echo -e "\n${BLUE}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${YELLOW}输入:${NC} $text"
    if [ -n "$expected" ]; then
        echo -e "${YELLOW}预期:${NC} $expected"
    fi
    echo -e "${BLUE}────────────────────────────────────────────────────────────────${NC}"

    # 构建请求体
    local body
    if [ -n "$session_id" ]; then
        body="{\"text\": \"$text\", \"session_id\": \"$session_id\"}"
    else
        body="{\"text\": \"$text\"}"
    fi

    # 调用 API（使用 timeout 限制等待时间）
    local response
    response=$(timeout 60 curl -s -X POST "$API_BASE/agent/voice-schedule/text" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "$body" 2>&1)

    # 解析响应
    echo -e "${GREEN}响应:${NC}"
    echo "$response" | while read -r line; do
        if [[ "$line" == data:* ]]; then
            local data="${line#data: }"
            if [[ "$data" != "[DONE]" ]]; then
                # 解析 JSON
                local type=$(echo "$data" | jq -r '.type // empty' 2>/dev/null)
                case "$type" in
                    "session_start")
                        local sid=$(echo "$data" | jq -r '.session_id' 2>/dev/null)
                        echo -e "  ${BLUE}[会话开始]${NC} session_id=$sid"
                        # 保存 session_id 到文件供后续使用
                        echo "$sid" > /tmp/voice_schedule_session_id
                        ;;
                    "transcript")
                        local txt=$(echo "$data" | jq -r '.text' 2>/dev/null)
                        echo -e "  ${BLUE}[识别文本]${NC} $txt"
                        ;;
                    "progress")
                        local msg=$(echo "$data" | jq -r '.message // .detail // empty' 2>/dev/null)
                        echo -e "  ${BLUE}[进度]${NC} $msg"
                        ;;
                    "schedule")
                        echo -e "  ${GREEN}[时刻表生成]${NC}"
                        local items=$(echo "$data" | jq -r '.items[]? | "    \(.start_time)-\(.end_time) \(.emoji) \(.status)"' 2>/dev/null)
                        echo "$items"
                        local reasoning=$(echo "$data" | jq -r '.reasoning[]?' 2>/dev/null)
                        if [ -n "$reasoning" ]; then
                            echo -e "  ${YELLOW}[推理]${NC}"
                            echo "$reasoning" | while read -r r; do
                                echo "    - $r"
                            done
                        fi
                        ;;
                    "current_status")
                        local emoji=$(echo "$data" | jq -r '.emoji' 2>/dev/null)
                        local status=$(echo "$data" | jq -r '.status_text' 2>/dev/null)
                        echo -e "  ${GREEN}[状态猜测]${NC} $emoji $status"
                        ;;
                    "clarify")
                        echo -e "  ${YELLOW}[需要澄清]${NC}"
                        echo "$data" | jq -r '.questions[]? | "    Q: \(.question)"' 2>/dev/null
                        ;;
                    "error")
                        local msg=$(echo "$data" | jq -r '.message' 2>/dev/null)
                        echo -e "  ${RED}[错误]${NC} $msg"
                        ;;
                esac
            fi
        fi
    done

    echo ""
}

# 获取上一次的 session_id
get_session_id() {
    if [ -f /tmp/voice_schedule_session_id ]; then
        cat /tmp/voice_schedule_session_id
    fi
}

# 清除 session
clear_session() {
    rm -f /tmp/voice_schedule_session_id
}

echo "═══════════════════════════════════════════════════════════════════════"
echo "              语音时刻表智能测试 - 多轮对话评估"
echo "═══════════════════════════════════════════════════════════════════════"

# ========== 测试 1: 睡眠时间推理 ==========
echo -e "\n${YELLOW}【测试1: 睡眠时间推理】${NC}"
clear_session
test_schedule "1-2点睡觉" "" "应该推理为 02:00-09:00 睡觉（入睡时间→起床时间）"

# ========== 测试 2: 基础时刻表创建 ==========
echo -e "\n${YELLOW}【测试2: 基础时刻表创建】${NC}"
clear_session
test_schedule "上午9点到12点工作，12点吃午饭，下午2点到6点开会" "" "应该创建3个时段，午餐默认1小时"

# ========== 测试 3: 修改保留上下文 ==========
echo -e "\n${YELLOW}【测试3: 修改保留上下文】${NC}"
# 先创建时刻表
clear_session
test_schedule "上午9点到12点工作，12点到1点午餐，下午2点到6点工作" "" "创建初始时刻表"
# 然后修改
SESSION_ID=$(get_session_id)
if [ -n "$SESSION_ID" ]; then
    test_schedule "把下午改成开会" "$SESSION_ID" "应该保留上午工作和午餐，只修改下午为开会"
fi

# ========== 测试 4: 活动时长推理 ==========
echo -e "\n${YELLOW}【测试4: 活动时长推理】${NC}"
clear_session
test_schedule "下午3点开会" "" "应该推理会议时长为2小时，即 15:00-17:00"

# ========== 测试 5: 连续活动衔接 ==========
echo -e "\n${YELLOW}【测试5: 连续活动衔接】${NC}"
clear_session
test_schedule "6点去健身，然后洗澡，然后吃晚饭" "" "应该自动衔接：18:00-19:30健身，19:30-20:00洗澡，20:00-21:00晚餐"

# ========== 测试 6: 多轮修改 ==========
echo -e "\n${YELLOW}【测试6: 多轮修改】${NC}"
clear_session
test_schedule "下午2点到4点开会" "" "创建初始时刻表"
SESSION_ID=$(get_session_id)
if [ -n "$SESSION_ID" ]; then
    test_schedule "不对，是3点开会" "$SESSION_ID" "应该修改为 15:00-17:00 开会"
fi

# ========== 测试 7: 模糊时间 ==========
echo -e "\n${YELLOW}【测试7: 模糊时间】${NC}"
clear_session
test_schedule "待会去吃饭" "" "应该询问具体时间，或根据当前时间推理"

# ========== 测试 8: 当前状态 ==========
echo -e "\n${YELLOW}【测试8: 当前状态猜测】${NC}"
clear_session
test_schedule "我在家休息" "" "应该猜测当前状态，不创建时刻表"

echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}测试完成！请根据上述输出评估 LLM 的智能程度${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════${NC}"
