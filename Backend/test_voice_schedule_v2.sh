#!/bin/bash

# 语音时刻表 v2 架构改进全面测试
# 测试目标：
# 1. 时间格式正确性（HH:MM 5字符格式）
# 2. 多轮对话上下文保持（时刻表累积）
# 3. 修改操作准确性
# 4. 连续活动衔接（"开完会去健身"）
# 5. 事件类型正确性

API_BASE="http://49.232.13.41:8080/api/v1"
# 测试账号
TEST_PHONE="13800000001"
TEST_CODE="111111"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 统计变量
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 获取 Token
get_token() {
    local response=$(curl -s -X POST "$API_BASE/auth/sms/verify" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$TEST_PHONE\",\"code\":\"$TEST_CODE\"}")
    echo "$response" | jq -r '.data.token // empty'
}

# 发送语音时刻表文本请求（解析 SSE 响应）
send_text() {
    local token="$1"
    local session_id="$2"
    local text="$3"

    local body="{\"text\":\"$text\""
    if [ -n "$session_id" ]; then
        body="$body,\"session_id\":\"$session_id\""
    fi
    body="$body}"

    # 获取 SSE 响应并解析
    local raw_response=$(curl -s -X POST "$API_BASE/agent/voice-schedule/text" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "$body" 2>&1)

    # 从 SSE 响应中提取 JSON 数据
    # 格式: data: {...}
    echo "$raw_response"
}

# 从 SSE 响应中提取 session_id
extract_session_id() {
    local sse_response="$1"
    echo "$sse_response" | grep 'data:.*session_start' | sed 's/data: //' | jq -r '.session_id // empty'
}

# 从 SSE 响应中提取 schedule items
extract_schedule() {
    local sse_response="$1"
    echo "$sse_response" | grep 'data:.*"type":"schedule"' | sed 's/data: //' | jq -r '.items // []'
}

# 从 SSE 响应中提取 event type
extract_event_type() {
    local sse_response="$1"
    # 获取最后一个有效事件的类型（排除 session_start, transcript, progress, DONE）
    echo "$sse_response" | grep 'data:' | grep -v 'session_start' | grep -v 'transcript' | grep -v 'progress' | grep -v '\[DONE\]' | tail -1 | sed 's/data: //' | jq -r '.type // empty'
}

# 从 SSE 响应中提取聊天消息
extract_message() {
    local sse_response="$1"
    echo "$sse_response" | grep 'data:.*"type":"chat"' | sed 's/data: //' | jq -r '.message // empty'
}

# 验证时间格式 HH:MM
validate_time_format() {
    local time="$1"
    if [[ "$time" =~ ^[0-2][0-9]:[0-5][0-9]$ ]]; then
        # 进一步验证小时范围
        local hour="${time:0:2}"
        if [ "$hour" -le 23 ]; then
            return 0
        fi
    fi
    return 1
}

# 验证时刻表中所有时间格式
validate_schedule_times() {
    local schedule="$1"
    local count=$(echo "$schedule" | jq 'length')

    if [ "$count" = "0" ] || [ -z "$count" ]; then
        echo "EMPTY_SCHEDULE"
        return 1
    fi

    for ((i=0; i<count; i++)); do
        local start_time=$(echo "$schedule" | jq -r ".[$i].start_time")
        local end_time=$(echo "$schedule" | jq -r ".[$i].end_time")

        if ! validate_time_format "$start_time"; then
            echo "INVALID_START:$start_time"
            return 1
        fi
        if ! validate_time_format "$end_time"; then
            echo "INVALID_END:$end_time"
            return 1
        fi
    done
    return 0
}

# 检查时刻表是否包含指定活动
check_activity_exists() {
    local schedule="$1"
    local keyword="$2"
    echo "$schedule" | jq -r '.[].status' 2>/dev/null | grep -qi "$keyword"
}

# 获取时刻表中活动的数量
get_schedule_count() {
    local schedule="$1"
    echo "$schedule" | jq 'length' 2>/dev/null || echo "0"
}

