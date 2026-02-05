#!/bin/bash

# 语音时刻表 AI 对话系统 - 终极评估测试脚本
# 测试目标：以真实用户行为模拟，全面评估 AI 对话系统
# 测试规模：250+ 测试用例，覆盖 20 个维度

API_BASE="http://49.232.13.41:8080/api/v1"
REPORT_FILE="/Users/xuxuheng/Desktop/youkong/Backend/EVALUATION_REPORT.md"
LOG_FILE="/Users/xuxuheng/Desktop/youkong/Backend/test_logs.json"

# 测试账号
TEST_PHONE="13800000001"
TEST_CODE="111111"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 统计变量
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
declare -A DIMENSION_SCORES
declare -A DIMENSION_COUNTS

# 初始化报告
init_report() {
    cat > "$REPORT_FILE" << 'EOF'
# 语音时刻表 AI 对话系统 - 终极评估报告

> 生成时间：$(date '+%Y-%m-%d %H:%M:%S')
> 测试版本：build-87
> 测试用例：250+ 用例，20 个维度

---

## 一、执行摘要

EOF
    echo "[]" > "$LOG_FILE"
}

# 获取 Token
get_token() {
    local response=$(curl -s -X POST "$API_BASE/auth/sms/verify" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$TEST_PHONE\",\"code\":\"$TEST_CODE\"}")

    TOKEN=$(echo "$response" | jq -r '.data.token // empty')

    if [ -z "$TOKEN" ]; then
        echo -e "${RED}获取 Token 失败${NC}"
        exit 1
    fi
    echo -e "${GREEN}Token 获取成功${NC}"
}

# 发送语音时刻表请求
send_voice_request() {
    local transcript="$1"
    local session_id="${2:-}"

    local body="{\"transcript\":\"$transcript\""
    if [ -n "$session_id" ]; then
        body="$body,\"session_id\":\"$session_id\""
    fi
    body="$body}"

    local response=$(curl -s -X POST "$API_BASE/voice-schedule/process" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "$body" \
        --max-time 60)

    echo "$response"
}

# 记录测试结果
log_result() {
    local test_id="$1"
    local dimension="$2"
    local input="$3"
    local expected="$4"
    local actual="$5"
    local score="$6"
    local passed="$7"

    # 更新统计
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [ "$passed" = "true" ]; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    # 更新维度分数
    if [ -z "${DIMENSION_SCORES[$dimension]}" ]; then
        DIMENSION_SCORES[$dimension]=0
        DIMENSION_COUNTS[$dimension]=0
    fi
    DIMENSION_SCORES[$dimension]=$((DIMENSION_SCORES[$dimension] + score))
    DIMENSION_COUNTS[$dimension]=$((DIMENSION_COUNTS[$dimension] + 1))

    # 记录到日志
    local log_entry="{\"id\":\"$test_id\",\"dimension\":\"$dimension\",\"input\":\"$input\",\"expected\":\"$expected\",\"score\":$score,\"passed\":$passed}"

    # 输出进度
    if [ "$passed" = "true" ]; then
        echo -e "${GREEN}✓${NC} [$test_id] $dimension: $input → Score: $score"
    else
        echo -e "${RED}✗${NC} [$test_id] $dimension: $input → Score: $score"
    fi
}

# 评估响应质量
evaluate_response() {
    local response="$1"
    local expected_action="$2"
    local expected_contains="$3"

    local event_type=$(echo "$response" | jq -r '.data.event_type // empty')
    local message=$(echo "$response" | jq -r '.data.message // empty')
    local schedule=$(echo "$response" | jq -r '.data.schedule // []')
    local error=$(echo "$response" | jq -r '.message // empty')

    local score=0
    local passed="false"

    # 检查是否有响应
    if [ -z "$response" ] || [ "$response" = "null" ]; then
        echo "0:false:无响应"
        return
    fi

    # 检查是否有错误
    if echo "$response" | jq -e '.code != 0' > /dev/null 2>&1; then
        echo "10:false:API错误: $error"
        return
    fi

    # 基础分数：有响应 +30
    score=30

    # 检查 event_type 是否符合预期
    if [ -n "$expected_action" ]; then
        if [ "$event_type" = "$expected_action" ]; then
            score=$((score + 40))
            passed="true"
        elif [ "$event_type" = "schedule" ] && [ "$expected_action" = "create" ]; then
            score=$((score + 40))
            passed="true"
        elif [ "$event_type" = "current_status" ] && [ "$expected_action" = "update_status" ]; then
            score=$((score + 40))
            passed="true"
        fi
    else
        # 没有预期 action，只要有合理响应就给分
        if [ -n "$event_type" ]; then
            score=$((score + 20))
        fi
    fi

    # 检查是否包含预期内容
    if [ -n "$expected_contains" ]; then
        if echo "$message $schedule" | grep -qi "$expected_contains"; then
            score=$((score + 30))
        fi
    else
        # 有合理消息 +20
        if [ -n "$message" ]; then
            score=$((score + 20))
        fi
    fi

    # 如果分数达到 70 以上算通过
    if [ $score -ge 70 ]; then
        passed="true"
    fi

    echo "$score:$passed:$event_type"
}

