#!/bin/bash

# 全面测试语音时刻表功能
# 模拟20个不同用户场景，每个用户20轮对话

API_BASE="http://49.232.13.41:8080/api/v1"
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWZjYmE0YWYtMjM0Yi00MTc5LTkyZGUtOTk1Nzg1N2ZmZjAzIiwiZXhwIjoxNzcyODE2MzQ2fQ.QOlGxFOqjWCOmPLy3gMTz7m5qAXNfX8BCyNJu7AXYOM"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 结果统计
declare -A USER_RESULTS
declare -A TURN_COUNTS
declare -A SUCCESS_COUNTS
declare -A FAIL_COUNTS

# 发送消息并获取响应
send_message() {
    local session_id="$1"
    local text="$2"
    local user_num="$3"
    local turn_num="$4"

    response=$(curl -s -X POST "$API_BASE/agent/voice-schedule/text" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"session_id\": \"$session_id\", \"text\": \"$text\"}" \
        --max-time 120)

    echo "$response"
}

# 解析SSE响应中的关键信息
parse_response() {
    local response="$1"

    # 提取 action 类型
    action=$(echo "$response" | grep -o '"action":"[^"]*"' | head -1 | cut -d'"' -f4)

    # 提取消息
    message=$(echo "$response" | grep -o '"message":"[^"]*"' | head -1 | cut -d'"' -f4)

    # 提取时刻表数量
    schedule_count=$(echo "$response" | grep -o '"schedule":\[' | wc -l)

    # 提取事件类型
    event_type=$(echo "$response" | grep -o '"type":"[^"]*"' | tail -1 | cut -d'"' -f4)

    echo "action=$action|message=$message|schedule_count=$schedule_count|event_type=$event_type"
}

# 评估单轮对话
evaluate_turn() {
    local user_num="$1"
    local turn_num="$2"
    local input="$3"
    local response="$4"

    local parsed=$(parse_response "$response")
    local action=$(echo "$parsed" | cut -d'|' -f1 | cut -d'=' -f2)
    local event_type=$(echo "$parsed" | cut -d'|' -f4 | cut -d'=' -f2)

    # 评估标准
    local score=0
    local issues=""

    # 1. 检查是否有响应
    if [ -z "$response" ] || [ "$response" == "null" ]; then
        issues+="无响应;"
        echo "FAIL|$issues"
        return
    fi

    # 2. 检查是否有错误
    if echo "$response" | grep -q '"type":"error"'; then
        issues+="返回错误;"
        echo "FAIL|$issues"
        return
    fi

    # 3. 检查响应类型是否合理
    case "$event_type" in
        "schedule"|"current_status"|"chat"|"clarify"|"confirmed"|"cancelled")
            score=$((score + 1))
            ;;
        *)
            issues+="未知事件类型:$event_type;"
            ;;
    esac

    # 4. 检查是否遵循"先计划再确认"框架
    # 如果用户输入包含时间安排相关内容，应该返回schedule或clarify
    if echo "$input" | grep -qE "点|时|分|早上|下午|晚上|明天|今天|后天"; then
        if [ "$event_type" == "schedule" ] || [ "$event_type" == "clarify" ]; then
            score=$((score + 1))
        else
            issues+="时间输入未返回schedule;"
        fi
    fi

    # 5. 检查确认类输入
    if echo "$input" | grep -qE "确认|好的|是的|没问题|可以|行|对"; then
        if [ "$event_type" == "confirmed" ] || echo "$response" | grep -q "tool_confirm"; then
            score=$((score + 2))  # Tool Calling 成功加分
        else
            issues+="确认输入未触发确认;"
        fi
    fi

    if [ -z "$issues" ]; then
        echo "PASS|score=$score"
    else
        echo "PARTIAL|$issues|score=$score"
    fi
}