# 打印测试结果
print_result() {
    local test_name="$1"
    local passed="$2"
    local details="$3"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if [ "$passed" = "true" ]; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        echo -e "${GREEN}[PASS]${NC} $test_name"
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo -e "${RED}[FAIL]${NC} $test_name"
        if [ -n "$details" ]; then
            echo -e "       ${YELLOW}详情: $details${NC}"
        fi
    fi
}

echo "================================================"
echo "   语音时刻表 v2 架构改进全面测试"
echo "================================================"
echo ""

# 获取 Token
echo -e "${BLUE}[准备] 获取测试 Token...${NC}"
TOKEN=$(get_token)
if [ -z "$TOKEN" ]; then
    echo -e "${RED}[错误] 无法获取 Token，测试终止${NC}"
    exit 1
fi
echo -e "${GREEN}[准备] Token 获取成功${NC}"
echo ""

# ============================================================
# 测试场景 1: 基础时刻表创建（单轮）
# ============================================================
echo "=============================================="
echo "测试场景 1: 基础时刻表创建"
echo "=============================================="

response=$(send_text "$TOKEN" "" "下午3点开会")
session_id=$(extract_session_id "$response")
schedule=$(extract_schedule "$response")
echo -e "${CYAN}响应: session_id=$session_id${NC}"
echo -e "${CYAN}时刻表: $schedule${NC}"

# 检查时间格式
time_result=$(validate_schedule_times "$schedule")
if [ $? -eq 0 ]; then
    print_result "1.1 时间格式正确性" "true"
else
    print_result "1.1 时间格式正确性" "false" "$time_result"
fi

# 检查活动存在
if check_activity_exists "$schedule" "会"; then
    print_result "1.2 活动识别-开会" "true"
else
    print_result "1.2 活动识别-开会" "false" "未识别到开会活动"
fi

echo ""

# ============================================================
# 测试场景 2: 多轮对话累积（核心测试）
# ============================================================
echo "=============================================="
echo "测试场景 2: 多轮对话累积（v2核心改进）"
echo "=============================================="

# 第1轮
response=$(send_text "$TOKEN" "" "今天下午3点开会")
session_id=$(extract_session_id "$response")
schedule=$(extract_schedule "$response")
count1=$(get_schedule_count "$schedule")
echo -e "${BLUE}第1轮: 下午3点开会 -> 时段数: $count1${NC}"

# 第2轮 - 关键测试：新增活动时是否保留旧活动
sleep 2
response=$(send_text "$TOKEN" "$session_id" "晚上7点健身")
schedule=$(extract_schedule "$response")
count2=$(get_schedule_count "$schedule")
echo -e "${BLUE}第2轮: 晚上7点健身 -> 时段数: $count2${NC}"
echo -e "${CYAN}时刻表: $schedule${NC}"

# 验证：第2轮应该有2个时段（开会 + 健身）
if [ "$count2" -ge 2 ]; then
    print_result "2.1 多轮累积-时段数量" "true"
else
    print_result "2.1 多轮累积-时段数量" "false" "期望>=2，实际=$count2"
fi

# 验证：开会活动是否保留
if check_activity_exists "$schedule" "会"; then
    print_result "2.2 多轮累积-保留开会" "true"
else
    print_result "2.2 多轮累积-保留开会" "false" "开会活动丢失"
fi

# 验证：健身活动是否添加
if check_activity_exists "$schedule" "健身"; then
    print_result "2.3 多轮累积-添加健身" "true"
else
    print_result "2.3 多轮累积-添加健身" "false" "健身活动未添加"
fi

# 第3轮 - 继续累积
sleep 2
response=$(send_text "$TOKEN" "$session_id" "晚上9点洗澡")
schedule=$(extract_schedule "$response")
count3=$(get_schedule_count "$schedule")
echo -e "${BLUE}第3轮: 晚上9点洗澡 -> 时段数: $count3${NC}"
echo -e "${CYAN}时刻表: $schedule${NC}"

if [ "$count3" -ge 3 ]; then
    print_result "2.4 多轮累积-三个活动" "true"
else
    print_result "2.4 多轮累积-三个活动" "false" "期望>=3，实际=$count3"