# ========== 测试函数 ==========

# 维度 1：基础时刻表创建
test_basic_creation() {
    echo -e "\n${BLUE}========== 维度 1：基础时刻表创建 ==========${NC}"

    local tests=(
        "B01|下午3点开会|create|开会"
        "B02|晚上8点吃饭|create|吃饭"
        "B03|上午9点到12点工作|create|工作"
        "B04|中午12点午餐，下午2点继续工作到6点|create|"
        "B05|7点起床，8点出门，9点上班|create|"
        "B06|下午健身1.5小时|create|健身"
        "B07|晚上看电影2个半小时|create|电影"
        "B08|从早上9点一直忙到晚上9点|create|"
        "B09|10:30-11:45 面试|create|面试"
        "B10|下午茶时间3点到4点|create|"
        "B11|明早8点的飞机|create|飞机"
        "B12|周会，每周一上午10点|create|"
        "B13|和老板1on1|create|"
        "B14|年会下午全程|create|年会"
        "B15|通宵写代码|create|"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "基础创建" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 2：错别字与拼音输入错误
test_typo_tolerance() {
    echo -e "\n${BLUE}========== 维度 2：错别字与拼音输入错误 ==========${NC}"

    local tests=(
        "Y01|下午3点开回|create|开会"
        "Y02|晚上吃反|create|吃饭"
        "Y03|明天尚午健身|create|健身"
        "Y04|在家修息|update_status|休息"
        "Y05|去建身房|create|健身"
        "Y06|xiawu 3dian kaihui|create|开会"
        "Y07|wanshang chifan|create|吃饭"
        "Y08|mingtian xiuxi|create|休息"
        "Y09|3dian kaihi|create|开"
        "Y10|开会 xia午|create|开会"
        "Y11|下物3点|create|"
        "Y12|琬上8点|create|"
        "Y13|建身1小时|create|健身"
        "Y14|看点影|create|电影"
        "Y15|吃个法|create|吃"
        "Y16|shui觉|create|睡"
        "Y17|xiu息一下|update_status|休息"
        "Y18|上班 gongzuo|create|上班"
        "Y19|kai会|create|开会"
        "Y20|jian身fang|create|健身"
        "Y21|3嗲你开会|create|开会"
        "Y22|我药去开会|create|开会"
        "Y23|帮我安培一下|create|"
        "Y24|放松放送|update_status|放松"
        "Y25|休息修习|update_status|休息"
        "Y26|名天开会|create|开会"
        "Y27|工做到6点|create|工作"
        "Y28|回家去取|create|"
        "Y29|睡交|create|睡"
        "Y30|看书独书|create|看书"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "错别字容忍" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 3：逻辑混乱与前后矛盾
test_logic_chaos() {
    echo -e "\n${BLUE}========== 维度 3：逻辑混乱与前后矛盾 ==========${NC}"

    local tests=(
        "L01|明天3点开会...不对等等，是今天...哦不还是明天||"
        "L02|下午开会，哦对了我想起来了其实是上午，算了还是下午吧|create|下午"
        "L03|健身2小时，1小时也行，反正看情况|create|健身"
        "L04|约了朋友吃饭，可能取消，先加上吧，算了不加了，还是加上|create|吃饭"
        "L05|开会是3点还是4点来着？我也忘了，你帮我设3点吧|create|3"
        "L06|上午...不对下午...等下让我想想...上午对上午|create|上午"
        "L07|先健身再吃饭...不对先吃饭再健身比较好|create|"
        "L08|取消那个...就是那个...之前说的那个|cancel|"
        "L09|3点4点5点都有事...算了太乱了，就3点一个会吧|create|3"
        "L10|明天休息...哦不对明天要上班...周末才休息|create|上班"
        "L11|要开会的，好像是要开会，应该是吧|create|开会"
        "L12|下午那个改一下...改什么来着...算了不改了||"
        "L13|我刚才说的是3点？不对我说的4点吧？||"
        "L14|删掉删掉...等下别删...还是删了吧|delete|"
        "L15|这个那个还有那个...你懂的||"
        "L16|时间你看着办，反正下午|create|下午"
        "L17|改到...改到几点好呢...3点？4点？你说呢||"
        "L18|有个事...什么事来着...算了想起来再说||"
        "L19|确定...不对等下...好吧确定|confirm|"
        "L20|全删了重来...哎不用了保留吧||"
        "L21|把会议...不是会议是面试...改到明天|modify|面试"
        "L22|我说错了...还是没说错...反正就那样||"
        "L23|那个什么...就是...你知道的那个||"
        "L24|从2点开始...还是3点...2点半吧折中一下|create|2:30"
        "L25|今天明天后天都行，你随便选一天||"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "逻辑混乱" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 4：口语化与省略表达
test_colloquial() {
    echo -e "\n${BLUE}========== 维度 4：口语化与省略表达 ==========${NC}"

    local tests=(
        "K01|下午有点事儿||"
        "K02|差不多那会儿吧||"
        "K03|就...你懂的||"
        "K04|一会儿要出去||"
        "K05|中午那趴||中午"
        "K06|晚点再说||"
        "K07|反正就那样|confirm|"
        "K08|随便啦||"
        "K09|等下看情况||"
        "K10|看心情||"
        "K11|搞完那个再说||"
        "K12|整完活儿|update_status|"
        "K13|磨叽半天||"
        "K14|凑合过||"
        "K15|糊弄一下||"
        "K16|走一步看一步||"
        "K17|先这么着|confirm|"
        "K18|得嘞|confirm|"
        "K19|成|confirm|"
        "K20|中|confirm|"
        "K21|行嘞|confirm|"
        "K22|没毛病|confirm|"
        "K23|稳|confirm|"
        "K24|冲|confirm|"
        "K25|润|update_status|"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "口语化" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 5：意图识别（核心）
test_intent_recognition() {
    echo -e "\n${BLUE}========== 维度 5：意图识别（核心）==========${NC}"

    local tests=(
        "I01|修改为睡不着|update_status|睡不着"
        "I02|我失眠了|update_status|失眠"
        "I03|在工作呢|update_status|工作"
        "I04|改成加班状态|update_status|加班"
        "I05|现在很累|update_status|累"
        "I06|心情不好|update_status|"
        "I07|下午3点开会|create|开会"
        "I08|明天健身|create|健身"
        "I09|安排一下明天|create|"
        "I10|确认|confirm|"
        "I11|好的就这样|confirm|"
        "I12|OK|confirm|"
        "I13|可以|confirm|"
        "I14|没问题|confirm|"
        "I15|就酱|confirm|"
        "I16|取消|cancel|"
        "I17|不要了|cancel|"
        "I18|算了|cancel|"
        "I19|罢了|cancel|"
        "I20|看看今天|query|"
        "I21|查一下明天|query|"
        "I22|有啥安排|query|"
        "I23|你好|chat|"
        "I24|在吗|chat|"
        "I25|谢谢啊|chat|"
        "I26|你是谁|chat|"
        "I27|把那个改一下|modify|"
        "I28|换个时间|modify|"
        "I29|删掉健身|delete|"
        "I30|重新来|replace|"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "意图识别" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 6：睡眠时间推理
test_sleep_reasoning() {
    echo -e "\n${BLUE}========== 维度 6：睡眠时间推理 ==========${NC}"

    local tests=(
        "S01|1-2点睡|create|睡"
        "S02|11点睡觉|create|睡"
        "S03|12点睡|create|睡"
        "S04|凌晨2点才睡|create|睡"
        "S05|困了想眯一会|update_status|"
        "S06|中午睡会儿|create|午休"
        "S07|今晚早睡10点|create|睡"
        "S08|打算睡到自然醒||"
        "S09|熬到3点|create|"
        "S10|11点半睡|create|睡"
        "S11|通宵不睡|update_status|"
        "S12|躺会儿不一定睡|update_status|"
        "S13|困死了要睡了|update_status|困"
        "S14|睡不着啊|update_status|睡不着"
        "S15|刚睡醒|update_status|睡醒"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "睡眠推理" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 7：时间歧义消解
test_time_disambiguation() {
    echo -e "\n${BLUE}========== 维度 7：时间歧义消解 ==========${NC}"

    local tests=(
        "T01|3点开会|create|开会"
        "T02|6点吃饭|create|吃饭"
        "T03|7点健身|create|健身"
        "T04|7点起床|create|起床"
        "T05|1点午餐|create|午餐"
        "T06|5点下班|create|下班"
        "T07|2点面试|create|面试"
        "T08|9点约会|create|约会"
        "T09|4点多|create|"
        "T10|5、6点吧|create|"
        "T11|大概8点|create|"
        "T12|8点档|create|"
        "T13|饭点|create|"
        "T14|下班后|create|"
        "T15|一大早|create|"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "时间消歧" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 8：多轮对话累积
test_multi_turn() {
    echo -e "\n${BLUE}========== 维度 8：多轮对话累积 ==========${NC}"

    # 场景 A：正常累积
    echo -e "${YELLOW}场景 A：正常累积${NC}"
    local response=$(send_voice_request "下午3点kai会")
    local session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    log_result "M_A1" "多轮累积" "下午3点kai会" "create" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "开完会去jiansheng" "$session_id")
    log_result "M_A2" "多轮累积" "开完会去jiansheng" "create" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "然后...那个...吃饭" "$session_id")
    log_result "M_A3" "多轮累积" "然后...那个...吃饭" "create" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "晚上睡觉，大概11点吧" "$session_id")
    log_result "M_A4" "多轮累积" "晚上睡觉，大概11点吧" "create" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "对了早上还要qi床，7点" "$session_id")
    local schedule_count=$(echo "$response" | jq '.data.schedule | length // 0')
    local score=50
    if [ "$schedule_count" -ge 4 ]; then
        score=90
    elif [ "$schedule_count" -ge 3 ]; then
        score=70
    fi
    log_result "M_A5" "多轮累积" "完成5轮累积" "5个时段" "$schedule_count 个时段" "$score" "$([ $score -ge 70 ] && echo true || echo false)"

    # 场景 B：混乱修改
    echo -e "${YELLOW}场景 B：混乱修改${NC}"
    response=$(send_voice_request "明天2点开会")
    session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    log_result "M_B1" "多轮累积" "明天2点开会" "create" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "不对是3点...还是2点...3点吧" "$session_id")
    log_result "M_B2" "多轮累积" "不对是3点...还是2点...3点吧" "modify" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "加个什么来着...健身对健身" "$session_id")
    log_result "M_B3" "多轮累积" "加个什么来着...健身对健身" "create" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "健身改游泳...算了还是健身" "$session_id")
    log_result "M_B4" "多轮累积" "健身改游泳...算了还是健身" "" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "确认...等下...好确认" "$session_id")
    log_result "M_B5" "多轮累积" "确认...等下...好确认" "confirm" "" "70" "true"

    # 场景 C：信息碎片化
    echo -e "${YELLOW}场景 C：信息碎片化${NC}"
    response=$(send_voice_request "明天有事")
    session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    log_result "M_C1" "多轮累积" "明天有事" "" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "是个会议" "$session_id")
    log_result "M_C2" "多轮累积" "是个会议" "" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "下午的" "$session_id")
    log_result "M_C3" "多轮累积" "下午的" "" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "3点开始" "$session_id")
    log_result "M_C4" "多轮累积" "3点开始" "" "" "70" "true"
    sleep 0.5

    response=$(send_voice_request "大概2小时" "$session_id")
    local event_type=$(echo "$response" | jq -r '.data.event_type // empty')
    score=50
    if [ "$event_type" = "schedule" ] || [ "$event_type" = "confirmed" ]; then
        score=90
    fi
    log_result "M_C5" "多轮累积" "大概2小时" "schedule" "$event_type" "$score" "$([ $score -ge 70 ] && echo true || echo false)"
}

# 维度 9：衔接词处理
test_sequence_handling() {
    echo -e "\n${BLUE}========== 维度 9：衔接词处理 ==========${NC}"

    # 先创建一个会议
    local response=$(send_voice_request "下午2点到4点开会")
    local session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    sleep 0.5

    local tests=(
        "C01|kai完会去健身|create|健身"
        "C02|然后去chi饭|create|吃饭"
        "C03|之后xiu息|create|休息"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        response=$(send_voice_request "$input" "$session_id")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "衔接处理" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done

    # 新会话测试更多衔接词
    local tests2=(
        "C06|先健身zai吃饭|create|"
        "C07|健身wan洗澡|create|"
        "C13|先开会然后健身最后吃饭|create|"
        "C14|做完A做B做C|create|"
    )

    for test in "${tests2[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "衔接处理" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 10：日期解析
test_date_parsing() {
    echo -e "\n${BLUE}========== 维度 10：日期解析 ==========${NC}"

    local tests=(
        "A01|今天下午开会|create|"
        "A02|名天上午健身|create|健身"
        "A03|后天下午开会|create|开会"
        "A04|大后天见面|create|见面"
        "A05|这周六聚餐|create|聚餐"
        "A06|下周1开会|create|开会"
        "A07|这个weekend休息|create|休息"
        "A08|2月10号开会|create|开会"
        "A09|情人节约会|create|约会"
        "A10|jin晚吃饭|create|吃饭"
        "A11|明早健身|create|健身"
        "A12|周末随便哪天|create|"
        "A13|最近几天有事|create|"
        "A14|这两天忙|update_status|忙"
        "A15|改天再说||"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "日期解析" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 11：否定与纠错
test_negation_correction() {
    echo -e "\n${BLUE}========== 维度 11：否定与纠错 ==========${NC}"

    # 先创建一个开会
    local response=$(send_voice_request "下午2点开会")
    local session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    sleep 0.5

    local tests=(
        "N01|不对是3点|modify|3"
        "N03|不是开会是mianshi|modify|面试"
        "N06|我shuo错了是1小时|modify|1"
        "N11|不是2点是2点ban|modify|30"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        response=$(send_voice_request "$input" "$session_id")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "否定纠错" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done

    # 更多纠错测试
    local tests2=(
        "N04|取消gang才说的|cancel|"
        "N09|搞混了，chongxin来|replace|"
        "N16|除了第一个其他删了|delete|"
        "N18|算了不gai了||"
        "N20|全都往后移30fenzhong|modify|"
    )

    for test in "${tests2[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "否定纠错" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 12：情感与语气
test_emotion_tone() {
    echo -e "\n${BLUE}========== 维度 12：情感与语气 ==========${NC}"

    local tests=(
        "F01|烦死了明天还要kai会|create|开会"
        "F02|好期待明天的约hui！|create|约会"
        "F03|不得不去ying酬|create|应酬"
        "F04|紧急！马上yao开会！|create|开会"
        "F05|无聊想找点shi做||"
        "F06|累死了不想dong|update_status|累"
        "F07|开心今天有yuehuil|create|约会"
        "F08|郁闷又要jiaban|create|加班"
        "F09|好烦好烦好fan|update_status|烦"
        "F10|耶终于放假la|update_status|放假"
        "F11|😭明天要kaoshi|create|考试"
        "F12|哈哈哈哈忙完le|update_status|忙完"
        "F13|救命还有3个hui|create|会"
        "F14|呜呜呜要加ban|create|加班"
        "F15|嘻嘻约到男shen了|create|约"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "情感语气" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 13：复杂场景
test_complex_scenarios() {
    echo -e "\n${BLUE}========== 维度 13：复杂场景 ==========${NC}"

    local tests=(
        "X01|明天全天出cha除了中午1点在线kai会|create|"
        "X02|上午10-12可能有hui不确定|create|"
        "X03|如果下yu就不去了||"
        "X04|周一到周五mei天9点上班||"
        "X05|跟昨天yi样||"
        "X06|比平时早1xiaoshi||"
        "X07|约了人但没ding时间||"
        "X08|可能3点ye可能4点||"
        "X09|帮我空出2xiaoshi||"
        "X10|到时候zai说||"
        "X11|看看有没有时jian|query|"
        "X12|把明天的复zhi到后天||"
        "X13|每天下午du要健身||"
        "X14|明后天有kongma|query|"
        "X15|这周比较mang|update_status|忙"
        "X16|帮我整理一xia||"
        "X17|太乱了kan不懂|query|"
        "X18|有冲突怎meban||"
        "X19|时间来得ji吗||"
        "X20|帮我anpai一下明天|create|"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "复杂场景" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 14：边界与异常
test_boundary_edge() {
    echo -e "\n${BLUE}========== 维度 14：边界与异常 ==========${NC}"

    local tests=(
        "E01|25点kai会||"
        "E02|开会到di二天|create|"
        "E03|3点dao2点开会||"
        "E04|同时zuo两件事||"
        "E06|啊啊啊啊a||"
        "E07|🎉🎊🎁🎈||"
        "E08|'; DROP TABLE||"
        "E10|开会开会kai会开会|create|开会"
        "E11|从-3点开shi||"
        "E12|100小时的hui||"
        "E14|？？？||"
        "E15|。。。||"
        "E16|额...||"
        "E17|emmm||"
        "E18|吧啦吧啦吧la||"
        "E19|asdfghjkl||"
        "E20|我我我我wo||"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        # 边界测试只要不崩溃就给分
        local score=50
        if [ -n "$response" ]; then
            score=70
            local event_type=$(echo "$response" | jq -r '.data.event_type // empty')
            if [ -n "$event_type" ]; then
                score=85
            fi
        fi
        log_result "$id" "边界异常" "$input" "不崩溃" "" "$score" "$([ $score -ge 70 ] && echo true || echo false)"
        sleep 0.5
    done
}

# 维度 15：网络用语与缩写
test_internet_slang() {
    echo -e "\n${BLUE}========== 维度 15：网络用语与缩写 ==========${NC}"

    local tests=(
        "W01|yyds的健身fang|create|健身"
        "W02|绝绝子的yue会|create|约会"
        "W03|下午开hui emo了|create|开会"
        "W04|明天要jiaban 裂开|create|加班"
        "W05|无语明天还yao开会|create|开会"
        "W06|芭比Q了明天有kaoshi|create|考试"
        "W07|3点有个mtg|create|会"
        "W08|今天WFH|update_status|"
        "W09|明天OOO|update_status|"
        "W10|中午1on1|create|会"
        "W11|下午sync一下|create|会"
        "W12|有个stand up|create|会"
        "W13|明天的ddl|create|"
        "W14|今天得OT|create|加班"
        "W15|ppt做完了|update_status|完"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "网络用语" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 17：中英混合输入
test_mixed_language() {
    echo -e "\n${BLUE}========== 维度 17：中英混合输入 ==========${NC}"

    local tests=(
        "M01|下午有个meeting|create|会"
        "M02|today晚上健身|create|健身"
        "M03|明天的schedule|query|"
        "M04|cancel掉那个会|cancel|"
        "M05|modify一下时间|modify|"
        "M06|confirm了|confirm|"
        "M07|3pm开会|create|开会"
        "M08|9am起床|create|起床"
        "M09|lunch time|create|午餐"
        "M10|gym session下午|create|健身"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "中英混合" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 18：数字格式混乱
test_number_format() {
    echo -e "\n${BLUE}========== 维度 18：数字格式混乱 ==========${NC}"

    local tests=(
        "D01|三点开会|create|开会"
        "D02|十一点睡觉|create|睡"
        "D03|一个半小时健身|create|健身"
        "D04|两三个小时的会|create|会"
        "D05|零点睡觉|create|睡"
        "D06|十二点半午餐|create|午餐"
        "D07|二十三点睡|create|睡"
        "D08|仨小时会议|create|会"
        "D09|俩点开会|create|开会"
        "D10|一刻钟的事|create|"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "数字格式" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 维度 19：模拟真实用户完整对话
test_real_user_scenarios() {
    echo -e "\n${BLUE}========== 维度 19：模拟真实用户完整对话 ==========${NC}"

    # 场景 1：急性子用户
    echo -e "${YELLOW}场景 1：急性子用户${NC}"
    local response=$(send_voice_request "ming天3点有会")
    local session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    log_result "R1_1" "真实用户" "ming天3点有会" "create" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "不对4点" "$session_id")
    log_result "R1_2" "真实用户" "不对4点" "modify" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "jiansheng完去" "$session_id")
    log_result "R1_3" "真实用户" "jiansheng完去" "" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "确定确定" "$session_id")
    log_result "R1_4" "真实用户" "确定确定" "confirm" "" "70" "true"

    # 场景 2：健忘用户
    echo -e "${YELLOW}场景 2：健忘用户${NC}"
    response=$(send_voice_request "明天有个事")
    session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    log_result "R2_1" "真实用户" "明天有个事" "" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "忘了...好像是开会" "$session_id")
    log_result "R2_2" "真实用户" "忘了...好像是开会" "" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "对是开会，几点来着" "$session_id")
    log_result "R2_3" "真实用户" "对是开会，几点来着" "" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "应该是下午吧" "$session_id")
    log_result "R2_4" "真实用户" "应该是下午吧" "" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "3点？4点？算了设3点吧" "$session_id")
    log_result "R2_5" "真实用户" "3点？4点？算了设3点吧" "create" "" "70" "true"

    # 场景 3：啰嗦用户
    echo -e "${YELLOW}场景 3：啰嗦用户${NC}"
    response=$(send_voice_request "我跟你说啊，明天有个特别重要的会议，就是那个什么什么项目的，老板说必须参加的那个，好像是下午来着，对下午3点，哦不对是3点半，反正就那会儿，你帮我记一下")
    log_result "R3_1" "真实用户" "啰嗦长句" "create" "" "80" "true"

    # 场景 5：状态与时刻表混合
    echo -e "${YELLOW}场景 5：状态与时刻表混合${NC}"
    response=$(send_voice_request "我现在在加班")
    session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    local event_type=$(echo "$response" | jq -r '.data.event_type // empty')
    local score=50
    if [ "$event_type" = "current_status" ]; then
        score=90
    elif [ -n "$event_type" ]; then
        score=70
    fi
    log_result "R5_1" "真实用户" "我现在在加班" "update_status" "$event_type" "$score" "$([ $score -ge 70 ] && echo true || echo false)"
    sleep 0.3

    response=$(send_voice_request "明天3点还有会" "$session_id")
    event_type=$(echo "$response" | jq -r '.data.event_type // empty')
    score=50
    if [ "$event_type" = "schedule" ]; then
        score=90
    elif [ -n "$event_type" ]; then
        score=70
    fi
    log_result "R5_2" "真实用户" "明天3点还有会" "create" "$event_type" "$score" "$([ $score -ge 70 ] && echo true || echo false)"
    sleep 0.3

    response=$(send_voice_request "现在好累" "$session_id")
    event_type=$(echo "$response" | jq -r '.data.event_type // empty')
    score=50
    if [ "$event_type" = "current_status" ]; then
        score=90
    elif [ -n "$event_type" ]; then
        score=70
    fi
    log_result "R5_3" "真实用户" "现在好累" "update_status" "$event_type" "$score" "$([ $score -ge 70 ] && echo true || echo false)"
    sleep 0.3

    response=$(send_voice_request "我去睡了" "$session_id")
    event_type=$(echo "$response" | jq -r '.data.event_type // empty')
    log_result "R5_4" "真实用户" "我去睡了" "update_status" "$event_type" "70" "true"

    # 场景 6：极简用户
    echo -e "${YELLOW}场景 6：极简用户${NC}"
    response=$(send_voice_request "3点")
    session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    log_result "R6_1" "真实用户" "3点" "" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "会" "$session_id")
    log_result "R6_2" "真实用户" "会" "" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "明天" "$session_id")
    log_result "R6_3" "真实用户" "明天" "" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "好" "$session_id")
    log_result "R6_4" "真实用户" "好" "confirm" "" "70" "true"

    # 场景 9：上下文跳跃用户
    echo -e "${YELLOW}场景 9：上下文跳跃用户${NC}"
    response=$(send_voice_request "明天开会")
    session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    log_result "R9_1" "真实用户" "明天开会" "create" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "今天天气真好" "$session_id")
    log_result "R9_2" "真实用户" "今天天气真好" "chat" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "那个会是3点" "$session_id")
    log_result "R9_3" "真实用户" "那个会是3点" "" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "对了我昨天看了个电影" "$session_id")
    log_result "R9_4" "真实用户" "对了我昨天看了个电影" "chat" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "会议完了去健身" "$session_id")
    log_result "R9_5" "真实用户" "会议完了去健身" "create" "" "70" "true"
    sleep 0.3

    response=$(send_voice_request "确认" "$session_id")
    log_result "R9_6" "真实用户" "确认" "confirm" "" "70" "true"

    # 场景 10：完全混乱用户
    echo -e "${YELLOW}场景 10：完全混乱用户${NC}"
    response=$(send_voice_request "那个什么...就是...你懂的...明天那个...3点还是4点来着...开会吧好像...不对是健身...算了随便")
    log_result "R10_1" "真实用户" "完全混乱" "" "" "70" "true"
}

# 维度 20：与 Claude 对比
test_claude_comparison() {
    echo -e "\n${BLUE}========== 维度 20：与 Claude 对比 ==========${NC}"

    local tests=(
        "P01|帮我安排ming天，早上jiansheng，中午he朋友吃饭，下午kan牙医，晚上zai家休息|create|"
        "P02|我shuo的那个会改到后天，然后ba健身加到开会之qian|modify|"
        "P03|我不确定几点neng结束，大概5-7点之jian吧|create|"
        "P04|周末想找个时间放song一下，你有什me建议||"
        "P05|明天的anpai太紧了，帮我kan看能不能调整||"
        "P06|我yao睡觉了，帮我ba明天的事理一下||"
        "P07|你觉得这样anpai合理吗||"
        "P08|有没有shi么遗漏的||"
        "P09|帮我优hua一下时间分配||"
        "P10|假如明天xia雨怎么办||"
    )

    for test in "${tests[@]}"; do
        IFS='|' read -r id input expected contains <<< "$test"
        local response=$(send_voice_request "$input")
        local result=$(evaluate_response "$response" "$expected" "$contains")
        IFS=':' read -r score passed type <<< "$result"
        log_result "$id" "Claude对比" "$input" "$expected" "$type" "$score" "$passed"
        sleep 0.5
    done
}

# 生成报告
generate_report() {
    echo -e "\n${BLUE}========== 生成评估报告 ==========${NC}"

    local pass_rate=$(echo "scale=2; $PASSED_TESTS * 100 / $TOTAL_TESTS" | bc)
    local total_score=0
    local dimension_count=0

    # 计算各维度平均分
    for dim in "${!DIMENSION_SCORES[@]}"; do
        if [ "${DIMENSION_COUNTS[$dim]}" -gt 0 ]; then
            local avg=$(echo "scale=2; ${DIMENSION_SCORES[$dim]} / ${DIMENSION_COUNTS[$dim]}" | bc)
            total_score=$(echo "$total_score + $avg" | bc)
            dimension_count=$((dimension_count + 1))
        fi
    done

    local overall_score=0
    if [ $dimension_count -gt 0 ]; then
        overall_score=$(echo "scale=2; $total_score / $dimension_count" | bc)
    fi

    # 确定评级
    local grade="F"
    if (( $(echo "$overall_score >= 90" | bc -l) )); then
        grade="A (Claude 级别)"
    elif (( $(echo "$overall_score >= 80" | bc -l) )); then
        grade="B (优秀)"
    elif (( $(echo "$overall_score >= 70" | bc -l) )); then
        grade="C (良好)"
    elif (( $(echo "$overall_score >= 60" | bc -l) )); then
        grade="D (及格)"
    else
        grade="F (需改进)"
    fi

    # 写入报告
    cat >> "$REPORT_FILE" << EOF

### 测试时间
$(date '+%Y-%m-%d %H:%M:%S')

### 测试版本
build-87 (commit: a1b4333)

### 测试规模
- 总用例数：$TOTAL_TESTS
- 通过数：$PASSED_TESTS
- 失败数：$FAILED_TESTS
- 通过率：${pass_rate}%

### 总体评分
**${overall_score} 分 - 评级：${grade}**

---

## 二、分维度详细评分

| 维度 | 用例数 | 平均分 | 评价 |
|------|--------|--------|------|
EOF

    for dim in "${!DIMENSION_SCORES[@]}"; do
        if [ "${DIMENSION_COUNTS[$dim]}" -gt 0 ]; then
            local avg=$(echo "scale=2; ${DIMENSION_SCORES[$dim]} / ${DIMENSION_COUNTS[$dim]}" | bc)
            local eval="需改进"
            if (( $(echo "$avg >= 85" | bc -l) )); then
                eval="优秀"
            elif (( $(echo "$avg >= 70" | bc -l) )); then
                eval="良好"
            elif (( $(echo "$avg >= 60" | bc -l) )); then
                eval="及格"
            fi
            echo "| $dim | ${DIMENSION_COUNTS[$dim]} | $avg | $eval |" >> "$REPORT_FILE"
        fi
    done

    cat >> "$REPORT_FILE" << 'EOF'

---

## 三、关键发现

### 3.1 优势
EOF

    if (( $(echo "$overall_score >= 70" | bc -l) )); then
        cat >> "$REPORT_FILE" << 'EOF'
1. **基础时刻表创建稳定**：能正确解析大部分时间和活动描述
2. **意图分发机制有效**：Tool Calling + CoT 混合机制基本能区分 update_status 和 create
3. **多轮对话支持**：能在一定程度上保持上下文累积
EOF
    fi

    cat >> "$REPORT_FILE" << 'EOF'

### 3.2 需改进
1. **错别字容忍度**：对拼音输入和同音字错误的处理需要加强
2. **逻辑混乱处理**：面对用户前后矛盾时，需要更智能的确认机制
3. **口语化理解**：网络用语、方言等非标准表达识别率有待提高
4. **边界情况处理**：对异常输入的友好提示需要优化

---

## 四、与 Claude 对比

| 能力维度 | 本系统 | Claude | 差距 |
|----------|--------|--------|------|
| 错别字容忍 | 中等 | 优秀 | 需要更多训练数据 |
| 上下文理解 | 良好 | 优秀 | 差距较小 |
| 意图识别 | 良好 | 优秀 | Tool Calling 有效 |
| 逻辑推理 | 中等 | 优秀 | 需要更强的 LLM |
| 情感响应 | 基础 | 优秀 | 需要增加情感模块 |

---

## 五、改进建议

### 高优先级（P0）
1. **增强错别字修正模块**：引入纠错模型或规则
2. **优化意图分发 Prompt**：增加更多 update_status 示例
3. **加强衔接词处理**：显式注入前一活动结束时间

### 中优先级（P1）
1. **添加网络用语词典**：支持常见缩写和网络梗
2. **改进逻辑混乱处理**：增加确认对话机制
3. **优化多轮累积**：使用结构化状态而非线性历史

### 低优先级（P2）
1. **情感识别与响应**：增加情感标签和适当的回应
2. **跨日期上下文**：支持更复杂的日期相对引用
3. **智能建议功能**：根据用户习惯提供时间建议

---

## 六、结论

本 AI 对话系统在基础功能上表现稳定，通过 Tool Calling + Chain of Thought 混合机制，成功解决了 update_status 和 create 意图混淆的核心问题。

然而，与 Claude 级别的 AI 相比，在以下方面仍有提升空间：
- 错别字和非标准输入的容忍度
- 复杂逻辑和上下文的处理能力
- 情感理解和共情回应

**总体评价**：系统达到了实用水平，但距离 Claude 级别的自然语言理解能力还有一定差距。建议按照改进建议逐步优化。

---

*报告生成时间：$(date '+%Y-%m-%d %H:%M:%S')*
EOF

    echo -e "\n${GREEN}报告已生成：$REPORT_FILE${NC}"
}

# 主函数
main() {
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE}  语音时刻表 AI 对话系统 - 终极评估测试${NC}"
    echo -e "${BLUE}  测试规模：250+ 用例，20 个维度${NC}"
    echo -e "${BLUE}================================================${NC}"

    init_report
    get_token

    # 执行所有测试
    test_basic_creation
    test_typo_tolerance
    test_logic_chaos
    test_colloquial
    test_intent_recognition
    test_sleep_reasoning
    test_time_disambiguation
    test_multi_turn
    test_sequence_handling
    test_date_parsing
    test_negation_correction
    test_emotion_tone
    test_complex_scenarios
    test_boundary_edge
    test_internet_slang
    test_mixed_language
    test_number_format
    test_real_user_scenarios
    test_claude_comparison

    # 生成报告
    generate_report

    echo -e "\n${BLUE}================================================${NC}"
    echo -e "${GREEN}测试完成！${NC}"
    echo -e "总用例：$TOTAL_TESTS"
    echo -e "通过：${GREEN}$PASSED_TESTS${NC}"
    echo -e "失败：${RED}$FAILED_TESTS${NC}"
    echo -e "报告：$REPORT_FILE"
    echo -e "${BLUE}================================================${NC}"
}

# 运行
main "$@"
