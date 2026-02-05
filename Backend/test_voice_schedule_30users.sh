#!/bin/bash

# ============================================================
# 语音时刻表全面测试 - 30个用户场景，每个至少10轮对话
# 测试目标：
# 1. 时间格式正确性（HH:MM 5字符格式）
# 2. 多轮对话上下文保持
# 3. 修改操作准确性
# 4. 事件类型正确性
# 5. 人性化对话体验
# ============================================================

API_BASE="http://49.232.13.41:8080/api/v1"
# 先获取有效 Token（测试账号）
TOKEN=""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# 日志文件
LOG_FILE="test_results_$(date +%Y%m%d_%H%M%S).log"
DETAIL_LOG="test_details_$(date +%Y%m%d_%H%M%S).json"

# 统计变量
declare -A USER_RESULTS
TOTAL_TURNS=0
TOTAL_SUCCESS=0
TOTAL_PARTIAL=0
TOTAL_FAIL=0

# 时间格式错误计数
TIME_FORMAT_ERRORS=0
TIME_FORMAT_TOTAL=0

# 上下文保持错误计数
CONTEXT_ERRORS=0
CONTEXT_TOTAL=0

# 获取测试 Token
get_token() {
    echo -e "${CYAN}正在获取测试 Token...${NC}"
    response=$(curl -s -X POST "$API_BASE/auth/sms/verify" \
        -H "Content-Type: application/json" \
        -d '{"phone": "13800000001", "code": "111111"}')

    TOKEN=$(echo "$response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

    if [ -z "$TOKEN" ]; then
        echo -e "${RED}获取 Token 失败！${NC}"
        echo "响应: $response"
        exit 1
    fi

    echo -e "${GREEN}Token 获取成功${NC}"
}

# 发送消息并获取完整响应
send_message() {
    local session_id="$1"
    local text="$2"

    response=$(curl -s -X POST "$API_BASE/agent/voice-schedule/text" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"session_id\": \"$session_id\", \"text\": \"$text\"}" \
        --max-time 120)

    echo "$response"
}

# 提取时间格式并验证
validate_time_format() {
    local response="$1"
    local errors=0
    local total=0

    # 提取所有 start_time 和 end_time
    times=$(echo "$response" | grep -oE '"(start_time|end_time)":"[^"]*"' | cut -d'"' -f4)

    for time in $times; do
        total=$((total + 1))
        TIME_FORMAT_TOTAL=$((TIME_FORMAT_TOTAL + 1))

        # 检查是否是 HH:MM 格式（5字符）
        if [[ ! "$time" =~ ^[0-2][0-9]:[0-5][0-9]$ ]]; then
            errors=$((errors + 1))
            TIME_FORMAT_ERRORS=$((TIME_FORMAT_ERRORS + 1))
            echo "TIME_ERROR:$time"
        fi
    done

    echo "TIME_CHECK:total=$total,errors=$errors"
}

# 提取关键信息
extract_info() {
    local response="$1"

    # 提取 session_id
    session_id=$(echo "$response" | grep -o '"session_id":"[^"]*"' | head -1 | cut -d'"' -f4)

    # 提取事件类型
    event_type=$(echo "$response" | grep -o '"type":"[^"]*"' | tail -1 | cut -d'"' -f4)

    # 提取消息
    message=$(echo "$response" | grep -o '"message":"[^"]*"' | head -1 | cut -d'"' -f4)

    # 提取时刻表数量
    schedule_count=$(echo "$response" | grep -c '"start_time"')

    # 提取 action
    action=$(echo "$response" | grep -o '"action":"[^"]*"' | head -1 | cut -d'"' -f4)

    echo "session_id=$session_id|event_type=$event_type|schedule_count=$schedule_count|action=$action|message=$message"
}

# 评估单轮对话质量
evaluate_turn() {
    local input="$1"
    local response="$2"
    local expected_type="$3"
    local context_check="$4"

    local score=0
    local max_score=10
    local issues=""

    # 1. 检查响应有效性 (2分)
    if [ -z "$response" ] || echo "$response" | grep -q '"type":"error"'; then
        issues+="响应无效;"
        echo "FAIL|$issues|0/$max_score"
        return
    fi
    score=$((score + 2))

    # 2. 时间格式检查 (2分)
    time_check=$(validate_time_format "$response" | grep "TIME_CHECK")
    time_errors=$(echo "$time_check" | grep -o 'errors=[0-9]*' | cut -d'=' -f2)
    if [ "$time_errors" == "0" ] || [ -z "$time_errors" ]; then
        score=$((score + 2))
    else
        issues+="时间格式错误:$time_errors个;"
    fi

    # 3. 事件类型检查 (2分)
    if [ -n "$expected_type" ]; then
        info=$(extract_info "$response")
        actual_type=$(echo "$info" | grep -o 'event_type=[^|]*' | cut -d'=' -f2)
        if [ "$actual_type" == "$expected_type" ]; then
            score=$((score + 2))
        else
            issues+="期望类型:$expected_type,实际:$actual_type;"
        fi
    else
        score=$((score + 2))
    fi

    # 4. 上下文保持检查 (2分)
    if [ -n "$context_check" ]; then
        CONTEXT_TOTAL=$((CONTEXT_TOTAL + 1))
        if echo "$response" | grep -q "$context_check"; then
            score=$((score + 2))
        else
            issues+="上下文丢失:$context_check;"
            CONTEXT_ERRORS=$((CONTEXT_ERRORS + 1))
        fi
    else
        score=$((score + 2))
    fi

    # 5. 消息自然度检查 (2分) - 检查是否有回复消息
    if echo "$response" | grep -q '"message":"[^"]\{5,\}"'; then
        score=$((score + 2))
    else
        issues+="消息过短或缺失;"
    fi

    # 输出评估结果
    if [ $score -ge 8 ]; then
        echo "PASS|$issues|$score/$max_score"
    elif [ $score -ge 5 ]; then
        echo "PARTIAL|$issues|$score/$max_score"
    else
        echo "FAIL|$issues|$score/$max_score"
    fi
}

# 运行用户场景
run_user_scenario() {
    local user_num="$1"
    local scenario_name="$2"
    shift 2

    echo -e "\n${BLUE}══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  用户 $user_num: $scenario_name${NC}"
    echo -e "${BLUE}══════════════════════════════════════════════════════════════${NC}"

    local session_id=""
    local turn=0
    local success=0
    local fail=0
    local partial=0
    local total_score=0
    local max_total=0

    # 解析参数：每4个一组 (input, expected_type, context_check, description)
    while [ $# -ge 1 ]; do
        local input="$1"
        local expected_type="${2:-}"
        local context_check="${3:-}"
        local description="${4:-}"
        shift
        [ $# -ge 1 ] && shift
        [ $# -ge 1 ] && shift
        [ $# -ge 1 ] && shift

        turn=$((turn + 1))
        TOTAL_TURNS=$((TOTAL_TURNS + 1))
        max_total=$((max_total + 10))

        echo -e "\n${YELLOW}[Turn $turn] 用户: $input${NC}"
        [ -n "$description" ] && echo -e "${CYAN}  说明: $description${NC}"

        response=$(send_message "$session_id" "$input")

        # 更新 session_id
        new_session=$(echo "$response" | grep -o '"session_id":"[^"]*"' | head -1 | cut -d'"' -f4)
        [ -n "$new_session" ] && session_id="$new_session"

        # 显示响应摘要
        info=$(extract_info "$response")
        echo -e "${GREEN}  响应: $info${NC}"

        # 评估
        eval_result=$(evaluate_turn "$input" "$response" "$expected_type" "$context_check")
        eval_status=$(echo "$eval_result" | cut -d'|' -f1)
        eval_issues=$(echo "$eval_result" | cut -d'|' -f2)
        eval_score=$(echo "$eval_result" | cut -d'|' -f3)

        score_num=$(echo "$eval_score" | cut -d'/' -f1)
        total_score=$((total_score + score_num))

        case "$eval_status" in
            "PASS")
                success=$((success + 1))
                TOTAL_SUCCESS=$((TOTAL_SUCCESS + 1))
                echo -e "${GREEN}  ✓ 评估: 通过 ($eval_score)${NC}"
                ;;
            "PARTIAL")
                partial=$((partial + 1))
                TOTAL_PARTIAL=$((TOTAL_PARTIAL + 1))
                echo -e "${YELLOW}  △ 评估: 部分通过 - $eval_issues ($eval_score)${NC}"
                ;;
            "FAIL")
                fail=$((fail + 1))
                TOTAL_FAIL=$((TOTAL_FAIL + 1))
                echo -e "${RED}  ✗ 评估: 失败 - $eval_issues ($eval_score)${NC}"
                ;;
        esac

        # 延迟避免请求过快
        sleep 0.5
    done

    # 用户总结
    local pass_rate=$(echo "scale=1; $success * 100 / $turn" | bc)
    local quality_score=$(echo "scale=1; $total_score * 100 / $max_total" | bc)

    echo -e "\n${MAGENTA}─── 用户 $user_num 总结 ───${NC}"
    echo -e "  总轮数: $turn | 通过: ${GREEN}$success${NC} | 部分: ${YELLOW}$partial${NC} | 失败: ${RED}$fail${NC}"
    echo -e "  通过率: ${pass_rate}% | 质量分: ${quality_score}%"

    # 记录结果
    USER_RESULTS[$user_num]="$scenario_name|$turn|$success|$partial|$fail|$total_score|$max_total"
}

# ============================================================
# 开始测试
# ============================================================

echo -e "${MAGENTA}"
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║          语音时刻表全面测试 - 30个用户场景                     ║"
echo "║          每个用户至少10轮对话，验证对话质量                    ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# 获取 Token
get_token

# ============================================================
# 30个用户场景定义
# 格式: input, expected_type, context_check, description
# ============================================================

# 用户1: 基础时刻表创建（10轮）
run_user_scenario 1 "基础时刻表创建" \
    "今天下午3点开会" "schedule" "" "创建第一个时段" \
    "会议2小时" "schedule" "" "补充时长" \
    "开完会去健身" "schedule" "" "添加第二个时段" \
    "健身1小时" "schedule" "" "补充健身时长" \
    "然后回家吃饭" "schedule" "" "添加第三个时段" \
    "吃饭到8点" "schedule" "" "补充吃饭时间" \
    "晚上看电影" "schedule" "" "添加第四个时段" \
    "看到10点" "schedule" "" "补充电影时间" \
    "确认" "confirmed" "" "确认时刻表" \
    "查看我的安排" "" "开会" "查询应包含开会"

# 用户2: 修改操作测试（12轮）
run_user_scenario 2 "修改操作测试" \
    "明天9点开会" "schedule" "" "创建" \
    "开到11点" "schedule" "" "时长" \
    "下午2点健身" "schedule" "" "第二个时段" \
    "健身2小时" "schedule" "" "时长" \
    "确认" "confirmed" "" "确认" \
    "把开会改到10点" "schedule" "10:00" "修改开始时间" \
    "确认" "confirmed" "" "确认修改" \
    "把健身改成游泳" "schedule" "游泳" "修改内容" \
    "确认" "confirmed" "" "确认修改" \
    "把游泳时间改到3点" "schedule" "15:00" "修改第二个时段时间" \
    "确认" "confirmed" "" "确认" \
    "查看明天安排" "" "游泳" "确认修改生效"

# 用户3: 取消操作测试（10轮）
run_user_scenario 3 "取消操作测试" \
    "今天晚上7点吃饭" "schedule" "" "创建" \
    "吃到9点" "schedule" "" "时长" \
    "然后看电影" "schedule" "" "第二个时段" \
    "看到11点" "schedule" "" "时长" \
    "确认" "confirmed" "" "确认" \
    "取消电影" "schedule" "" "取消一个时段" \
    "确认" "confirmed" "" "确认取消" \
    "查看安排" "" "吃饭" "应该只剩吃饭" \
    "重新加上电影" "schedule" "" "重新添加" \
    "确认" "confirmed" "" "确认"

# 用户4: 跨午夜场景（11轮）
run_user_scenario 4 "跨午夜场景" \
    "今晚10点打游戏" "schedule" "" "创建晚间活动" \
    "玩到凌晨2点" "schedule" "02:00" "跨午夜时间" \
    "然后睡觉" "schedule" "" "添加睡眠" \
    "睡到明天10点" "schedule" "10:00" "跨午夜睡眠" \
    "确认" "confirmed" "" "确认" \
    "把游戏改到11点开始" "schedule" "23:00" "修改" \
    "确认" "confirmed" "" "确认修改" \
    "明天下午3点健身" "schedule" "" "明天的安排" \
    "2小时" "schedule" "" "时长" \
    "确认" "confirmed" "" "确认" \
    "查看完整安排" "" "游戏" "验证跨午夜保持"

# 用户5: 模糊时间处理（10轮）
run_user_scenario 5 "模糊时间处理" \
    "待会儿去吃饭" "schedule" "" "模糊时间" \
    "大概半小时后吧" "schedule" "" "补充" \
    "吃完去逛街" "schedule" "" "第二个时段" \
    "逛到傍晚" "schedule" "" "模糊结束时间" \
    "确认" "confirmed" "" "确认" \
    "晚点出去喝酒" "schedule" "" "模糊开始" \
    "大概8、9点" "schedule" "" "模糊时间范围" \
    "玩到很晚" "schedule" "" "模糊结束" \
    "确认" "confirmed" "" "确认" \
    "查看今天安排" "" "" "查询"

# 用户6: 连续活动衔接（12轮）
run_user_scenario 6 "连续活动衔接" \
    "下午3点开会" "schedule" "" "开始" \
    "开1小时" "schedule" "" "时长" \
    "开完会直接去健身" "schedule" "" "衔接" \
    "健身1小时" "schedule" "" "时长" \
    "健身完去超市" "schedule" "" "衔接" \
    "买半小时东西" "schedule" "" "时长" \
    "然后回家做饭" "schedule" "" "衔接" \
    "做1小时" "schedule" "" "时长" \
    "确认" "confirmed" "" "确认" \
    "在健身和超市之间加个休息" "schedule" "" "插入" \
    "休息15分钟" "schedule" "" "时长" \
    "确认" "confirmed" "" "确认"

# 用户7: 工作日程测试（14轮）
run_user_scenario 7 "工作日程测试" \
    "明天有很多会议" "" "" "开场" \
    "早上9点产品评审" "schedule" "" "第一个" \
    "10点半技术讨论" "schedule" "" "第二个" \
    "中午12点和客户吃饭" "schedule" "" "第三个" \
    "下午2点项目汇报" "schedule" "" "第四个" \
    "4点团队会议" "schedule" "" "第五个" \
    "6点下班" "schedule" "" "结束" \
    "确认" "confirmed" "" "确认" \
    "产品评审改到9点半" "schedule" "09:30" "修改" \
    "确认" "confirmed" "" "确认" \
    "取消技术讨论" "schedule" "" "取消" \
    "确认" "confirmed" "" "确认" \
    "加一个3点的面试" "schedule" "" "添加" \
    "确认" "confirmed" "" "确认"

# 用户8: 周末休闲测试（10轮）
run_user_scenario 8 "周末休闲测试" \
    "周六想睡懒觉" "schedule" "" "开始" \
    "睡到中午12点" "schedule" "" "时间" \
    "起来吃brunch" "schedule" "" "第二个" \
    "下午去公园" "schedule" "" "第三个" \
    "晚上看电影" "schedule" "" "第四个" \
    "确认" "confirmed" "" "确认" \
    "把公园改成逛街" "schedule" "逛街" "修改内容" \
    "确认" "confirmed" "" "确认" \
    "晚上电影改成酒吧" "schedule" "酒吧" "修改" \
    "确认" "confirmed" "" "确认"

# 用户9: 状态即时更新（10轮）
run_user_scenario 9 "状态即时更新" \
    "我现在在工作" "current_status" "" "即时状态" \
    "更新到首页" "" "" "确认更新" \
    "开会了" "current_status" "" "状态变化" \
    "同步状态" "" "" "确认同步" \
    "会开完了" "" "" "状态结束" \
    "现在休息" "current_status" "" "新状态" \
    "更新" "" "" "确认更新" \
    "下午继续工作" "current_status" "" "新状态" \
    "同步" "" "" "确认" \
    "下班回家了" "current_status" "" "状态变化"

# 用户10: 简短指令测试（12轮）
run_user_scenario 10 "简短指令测试" \
    "开会" "" "" "最简指令" \
    "3点" "schedule" "" "补时间" \
    "2小时" "schedule" "" "补时长" \
    "确认" "confirmed" "" "确认" \
    "吃饭" "" "" "新任务" \
    "6点" "schedule" "" "时间" \
    "1小时" "schedule" "" "时长" \
    "确认" "confirmed" "" "确认" \
    "改开会时间" "" "" "修改" \
    "改到4点" "schedule" "16:00" "新时间" \
    "好" "confirmed" "" "确认" \
    "查看" "" "" "查询"

# 用户11: 多日规划（12轮）
run_user_scenario 11 "多日规划" \
    "帮我规划这周" "" "" "开始" \
    "今天晚上休息" "schedule" "" "今天" \
    "确认今天的" "confirmed" "" "确认" \
    "明天9点到6点上班" "schedule" "" "明天" \
    "中午休息1小时" "schedule" "" "补充" \
    "确认" "confirmed" "" "确认" \
    "后天有个会议" "schedule" "" "后天" \
    "上午10点" "schedule" "" "时间" \
    "3小时" "schedule" "" "时长" \
    "下午自由" "schedule" "" "下午" \
    "确认" "confirmed" "" "确认" \
    "查看这周安排" "" "" "查询"

# 用户12: 复杂时间推理（10轮）
run_user_scenario 12 "复杂时间推理" \
    "午饭后去开会" "schedule" "" "相对时间" \
    "午饭12点吃" "schedule" "" "锚点时间" \
    "会议2小时" "schedule" "" "时长" \
    "开完会去健身" "schedule" "" "衔接" \
    "健身1小时" "schedule" "" "时长" \
    "确认" "confirmed" "" "确认" \
    "把午饭提前到11点半" "schedule" "11:30" "修改基准时间" \
    "后面的时间都调整" "schedule" "" "联动调整" \
    "确认" "confirmed" "" "确认" \
    "查看安排" "" "" "验证"

# 用户13: 情绪化表达（10轮）
run_user_scenario 13 "情绪化表达" \
    "好累啊想休息" "schedule" "" "情绪表达" \
    "下午不想动" "schedule" "" "模糊意愿" \
    "就想躺着" "schedule" "" "状态描述" \
    "确认" "confirmed" "" "确认" \
    "晚上心情好再出去" "schedule" "" "条件性" \
    "大概7、8点吧" "schedule" "" "不确定时间" \
    "去散散步" "schedule" "" "活动" \
    "确认" "confirmed" "" "确认" \
    "查看安排" "" "" "查询" \
    "挺好的谢谢" "" "" "结束"

# 用户14: 地点相关（10轮）
run_user_scenario 14 "地点相关安排" \
    "下午去公司开会" "schedule" "" "带地点" \
    "3点到5点" "schedule" "" "时间" \
    "然后去三里屯吃饭" "schedule" "" "带地点" \
    "吃完去酒吧" "schedule" "" "衔接" \
    "确认" "confirmed" "" "确认" \
    "开会地点改成咖啡厅" "schedule" "" "修改地点" \
    "确认" "confirmed" "" "确认" \
    "取消酒吧" "schedule" "" "取消" \
    "直接回家" "schedule" "" "替换" \
    "确认" "confirmed" "" "确认"

# 用户15: 健身计划（10轮）
run_user_scenario 15 "健身计划" \
    "今天去健身" "schedule" "" "开始" \
    "下午4点" "schedule" "" "时间" \
    "先跑步30分钟" "schedule" "" "细分1" \
    "然后力量训练1小时" "schedule" "" "细分2" \
    "最后拉伸15分钟" "schedule" "" "细分3" \
    "确认" "confirmed" "" "确认" \
    "力量训练改成45分钟" "schedule" "" "修改子项" \
    "确认" "confirmed" "" "确认" \
    "健身后加个蛋白餐" "schedule" "" "添加" \
    "确认" "confirmed" "" "确认"

# 用户16: 学习计划（10轮）
run_user_scenario 16 "学习计划" \
    "今晚要学习" "schedule" "" "开始" \
    "7点开始" "schedule" "" "时间" \
    "先复习数学1小时" "schedule" "" "第一科" \
    "然后英语1小时" "schedule" "" "第二科" \
    "最后做题" "schedule" "" "第三项" \
    "确认" "confirmed" "" "确认" \
    "数学改成半小时" "schedule" "" "修改" \
    "英语加到1个半小时" "schedule" "" "修改" \
    "确认" "confirmed" "" "确认" \
    "查看学习计划" "" "" "查询"

# 用户17: 社交活动（12轮）
run_user_scenario 17 "社交活动" \
    "周末有聚会" "" "" "开场" \
    "周六中午同学聚餐" "schedule" "" "第一个" \
    "下午和闺蜜喝下午茶" "schedule" "" "第二个" \
    "晚上家庭聚餐" "schedule" "" "第三个" \
    "确认周六" "confirmed" "" "确认" \
    "周日和男朋友约会" "schedule" "" "周日" \
    "上午10点见面" "schedule" "" "时间" \
    "中午一起吃饭" "schedule" "" "衔接" \
    "下午看电影" "schedule" "" "衔接" \
    "晚上各回各家" "schedule" "" "结束" \
    "确认" "confirmed" "" "确认" \
    "查看周末安排" "" "" "查询"

# 用户18: 出差安排（10轮）
run_user_scenario 18 "出差安排" \
    "后天出差" "schedule" "" "开始" \
    "早上6点的飞机" "schedule" "06:00" "早班机" \
    "9点到目的地" "schedule" "" "到达" \
    "10点开会" "schedule" "" "会议" \
    "中午和客户吃饭" "schedule" "" "午餐" \
    "下午继续谈" "schedule" "" "下午" \
    "晚上9点飞回来" "schedule" "21:00" "返程" \
    "确认" "confirmed" "" "确认" \
    "飞机改成8点的" "schedule" "08:00" "修改" \
    "确认" "confirmed" "" "确认"

# 用户19: 极端输入测试（10轮）
run_user_scenario 19 "极端输入测试" \
    "嗯" "" "" "最短输入" \
    "？" "" "" "特殊字符" \
    "帮我安排" "" "" "模糊请求" \
    "今天下午" "schedule" "" "时间" \
    "3点健身" "schedule" "" "活动" \
    "好" "confirmed" "" "确认" \
    "改一下" "" "" "模糊修改" \
    "改成游泳" "schedule" "游泳" "具体内容" \
    "行" "confirmed" "" "确认" \
    "就这样" "" "" "结束"

# 用户20: 长对话测试（15轮）
run_user_scenario 20 "长对话上下文保持" \
    "早上7点起床" "schedule" "" "第1个时段" \
    "8点吃早餐" "schedule" "" "第2个时段" \
    "9点开始工作" "schedule" "" "第3个时段" \
    "12点午餐" "schedule" "" "第4个时段" \
    "1点继续工作" "schedule" "" "第5个时段" \
    "3点开会" "schedule" "" "第6个时段" \
    "5点健身" "schedule" "" "第7个时段" \
    "7点晚餐" "schedule" "" "第8个时段" \
    "8点看书" "schedule" "" "第9个时段" \
    "10点睡觉" "schedule" "" "第10个时段" \
    "确认" "confirmed" "" "确认所有" \
    "把第3个改成在家办公" "schedule" "" "修改指定索引" \
    "确认" "confirmed" "" "确认" \
    "查看完整安排" "" "起床" "验证第1个保持" \
    "没问题" "" "" "结束"

# 用户21: 取消后重建（10轮）
run_user_scenario 21 "取消后重建" \
    "明天10点开会" "schedule" "" "创建" \
    "确认" "confirmed" "" "确认" \
    "取消" "cancelled" "" "取消" \
    "重新安排" "" "" "重建" \
    "改成11点" "schedule" "11:00" "新时间" \
    "加个下午的培训" "schedule" "" "添加" \
    "3点开始" "schedule" "" "时间" \
    "2小时" "schedule" "" "时长" \
    "确认" "confirmed" "" "确认" \
    "查看安排" "" "培训" "验证重建"

# 用户22: 时间冲突测试（10轮）
run_user_scenario 22 "时间冲突处理" \
    "下午2点到4点开会" "schedule" "" "创建" \
    "下午3点健身" "schedule" "" "冲突输入" \
    "2小时" "schedule" "" "时长" \
    "确认" "confirmed" "" "确认" \
    "把健身改到5点" "schedule" "17:00" "解决冲突" \
    "确认" "confirmed" "" "确认" \
    "再加个4点的下午茶" "schedule" "" "边界时间" \
    "半小时" "schedule" "" "时长" \
    "确认" "confirmed" "" "确认" \
    "查看安排" "" "开会" "验证"

# 用户23: 分钟精度测试（10轮）
run_user_scenario 23 "分钟精度测试" \
    "下午2点15分开会" "schedule" "14:15" "非整点" \
    "开到3点45" "schedule" "15:45" "非整点结束" \
    "确认" "confirmed" "" "确认" \
    "加个4点05分的电话" "schedule" "16:05" "零几分" \
    "打10分钟" "schedule" "" "短时长" \
    "确认" "confirmed" "" "确认" \
    "再加个5点半的健身" "schedule" "17:30" "半点" \
    "1小时" "schedule" "" "时长" \
    "确认" "confirmed" "" "确认" \
    "查看安排" "" "" "验证"

# 用户24: 全天安排（10轮）
run_user_scenario 24 "全天安排" \
    "明天全天有活动" "" "" "开场" \
    "凌晨0点到6点睡觉" "schedule" "00:00" "从0点开始" \
    "6点起床" "schedule" "" "起床" \
    "7点到22点工作" "schedule" "" "主要时间" \
    "22点到24点休息" "schedule" "" "到24点" \
    "确认" "confirmed" "" "确认" \
    "把工作改成9点开始" "schedule" "09:00" "修改" \
    "早上加个晨跑" "schedule" "" "添加" \
    "确认" "confirmed" "" "确认" \
    "查看完整一天" "" "" "验证"

# 用户25: 重复确认测试（10轮）
run_user_scenario 25 "重复确认测试" \
    "明天9点开会" "schedule" "" "创建" \
    "2小时" "schedule" "" "时长" \
    "确认" "confirmed" "" "第一次确认" \
    "再确认一下" "" "" "重复确认" \
    "是的确认" "" "" "再次确认" \
    "对" "" "" "简短确认" \
    "查看安排" "" "开会" "验证" \
    "加个下午的健身" "schedule" "" "新增" \
    "确认" "confirmed" "" "确认" \
    "没问题就这样" "" "" "结束"

# 用户26: 删除特定项（10轮）
run_user_scenario 26 "删除特定项" \
    "明天上午开会" "schedule" "" "第1项" \
    "下午健身" "schedule" "" "第2项" \
    "晚上吃饭" "schedule" "" "第3项" \
    "确认" "confirmed" "" "确认" \
    "删除健身" "schedule" "" "删除中间项" \
    "确认" "confirmed" "" "确认" \
    "查看安排" "" "开会" "验证第1项在" \
    "再删除吃饭" "schedule" "" "再删除" \
    "确认" "confirmed" "" "确认" \
    "查看" "" "" "验证"

# 用户27: 24小时制测试（10轮）
run_user_scenario 27 "24小时制测试" \
    "凌晨1点睡觉" "schedule" "01:00" "凌晨1点" \
    "早上8点起床" "schedule" "08:00" "上午" \
    "中午12点吃饭" "schedule" "12:00" "正午" \
    "下午14点开会" "schedule" "14:00" "24小时制" \
    "晚上20点健身" "schedule" "20:00" "24小时制" \
    "深夜23点休息" "schedule" "23:00" "深夜" \
    "确认" "confirmed" "" "确认" \
    "把健身改到19点" "schedule" "19:00" "修改" \
    "确认" "confirmed" "" "确认" \
    "查看安排" "" "" "验证"

# 用户28: 口语化表达（10轮）
run_user_scenario 28 "口语化表达" \
    "下午茶时间逛街" "schedule" "" "口语化" \
    "傍晚回家" "schedule" "" "傍晚" \
    "晚饭后看电视" "schedule" "" "相对时间" \
    "睡前看会书" "schedule" "" "睡前" \
    "确认" "confirmed" "" "确认" \
    "上午茶时间改成喝咖啡" "schedule" "" "修改" \
    "确认" "confirmed" "" "确认" \
    "深夜追剧" "schedule" "" "深夜" \
    "确认" "confirmed" "" "确认" \
    "查看" "" "" "验证"

# 用户29: 修改时保持其他项（12轮）
run_user_scenario 29 "修改时保持其他项" \
    "9点工作" "schedule" "" "第1项" \
    "12点午餐" "schedule" "" "第2项" \
    "14点开会" "schedule" "" "第3项" \
    "17点健身" "schedule" "" "第4项" \
    "19点晚餐" "schedule" "" "第5项" \
    "确认" "confirmed" "" "确认" \
    "把开会改到15点" "schedule" "15:00" "修改第3项" \
    "确认" "confirmed" "" "确认" \
    "查看安排" "" "工作" "验证第1项保持" \
    "再查看" "" "午餐" "验证第2项保持" \
    "继续查看" "" "健身" "验证第4项保持" \
    "最后确认" "" "晚餐" "验证第5项保持"

# 用户30: 综合压力测试（15轮）
run_user_scenario 30 "综合压力测试" \
    "今天很忙帮我安排" "" "" "开场" \
    "早上7点起床洗漱" "schedule" "07:00" "第1个" \
    "8点吃早餐半小时" "schedule" "" "第2个" \
    "9点到12点工作" "schedule" "" "第3个" \
    "12点到1点午餐午休" "schedule" "" "第4个" \
    "下午1点到5点继续工作" "schedule" "" "第5个" \
    "5点到6点健身" "schedule" "" "第6个" \
    "6点半到8点和朋友吃饭" "schedule" "" "第7个" \
    "8点到10点看电影" "schedule" "" "第8个" \
    "10点到11点回家洗漱" "schedule" "" "第9个" \
    "11点睡觉" "schedule" "" "第10个" \
    "确认所有安排" "confirmed" "" "确认全部" \
    "把健身改成瑜伽" "schedule" "瑜伽" "修改" \
    "确认" "confirmed" "" "确认" \
    "查看完整一天的安排" "" "起床" "验证完整性"

# ============================================================
# 生成测试报告
# ============================================================

echo -e "\n\n"
echo -e "${MAGENTA}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${MAGENTA}║                      测试总结报告                              ║${NC}"
echo -e "${MAGENTA}╚════════════════════════════════════════════════════════════════╝${NC}"

echo -e "\n${CYAN}═══ 总体统计 ═══${NC}"
echo -e "总对话轮数: ${TOTAL_TURNS}"
echo -e "通过: ${GREEN}${TOTAL_SUCCESS}${NC}"
echo -e "部分通过: ${YELLOW}${TOTAL_PARTIAL}${NC}"
echo -e "失败: ${RED}${TOTAL_FAIL}${NC}"

if [ $TOTAL_TURNS -gt 0 ]; then
    PASS_RATE=$(echo "scale=1; $TOTAL_SUCCESS * 100 / $TOTAL_TURNS" | bc)
    EFFECTIVE_RATE=$(echo "scale=1; ($TOTAL_SUCCESS + $TOTAL_PARTIAL) * 100 / $TOTAL_TURNS" | bc)
    echo -e "严格通过率: ${GREEN}${PASS_RATE}%${NC}"
    echo -e "有效率(含部分通过): ${GREEN}${EFFECTIVE_RATE}%${NC}"
fi

echo -e "\n${CYAN}═══ 时间格式统计 ═══${NC}"
echo -e "检查的时间字段: ${TIME_FORMAT_TOTAL}"
echo -e "格式错误数: ${RED}${TIME_FORMAT_ERRORS}${NC}"
if [ $TIME_FORMAT_TOTAL -gt 0 ]; then
    TIME_CORRECT_RATE=$(echo "scale=1; ($TIME_FORMAT_TOTAL - $TIME_FORMAT_ERRORS) * 100 / $TIME_FORMAT_TOTAL" | bc)
    echo -e "时间格式正确率: ${GREEN}${TIME_CORRECT_RATE}%${NC}"
fi

echo -e "\n${CYAN}═══ 上下文保持统计 ═══${NC}"
echo -e "上下文检查次数: ${CONTEXT_TOTAL}"
echo -e "上下文丢失次数: ${RED}${CONTEXT_ERRORS}${NC}"
if [ $CONTEXT_TOTAL -gt 0 ]; then
    CONTEXT_RATE=$(echo "scale=1; ($CONTEXT_TOTAL - $CONTEXT_ERRORS) * 100 / $CONTEXT_TOTAL" | bc)
    echo -e "上下文保持率: ${GREEN}${CONTEXT_RATE}%${NC}"
fi

echo -e "\n${CYAN}═══ 各用户详情 ═══${NC}"
echo "┌──────┬────────────────────────┬──────┬──────┬──────┬──────┬────────┐"
echo "│ 用户 │ 场景                   │ 轮数 │ 通过 │ 部分 │ 失败 │ 质量分 │"
echo "├──────┼────────────────────────┼──────┼──────┼──────┼──────┼────────┤"

for i in {1..30}; do
    if [ -n "${USER_RESULTS[$i]}" ]; then
        IFS='|' read -r scenario turns success partial fail score max_score <<< "${USER_RESULTS[$i]}"
        quality=$(echo "scale=0; $score * 100 / $max_score" | bc)
        printf "│ %4d │ %-22s │ %4d │ %4d │ %4d │ %4d │ %5d%% │\n" "$i" "${scenario:0:22}" "$turns" "$success" "$partial" "$fail" "$quality"
    fi
done

echo "└──────┴────────────────────────┴──────┴──────┴──────┴──────┴────────┘"

# 计算总质量分
TOTAL_SCORE=0
TOTAL_MAX=0
for i in {1..30}; do
    if [ -n "${USER_RESULTS[$i]}" ]; then
        IFS='|' read -r scenario turns success partial fail score max_score <<< "${USER_RESULTS[$i]}"
        TOTAL_SCORE=$((TOTAL_SCORE + score))
        TOTAL_MAX=$((TOTAL_MAX + max_score))
    fi
done

if [ $TOTAL_MAX -gt 0 ]; then
    OVERALL_QUALITY=$(echo "scale=1; $TOTAL_SCORE * 100 / $TOTAL_MAX" | bc)
    echo -e "\n${MAGENTA}═══ 最终评分 ═══${NC}"
    echo -e "总质量分: ${GREEN}${OVERALL_QUALITY}%${NC} ($TOTAL_SCORE / $TOTAL_MAX)"

    # 评级
    if (( $(echo "$OVERALL_QUALITY >= 90" | bc -l) )); then
        echo -e "评级: ${GREEN}★★★★★ 优秀${NC}"
    elif (( $(echo "$OVERALL_QUALITY >= 80" | bc -l) )); then
        echo -e "评级: ${GREEN}★★★★☆ 良好${NC}"
    elif (( $(echo "$OVERALL_QUALITY >= 70" | bc -l) )); then
        echo -e "评级: ${YELLOW}★★★☆☆ 中等${NC}"
    elif (( $(echo "$OVERALL_QUALITY >= 60" | bc -l) )); then
        echo -e "评级: ${YELLOW}★★☆☆☆ 及格${NC}"
    else
        echo -e "评级: ${RED}★☆☆☆☆ 需改进${NC}"
    fi
fi

echo -e "\n${CYAN}测试完成！${NC}"
echo "日志已保存到: $LOG_FILE"