fi

echo ""

# ============================================================
# 测试场景 3: 连续活动衔接（v2核心改进）
# ============================================================
echo "=============================================="
echo "测试场景 3: 连续活动衔接"
echo "=============================================="

# 新会话
response=$(send_text "$TOKEN" "" "下午3点到5点开会")
session_id=$(extract_session_id "$response")
schedule=$(extract_schedule "$response")
meeting_end=$(echo "$schedule" | jq -r '.[0].end_time // empty')
echo -e "${BLUE}第1轮: 开会 -> 结束时间: $meeting_end${NC}"

# 衔接测试
sleep 2
response=$(send_text "$TOKEN" "$session_id" "开完会去健身")
schedule=$(extract_schedule "$response")
gym_start=$(echo "$schedule" | jq -r '.[] | select(.status | test("健身|运动")) | .start_time' 2>/dev/null | head -1)
echo -e "${BLUE}第2轮: 开完会去健身 -> 健身开始: $gym_start${NC}"
echo -e "${CYAN}时刻表: $schedule${NC}"

# 验证：健身开始时间应该等于开会结束时间
if [ "$gym_start" = "$meeting_end" ]; then
    print_result "3.1 衔接时间正确" "true"
else
    print_result "3.1 衔接时间正确" "false" "期望健身从$meeting_end开始，实际=$gym_start"
fi

# 验证：两个活动都存在
count=$(get_schedule_count "$schedule")
if [ "$count" -ge 2 ]; then
    print_result "3.2 衔接保留前置活动" "true"
else
    print_result "3.2 衔接保留前置活动" "false" "开会活动丢失，当前$count个时段"
fi

echo ""

# ============================================================
# 测试场景 4: 修改操作
# ============================================================
echo "=============================================="
echo "测试场景 4: 修改操作"
echo "=============================================="

# 新会话
response=$(send_text "$TOKEN" "" "上午10点开会，下午2点午休")
session_id=$(extract_session_id "$response")
schedule=$(extract_schedule "$response")
original_count=$(get_schedule_count "$schedule")
echo -e "${BLUE}原始时刻表: $original_count 个时段${NC}"
echo -e "${CYAN}$schedule${NC}"

# 修改测试
sleep 2
response=$(send_text "$TOKEN" "$session_id" "把开会改到11点")
schedule=$(extract_schedule "$response")
new_meeting_start=$(echo "$schedule" | jq -r '.[] | select(.status | test("会|开会")) | .start_time' 2>/dev/null | head -1)
new_count=$(get_schedule_count "$schedule")
echo -e "${BLUE}修改后: 开会开始=$new_meeting_start, 时段数=$new_count${NC}"
echo -e "${CYAN}$schedule${NC}"

# 验证：开会时间改为11点
if [ "$new_meeting_start" = "11:00" ]; then
    print_result "4.1 修改时间正确" "true"
else
    print_result "4.1 修改时间正确" "false" "期望11:00，实际=$new_meeting_start"
fi

# 验证：其他活动保留
if check_activity_exists "$schedule" "午休"; then
    print_result "4.2 修改保留其他活动" "true"
else
    print_result "4.2 修改保留其他活动" "false" "午休活动丢失"
fi

echo ""

# ============================================================
# 测试场景 5: 查询操作（event_type 正确性）
# ============================================================
echo "=============================================="
echo "测试场景 5: 查询操作"
echo "=============================================="

response=$(send_text "$TOKEN" "" "查看今天的时刻表")
event_type=$(extract_event_type "$response")
echo -e "${CYAN}事件类型: $event_type${NC}"

if [ "$event_type" = "schedule" ] || [ "$event_type" = "chat" ]; then
    print_result "5.1 查询返回正确事件类型" "true"
else
    print_result "5.1 查询返回正确事件类型" "false" "event_type=$event_type"
fi

echo ""

# ============================================================
# 测试场景 6: 聊天识别（不创建时刻表）
# ============================================================
echo "=============================================="
echo "测试场景 6: 聊天识别"
echo "=============================================="

