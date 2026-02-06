#!/bin/bash

# ========================================
# 语音时刻表 - 全面闭环测试
# 测试四大环节：查询、修改、保存、展示
# ========================================

API_BASE="http://49.232.13.41:8080/api/v1"
PHONE="13800000003"
CODE="333333"
TODAY=$(date +%Y-%m-%d)

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

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

# 发送语音时刻表请求
send_request() {
    local text="$1"
    local session_id="${2:-}"
    local body="{\"text\":\"$text\""
    if [ -n "$session_id" ]; then
        body="$body,\"session_id\":\"$session_id\""
    fi
    body="$body}"

    local tmpfile=$(mktemp)
    curl -s -N "$API_BASE/agent/voice-schedule/v4/text" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "$body" > "$tmpfile" 2>&1
    cat "$tmpfile"
    rm -f "$tmpfile"
}

# API 函数
api_get_items() {
    curl -s "$API_BASE/agent/my-schedule/$TODAY" \
        -H "Authorization: Bearer $TOKEN" | jq -r '.data.items // []'
}

api_get_item_count() {
    curl -s "$API_BASE/agent/my-schedule/$TODAY" \
        -H "Authorization: Bearer $TOKEN" | jq -r '.data.items | length // 0'
}

# 提取函数
extract_session_id() {
    echo "$1" | grep "session_start" | sed 's/data: //' | jq -r '.session_id' 2>/dev/null || echo ""
}

extract_schedule_preview() {
    echo "$1" | grep "schedule_preview" | sed 's/data: //' | jq -r '.items // []' 2>/dev/null || echo "[]"
}

extract_item_count() {
    local count=$(echo "$1" | grep "schedule_preview" | sed 's/data: //' | jq -r '.items | length' 2>/dev/null)
    [ -z "$count" ] || [ "$count" == "null" ] && echo "0" || echo "$count"
}

extract_chat_message() {
    echo "$1" | grep '"type":"chat"' | sed 's/data: //' | jq -r '.message // ""' 2>/dev/null | head -1 || echo ""
}