# 用户场景定义
run_user_scenario() {
    local user_num="$1"
    local scenario_name="$2"
    shift 2
    local messages=("$@")

    echo -e "\n${BLUE}========== 用户 $user_num: $scenario_name ==========${NC}"

    local session_id=""
    local turn=0
    local success=0
    local fail=0
    local partial=0

    for msg in "${messages[@]}"; do
        turn=$((turn + 1))
        echo -e "\n${YELLOW}[Turn $turn] 用户: $msg${NC}"

        response=$(send_message "$session_id" "$msg" "$user_num" "$turn")

        # 提取新的 session_id
        new_session=$(echo "$response" | grep -o '"session_id":"[^"]*"' | head -1 | cut -d'"' -f4)
        if [ -n "$new_session" ]; then
            session_id="$new_session"
        fi

        # 显示响应摘要
        parsed=$(parse_response "$response")
        echo -e "${GREEN}AI响应: $parsed${NC}"

        # 评估
        eval_result=$(evaluate_turn "$user_num" "$turn" "$msg" "$response")
        eval_status=$(echo "$eval_result" | cut -d'|' -f1)

        case "$eval_status" in
            "PASS")
                success=$((success + 1))
                echo -e "${GREEN}✓ 评估: 通过${NC}"
                ;;
            "PARTIAL")
                partial=$((partial + 1))
                echo -e "${YELLOW}△ 评估: 部分通过 - $(echo "$eval_result" | cut -d'|' -f2)${NC}"
                ;;
            "FAIL")
                fail=$((fail + 1))
                echo -e "${RED}✗ 评估: 失败 - $(echo "$eval_result" | cut -d'|' -f2)${NC}"
                ;;
        esac

        # 短暂延迟避免请求过快
        sleep 1
    done

    echo -e "\n${BLUE}--- 用户 $user_num 总结 ---${NC}"
    echo "总轮数: $turn, 通过: $success, 部分通过: $partial, 失败: $fail"
    echo "通过率: $(echo "scale=2; $success * 100 / $turn" | bc)%"

    # 记录结果
    USER_RESULTS[$user_num]="$scenario_name|$turn|$success|$partial|$fail"
}

echo "============================================"
echo "语音时刻表全面测试 - 20个用户场景"
echo "============================================"

# ============== 用户场景定义 ==============

# 用户1: 简单的今日安排
run_user_scenario 1 "简单今日安排" \
    "今天下午3点开会" \
    "会议大概2小时" \
    "开完会去健身" \
    "健身1小时" \
    "然后回家吃饭" \
    "确认" \
    "晚上8点看电影" \
    "看到10点" \
    "然后睡觉" \
    "确认" \
    "查看我今天的安排" \
    "把健身改成游泳" \
    "确认" \
    "取消今天的电影" \
    "确认" \
    "重新安排晚上" \
    "晚上7点和朋友吃饭" \
    "吃到9点" \
    "然后回家休息" \
    "确认"

# 用户2: 明天的完整安排
run_user_scenario 2 "明天完整安排" \
    "帮我安排明天的一天" \
    "早上7点起床" \
    "8点吃早餐" \
    "9点到12点工作" \
    "中午吃饭休息" \
    "下午继续工作到6点" \
    "晚上和家人吃饭" \
    "然后看书" \
    "11点睡觉" \
    "确认这个安排" \
    "把早餐时间改到7点半" \
    "好的" \
    "下午3点加一个会议" \
    "会议1小时" \
    "确认" \
    "查看明天的安排" \
    "把晚餐时间提前到6点" \
    "确认" \
    "取消下午的会议" \
    "确认"

# 用户3: 跨午夜安排（重点测试）
run_user_scenario 3 "跨午夜安排" \
    "今晚10点开始打游戏" \
    "可能玩到凌晨2点" \
    "然后睡觉" \
    "睡到明天10点" \
    "确认" \
    "把游戏时间改到11点开始" \
    "确认" \
    "明天下午有什么安排吗" \
    "明天下午3点健身" \
    "2小时" \
    "确认" \
    "晚上呢" \
    "晚上和朋友聚餐" \
    "大概7点到10点" \
    "然后回家" \
    "11点睡觉" \
    "睡到后天8点" \
    "确认" \
    "查看明天完整安排" \
    "没问题"

# 用户4: 模糊时间表达
run_user_scenario 4 "模糊时间表达" \
    "待会儿去吃饭" \
    "大概半小时后" \
    "吃完饭去逛街" \
    "逛到差不多傍晚" \
    "然后回家" \
    "确认" \
    "晚点还要出去" \
    "大概8、9点的样子" \
    "去酒吧" \
    "玩到比较晚" \
    "确认" \
    "把逛街改成看电影" \
    "行" \
    "电影大概2小时" \
    "对" \
    "查看今天安排" \
    "酒吧改成KTV" \
    "可以" \
    "取消KTV" \
    "算了不去了"

# 用户5: 工作日程安排
run_user_scenario 5 "工作日程" \
    "明天有很多会议" \
    "早上9点产品评审" \
    "10点半技术讨论" \
    "中午和客户吃饭" \
    "下午2点项目汇报" \
    "4点团队会议" \
    "6点下班" \
    "确认" \
    "产品评审改到9点半" \
    "好" \
    "技术讨论取消" \
    "确认" \
    "下午加一个面试" \
    "3点开始" \
    "1小时" \
    "确认" \
    "查看明天安排" \
    "把项目汇报和面试对调" \
    "确认" \
    "没问题"