response=$(send_text "$TOKEN" "" "你好")
event_type=$(extract_event_type "$response")
echo -e "${CYAN}\"你好\" -> 事件类型: $event_type${NC}"

if [ "$event_type" = "chat" ]; then
    print_result "6.1 聊天正确识别" "true"
else
    print_result "6.1 聊天正确识别" "false" "期望chat，实际=$event_type"
fi

sleep 1
response=$(send_text "$TOKEN" "" "今天天气怎么样")
event_type=$(extract_event_type "$response")
echo -e "${CYAN}\"今天天气怎么样\" -> 事件类型: $event_type${NC}"

if [ "$event_type" = "chat" ]; then
    print_result "6.2 问答正确识别" "true"
else
    print_result "6.2 问答正确识别" "false" "期望chat，实际=$event_type"
fi

echo ""

# ============================================================
# 测试场景 7: 复杂多轮对话（10轮）
# ============================================================
echo "=============================================="
echo "测试场景 7: 复杂多轮对话（10轮）"
echo "=============================================="

# 第1轮
response=$(send_text "$TOKEN" "" "早上8点起床")
session_id=$(extract_session_id "$response")
echo -e "${BLUE}轮次1: 早上8点起床${NC}"

# 第2轮
sleep 1
response=$(send_text "$TOKEN" "$session_id" "然后9点吃早餐")
echo -e "${BLUE}轮次2: 然后9点吃早餐${NC}"

# 第3轮
sleep 1
response=$(send_text "$TOKEN" "$session_id" "10点开始工作")
echo -e "${BLUE}轮次3: 10点开始工作${NC}"

# 第4轮
sleep 1
response=$(send_text "$TOKEN" "$session_id" "12点午餐")
echo -e "${BLUE}轮次4: 12点午餐${NC}"

# 第5轮
sleep 1
response=$(send_text "$TOKEN" "$session_id" "下午1点午休")
echo -e "${BLUE}轮次5: 下午1点午休${NC}"

# 第6轮
sleep 1
response=$(send_text "$TOKEN" "$session_id" "2点继续工作")
echo -e "${BLUE}轮次6: 2点继续工作${NC}"

# 第7轮
sleep 1
response=$(send_text "$TOKEN" "$session_id" "5点下班")
echo -e "${BLUE}轮次7: 5点下班${NC}"

# 第8轮
sleep 1
response=$(send_text "$TOKEN" "$session_id" "6点健身")
echo -e "${BLUE}轮次8: 6点健身${NC}"

# 第9轮
sleep 1
response=$(send_text "$TOKEN" "$session_id" "7点半晚餐")
echo -e "${BLUE}轮次9: 7点半晚餐${NC}"

# 第10轮
sleep 1
response=$(send_text "$TOKEN" "$session_id" "9点看电影")
schedule=$(extract_schedule "$response")
final_count=$(get_schedule_count "$schedule")
echo -e "${BLUE}轮次10: 9点看电影 -> 最终时段数: $final_count${NC}"
echo -e "${CYAN}最终时刻表: $schedule${NC}"

# 验证：应该有多个时段
if [ "$final_count" -ge 5 ]; then
    print_result "7.1 10轮对话累积成功" "true"
else
    print_result "7.1 10轮对话累积成功" "false" "期望>=5，实际=$final_count"
fi

# 验证时间格式
time_result=$(validate_schedule_times "$schedule")
if [ $? -eq 0 ]; then
    print_result "7.2 10轮后时间格式正确" "true"
else
    print_result "7.2 10轮后时间格式正确" "false" "$time_result"
fi

echo ""

# ============================================================
# 测试场景 8: 时间格式边界测试
# ============================================================
echo "=============================================="
echo "测试场景 8: 时间格式边界测试"
echo "=============================================="

# 凌晨时间
response=$(send_text "$TOKEN" "" "凌晨2点睡觉")
schedule=$(extract_schedule "$response")
start_time=$(echo "$schedule" | jq -r '.[0].start_time // empty')
echo -e "${CYAN}凌晨2点睡觉 -> start_time: $start_time${NC}"
if validate_time_format "$start_time"; then
    print_result "8.1 凌晨时间格式($start_time)" "true"
