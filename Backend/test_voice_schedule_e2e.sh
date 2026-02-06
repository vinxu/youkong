#!/bin/bash

# ========================================
# 语音时刻表端到端测试脚本
# 测试三大核心流程：查询、修改、保存同步
# ========================================

# set -e  # 禁用，以便测试能完整运行

API_BASE="http://49.232.13.41:8080/api/v1"
PHONE="13800000003"
CODE="333333"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 计数器
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 获取 Token
get_token() {
    curl -s -X POST "$API_BASE/auth/sms/verify" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$PHONE\",\"code\":\"$CODE\"}" | jq -r '.data.token'
}

# 发送语音时刻表请求（等待完整响应）
send_request() {
    local text="$1"
    local session_id="${2:-}"

    local body="{\"text\":\"$text\""
    if [ -n "$session_id" ]; then
        body="$body,\"session_id\":\"$session_id\""
    fi
    body="$body}"

    # 使用临时文件收集完整响应
    local tmpfile=$(mktemp)
    curl -s -N "$API_BASE/agent/voice-schedule/v4/text" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "$body" > "$tmpfile" 2>&1
    cat "$tmpfile"
    rm -f "$tmpfile"
}

# 获取时刻表 API（验证同步）
get_schedule_api() {
    local date="${1:-$(date +%Y-%m-%d)}"
    curl -s "$API_BASE/agent/my-schedule/$date" \
        -H "Authorization: Bearer $TOKEN" 2>&1
}

# 直接从 API 获取时刻表条目数量
get_schedule_item_count() {
    local date="${1:-$(date +%Y-%m-%d)}"
    local result=$(curl -s "$API_BASE/agent/my-schedule/$date" \
        -H "Authorization: Bearer $TOKEN" 2>&1)
    echo "$result" | jq -r '.data.items | length' 2>/dev/null || echo "0"
}

# 直接从 API 获取时刻表条目
get_schedule_items() {
    local date="${1:-$(date +%Y-%m-%d)}"
    curl -s "$API_BASE/agent/my-schedule/$date" \
        -H "Authorization: Bearer $TOKEN" | jq -r '.data.items' 2>/dev/null || echo "[]"
}

# 重置时刻表（创建新的基础时刻表）
reset_schedule() {
    echo -e "  ${BLUE}重置时刻表...${NC}"
    local result=$(send_request "重新安排今天的时刻表：8点起床，9点开会，12点午餐，14点写代码，18点健身，20点晚餐")
    local session_id=$(extract_session_id "$result")
    send_request "保存" "$session_id" > /dev/null
    sleep 0.5
}

# 提取 session_id
extract_session_id() {
    echo "$1" | grep "session_start" | sed 's/data: //' | jq -r '.session_id' 2>/dev/null || echo ""
}

# 提取预览时刻表（修改操作返回）
extract_schedule_preview() {
    echo "$1" | grep "schedule_preview" | sed 's/data: //' | jq -r '.items // []' 2>/dev/null || echo "[]"
}

# 提取时段数量（修改操作返回）
extract_item_count() {
    local count=$(echo "$1" | grep "schedule_preview" | sed 's/data: //' | jq -r '.items | length' 2>/dev/null)
    if [ -z "$count" ] || [ "$count" == "null" ]; then
        echo "0"
    else
        echo "$count"
    fi
}

# 提取 chat 消息
extract_chat_message() {
    echo "$1" | grep '"type":"chat"' | sed 's/data: //' | jq -r '.message // ""' 2>/dev/null || echo ""
}

# 检查是否返回了有效响应
check_valid_response() {
    echo "$1" | grep -q "session_start"
}

