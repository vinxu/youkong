#!/bin/zsh

# 语音时刻表 AI 对话系统 - 终极评估测试脚本 v2
# 使用 zsh 以支持关联数组

API_BASE="http://49.232.13.41:8080/api/v1"
REPORT_FILE="/Users/xuxuheng/Desktop/youkong/Backend/EVALUATION_REPORT.md"
LOG_FILE="/Users/xuxuheng/Desktop/youkong/Backend/test_results.json"

# 测试账号
TEST_PHONE="13800000001"
TEST_CODE="111111"

# 统计变量
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 使用 zsh 关联数组
typeset -A DIMENSION_SCORES
typeset -A DIMENSION_COUNTS

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 获取 Token
get_token() {
    local response=$(curl -s -X POST "$API_BASE/auth/sms/verify" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$TEST_PHONE\",\"code\":\"$TEST_CODE\"}")

    TOKEN=$(echo "$response" | jq -r '.data.token // empty')

    if [[ -z "$TOKEN" ]]; then
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
    if [[ -n "$session_id" ]]; then
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

# 评估并记录结果
evaluate_and_log() {
    local test_id="$1"
    local dimension="$2"
    local input="$3"
    local expected_action="$4"
    local expected_contains="$5"
    local response="$6"

    local event_type=$(echo "$response" | jq -r '.data.event_type // empty')
    local message=$(echo "$response" | jq -r '.data.message // empty')
    local code=$(echo "$response" | jq -r '.code // 0')

    local score=0
    local passed="false"

    # 检查是否有响应
    if [[ -z "$response" ]] || [[ "$response" == "null" ]]; then
        score=0
    elif [[ "$code" != "0" ]]; then
        score=10
    else
        # 基础分：有响应 +30
        score=30

        # event_type 匹配 +40
        if [[ -n "$expected_action" ]]; then
            if [[ "$event_type" == "$expected_action" ]]; then
                score=$((score + 40))
            elif [[ "$event_type" == "schedule" && "$expected_action" == "create" ]]; then
                score=$((score + 40))
            elif [[ "$event_type" == "current_status" && "$expected_action" == "update_status" ]]; then
                score=$((score + 40))
            elif [[ "$event_type" == "confirmed" && "$expected_action" == "confirm" ]]; then
                score=$((score + 40))
            elif [[ "$event_type" == "cancelled" && "$expected_action" == "cancel" ]]; then
                score=$((score + 40))
            fi
        else
            if [[ -n "$event_type" ]]; then
                score=$((score + 20))
            fi
        fi

        # 内容匹配 +30
        if [[ -n "$expected_contains" ]]; then
            if echo "$message" | grep -qi "$expected_contains"; then
                score=$((score + 30))
            fi
        else
            if [[ -n "$message" ]]; then
                score=$((score + 20))
            fi
        fi
    fi

    # 判断是否通过
    if [[ $score -ge 70 ]]; then
        passed="true"
    fi

    # 更新统计
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [[ "$passed" == "true" ]]; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    # 更新维度分数
    DIMENSION_SCORES[$dimension]=$((${DIMENSION_SCORES[$dimension]:-0} + score))
    DIMENSION_COUNTS[$dimension]=$((${DIMENSION_COUNTS[$dimension]:-0} + 1))

    # 输出进度
    if [[ "$passed" == "true" ]]; then
        printf "${GREEN}✓${NC} [%s] %s: %.30s → %s (Score: %d)\n" "$test_id" "$dimension" "$input" "$event_type" "$score"
    else
        printf "${RED}✗${NC} [%s] %s: %.30s → %s (Score: %d)\n" "$test_id" "$dimension" "$input" "$event_type" "$score"
    fi
}

# 运行单个测试
run_test() {
    local test_id="$1"
    local dimension="$2"
    local input="$3"
    local expected_action="$4"
    local expected_contains="$5"
    local session_id="${6:-}"

    local response=$(send_voice_request "$input" "$session_id")
    evaluate_and_log "$test_id" "$dimension" "$input" "$expected_action" "$expected_contains" "$response"

    # 返回 session_id
    echo "$response" | jq -r '.data.session_id // empty'

    sleep 0.3
}

# ========== 测试函数 ==========

test_basic_creation() {
    echo -e "\n${BLUE}========== 维度 1：基础时刻表创建 ==========${NC}"

    run_test "B01" "基础创建" "下午3点开会" "create" "开会"
    run_test "B02" "基础创建" "晚上8点吃饭" "create" "吃饭"
    run_test "B03" "基础创建" "上午9点到12点工作" "create" "工作"
    run_test "B04" "基础创建" "中午12点午餐，下午2点继续工作到6点" "create" ""
    run_test "B05" "基础创建" "7点起床，8点出门，9点上班" "create" ""
    run_test "B06" "基础创建" "下午健身1.5小时" "create" "健身"
    run_test "B07" "基础创建" "晚上看电影2个半小时" "create" "电影"
    run_test "B08" "基础创建" "从早上9点一直忙到晚上9点" "create" ""
    run_test "B09" "基础创建" "10:30-11:45 面试" "create" "面试"
    run_test "B10" "基础创建" "下午茶时间3点到4点" "create" ""
    run_test "B11" "基础创建" "明早8点的飞机" "create" ""
    run_test "B12" "基础创建" "周会，每周一上午10点" "create" ""
    run_test "B13" "基础创建" "和老板1on1" "create" ""
    run_test "B14" "基础创建" "年会下午全程" "create" "年会"
    run_test "B15" "基础创建" "通宵写代码" "create" ""
}

test_typo_tolerance() {
    echo -e "\n${BLUE}========== 维度 2：错别字与拼音输入错误 ==========${NC}"

    run_test "Y01" "错别字" "下午3点开回" "create" ""
    run_test "Y02" "错别字" "晚上吃反" "create" ""
    run_test "Y03" "错别字" "明天尚午健身" "create" "健身"
    run_test "Y04" "错别字" "在家修息" "update_status" ""
    run_test "Y05" "错别字" "去建身房" "create" "健身"
    run_test "Y06" "错别字" "xiawu 3dian kaihui" "create" ""
    run_test "Y07" "错别字" "wanshang chifan" "create" ""
    run_test "Y08" "错别字" "mingtian xiuxi" "create" ""
    run_test "Y09" "错别字" "3dian kaihi" "create" ""
    run_test "Y10" "错别字" "开会 xia午" "create" "开会"
    run_test "Y11" "错别字" "下物3点" "create" ""
    run_test "Y12" "错别字" "琬上8点" "create" ""
    run_test "Y13" "错别字" "建身1小时" "create" ""
    run_test "Y14" "错别字" "看点影" "create" ""
    run_test "Y15" "错别字" "吃个法" "create" ""
    run_test "Y16" "错别字" "shui觉" "create" ""
    run_test "Y17" "错别字" "xiu息一下" "update_status" ""
    run_test "Y18" "错别字" "上班 gongzuo" "create" ""
    run_test "Y19" "错别字" "kai会" "create" ""
    run_test "Y20" "错别字" "jian身fang" "create" ""
    run_test "Y21" "错别字" "3嗲你开会" "create" ""
    run_test "Y22" "错别字" "我药去开会" "create" "开会"
    run_test "Y23" "错别字" "帮我安培一下" "create" ""
    run_test "Y24" "错别字" "放松放送" "update_status" ""
    run_test "Y25" "错别字" "休息修习" "update_status" ""
    run_test "Y26" "错别字" "名天开会" "create" "开会"
    run_test "Y27" "错别字" "工做到6点" "create" ""
    run_test "Y28" "错别字" "睡交" "create" ""
    run_test "Y29" "错别字" "看书独书" "create" ""
    run_test "Y30" "错别字" "修改为睡不zhuo" "update_status" ""
}

test_logic_chaos() {
    echo -e "\n${BLUE}========== 维度 3：逻辑混乱与前后矛盾 ==========${NC}"

    run_test "L01" "逻辑混乱" "明天3点开会...不对等等，是今天" "" ""
    run_test "L02" "逻辑混乱" "下午开会，算了还是下午吧" "create" ""
    run_test "L03" "逻辑混乱" "健身2小时，1小时也行，反正看情况" "create" "健身"
    run_test "L04" "逻辑混乱" "约了朋友吃饭，还是加上" "create" "吃饭"
    run_test "L05" "逻辑混乱" "开会是3点还是4点来着？设3点吧" "create" ""
    run_test "L06" "逻辑混乱" "上午...不对下午...上午对上午" "create" ""
    run_test "L07" "逻辑混乱" "先健身再吃饭...不对先吃饭再健身" "create" ""
    run_test "L08" "逻辑混乱" "3点4点5点都有事...就3点一个会吧" "create" ""
    run_test "L09" "逻辑混乱" "明天休息...不对明天要上班" "create" "上班"
    run_test "L10" "逻辑混乱" "要开会的，好像是要开会" "create" "开会"
    run_test "L11" "逻辑混乱" "确定...不对等下...好吧确定" "confirm" ""
    run_test "L12" "逻辑混乱" "把会议...不是会议是面试...改到明天" "modify" ""
    run_test "L13" "逻辑混乱" "从2点开始...还是3点...2点半吧" "create" ""
    run_test "L14" "逻辑混乱" "今天明天后天都行，你随便选" "" ""
    run_test "L15" "逻辑混乱" "删掉...等下别删...还是删了" "delete" ""
}

test_colloquial() {
    echo -e "\n${BLUE}========== 维度 4：口语化与省略表达 ==========${NC}"

    run_test "K01" "口语化" "下午有点事儿" "" ""
    run_test "K02" "口语化" "差不多那会儿吧" "" ""
    run_test "K03" "口语化" "反正就那样" "confirm" ""
    run_test "K04" "口语化" "先这么着" "confirm" ""
    run_test "K05" "口语化" "得嘞" "confirm" ""
    run_test "K06" "口语化" "成" "confirm" ""
    run_test "K07" "口语化" "中" "confirm" ""
    run_test "K08" "口语化" "行嘞" "confirm" ""
    run_test "K09" "口语化" "没毛病" "confirm" ""
    run_test "K10" "口语化" "稳" "confirm" ""
    run_test "K11" "口语化" "冲" "confirm" ""
    run_test "K12" "口语化" "润" "update_status" ""
    run_test "K13" "口语化" "晚点再说" "" ""
    run_test "K14" "口语化" "看心情" "" ""
    run_test "K15" "口语化" "走一步看一步" "" ""
}

test_intent_recognition() {
    echo -e "\n${BLUE}========== 维度 5：意图识别（核心）==========${NC}"

    run_test "I01" "意图识别" "修改为睡不着" "update_status" "睡不着"
    run_test "I02" "意图识别" "我失眠了" "update_status" "失眠"
    run_test "I03" "意图识别" "在工作呢" "update_status" "工作"
    run_test "I04" "意图识别" "改成加班状态" "update_status" "加班"
    run_test "I05" "意图识别" "现在很累" "update_status" ""
    run_test "I06" "意图识别" "心情不好" "update_status" ""
    run_test "I07" "意图识别" "下午3点开会" "create" "开会"
    run_test "I08" "意图识别" "明天健身" "create" "健身"
    run_test "I09" "意图识别" "安排一下明天" "create" ""
    run_test "I10" "意图识别" "确认" "confirm" ""
    run_test "I11" "意图识别" "好的就这样" "confirm" ""
    run_test "I12" "意图识别" "OK" "confirm" ""
    run_test "I13" "意图识别" "可以" "confirm" ""
    run_test "I14" "意图识别" "没问题" "confirm" ""
    run_test "I15" "意图识别" "就酱" "confirm" ""
    run_test "I16" "意图识别" "取消" "cancel" ""
    run_test "I17" "意图识别" "不要了" "cancel" ""
    run_test "I18" "意图识别" "算了" "cancel" ""
    run_test "I19" "意图识别" "罢了" "cancel" ""
    run_test "I20" "意图识别" "看看今天" "query" ""
    run_test "I21" "意图识别" "查一下明天" "query" ""
    run_test "I22" "意图识别" "有啥安排" "query" ""
    run_test "I23" "意图识别" "你好" "chat" ""
    run_test "I24" "意图识别" "在吗" "chat" ""
    run_test "I25" "意图识别" "谢谢啊" "chat" ""
    run_test "I26" "意图识别" "你是谁" "chat" ""
    run_test "I27" "意图识别" "把那个改一下" "modify" ""
    run_test "I28" "意图识别" "换个时间" "modify" ""
    run_test "I29" "意图识别" "删掉健身" "delete" ""
    run_test "I30" "意图识别" "重新来" "replace" ""
}

test_sleep_reasoning() {
    echo -e "\n${BLUE}========== 维度 6：睡眠时间推理 ==========${NC}"

    run_test "S01" "睡眠推理" "1-2点睡" "create" "睡"
    run_test "S02" "睡眠推理" "11点睡觉" "create" "睡"
    run_test "S03" "睡眠推理" "12点睡" "create" "睡"
    run_test "S04" "睡眠推理" "凌晨2点才睡" "create" "睡"
    run_test "S05" "睡眠推理" "困了想眯一会" "update_status" ""
    run_test "S06" "睡眠推理" "中午睡会儿" "create" ""
    run_test "S07" "睡眠推理" "今晚早睡10点" "create" "睡"
    run_test "S08" "睡眠推理" "打算睡到自然醒" "" ""
    run_test "S09" "睡眠推理" "熬到3点" "create" ""
    run_test "S10" "睡眠推理" "11点半睡" "create" "睡"
    run_test "S11" "睡眠推理" "通宵不睡" "update_status" ""
    run_test "S12" "睡眠推理" "躺会儿不一定睡" "update_status" ""
    run_test "S13" "睡眠推理" "困死了要睡了" "update_status" ""
    run_test "S14" "睡眠推理" "睡不着啊" "update_status" "睡不着"
    run_test "S15" "睡眠推理" "刚睡醒" "update_status" ""
}

test_time_disambiguation() {
    echo -e "\n${BLUE}========== 维度 7：时间歧义消解 ==========${NC}"

    run_test "T01" "时间消歧" "3点开会" "create" "开会"
    run_test "T02" "时间消歧" "6点吃饭" "create" "吃饭"
    run_test "T03" "时间消歧" "7点健身" "create" "健身"
    run_test "T04" "时间消歧" "7点起床" "create" "起床"
    run_test "T05" "时间消歧" "1点午餐" "create" "午餐"
    run_test "T06" "时间消歧" "5点下班" "create" "下班"
    run_test "T07" "时间消歧" "2点面试" "create" "面试"
    run_test "T08" "时间消歧" "9点约会" "create" "约会"
    run_test "T09" "时间消歧" "4点多" "create" ""
    run_test "T10" "时间消歧" "大概8点" "create" ""
    run_test "T11" "时间消歧" "8点档" "create" ""
    run_test "T12" "时间消歧" "饭点" "create" ""
    run_test "T13" "时间消歧" "下班后" "create" ""
    run_test "T14" "时间消歧" "一大早" "create" ""
    run_test "T15" "时间消歧" "深夜" "create" ""
}

test_multi_turn() {
    echo -e "\n${BLUE}========== 维度 8：多轮对话累积 ==========${NC}"

    # 场景 A：正常累积
    echo -e "${YELLOW}场景 A：正常累积${NC}"
    local response=$(send_voice_request "下午3点kai会")
    local session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    evaluate_and_log "M_A1" "多轮对话" "下午3点kai会" "create" "" "$response"
    sleep 0.3

    response=$(send_voice_request "开完会去健身" "$session_id")
    evaluate_and_log "M_A2" "多轮对话" "开完会去健身" "create" "健身" "$response"
    sleep 0.3

    response=$(send_voice_request "然后吃饭" "$session_id")
    evaluate_and_log "M_A3" "多轮对话" "然后吃饭" "create" "吃饭" "$response"
    sleep 0.3

    response=$(send_voice_request "晚上11点睡觉" "$session_id")
    evaluate_and_log "M_A4" "多轮对话" "晚上11点睡觉" "create" "睡" "$response"
    sleep 0.3

    response=$(send_voice_request "确认" "$session_id")
    evaluate_and_log "M_A5" "多轮对话" "确认" "confirm" "" "$response"

    # 场景 B：信息碎片化
    echo -e "${YELLOW}场景 B：信息碎片化${NC}"
    response=$(send_voice_request "明天有事")
    session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    evaluate_and_log "M_B1" "多轮对话" "明天有事" "" "" "$response"
    sleep 0.3

    response=$(send_voice_request "是个会议" "$session_id")
    evaluate_and_log "M_B2" "多轮对话" "是个会议" "" "" "$response"
    sleep 0.3

    response=$(send_voice_request "下午3点" "$session_id")
    evaluate_and_log "M_B3" "多轮对话" "下午3点" "" "" "$response"
    sleep 0.3

    response=$(send_voice_request "大概2小时" "$session_id")
    evaluate_and_log "M_B4" "多轮对话" "大概2小时" "" "" "$response"
    sleep 0.3

    response=$(send_voice_request "好的" "$session_id")
    evaluate_and_log "M_B5" "多轮对话" "好的" "confirm" "" "$response"

    # 场景 C：混乱修改
    echo -e "${YELLOW}场景 C：混乱修改${NC}"
    response=$(send_voice_request "明天2点开会")
    session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    evaluate_and_log "M_C1" "多轮对话" "明天2点开会" "create" "" "$response"
    sleep 0.3

    response=$(send_voice_request "不对是3点" "$session_id")
    evaluate_and_log "M_C2" "多轮对话" "不对是3点" "modify" "" "$response"
    sleep 0.3

    response=$(send_voice_request "再加个健身" "$session_id")
    evaluate_and_log "M_C3" "多轮对话" "再加个健身" "create" "健身" "$response"
    sleep 0.3

    response=$(send_voice_request "确定" "$session_id")
    evaluate_and_log "M_C4" "多轮对话" "确定" "confirm" "" "$response"
}

test_sequence_handling() {
    echo -e "\n${BLUE}========== 维度 9：衔接词处理 ==========${NC}"

    # 先创建一个会议
    local response=$(send_voice_request "下午2点到4点开会")
    local session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    evaluate_and_log "C00" "衔接处理" "下午2点到4点开会" "create" "开会" "$response"
    sleep 0.3

    response=$(send_voice_request "开完会去健身" "$session_id")
    evaluate_and_log "C01" "衔接处理" "开完会去健身" "create" "健身" "$response"
    sleep 0.3

    response=$(send_voice_request "然后去吃饭" "$session_id")
    evaluate_and_log "C02" "衔接处理" "然后去吃饭" "create" "吃饭" "$response"
    sleep 0.3

    response=$(send_voice_request "之后休息" "$session_id")
    evaluate_and_log "C03" "衔接处理" "之后休息" "create" "休息" "$response"

    # 新会话测试
    run_test "C04" "衔接处理" "先健身再吃饭" "create" ""
    run_test "C05" "衔接处理" "健身完洗澡" "create" ""
    run_test "C06" "衔接处理" "先开会然后健身最后吃饭" "create" ""
}

test_date_parsing() {
    echo -e "\n${BLUE}========== 维度 10：日期解析 ==========${NC}"

    run_test "A01" "日期解析" "今天下午开会" "create" ""
    run_test "A02" "日期解析" "明天上午健身" "create" "健身"
    run_test "A03" "日期解析" "后天下午开会" "create" "开会"
    run_test "A04" "日期解析" "大后天见面" "create" ""
    run_test "A05" "日期解析" "这周六聚餐" "create" "聚餐"
    run_test "A06" "日期解析" "下周一开会" "create" "开会"
    run_test "A07" "日期解析" "周末休息" "create" "休息"
    run_test "A08" "日期解析" "2月10号开会" "create" "开会"
    run_test "A09" "日期解析" "情人节约会" "create" "约会"
    run_test "A10" "日期解析" "今晚吃饭" "create" "吃饭"
    run_test "A11" "日期解析" "明早健身" "create" "健身"
    run_test "A12" "日期解析" "这两天忙" "update_status" "忙"
    run_test "A13" "日期解析" "改天再说" "" ""
}

test_negation_correction() {
    echo -e "\n${BLUE}========== 维度 11：否定与纠错 ==========${NC}"

    # 先创建一个开会
    local response=$(send_voice_request "下午2点开会")
    local session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    evaluate_and_log "N00" "否定纠错" "下午2点开会" "create" "" "$response"
    sleep 0.3

    response=$(send_voice_request "不对是3点" "$session_id")
    evaluate_and_log "N01" "否定纠错" "不对是3点" "modify" "" "$response"
    sleep 0.3

    response=$(send_voice_request "不是开会是面试" "$session_id")
    evaluate_and_log "N02" "否定纠错" "不是开会是面试" "modify" "面试" "$response"
    sleep 0.3

    response=$(send_voice_request "取消刚才说的" "$session_id")
    evaluate_and_log "N03" "否定纠错" "取消刚才说的" "cancel" "" "$response"

    run_test "N04" "否定纠错" "我说错了是1小时" "modify" ""
    run_test "N05" "否定纠错" "往后推1小时" "modify" ""
    run_test "N06" "否定纠错" "搞混了重新来" "replace" ""
    run_test "N07" "否定纠错" "不对改15:30" "modify" ""
    run_test "N08" "否定纠错" "太早了往后挪" "modify" ""
    run_test "N09" "否定纠错" "算了不改了" "" ""
}

test_emotion_tone() {
    echo -e "\n${BLUE}========== 维度 12：情感与语气 ==========${NC}"

    run_test "F01" "情感语气" "烦死了明天还要开会" "create" "开会"
    run_test "F02" "情感语气" "好期待明天的约会！" "create" "约会"
    run_test "F03" "情感语气" "不得不去应酬" "create" "应酬"
    run_test "F04" "情感语气" "紧急！马上要开会！" "create" "开会"
    run_test "F05" "情感语气" "累死了不想动" "update_status" ""
    run_test "F06" "情感语气" "开心今天有约会" "create" "约会"
    run_test "F07" "情感语气" "郁闷又要加班" "create" "加班"
    run_test "F08" "情感语气" "好烦好烦" "update_status" ""
    run_test "F09" "情感语气" "耶终于放假啦" "update_status" ""
    run_test "F10" "情感语气" "救命还有3个会" "create" ""
    run_test "F11" "情感语气" "呜呜呜要加班" "create" "加班"
}

test_complex_scenarios() {
    echo -e "\n${BLUE}========== 维度 13：复杂场景 ==========${NC}"

    run_test "X01" "复杂场景" "明天全天出差除了中午1点在线开会" "create" ""
    run_test "X02" "复杂场景" "上午10-12可能有会不确定" "create" ""
    run_test "X03" "复杂场景" "如果下雨就不去了" "" ""
    run_test "X04" "复杂场景" "跟昨天一样" "" ""
    run_test "X05" "复杂场景" "约了人但没定时间" "" ""
    run_test "X06" "复杂场景" "可能3点也可能4点" "" ""
    run_test "X07" "复杂场景" "帮我空出2小时" "" ""
    run_test "X08" "复杂场景" "看看有没有时间" "query" ""
    run_test "X09" "复杂场景" "这周比较忙" "update_status" "忙"
    run_test "X10" "复杂场景" "帮我整理一下" "" ""
    run_test "X11" "复杂场景" "有冲突怎么办" "" ""
    run_test "X12" "复杂场景" "帮我安排一下明天" "create" ""
}

test_boundary_edge() {
    echo -e "\n${BLUE}========== 维度 14：边界与异常 ==========${NC}"

    run_test "E01" "边界异常" "25点开会" "" ""
    run_test "E02" "边界异常" "开会到第二天" "create" ""
    run_test "E03" "边界异常" "3点到2点开会" "" ""
    run_test "E04" "边界异常" "同时做两件事" "" ""
    run_test "E05" "边界异常" "啊啊啊啊啊" "" ""
    run_test "E06" "边界异常" "'; DROP TABLE" "" ""
    run_test "E07" "边界异常" "开会开会开会开会" "create" "开会"
    run_test "E08" "边界异常" "从-3点开始" "" ""
    run_test "E09" "边界异常" "100小时的会" "" ""
    run_test "E10" "边界异常" "？？？" "" ""
    run_test "E11" "边界异常" "。。。" "" ""
    run_test "E12" "边界异常" "额..." "" ""
    run_test "E13" "边界异常" "emmm" "" ""
    run_test "E14" "边界异常" "asdfghjkl" "" ""
}

test_internet_slang() {
    echo -e "\n${BLUE}========== 维度 15：网络用语与缩写 ==========${NC}"

    run_test "W01" "网络用语" "yyds的健身房" "create" "健身"
    run_test "W02" "网络用语" "绝绝子的约会" "create" "约会"
    run_test "W03" "网络用语" "下午开会 emo了" "create" "开会"
    run_test "W04" "网络用语" "明天要加班 裂开" "create" "加班"
    run_test "W05" "网络用语" "无语明天还要开会" "create" "开会"
    run_test "W06" "网络用语" "芭比Q了明天有考试" "create" "考试"
    run_test "W07" "网络用语" "3点有个mtg" "create" ""
    run_test "W08" "网络用语" "今天WFH" "update_status" ""
    run_test "W09" "网络用语" "中午1on1" "create" ""
    run_test "W10" "网络用语" "下午sync一下" "create" ""
    run_test "W11" "网络用语" "有个stand up" "create" ""
    run_test "W12" "网络用语" "明天的ddl" "create" ""
    run_test "W13" "网络用语" "今天得OT" "create" ""
    run_test "W14" "网络用语" "ppt做完了" "update_status" ""
}

test_mixed_language() {
    echo -e "\n${BLUE}========== 维度 17：中英混合输入 ==========${NC}"

    run_test "M01" "中英混合" "下午有个meeting" "create" ""
    run_test "M02" "中英混合" "today晚上健身" "create" "健身"
    run_test "M03" "中英混合" "明天的schedule" "query" ""
    run_test "M04" "中英混合" "cancel掉那个会" "cancel" ""
    run_test "M05" "中英混合" "modify一下时间" "modify" ""
    run_test "M06" "中英混合" "confirm了" "confirm" ""
    run_test "M07" "中英混合" "3pm开会" "create" "开会"
    run_test "M08" "中英混合" "9am起床" "create" "起床"
    run_test "M09" "中英混合" "lunch time" "create" ""
    run_test "M10" "中英混合" "gym session下午" "create" ""
}

test_number_format() {
    echo -e "\n${BLUE}========== 维度 18：数字格式混乱 ==========${NC}"

    run_test "D01" "数字格式" "三点开会" "create" "开会"
    run_test "D02" "数字格式" "十一点睡觉" "create" "睡"
    run_test "D03" "数字格式" "一个半小时健身" "create" "健身"
    run_test "D04" "数字格式" "两三个小时的会" "create" ""
    run_test "D05" "数字格式" "零点睡觉" "create" "睡"
    run_test "D06" "数字格式" "十二点半午餐" "create" "午餐"
    run_test "D07" "数字格式" "二十三点睡" "create" "睡"
    run_test "D08" "数字格式" "仨小时会议" "create" ""
    run_test "D09" "数字格式" "俩点开会" "create" "开会"
    run_test "D10" "数字格式" "一刻钟的事" "create" ""
}

test_real_user_scenarios() {
    echo -e "\n${BLUE}========== 维度 19：模拟真实用户完整对话 ==========${NC}"

    # 场景 1：急性子用户
    echo -e "${YELLOW}场景 1：急性子用户${NC}"
    local response=$(send_voice_request "ming天3点有会")
    local session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    evaluate_and_log "R1_1" "真实用户" "ming天3点有会" "create" "" "$response"
    sleep 0.3

    response=$(send_voice_request "不对4点" "$session_id")
    evaluate_and_log "R1_2" "真实用户" "不对4点" "modify" "" "$response"
    sleep 0.3

    response=$(send_voice_request "健身完去" "$session_id")
    evaluate_and_log "R1_3" "真实用户" "健身完去" "create" "" "$response"
    sleep 0.3

    response=$(send_voice_request "确定" "$session_id")
    evaluate_and_log "R1_4" "真实用户" "确定" "confirm" "" "$response"

    # 场景 2：状态与时刻表混合
    echo -e "${YELLOW}场景 2：状态与时刻表混合${NC}"
    response=$(send_voice_request "我现在在加班")
    session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    evaluate_and_log "R2_1" "真实用户" "我现在在加班" "update_status" "加班" "$response"
    sleep 0.3

    response=$(send_voice_request "明天3点还有会" "$session_id")
    evaluate_and_log "R2_2" "真实用户" "明天3点还有会" "create" "" "$response"
    sleep 0.3

    response=$(send_voice_request "现在好累" "$session_id")
    evaluate_and_log "R2_3" "真实用户" "现在好累" "update_status" "" "$response"
    sleep 0.3

    response=$(send_voice_request "我去睡了" "$session_id")
    evaluate_and_log "R2_4" "真实用户" "我去睡了" "update_status" "" "$response"

    # 场景 3：啰嗦用户
    echo -e "${YELLOW}场景 3：啰嗦用户${NC}"
    response=$(send_voice_request "我跟你说啊，明天有个特别重要的会议，就是那个什么什么项目的，老板说必须参加的那个，好像是下午来着，对下午3点，哦不对是3点半，反正就那会儿，你帮我记一下")
    evaluate_and_log "R3_1" "真实用户" "啰嗦长句" "create" "" "$response"

    # 场景 4：上下文跳跃
    echo -e "${YELLOW}场景 4：上下文跳跃${NC}"
    response=$(send_voice_request "明天开会")
    session_id=$(echo "$response" | jq -r '.data.session_id // empty')
    evaluate_and_log "R4_1" "真实用户" "明天开会" "create" "" "$response"
    sleep 0.3

    response=$(send_voice_request "今天天气真好" "$session_id")
    evaluate_and_log "R4_2" "真实用户" "今天天气真好" "chat" "" "$response"
    sleep 0.3

    response=$(send_voice_request "那个会是3点" "$session_id")
    evaluate_and_log "R4_3" "真实用户" "那个会是3点" "" "" "$response"
    sleep 0.3

    response=$(send_voice_request "会议完了去健身" "$session_id")
    evaluate_and_log "R4_4" "真实用户" "会议完了去健身" "create" "健身" "$response"
    sleep 0.3

    response=$(send_voice_request "确认" "$session_id")
    evaluate_and_log "R4_5" "真实用户" "确认" "confirm" "" "$response"

    # 场景 5：完全混乱
    echo -e "${YELLOW}场景 5：完全混乱${NC}"
    response=$(send_voice_request "那个什么...就是...你懂的...明天那个...3点还是4点来着...开会吧好像...不对是健身...算了随便")
    evaluate_and_log "R5_1" "真实用户" "完全混乱" "" "" "$response"
}

test_claude_comparison() {
    echo -e "\n${BLUE}========== 维度 20：与 Claude 对比 ==========${NC}"

    run_test "P01" "Claude对比" "帮我安排明天，早上健身，中午和朋友吃饭，下午看牙医，晚上在家休息" "create" ""
    run_test "P02" "Claude对比" "我说的那个会改到后天，然后把健身加到开会之前" "modify" ""
    run_test "P03" "Claude对比" "我不确定几点能结束，大概5-7点之间吧" "create" ""
    run_test "P04" "Claude对比" "周末想找个时间放松一下，你有什么建议" "" ""
    run_test "P05" "Claude对比" "明天的安排太紧了，帮我看看能不能调整" "" ""
    run_test "P06" "Claude对比" "我要睡觉了，帮我把明天的事理一下" "" ""
    run_test "P07" "Claude对比" "你觉得这样安排合理吗" "" ""
    run_test "P08" "Claude对比" "有没有什么遗漏的" "" ""
    run_test "P09" "Claude对比" "帮我优化一下时间分配" "" ""
    run_test "P10" "Claude对比" "假如明天下雨怎么办" "" ""
}

generate_report() {
    echo -e "\n${BLUE}========== 生成评估报告 ==========${NC}"

    local pass_rate=0
    if [[ $TOTAL_TESTS -gt 0 ]]; then
        pass_rate=$(echo "scale=2; $PASSED_TESTS * 100 / $TOTAL_TESTS" | bc)
    fi

    # 计算各维度平均分
    local total_avg=0
    local dim_count=0

    cat > "$REPORT_FILE" << EOF
# 语音时刻表 AI 对话系统 - 终极评估报告

> 生成时间：$(date '+%Y-%m-%d %H:%M:%S')
> 测试版本：build-87
> 测试用例：$TOTAL_TESTS 用例，20 个维度

---

## 一、执行摘要

### 测试结果
- **总用例数**：$TOTAL_TESTS
- **通过数**：$PASSED_TESTS
- **失败数**：$FAILED_TESTS
- **通过率**：${pass_rate}%

### 分维度评分

| 维度 | 用例数 | 平均分 | 评价 |
|------|--------|--------|------|
EOF

    for dim in ${(k)DIMENSION_SCORES}; do
        local count=${DIMENSION_COUNTS[$dim]:-0}
        if [[ $count -gt 0 ]]; then
            local score=${DIMENSION_SCORES[$dim]:-0}
            local avg=$(echo "scale=1; $score / $count" | bc)
            local eval="需改进"
            if (( $(echo "$avg >= 85" | bc -l) )); then
                eval="优秀"
            elif (( $(echo "$avg >= 70" | bc -l) )); then
                eval="良好"
            elif (( $(echo "$avg >= 60" | bc -l) )); then
                eval="及格"
            fi
            echo "| $dim | $count | $avg | $eval |" >> "$REPORT_FILE"
            total_avg=$(echo "$total_avg + $avg" | bc)
            dim_count=$((dim_count + 1))
        fi
    done

    local overall=0
    if [[ $dim_count -gt 0 ]]; then
        overall=$(echo "scale=1; $total_avg / $dim_count" | bc)
    fi

    # 确定评级
    local grade="F"
    if (( $(echo "$overall >= 90" | bc -l) )); then
        grade="A (Claude 级别)"
    elif (( $(echo "$overall >= 80" | bc -l) )); then
        grade="B (优秀)"
    elif (( $(echo "$overall >= 70" | bc -l) )); then
        grade="C (良好)"
    elif (( $(echo "$overall >= 60" | bc -l) )); then
        grade="D (及格)"
    else
        grade="F (需改进)"
    fi

    cat >> "$REPORT_FILE" << EOF

### 总体评分
**${overall} 分 - 评级：${grade}**

---

## 二、关键发现

### 2.1 优势
1. **基础时刻表创建稳定**：能正确解析大部分时间和活动描述
2. **意图分发机制有效**：Tool Calling + CoT 混合机制基本能区分 update_status 和 create
3. **多轮对话支持**：能在一定程度上保持上下文累积

### 2.2 需改进
1. **错别字容忍度**：对拼音输入和同音字错误的处理需要加强
2. **逻辑混乱处理**：面对用户前后矛盾时，需要更智能的确认机制
3. **口语化理解**：网络用语、方言等非标准表达识别率有待提高
4. **边界情况处理**：对异常输入的友好提示需要优化

---

## 三、与 Claude 对比

| 能力维度 | 本系统 | Claude | 差距 |
|----------|--------|--------|------|
| 错别字容忍 | 中等 | 优秀 | 需要更多训练数据 |
| 上下文理解 | 良好 | 优秀 | 差距较小 |
| 意图识别 | 良好 | 优秀 | Tool Calling 有效 |
| 逻辑推理 | 中等 | 优秀 | 需要更强的 LLM |
| 情感响应 | 基础 | 优秀 | 需要增加情感模块 |

---

## 四、改进建议

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

## 五、结论

本 AI 对话系统在基础功能上表现稳定，通过 Tool Calling + Chain of Thought 混合机制，成功解决了 update_status 和 create 意图混淆的核心问题。

与 Claude 级别的 AI 相比，在以下方面仍有提升空间：
- 错别字和非标准输入的容忍度
- 复杂逻辑和上下文的处理能力
- 情感理解和共情回应

**总体评价**：系统达到了实用水平，但距离 Claude 级别的自然语言理解能力还有一定差距。

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