# 用户6: 周末休闲安排
run_user_scenario 6 "周末休闲" \
    "周六想睡个懒觉" \
    "睡到中午" \
    "起来吃个brunch" \
    "下午去公园" \
    "晚上看电影" \
    "确认" \
    "周日呢" \
    "周日早点起来" \
    "去爬山" \
    "下午回来休息" \
    "晚上准备下周工作" \
    "确认" \
    "周六晚上改成去酒吧" \
    "行" \
    "周日爬山改成骑车" \
    "可以" \
    "查看周末安排" \
    "周六下午加个下午茶" \
    "3点" \
    "确认"

# 用户7: 连续活动衔接
run_user_scenario 7 "连续活动" \
    "下午开完会直接去健身" \
    "健身完去超市" \
    "买完东西回家做饭" \
    "吃完饭休息一下" \
    "然后看书" \
    "看到睡觉" \
    "确认" \
    "会议几点" \
    "3点开始" \
    "大概1小时" \
    "确认" \
    "健身和超市之间加个咖啡" \
    "喝半小时" \
    "好" \
    "把看书改成看电影" \
    "确认" \
    "取消超市" \
    "直接回家" \
    "行" \
    "确认"

# 用户8: 多日规划
run_user_scenario 8 "多日规划" \
    "帮我规划这周剩下的时间" \
    "今天晚上休息" \
    "明天正常上班" \
    "后天有个重要会议" \
    "周五提前下班" \
    "确认今天的" \
    "明天具体安排" \
    "9点到6点工作" \
    "中午休息1小时" \
    "确认" \
    "后天的会议几点" \
    "上午10点" \
    "大概3小时" \
    "下午自由" \
    "确认" \
    "周五几点下班" \
    "4点" \
    "下班后去聚餐" \
    "确认" \
    "查看这周安排"

# 用户9: 状态更新测试
run_user_scenario 9 "状态更新" \
    "我现在在工作" \
    "更新到首页" \
    "等下要开会" \
    "会议开始了更新一下" \
    "开会中" \
    "更新状态" \
    "会开完了" \
    "现在休息" \
    "同步到首页" \
    "下午继续工作" \
    "更新" \
    "快下班了" \
    "准备回家" \
    "更新状态" \
    "到家了" \
    "在家休息" \
    "更新" \
    "吃完饭了" \
    "看电视中" \
    "更新到首页"

# 用户10: 取消和修改测试
run_user_scenario 10 "取消修改" \
    "明天10点开会" \
    "确认" \
    "取消这个会议" \
    "重新安排" \
    "改到11点" \
    "确认" \
    "再取消" \
    "算了不开了" \
    "下午有个培训" \
    "3点到5点" \
    "确认" \
    "培训取消了" \
    "改成自习" \
    "确认" \
    "晚上加个聚餐" \
    "7点" \
    "确认" \
    "聚餐推迟到8点" \
    "行" \
    "确认"

# 用户11: 简短指令测试
run_user_scenario 11 "简短指令" \
    "开会" \
    "几点" \
    "3点" \
    "多久" \
    "2小时" \
    "确认" \
    "吃饭" \
    "6点" \
    "1小时" \
    "确认" \
    "睡觉" \
    "11点" \
    "确认" \
    "改一下" \
    "开会改4点" \
    "行" \
    "查看" \
    "没问题" \
    "取消吃饭" \
    "确认"

# 用户12: 复杂时间推理
run_user_scenario 12 "时间推理" \
    "吃完午饭去开会" \
    "午饭12点吃" \
    "会议2小时" \
    "开完会去健身" \
    "健身1小时" \
    "确认" \
    "健身完直接去约会" \
    "约会到晚上9点" \
    "确认" \
    "把午饭提前到11点半" \
    "后面的时间都要调整吗" \
    "是的" \
    "确认" \
    "在会议前加个准备时间" \
    "半小时" \
    "确认" \
    "约会后回家" \
    "10点到家" \
    "然后休息" \
    "确认"

# 用户13: 情绪化表达
run_user_scenario 13 "情绪表达" \
    "好累啊想休息" \
    "下午不想工作了" \
    "就想躺着" \
    "确认" \
    "晚上心情好点再出去" \
    "大概7、8点" \
    "去散步" \
    "确认" \
    "明天不想早起" \
    "睡到自然醒" \
    "确认" \
    "起来后慢慢吃早餐" \
    "然后看看书" \
    "下午再说" \
    "确认" \
    "查看安排" \
    "挺好的" \
    "就这样" \
    "没问题" \
    "谢谢"

# 用户14: 地点相关安排
run_user_scenario 14 "地点安排" \
    "下午去公司开会" \
    "3点到5点" \
    "然后去三里屯" \
    "和朋友吃饭" \
    "吃完去酒吧" \
    "确认" \
    "开会地点改到咖啡厅" \
    "时间不变" \
    "确认" \
    "吃饭地点改成国贸" \
    "行" \
    "酒吧取消" \
    "直接回家" \
    "确认" \
    "查看安排" \
    "在回家前加个超市" \
    "买点东西" \
    "半小时" \
    "确认" \
    "没问题"