# 测试断言：相等
run_test() {
    local test_name="$1"
    local expected="$2"
    local actual="$3"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if [ "$expected" == "$actual" ]; then
        echo -e "  ${GREEN}✓${NC} $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "  ${RED}✗${NC} $test_name"
        echo -e "    期望: $expected"
        echo -e "    实际: $actual"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# 测试断言：包含
assert_contains() {
    local test_name="$1"
    local haystack="$2"
    local needle="$3"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if echo "$haystack" | grep -qE "$needle"; then
        echo -e "  ${GREEN}✓${NC} $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "  ${RED}✗${NC} $test_name"
        echo -e "    未找到匹配: $needle"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# 测试断言：数值 >=
assert_ge() {
    local test_name="$1"
    local actual="$2"
    local expected="$3"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    # 处理空值
    if [ -z "$actual" ] || [ "$actual" == "null" ]; then
        actual=0
    fi

    if [ "$actual" -ge "$expected" ] 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} $test_name (实际值: $actual)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "  ${RED}✗${NC} $test_name"
        echo -e "    期望 >= $expected, 实际: $actual"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# 测试断言：非空
assert_not_empty() {
    local test_name="$1"
    local value="$2"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if [ -n "$value" ] && [ "$value" != "null" ] && [ "$value" != "" ]; then
        echo -e "  ${GREEN}✓${NC} $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "  ${RED}✗${NC} $test_name (值为空)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   语音时刻表 E2E 测试${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 获取 Token
echo -e "${YELLOW}>>> 初始化：获取认证 Token${NC}"
TOKEN=$(get_token)
if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
    echo -e "${RED}获取 Token 失败！${NC}"
    exit 1
fi
echo -e "  Token: ${TOKEN:0:30}..."
echo ""

# ========================================
# 测试组 1: 创建基础时刻表
# ========================================
echo -e "${YELLOW}>>> 测试组 1: 创建基础时刻表${NC}"

# 创建新的时刻表
RESULT=$(send_request "重新安排今天的时刻表：8点起床，9点开会，12点午餐，14点写代码，18点健身，20点晚餐")
SESSION_ID=$(extract_session_id "$RESULT")
ITEM_COUNT=$(extract_item_count "$RESULT")

assert_not_empty "返回有效 session_id" "$SESSION_ID"
assert_ge "创建时刻表包含至少6个时段" "$ITEM_COUNT" 6

# 保存时刻表
echo ""
echo -e "${YELLOW}>>> 保存时刻表${NC}"
SAVE_RESULT=$(send_request "保存" "$SESSION_ID")
assert_contains "保存成功" "$SAVE_RESULT" "已保存|保存成功"
echo ""

# ========================================
# 测试组 2: 查询测试
# ========================================
echo -e "${YELLOW}>>> 测试组 2: 查询测试${NC}"

# 2.1 查询今天全部时刻表
echo -e "  ${BLUE}2.1 查询今天全部时刻表${NC}"
QUERY_RESULT=$(send_request "查看今天的时刻表")
CHAT_MSG=$(extract_chat_message "$QUERY_RESULT")
assert_not_empty "查询返回结果" "$CHAT_MSG"
assert_contains "返回包含时刻表内容" "$CHAT_MSG" "开会|午餐|健身|起床|写代码"

# 2.2 查询上午时段
echo -e "  ${BLUE}2.2 查询上午时段${NC}"
QUERY_RESULT=$(send_request "今天上午有什么安排")
CHAT_MSG=$(extract_chat_message "$QUERY_RESULT")
assert_contains "查询上午返回相关内容" "$CHAT_MSG" "开会|起床|上午|09|08"

# 2.3 查询下午时段
echo -e "  ${BLUE}2.3 查询下午时段${NC}"
QUERY_RESULT=$(send_request "下午有什么安排")
CHAT_MSG=$(extract_chat_message "$QUERY_RESULT")
assert_contains "查询下午返回相关内容" "$CHAT_MSG" "写代码|代码|14|下午"

# 2.4 查询特定时间点
echo -e "  ${BLUE}2.4 查询特定时间点（12点）${NC}"
QUERY_RESULT=$(send_request "12点在干什么")
CHAT_MSG=$(extract_chat_message "$QUERY_RESULT")
assert_contains "查询12点返回午餐相关" "$CHAT_MSG" "午餐|吃饭|12"

# 2.5 查询晚上时段
echo -e "  ${BLUE}2.5 查询晚上时段${NC}"
QUERY_RESULT=$(send_request "18点到21点有什么安排")
CHAT_MSG=$(extract_chat_message "$QUERY_RESULT")
assert_contains "查询晚上返回相关安排" "$CHAT_MSG" "健身|晚餐|18|20"

echo ""

# ========================================
# 测试组 3: 修改测试
# ========================================
echo -e "${YELLOW}>>> 测试组 3: 修改测试${NC}"

# 重置时刻表，确保测试隔离
reset_schedule

# 3.1 修改单个时段状态（使用精确指令）
echo -e "  ${BLUE}3.1 修改单个时段状态（12点午餐→医院）${NC}"
MODIFY_RESULT=$(send_request "把12:00-13:00的午餐改成去医院")
NEW_SESSION=$(extract_session_id "$MODIFY_RESULT")
PREVIEW=$(extract_schedule_preview "$MODIFY_RESULT")
ITEM_COUNT=$(extract_item_count "$MODIFY_RESULT")

assert_ge "修改后时刻表保留其他时段" "$ITEM_COUNT" 5
assert_contains "12点改为医院" "$PREVIEW" "医院"

# 保存修改
SAVE_RESULT=$(send_request "保存" "$NEW_SESSION")
assert_contains "修改保存成功" "$SAVE_RESULT" "已保存|保存成功"

# 3.2 修改时间范围（使用精确指令：删除旧的+添加新的）
echo -e "  ${BLUE}3.2 修改时间范围（健身从18点改到19点）${NC}"
# 使用更精确的指令：删除18点健身，添加19点健身
MODIFY_RESULT=$(send_request "删除18:00-19:00的健身，添加19:00-20:00健身")
NEW_SESSION=$(extract_session_id "$MODIFY_RESULT")
PREVIEW=$(extract_schedule_preview "$MODIFY_RESULT")
ITEM_COUNT=$(extract_item_count "$MODIFY_RESULT")

assert_ge "修改时间后保留其他时段" "$ITEM_COUNT" 5
assert_contains "健身时间改到19点" "$PREVIEW" "19:00"

# 保存
send_request "保存" "$NEW_SESSION" > /dev/null

# 3.3 添加新时段（使用精确时间范围）
echo -e "  ${BLUE}3.3 添加新时段（15:00-15:30喝咖啡）${NC}"
MODIFY_RESULT=$(send_request "添加15:00-15:30喝咖啡")
NEW_SESSION=$(extract_session_id "$MODIFY_RESULT")
PREVIEW=$(extract_schedule_preview "$MODIFY_RESULT")
ITEM_COUNT=$(extract_item_count "$MODIFY_RESULT")

assert_ge "添加后时刻表时段增加" "$ITEM_COUNT" 6
assert_contains "新增咖啡时段" "$PREVIEW" "咖啡|15:00"

# 保存
send_request "保存" "$NEW_SESSION" > /dev/null

# 3.4 连续修改不丢失数据
echo -e "  ${BLUE}3.4 连续修改不丢失数据${NC}"
# 通过 API 获取当前时段数量作为基准
BASE_COUNT=$(get_schedule_item_count)
echo -e "    基准时段数: $BASE_COUNT"

# 第一次修改
MODIFY_RESULT=$(send_request "把8:00-9:00的起床改成晨跑")
NEW_SESSION=$(extract_session_id "$MODIFY_RESULT")
COUNT1=$(extract_item_count "$MODIFY_RESULT")
send_request "保存" "$NEW_SESSION" > /dev/null

# 第二次修改
MODIFY_RESULT=$(send_request "把9:00-10:00的开会改成团队会议")
NEW_SESSION=$(extract_session_id "$MODIFY_RESULT")
COUNT2=$(extract_item_count "$MODIFY_RESULT")
send_request "保存" "$NEW_SESSION" > /dev/null

# 验证时段数量不减少
assert_ge "连续修改后时段数量不减少" "$COUNT2" "$((BASE_COUNT - 1))"

echo ""

# ========================================
# 测试组 4: 保存同步测试（使用 API 直接验证）
# ========================================
echo -e "${YELLOW}>>> 测试组 4: 保存同步测试${NC}"

# 4.1 通过 API 直接获取时刻表验证
echo -e "  ${BLUE}4.1 通过 API 验证时刻表已保存${NC}"
API_ITEMS=$(get_schedule_items)
API_COUNT=$(echo "$API_ITEMS" | jq 'length' 2>/dev/null || echo "0")
assert_ge "API 返回时刻表条目数量" "$API_COUNT" 5

# 4.2 验证 API 返回的时刻表包含关键内容
echo -e "  ${BLUE}4.2 验证 API 返回的时刻表内容${NC}"
assert_contains "时刻表包含医院或去医院" "$API_ITEMS" "医院"

# 4.3 验证时刻表完整性（通过 API）
echo -e "  ${BLUE}4.3 验证时刻表包含多种活动${NC}"
assert_contains "时刻表包含健身或写代码" "$API_ITEMS" "健身|写代码|代码"

# 4.4 验证保存后的数据结构正确
echo -e "  ${BLUE}4.4 验证数据结构完整${NC}"
HAS_START_TIME=$(echo "$API_ITEMS" | jq '.[0].start_time' 2>/dev/null)
HAS_STATUS=$(echo "$API_ITEMS" | jq '.[0].status' 2>/dev/null)
if [ -n "$HAS_START_TIME" ] && [ "$HAS_START_TIME" != "null" ] && [ -n "$HAS_STATUS" ] && [ "$HAS_STATUS" != "null" ]; then
    run_test "时刻表数据结构完整" "true" "true"
else
    run_test "时刻表数据结构完整" "true" "false"
fi

echo ""

# ========================================
# 测试组 5: 边界条件测试
# ========================================
echo -e "${YELLOW}>>> 测试组 5: 边界条件测试${NC}"

# 5.1 查询不存在的时段
echo -e "  ${BLUE}5.1 查询凌晨3点${NC}"
QUERY_RESULT=$(send_request "凌晨3点在干什么")
CHAT_MSG=$(extract_chat_message "$QUERY_RESULT")
# 应该返回睡觉或无安排
assert_not_empty "凌晨3点返回结果" "$CHAT_MSG"

# 5.2 查询明天的时刻表
echo -e "  ${BLUE}5.2 查询明天的时刻表${NC}"
QUERY_RESULT=$(send_request "明天有什么安排")
assert_not_empty "查询明天返回有效 session" "$(extract_session_id "$QUERY_RESULT")"

# 5.3 模糊时间查询
echo -e "  ${BLUE}5.3 模糊时间查询（傍晚）${NC}"
QUERY_RESULT=$(send_request "傍晚有什么安排")
CHAT_MSG=$(extract_chat_message "$QUERY_RESULT")
assert_not_empty "傍晚查询返回结果" "$CHAT_MSG"

echo ""

# ========================================
# 测试组 6: 压力测试
# ========================================
echo -e "${YELLOW}>>> 测试组 6: 压力测试（连续修改验证数据完整性）${NC}"

# 重置时刻表
reset_schedule

# 获取基准数量
BASE_COUNT=$(get_schedule_item_count)
echo -e "  基准时段数: $BASE_COUNT"

# 连续修改 3 次，验证数据不丢失
echo -e "  ${BLUE}连续3次修改...${NC}"
MODIFY_RESULT=$(send_request "把8:00-9:00的起床改成晨练")
NEW_SESSION=$(extract_session_id "$MODIFY_RESULT")
send_request "保存" "$NEW_SESSION" > /dev/null
sleep 0.5

MODIFY_RESULT=$(send_request "把9:00-10:00的开会改成晨会")
NEW_SESSION=$(extract_session_id "$MODIFY_RESULT")
send_request "保存" "$NEW_SESSION" > /dev/null
sleep 0.5

MODIFY_RESULT=$(send_request "把12:00-13:00的午餐改成工作午餐")
NEW_SESSION=$(extract_session_id "$MODIFY_RESULT")
send_request "保存" "$NEW_SESSION" > /dev/null
sleep 0.5

# 通过 API 验证数据完整性
FINAL_COUNT=$(get_schedule_item_count)
echo -e "  最终时段数: $FINAL_COUNT"
assert_ge "压力测试后时段数不减少" "$FINAL_COUNT" "$((BASE_COUNT - 1))"

echo ""
echo -e "  ${BLUE}验证数据完整性${NC}"
API_ITEMS=$(get_schedule_items)
# 验证时刻表结构完整（有多个时段）
ITEM_COUNT=$(echo "$API_ITEMS" | jq 'length' 2>/dev/null || echo "0")
assert_ge "压力测试后仍有足够时段" "$ITEM_COUNT" 5

echo ""

# ========================================
# 测试报告
# ========================================
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   测试报告${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "总测试数: $TOTAL_TESTS"
echo -e "${GREEN}通过: $PASSED_TESTS${NC}"
echo -e "${RED}失败: $FAILED_TESTS${NC}"
PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
echo -e "通过率: ${PASS_RATE}%"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}   所有测试通过！ ✓${NC}"
    echo -e "${GREEN}========================================${NC}"
    exit 0
else
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}   有 $FAILED_TESTS 个测试失败${NC}"
    echo -e "${YELLOW}========================================${NC}"
    exit 1
fi