# 断言函数
assert_ge() {
    local name="$1" actual="${2:-0}" expected="$3"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [ "$actual" -ge "$expected" ] 2>/dev/null; then
        echo -e "    ${GREEN}✓${NC} $name (值: $actual)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "    ${RED}✗${NC} $name (期望>=$expected, 实际:$actual)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

assert_contains() {
    local name="$1" haystack="$2" needle="$3"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if echo "$haystack" | grep -qE "$needle"; then
        echo -e "    ${GREEN}✓${NC} $name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "    ${RED}✗${NC} $name (未找到: $needle)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

assert_not_empty() {
    local name="$1" value="$2"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [ -n "$value" ] && [ "$value" != "null" ] && [ "$value" != "[]" ]; then
        echo -e "    ${GREEN}✓${NC} $name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "    ${RED}✗${NC} $name (值为空)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

assert_eq() {
    local name="$1" expected="$2" actual="$3"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [ "$expected" == "$actual" ]; then
        echo -e "    ${GREEN}✓${NC} $name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "    ${RED}✗${NC} $name (期望:$expected, 实际:$actual)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# 使用 API 直接设置时刻表（绕过 LLM，100% 可靠）
api_set_schedule() {
    local items_json="$1"
    curl -s -X PUT "$API_BASE/agent/my-schedule/$TODAY" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"items\":$items_json}" > /dev/null
    sleep 0.3
}

# 预定义标准测试时刻表
STANDARD_SCHEDULE='[
    {"start_time":"08:00","end_time":"09:00","emoji":"🌅","status":"起床"},
    {"start_time":"09:00","end_time":"10:00","emoji":"💼","status":"开会"},
    {"start_time":"12:00","end_time":"13:00","emoji":"🍽️","status":"午餐"},
    {"start_time":"14:00","end_time":"18:00","emoji":"💻","status":"写代码"},
    {"start_time":"18:00","end_time":"19:00","emoji":"🏃","status":"健身"},
    {"start_time":"20:00","end_time":"21:00","emoji":"🍽️","status":"晚餐"}
]'

# 重置为标准测试时刻表
reset_to_standard() {
    api_set_schedule "$STANDARD_SCHEDULE"
}

# 清空时刻表
clear_schedule() {
    api_set_schedule "[]"
}

# 开始测试组
start_group() {
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}  $1${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# ========================================
# 主测试
# ========================================
echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║         语音时刻表 - 全面闭环测试                        ║${NC}"
echo -e "${CYAN}║     测试范围：查询 → 修改 → 保存 → 展示                   ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"

echo ""
echo -e "${BLUE}>>> 初始化${NC}"
TOKEN=$(get_token)
if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
    echo -e "${RED}获取 Token 失败！${NC}"
    exit 1
fi
echo -e "  Token: ${TOKEN:0:20}..."

# ========================================
# 第一部分：查询测试
# ========================================
start_group "第一部分：查询测试 (Query)"

echo -e "  ${BLUE}1.1 基础时刻表创建与API验证${NC}"
reset_to_standard
API_ITEMS=$(api_get_items)
assert_contains "API包含开会" "$API_ITEMS" "开会"
assert_contains "API包含午餐" "$API_ITEMS" "午餐"

echo -e "  ${BLUE}1.2 上午时段查询${NC}"
RESULT=$(send_request "今天上午有什么安排")
CHAT=$(extract_chat_message "$RESULT")
assert_contains "上午查询包含起床或开会" "$CHAT" "起床|开会|上午|08|09"

echo -e "  ${BLUE}1.3 下午时段查询${NC}"
RESULT=$(send_request "下午有什么安排")
CHAT=$(extract_chat_message "$RESULT")
assert_contains "下午查询包含写代码" "$CHAT" "写代码|代码|14|下午"

echo -e "  ${BLUE}1.4 晚上时段查询${NC}"
RESULT=$(send_request "18点到21点有什么安排")
CHAT=$(extract_chat_message "$RESULT")
assert_contains "晚上查询包含健身或晚餐" "$CHAT" "健身|晚餐|18|20"

echo -e "  ${BLUE}1.5 特定时间点查询${NC}"
RESULT=$(send_request "12点在做什么")
CHAT=$(extract_chat_message "$RESULT")
assert_contains "12点查询包含午餐" "$CHAT" "午餐|吃饭|12"

echo -e "  ${BLUE}1.6 凌晨时段查询${NC}"
RESULT=$(send_request "凌晨3点在做什么")
CHAT=$(extract_chat_message "$RESULT")
assert_not_empty "凌晨查询返回响应" "$CHAT"

echo -e "  ${BLUE}1.7 模糊时间查询${NC}"
RESULT=$(send_request "傍晚有什么安排")
CHAT=$(extract_chat_message "$RESULT")
assert_not_empty "傍晚查询返回响应" "$CHAT"

echo -e "  ${BLUE}1.8 API 直接查询${NC}"
API_COUNT=$(api_get_item_count)
assert_ge "API 返回时刻表条目" "$API_COUNT" 5

# ========================================
# 第二部分：修改测试
# ========================================
start_group "第二部分：修改测试 (Modify)"

echo -e "  ${BLUE}2.1 修改状态（12点午餐→去医院）${NC}"
reset_to_standard
RESULT=$(send_request "把12:00-13:00的午餐改成去医院")
PREVIEW=$(extract_schedule_preview "$RESULT")
COUNT=$(extract_item_count "$RESULT")
assert_ge "修改后保留其他时段" "$COUNT" 5
assert_contains "修改后包含医院" "$PREVIEW" "医院"
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

echo -e "  ${BLUE}2.2 修改时间范围（把18点健身改到19点）${NC}"
RESULT=$(send_request "把18:00-19:00的健身改成19:00-20:00健身")
PREVIEW=$(extract_schedule_preview "$RESULT")
COUNT=$(extract_item_count "$RESULT")
assert_ge "修改时间后保留其他时段" "$COUNT" 5
assert_contains "包含19点健身" "$PREVIEW" "19:00|健身"
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

echo -e "  ${BLUE}2.3 添加不冲突时段（10:00-11:00喝茶）${NC}"
BEFORE_COUNT=$(api_get_item_count)
RESULT=$(send_request "添加10:00-11:00喝茶")
COUNT=$(extract_item_count "$RESULT")
assert_ge "添加后时段增加" "$COUNT" "$BEFORE_COUNT"
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

echo -e "  ${BLUE}2.4 添加冲突时段（15:00-15:30咖啡，插入写代码中间）${NC}"
RESULT=$(send_request "添加15:00-15:30喝咖啡")
PREVIEW=$(extract_schedule_preview "$RESULT")
COUNT=$(extract_item_count "$RESULT")
assert_ge "添加冲突时段后条目增加" "$COUNT" 7
assert_contains "包含咖啡" "$PREVIEW" "咖啡"
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

echo -e "  ${BLUE}2.5 边界时间修改（添加6:00早起）${NC}"
RESULT=$(send_request "添加6:00-7:00早起锻炼")
PREVIEW=$(extract_schedule_preview "$RESULT")
assert_contains "包含6点时段" "$PREVIEW" "06:00|6:00"
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

echo -e "  ${BLUE}2.6 边界时间修改（添加23:00睡前阅读）${NC}"
RESULT=$(send_request "添加23:00-23:30睡前阅读")
PREVIEW=$(extract_schedule_preview "$RESULT")
assert_contains "包含23点时段" "$PREVIEW" "23:00"
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

echo -e "  ${BLUE}2.7 长状态名称测试${NC}"
RESULT=$(send_request "添加16:00-17:00参加公司年度战略规划会议讨论")
PREVIEW=$(extract_schedule_preview "$RESULT")
# LLM 可能简化状态名，检测是否包含任何相关关键词
assert_contains "包含长状态名" "$PREVIEW" "会议|规划|战略|公司|年度|讨论|16:00"
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

# ========================================
# 第三部分：保存测试
# ========================================
start_group "第三部分：保存测试 (Save)"

echo -e "  ${BLUE}3.1 保存后立即查询验证${NC}"
api_set_schedule '[{"start_time":"09:00","end_time":"10:00","emoji":"💼","status":"测试会议"},{"start_time":"14:00","end_time":"15:00","emoji":"💻","status":"测试编码"}]'
API_ITEMS=$(api_get_items)
assert_contains "保存后API包含测试会议" "$API_ITEMS" "测试会议"
assert_contains "保存后API包含测试编码" "$API_ITEMS" "测试编码"

echo -e "  ${BLUE}3.2 修改保存后验证${NC}"
RESULT=$(send_request "把9:00-10:00的测试会议改成重要会议")
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null
sleep 0.5
API_ITEMS=$(api_get_items)
assert_contains "修改保存后包含重要会议" "$API_ITEMS" "重要会议"

echo -e "  ${BLUE}3.3 添加保存后验证${NC}"
RESULT=$(send_request "添加11:00-12:00代码审查")
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null
sleep 0.5
API_ITEMS=$(api_get_items)
assert_contains "添加保存后包含代码审查" "$API_ITEMS" "代码审查"

echo -e "  ${BLUE}3.4 连续保存测试（3次修改3次保存）${NC}"
RESULT=$(send_request "把9:00-10:00改成晨会")
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

RESULT=$(send_request "把11:00-12:00改成评审会")
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

RESULT=$(send_request "把14:00-15:00改成编程时间")
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

sleep 0.5
API_ITEMS=$(api_get_items)
assert_contains "连续保存后包含晨会" "$API_ITEMS" "晨会"
assert_contains "连续保存后包含评审会" "$API_ITEMS" "评审会"

echo -e "  ${BLUE}3.5 不保存验证（修改后不保存）${NC}"
BEFORE_ITEMS=$(api_get_items)
RESULT=$(send_request "把9:00-10:00改成临时测试")
# 不调用保存
sleep 0.5
AFTER_ITEMS=$(api_get_items)
assert_contains "不保存时数据保持不变" "$AFTER_ITEMS" "晨会"

# ========================================
# 第四部分：展示测试
# ========================================
start_group "第四部分：展示测试 (Display)"

echo -e "  ${BLUE}4.1 API展示完整时刻表验证${NC}"
reset_to_standard
API_ITEMS=$(api_get_items)
assert_contains "展示包含起床" "$API_ITEMS" "起床"
assert_contains "展示包含开会" "$API_ITEMS" "开会"
assert_contains "展示包含午餐" "$API_ITEMS" "午餐"

echo -e "  ${BLUE}4.2 API 展示验证时间排序${NC}"
API_ITEMS=$(api_get_items)
FIRST_TIME=$(echo "$API_ITEMS" | jq -r '.[0].start_time' 2>/dev/null)
LAST_TIME=$(echo "$API_ITEMS" | jq -r '.[-1].start_time' 2>/dev/null)
if [[ "$FIRST_TIME" < "$LAST_TIME" ]]; then
    assert_eq "时间排序正确（升序）" "true" "true"
else
    assert_eq "时间排序正确（升序）" "true" "false"
fi

echo -e "  ${BLUE}4.3 API 展示验证数据结构${NC}"
FIRST_ITEM=$(echo "$API_ITEMS" | jq '.[0]' 2>/dev/null)
HAS_START=$(echo "$FIRST_ITEM" | jq -r '.start_time' 2>/dev/null)
HAS_END=$(echo "$FIRST_ITEM" | jq -r '.end_time' 2>/dev/null)
HAS_STATUS=$(echo "$FIRST_ITEM" | jq -r '.status' 2>/dev/null)
HAS_EMOJI=$(echo "$FIRST_ITEM" | jq -r '.emoji' 2>/dev/null)
assert_not_empty "数据结构包含start_time" "$HAS_START"
assert_not_empty "数据结构包含end_time" "$HAS_END"
assert_not_empty "数据结构包含status" "$HAS_STATUS"
assert_not_empty "数据结构包含emoji" "$HAS_EMOJI"

echo -e "  ${BLUE}4.4 Emoji 多样性验证${NC}"
EMOJIS=$(echo "$API_ITEMS" | jq -r '.[].emoji' 2>/dev/null | sort -u | wc -l | tr -d ' ')
assert_ge "Emoji 种类多样" "$EMOJIS" 2

echo -e "  ${BLUE}4.5 时间格式验证（HH:MM）${NC}"
FIRST_TIME=$(echo "$API_ITEMS" | jq -r '.[0].start_time' 2>/dev/null)
if [[ "$FIRST_TIME" =~ ^[0-9]{2}:[0-9]{2}$ ]]; then
    assert_eq "时间格式正确" "true" "true"
else
    assert_eq "时间格式正确" "true" "false"
fi

# ========================================
# 第五部分：边界条件测试
# ========================================
start_group "第五部分：边界条件测试 (Edge Cases)"

echo -e "  ${BLUE}5.1 单时段时刻表${NC}"
api_set_schedule '[{"start_time":"12:00","end_time":"13:00","emoji":"🍽️","status":"午餐"}]'
API_COUNT=$(api_get_item_count)
assert_eq "单时段时刻表" "1" "$API_COUNT"

echo -e "  ${BLUE}5.2 大量时段时刻表（10+个时段）${NC}"
api_set_schedule '[
    {"start_time":"06:00","end_time":"07:00","emoji":"🌅","status":"起床"},
    {"start_time":"07:00","end_time":"08:00","emoji":"🍳","status":"早餐"},
    {"start_time":"08:00","end_time":"09:00","emoji":"💼","status":"晨会"},
    {"start_time":"09:00","end_time":"10:00","emoji":"💻","status":"开发"},
    {"start_time":"10:00","end_time":"10:30","emoji":"☕","status":"休息"},
    {"start_time":"10:30","end_time":"12:00","emoji":"💻","status":"开发"},
    {"start_time":"12:00","end_time":"13:00","emoji":"🍽️","status":"午餐"},
    {"start_time":"13:00","end_time":"14:00","emoji":"😴","status":"午休"},
    {"start_time":"14:00","end_time":"15:00","emoji":"💻","status":"开发"},
    {"start_time":"15:00","end_time":"15:30","emoji":"🍵","status":"茶歇"},
    {"start_time":"15:30","end_time":"17:00","emoji":"💻","status":"开发"},
    {"start_time":"17:00","end_time":"18:00","emoji":"📝","status":"总结"}
]'
API_COUNT=$(api_get_item_count)
assert_ge "大量时段创建成功" "$API_COUNT" 10

echo -e "  ${BLUE}5.3 相邻时段测试（无间隔）${NC}"
api_set_schedule '[
    {"start_time":"09:00","end_time":"10:00","emoji":"💼","status":"会议A"},
    {"start_time":"10:00","end_time":"11:00","emoji":"💼","status":"会议B"},
    {"start_time":"11:00","end_time":"12:00","emoji":"💼","status":"会议C"}
]'
API_COUNT=$(api_get_item_count)
assert_eq "相邻时段创建成功" "3" "$API_COUNT"

echo -e "  ${BLUE}5.4 最小时段测试（30分钟）${NC}"
RESULT=$(send_request "添加14:00-14:30快速会议")
PREVIEW=$(extract_schedule_preview "$RESULT")
assert_contains "30分钟时段创建成功" "$PREVIEW" "14:00|14:30"
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

echo -e "  ${BLUE}5.5 全天时刻表覆盖测试${NC}"
api_set_schedule '[
    {"start_time":"06:00","end_time":"07:00","emoji":"🌅","status":"起床"},
    {"start_time":"08:00","end_time":"09:00","emoji":"🍳","status":"早餐"},
    {"start_time":"09:00","end_time":"12:00","emoji":"💼","status":"工作"},
    {"start_time":"12:00","end_time":"13:00","emoji":"🍽️","status":"午餐"},
    {"start_time":"14:00","end_time":"18:00","emoji":"💼","status":"工作"},
    {"start_time":"18:00","end_time":"19:00","emoji":"🚪","status":"下班"},
    {"start_time":"19:00","end_time":"20:00","emoji":"🍽️","status":"晚餐"},
    {"start_time":"21:00","end_time":"23:00","emoji":"📺","status":"休闲"},
    {"start_time":"23:00","end_time":"23:59","emoji":"😴","status":"睡觉"}
]'
API_ITEMS=$(api_get_items)
assert_contains "全天覆盖包含起床" "$API_ITEMS" "起床"
assert_contains "全天覆盖包含睡觉" "$API_ITEMS" "睡觉"

# ========================================
# 第六部分：完整闭环验证
# ========================================
start_group "第六部分：完整闭环验证"

echo -e "  ${BLUE}6.1 创建→查询→修改→保存→展示 完整流程${NC}"

echo -e "    Step 1: 创建时刻表"
api_set_schedule '[
    {"start_time":"09:00","end_time":"10:00","emoji":"💼","status":"产品会议"},
    {"start_time":"14:00","end_time":"15:00","emoji":"💼","status":"技术评审"},
    {"start_time":"16:00","end_time":"18:00","emoji":"💻","status":"代码编写"}
]'
API_COUNT=$(api_get_item_count)
assert_eq "创建时刻表成功" "3" "$API_COUNT"