else
    print_result "8.1 凌晨时间格式($start_time)" "false"
fi

# 午间时间
sleep 1
response=$(send_text "$TOKEN" "" "12点吃午饭")
schedule=$(extract_schedule "$response")
start_time=$(echo "$schedule" | jq -r '.[0].start_time // empty')
echo -e "${CYAN}12点吃午饭 -> start_time: $start_time${NC}"
if validate_time_format "$start_time"; then
    print_result "8.2 正午时间格式($start_time)" "true"
else
    print_result "8.2 正午时间格式($start_time)" "false"
fi

# 晚间时间
sleep 1
response=$(send_text "$TOKEN" "" "晚上11点睡觉")
schedule=$(extract_schedule "$response")
start_time=$(echo "$schedule" | jq -r '.[0].start_time // empty')
echo -e "${CYAN}晚上11点睡觉 -> start_time: $start_time${NC}"
if validate_time_format "$start_time"; then
    print_result "8.3 晚间时间格式($start_time)" "true"
else
    print_result "8.3 晚间时间格式($start_time)" "false"
fi

echo ""

# ============================================================
# 测试场景 9: 删除操作
# ============================================================
echo "=============================================="
echo "测试场景 9: 删除操作"
echo "=============================================="

# 创建多个活动
response=$(send_text "$TOKEN" "" "上午开会，下午健身，晚上看电影")
session_id=$(extract_session_id "$response")
schedule=$(extract_schedule "$response")
original_count=$(get_schedule_count "$schedule")
echo -e "${BLUE}原始: $original_count 个时段${NC}"
echo -e "${CYAN}$schedule${NC}"

# 删除一个
sleep 2
response=$(send_text "$TOKEN" "$session_id" "取消健身")
schedule=$(extract_schedule "$response")
new_count=$(get_schedule_count "$schedule")
echo -e "${BLUE}删除后: $new_count 个时段${NC}"
echo -e "${CYAN}$schedule${NC}"

if [ "$new_count" -lt "$original_count" ]; then
    print_result "9.1 删除操作成功" "true"
else
    print_result "9.1 删除操作成功" "false" "时段数未减少（原$original_count，现$new_count）"
fi

# 验证其他活动保留
if check_activity_exists "$schedule" "会" || check_activity_exists "$schedule" "电影"; then
    print_result "9.2 删除保留其他活动" "true"
else
    print_result "9.2 删除保留其他活动" "false"
fi

echo ""

# ============================================================
# 测试场景 10: 明天的时刻表
# ============================================================
echo "=============================================="
echo "测试场景 10: 明天的时刻表"
echo "=============================================="

response=$(send_text "$TOKEN" "" "明天下午3点开会")
schedule=$(extract_schedule "$response")
echo -e "${CYAN}$schedule${NC}"

# 验证时间格式
time_result=$(validate_schedule_times "$schedule")
if [ $? -eq 0 ]; then
    print_result "10.1 明天时刻表时间格式" "true"
else
    print_result "10.1 明天时刻表时间格式" "false" "$time_result"
fi

echo ""

# ============================================================
# 汇总报告
# ============================================================
echo "=============================================="
echo "               测试汇总报告"
echo "=============================================="
echo ""
echo -e "总测试数: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "通过: ${GREEN}$PASSED_TESTS${NC}"
echo -e "失败: ${RED}$FAILED_TESTS${NC}"

if [ $TOTAL_TESTS -gt 0 ]; then
    PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    echo -e "通过率: ${BLUE}$PASS_RATE%${NC}"

    if [ $PASS_RATE -ge 90 ]; then
        echo -e "\n${GREEN}[优秀] v2 架构改进效果显著！${NC}"
    elif [ $PASS_RATE -ge 70 ]; then
        echo -e "\n${YELLOW}[良好] 大部分功能正常，还有改进空间${NC}"
    else
        echo -e "\n${RED}[需改进] 通过率较低，需要排查问题${NC}"
    fi
fi

echo ""
echo "=============================================="