# 用户15: 健身计划
run_user_scenario 15 "健身计划" \
    "今天要去健身" \
    "下午4点" \
    "先跑步半小时" \
    "然后力量训练1小时" \
    "最后拉伸15分钟" \
    "确认" \
    "明天休息" \
    "后天继续练" \
    "换成游泳" \
    "游1小时" \
    "确认" \
    "今天健身后加个蛋白餐" \
    "6点吃" \
    "确认" \
    "把力量训练改成45分钟" \
    "行" \
    "查看健身安排" \
    "后天游泳改到上午" \
    "10点" \
    "确认"

# 用户16: 学习计划
run_user_scenario 16 "学习计划" \
    "今晚要学习" \
    "7点开始" \
    "先复习数学" \
    "1小时" \
    "然后看英语" \
    "也是1小时" \
    "最后做题" \
    "确认" \
    "数学太难了改半小时" \
    "英语多看点" \
    "1个半小时" \
    "确认" \
    "做题时间呢" \
    "1小时够了" \
    "确认" \
    "中间休息一下" \
    "每科之间休息10分钟" \
    "好" \
    "查看学习计划" \
    "没问题"

# 用户17: 社交活动
run_user_scenario 17 "社交活动" \
    "周末有很多聚会" \
    "周六中午同学聚餐" \
    "下午和闺蜜逛街" \
    "晚上家庭聚餐" \
    "确认周六" \
    "周日呢" \
    "上午和男朋友约会" \
    "中午一起吃饭" \
    "下午看电影" \
    "晚上各回各家" \
    "确认" \
    "周六闺蜜改成下午茶" \
    "3点到5点" \
    "确认" \
    "周日电影改成逛公园" \
    "行" \
    "查看周末安排" \
    "周六晚上家庭聚餐几点" \
    "7点开始" \
    "确认"

# 用户18: 出差安排
run_user_scenario 18 "出差安排" \
    "后天要出差" \
    "早上6点的飞机" \
    "9点到目的地" \
    "10点开会" \
    "中午和客户吃饭" \
    "下午继续谈" \
    "晚上飞回来" \
    "确认" \
    "飞机改成8点的" \
    "到的时间也变了" \
    "11点到" \
    "会议改到下午" \
    "2点开始" \
    "确认" \
    "晚上飞机几点" \
    "9点" \
    "到家大概12点" \
    "确认" \
    "查看出差安排" \
    "没问题"

# 用户19: 混合场景
run_user_scenario 19 "混合场景" \
    "今天很忙" \
    "上午在公司" \
    "中午出去办事" \
    "下午回来继续工作" \
    "晚上健身" \
    "然后回家" \
    "确认" \
    "明天呢" \
    "明天比较轻松" \
    "上午在家办公" \
    "下午出去逛逛" \
    "晚上和朋友吃饭" \
    "确认" \
    "今天健身取消" \
    "改成瑜伽" \
    "在家做" \
    "确认" \
    "明天逛街加个购物" \
    "买点东西" \
    "行"

# 用户20: 极端测试
run_user_scenario 20 "极端测试" \
    "啊啊啊" \
    "？？？" \
    "不知道" \
    "随便" \
    "都行" \
    "你说呢" \
    "帮我安排" \
    "今天下午" \
    "做点什么" \
    "3点吧" \
    "干嘛呢" \
    "健身" \
    "好" \
    "确认" \
    "改一下" \
    "不健身了" \
    "看电影" \
    "行" \
    "确认" \
    "就这样"

# ============== 生成总结报告 ==============

echo -e "\n\n${BLUE}============================================${NC}"
echo -e "${BLUE}              测试总结报告                  ${NC}"
echo -e "${BLUE}============================================${NC}\n"

total_turns=0
total_success=0
total_partial=0
total_fail=0

echo "| 用户 | 场景 | 总轮数 | 通过 | 部分通过 | 失败 | 通过率 |"
echo "|------|------|--------|------|----------|------|--------|"

for i in {1..20}; do
    if [ -n "${USER_RESULTS[$i]}" ]; then
        IFS='|' read -r scenario turns success partial fail <<< "${USER_RESULTS[$i]}"
        rate=$(echo "scale=1; $success * 100 / $turns" | bc)
        echo "| $i | $scenario | $turns | $success | $partial | $fail | ${rate}% |"

        total_turns=$((total_turns + turns))
        total_success=$((total_success + success))
        total_partial=$((total_partial + partial))
        total_fail=$((total_fail + fail))
    fi
done

echo ""
echo "总计: $total_turns 轮对话"
echo "通过: $total_success"
echo "部分通过: $total_partial"
echo "失败: $total_fail"
echo "总体通过率: $(echo "scale=1; $total_success * 100 / $total_turns" | bc)%"