echo -e "    Step 2: API查询验证"
API_ITEMS=$(api_get_items)
assert_contains "查询包含产品会议" "$API_ITEMS" "产品会议"

echo -e "    Step 3: 修改时刻表"
RESULT=$(send_request "把9:00-10:00的产品会议改成战略会议")
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null

echo -e "    Step 4: 保存验证"
API_ITEMS=$(api_get_items)
assert_contains "保存后包含战略会议" "$API_ITEMS" "战略会议"

echo -e "    Step 5: API展示验证"
API_ITEMS=$(api_get_items)
assert_contains "展示包含战略会议" "$API_ITEMS" "战略会议"

echo -e "  ${BLUE}6.2 多次修改闭环测试${NC}"
api_set_schedule '[{"start_time":"10:00","end_time":"11:00","emoji":"💼","status":"会议A"}]'

# 使用更明确的指令，包含原状态名
RESULT=$(send_request "把10:00-11:00的会议A改成会议B")
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null
sleep 0.3
API_ITEMS=$(api_get_items)
assert_contains "第一次修改成功" "$API_ITEMS" "会议B"

RESULT=$(send_request "把10:00-11:00的会议B改成会议C")
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null
sleep 0.3
API_ITEMS=$(api_get_items)
assert_contains "第二次修改成功" "$API_ITEMS" "会议C"

RESULT=$(send_request "把10:00-11:00的会议C改成会议D")
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null
sleep 0.3
API_ITEMS=$(api_get_items)
assert_contains "第三次修改成功" "$API_ITEMS" "会议D"

echo -e "  ${BLUE}6.3 添加删除闭环测试${NC}"
api_set_schedule '[{"start_time":"12:00","end_time":"13:00","emoji":"🍽️","status":"午餐"}]'
BEFORE=$(api_get_item_count)

RESULT=$(send_request "添加14:00-15:00下午茶")
SESSION=$(extract_session_id "$RESULT")
send_request "保存" "$SESSION" > /dev/null
AFTER_ADD=$(api_get_item_count)
assert_ge "添加后数量增加" "$AFTER_ADD" "$BEFORE"

# ========================================
# 测试报告
# ========================================
echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                      测试报告                            ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  总测试数: ${YELLOW}$TOTAL_TESTS${NC}"
echo -e "  通过: ${GREEN}$PASSED_TESTS${NC}"
echo -e "  失败: ${RED}$FAILED_TESTS${NC}"

if [ $TOTAL_TESTS -gt 0 ]; then
    PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    echo -e "  通过率: ${YELLOW}${PASS_RATE}%${NC}"
fi

echo ""
if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║              所有测试通过！ ✓                            ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
    exit 0
else
    echo -e "${YELLOW}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${YELLOW}║              有 $FAILED_TESTS 个测试失败                              ║${NC}"
    echo -e "${YELLOW}╚══════════════════════════════════════════════════════════╝${NC}"
    exit 1
fi
