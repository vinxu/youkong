package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"youkong/internal/model"
)

// ========== 测试评估框架 ==========
//
// 目标：评估语音时刻表的智能程度，对标 Claude CLI /plan 能力
//
// 评估维度：
// 1. 意图理解准确性
// 2. 时间推理能力
// 3. 上下文保持能力
// 4. 多轮对话连贯性
// 5. 边界场景处理
// 6. 输出格式规范性

// TestCase 测试用例
type TestCase struct {
	Name            string                 // 测试名称
	Category        string                 // 分类
	Input           string                 // 用户输入
	CurrentSchedule []model.ScheduleItem   // 当前时刻表（模拟已有状态）
	Context         *model.CompressedUserContext // 用户上下文
	ConversationHistory []model.ConversationTurn // 对话历史

	// 预期结果
	ExpectedAction    string              // 预期动作: create/modify/clarify/guess
	ExpectedSchedule  []ExpectedItem      // 预期时刻表
	ExpectedQuestions []string            // 预期澄清问题（如果 action=clarify）

	// 评估标准
	MustContain       []string            // prompt 必须包含的内容
	MustTriggerThinking bool              // 是否必须触发深度思考

	// 人类预期说明
	HumanExpectation  string              // 人类对这个输入的自然预期
	CommonMistake     string              // LLM 常见错误
}

// ExpectedItem 预期时刻表条目
type ExpectedItem struct {
	StartTime string
	EndTime   string
	Status    string
	Flexible  bool   // 时间是否可以有一定灵活性
}

// ========== 测试用例集 ==========

func getTestCases() []TestCase {
	return []TestCase{
		// ========== 1. 基础意图理解 ==========
		{
			Name:     "基础-创建单个时段",
			Category: "基础",
			Input:    "下午2点到4点开会",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "14:00", EndTime: "16:00", Status: "开会"},
			},
			HumanExpectation: "明确的时间段，应该直接创建，不需要提问",
			CommonMistake:    "询问不必要的问题",
		},
		{
			Name:     "基础-创建多个时段",
			Category: "基础",
			Input:    "上午9点到12点工作，12点到1点吃午饭，下午2点到6点继续工作",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "09:00", EndTime: "12:00", Status: "工作"},
				{StartTime: "12:00", EndTime: "13:00", Status: "午餐"},
				{StartTime: "14:00", EndTime: "18:00", Status: "工作"},
			},
			HumanExpectation: "连续描述多个时段，应该全部正确解析",
			CommonMistake:    "遗漏某个时段或时间解析错误",
		},
		{
			Name:     "基础-当前状态描述",
			Category: "基础",
			Input:    "我在家休息",
			ExpectedAction: "guess",
			HumanExpectation: "没有具体时间安排，应该猜测当前状态",
			CommonMistake:    "强行创建时刻表",
		},

		// ========== 2. 睡眠时间推理 ==========
		{
			Name:     "睡眠-凌晨入睡",
			Category: "睡眠推理",
			Input:    "1-2点睡觉",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "02:00", EndTime: "09:00", Status: "睡觉", Flexible: true},
			},
			MustTriggerThinking: true,
			HumanExpectation: "1-2点表示入睡时间范围，应推理为2点开始睡到早上9点",
			CommonMistake:    "直接设置为 01:00-02:00，只有1小时睡眠",
		},
		{
			Name:     "睡眠-深夜入睡",
			Category: "睡眠推理",
			Input:    "晚上11点睡觉",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "23:00", EndTime: "09:00", Status: "睡觉", Flexible: true},
			},
			MustTriggerThinking: true,
			HumanExpectation: "晚上11点睡，应该睡到第二天早上",
			CommonMistake:    "不知道结束时间而提问",
		},
		{
			Name:     "睡眠-午休",
			Category: "睡眠推理",
			Input:    "中午1点到2点午休",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "13:00", EndTime: "14:00", Status: "午休"},
			},
			HumanExpectation: "明确说了结束时间，不需要推理",
			CommonMistake:    "把1点理解为凌晨",
		},
		{
			Name:     "睡眠-模糊表达",
			Category: "睡眠推理",
			Input:    "12点睡",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "00:00", EndTime: "09:00", Status: "睡觉", Flexible: true},
			},
			MustTriggerThinking: true,
			HumanExpectation: "12点睡通常指凌晨12点（午夜），应该睡到早上",
			CommonMistake:    "理解为中午12点",
		},

		// ========== 3. 时间歧义处理 ==========
		{
			Name:     "歧义-下午时间",
			Category: "时间歧义",
			Input:    "3点开会",
			Context: &model.CompressedUserContext{
				ProfileSummary: "上班族，朝九晚五",
			},
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "15:00", EndTime: "17:00", Status: "开会", Flexible: true},
			},
			HumanExpectation: "上班族说3点开会，通常是下午3点",
			CommonMistake:    "不确定而提问是上午还是下午",
		},
		{
			Name:     "歧义-早上时间",
			Category: "时间歧义",
			Input:    "7点起床，8点出门",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "07:00", EndTime: "08:00", Status: "起床/准备"},
				{StartTime: "08:00", EndTime: "09:00", Status: "出门/通勤", Flexible: true},
			},
			HumanExpectation: "7点8点明显是早上",
			CommonMistake:    "询问是早上还是晚上",
		},

		// ========== 4. 活动时长推理 ==========
		{
			Name:     "时长-开会无结束时间",
			Category: "时长推理",
			Input:    "下午3点开会",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "15:00", EndTime: "17:00", Status: "开会", Flexible: true},
			},
			HumanExpectation: "没说结束时间，会议默认2小时",
			CommonMistake:    "询问结束时间",
		},
		{
			Name:     "时长-吃饭无时长",
			Category: "时长推理",
			Input:    "12点吃午饭",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "12:00", EndTime: "13:00", Status: "午餐"},
			},
			HumanExpectation: "午餐默认1小时",
			CommonMistake:    "询问吃多久",
		},
		{
			Name:     "时长-健身无时长",
			Category: "时长推理",
			Input:    "晚上7点去健身",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "19:00", EndTime: "20:30", Status: "健身", Flexible: true},
			},
			HumanExpectation: "健身默认1-1.5小时",
			CommonMistake:    "询问健身多久",
		},

		// ========== 5. 修改保留上下文 ==========
		{
			Name:     "修改-替换单个时段",
			Category: "修改保留",
			Input:    "把下午改成开会",
			CurrentSchedule: []model.ScheduleItem{
				{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "工作"},
				{StartTime: "12:00", EndTime: "13:00", Emoji: "🍚", Status: "午餐"},
				{StartTime: "14:00", EndTime: "18:00", Emoji: "💼", Status: "工作"},
			},
			ExpectedAction: "create", // 或 modify
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "09:00", EndTime: "12:00", Status: "工作"},
				{StartTime: "12:00", EndTime: "13:00", Status: "午餐"},
				{StartTime: "14:00", EndTime: "18:00", Status: "开会"},
			},
			MustTriggerThinking: true,
			MustContain: []string{"当前完整时刻表", "保留未提及的时段"},
			HumanExpectation: "只修改下午的工作为开会，保留上午和午餐",
			CommonMistake:    "只返回14:00-18:00开会，丢失其他时段",
		},
		{
			Name:     "修改-调整时间",
			Category: "修改保留",
			Input:    "午餐改成12点半",
			CurrentSchedule: []model.ScheduleItem{
				{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "工作"},
				{StartTime: "12:00", EndTime: "13:00", Emoji: "🍚", Status: "午餐"},
				{StartTime: "14:00", EndTime: "18:00", Emoji: "💼", Status: "工作"},
			},
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "09:00", EndTime: "12:30", Status: "工作"},
				{StartTime: "12:30", EndTime: "13:30", Status: "午餐"},
				{StartTime: "14:00", EndTime: "18:00", Status: "工作"},
			},
			MustTriggerThinking: true,
			HumanExpectation: "午餐推迟到12:30，工作也要相应调整",
			CommonMistake:    "只改午餐时间，不调整相邻时段",
		},
		{
			Name:     "修改-删除时段",
			Category: "修改保留",
			Input:    "取消午餐",
			CurrentSchedule: []model.ScheduleItem{
				{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "工作"},
				{StartTime: "12:00", EndTime: "13:00", Emoji: "🍚", Status: "午餐"},
				{StartTime: "14:00", EndTime: "18:00", Emoji: "💼", Status: "工作"},
			},
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "09:00", EndTime: "12:00", Status: "工作"},
				{StartTime: "14:00", EndTime: "18:00", Status: "工作"},
			},
			MustTriggerThinking: true,
			HumanExpectation: "删除午餐，保留其他时段",
			CommonMistake:    "返回空时刻表或只返回午餐时段",
		},
		{
			Name:     "修改-添加时段",
			Category: "修改保留",
			Input:    "下午3点加个茶歇",
			CurrentSchedule: []model.ScheduleItem{
				{StartTime: "14:00", EndTime: "18:00", Emoji: "💼", Status: "工作"},
			},
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "14:00", EndTime: "15:00", Status: "工作"},
				{StartTime: "15:00", EndTime: "15:30", Status: "茶歇", Flexible: true},
				{StartTime: "15:30", EndTime: "18:00", Status: "工作"},
			},
			MustTriggerThinking: true,
			HumanExpectation: "在原有工作中间插入茶歇，拆分工作时段",
			CommonMistake:    "只添加茶歇，丢失工作时段",
		},

		// ========== 6. 连续活动衔接 ==========
		{
			Name:     "衔接-开完会去吃饭",
			Category: "活动衔接",
			Input:    "下午2点开会，开完会去吃饭",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "14:00", EndTime: "16:00", Status: "开会", Flexible: true},
				{StartTime: "16:00", EndTime: "17:00", Status: "吃饭", Flexible: true},
			},
			MustTriggerThinking: true,
			HumanExpectation: "吃饭紧跟在开会之后，时间自动衔接",
			CommonMistake:    "询问吃饭几点",
		},
		{
			Name:     "衔接-然后语法",
			Category: "活动衔接",
			Input:    "先健身，然后洗澡，然后吃晚饭",
			ExpectedAction: "clarify", // 需要知道开始时间
			ExpectedQuestions: []string{"健身", "开始", "时间"},
			HumanExpectation: "连续活动但缺少起始时间，应该询问健身开始时间",
			CommonMistake:    "随意假设一个开始时间",
		},
		{
			Name:     "衔接-有起始时间的连续",
			Category: "活动衔接",
			Input:    "6点健身，然后洗澡，然后吃晚饭",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "18:00", EndTime: "19:30", Status: "健身", Flexible: true},
				{StartTime: "19:30", EndTime: "20:00", Status: "洗澡", Flexible: true},
				{StartTime: "20:00", EndTime: "21:00", Status: "晚餐", Flexible: true},
			},
			MustTriggerThinking: true,
			HumanExpectation: "有起始时间，后续活动自动衔接",
			CommonMistake:    "询问每个活动的时间",
		},

		// ========== 7. 复杂场景 ==========
		{
			Name:     "复杂-一天规划",
			Category: "复杂场景",
			Input:    "明天早上9点开始工作，中午吃饭，下午继续工作，5点下班后去健身，晚上回家看电影",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "09:00", EndTime: "12:00", Status: "工作", Flexible: true},
				{StartTime: "12:00", EndTime: "13:00", Status: "午餐"},
				{StartTime: "13:00", EndTime: "17:00", Status: "工作", Flexible: true},
				{StartTime: "17:00", EndTime: "18:30", Status: "健身", Flexible: true},
				{StartTime: "19:00", EndTime: "21:30", Status: "看电影", Flexible: true},
			},
			MustTriggerThinking: true,
			HumanExpectation: "完整的一天规划，应该合理填充时间",
			CommonMistake:    "遗漏某些活动或时间安排不合理",
		},
		{
			Name:     "复杂-带条件的安排",
			Category: "复杂场景",
			Input:    "如果3点前能开完会，就去健身，否则直接回家",
			ExpectedAction: "clarify",
			HumanExpectation: "条件性安排，应该询问用户想要哪种情况",
			CommonMistake:    "试图同时创建两种可能",
		},

		// ========== 8. 边界场景 ==========
		{
			Name:     "边界-跨天时间",
			Category: "边界场景",
			Input:    "今晚10点到明早6点值班",
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "22:00", EndTime: "06:00", Status: "值班"},
			},
			HumanExpectation: "跨越午夜的时间段应该正确处理",
			CommonMistake:    "无法处理跨天时间",
		},
		{
			Name:     "边界-时间重叠",
			Category: "边界场景",
			Input:    "2点到4点开会，3点有个电话会议",
			ExpectedAction: "clarify",
			HumanExpectation: "时间冲突，应该询问如何处理",
			CommonMistake:    "忽略冲突直接创建",
		},
		{
			Name:     "边界-模糊不清",
			Category: "边界场景",
			Input:    "下午有点事",
			ExpectedAction: "clarify",
			ExpectedQuestions: []string{"什么事", "几点"},
			HumanExpectation: "信息太模糊，需要澄清",
			CommonMistake:    "随意猜测一个安排",
		},

		// ========== 9. 多轮对话 ==========
		{
			Name:     "多轮-追加信息",
			Category: "多轮对话",
			Input:    "对了，开会前还要准备一下",
			CurrentSchedule: []model.ScheduleItem{
				{StartTime: "14:00", EndTime: "16:00", Emoji: "💼", Status: "开会"},
			},
			ConversationHistory: []model.ConversationTurn{
				{Role: "user", Content: "下午2点开会"},
				{Role: "assistant", Content: "已创建：14:00-16:00 开会"},
			},
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "13:30", EndTime: "14:00", Status: "准备会议", Flexible: true},
				{StartTime: "14:00", EndTime: "16:00", Status: "开会"},
			},
			MustTriggerThinking: true,
			HumanExpectation: "基于上下文，在开会前添加准备时间",
			CommonMistake:    "不知道'开会'指的是哪个，询问",
		},
		{
			Name:     "多轮-否定修改",
			Category: "多轮对话",
			Input:    "不对，是3点开会，不是2点",
			CurrentSchedule: []model.ScheduleItem{
				{StartTime: "14:00", EndTime: "16:00", Emoji: "💼", Status: "开会"},
			},
			ConversationHistory: []model.ConversationTurn{
				{Role: "user", Content: "下午2点开会"},
				{Role: "assistant", Content: "已创建：14:00-16:00 开会"},
			},
			ExpectedAction: "create",
			ExpectedSchedule: []ExpectedItem{
				{StartTime: "15:00", EndTime: "17:00", Status: "开会"},
			},
			MustTriggerThinking: true,
			HumanExpectation: "理解用户在纠正错误，修改时间",
			CommonMistake:    "创建两个开会时段",
		},
	}
}

// ========== 评估函数 ==========

// EvaluationResult 评估结果
type EvaluationResult struct {
	TestCase        *TestCase
	PromptGenerated string
	ThinkingTriggered bool
	MissingContents []string
	Score           int    // 0-100
	Issues          []string
	Suggestions     []string
}

// EvaluateTestCase 评估单个测试用例
func EvaluateTestCase(s *VoiceScheduleService, tc *TestCase) *EvaluationResult {
	result := &EvaluationResult{
		TestCase: tc,
		Score:    100,
		Issues:   []string{},
		Suggestions: []string{},
	}

	// 构建 session
	session := &model.VoiceScheduleSession{
		CurrentSchedule:     tc.CurrentSchedule,
		ConversationHistory: tc.ConversationHistory,
	}

	ctx := tc.Context
	if ctx == nil {
		ctx = &model.CompressedUserContext{}
	}

	// 生成 prompt
	prompt := s.buildPlanModePrompt(tc.Input, ctx, session)

	// 检查深度思考触发
	thinkingTriggered := s.shouldUseThinking(tc.Input, ctx, session)
	result.ThinkingTriggered = thinkingTriggered

	if thinkingTriggered {
		prompt = s.addDeepThinkingInstructions(prompt, tc.Input, session)
	}

	result.PromptGenerated = prompt

	// 评估1：必须触发深度思考
	if tc.MustTriggerThinking && !thinkingTriggered {
		result.Score -= 20
		result.Issues = append(result.Issues, "应该触发深度思考但没有触发")
		result.Suggestions = append(result.Suggestions, "在 shouldUseThinking 中添加对应检测逻辑")
	}

	// 评估2：必须包含的内容
	for _, content := range tc.MustContain {
		if !strings.Contains(prompt, content) {
			result.Score -= 10
			result.MissingContents = append(result.MissingContents, content)
			result.Issues = append(result.Issues, fmt.Sprintf("Prompt 缺少必要内容: %s", content))
		}
	}

	// 评估3：当前时刻表是否在 prompt 中
	if len(tc.CurrentSchedule) > 0 {
		foundSchedule := true
		for _, item := range tc.CurrentSchedule {
			if !strings.Contains(prompt, item.StartTime) || !strings.Contains(prompt, item.Status) {
				foundSchedule = false
				break
			}
		}
		if !foundSchedule {
			result.Score -= 15
			result.Issues = append(result.Issues, "当前时刻表未完整出现在 Prompt 中")
			result.Suggestions = append(result.Suggestions, "确保 buildPlanModePrompt 正确展示当前时刻表")
		}
	}

	// 评估4：时间推理规则检查
	timeReasoningKeywords := []string{"睡眠推理", "时间歧义", "活动时长", "修改时保留上下文"}
	foundRules := 0
	for _, kw := range timeReasoningKeywords {
		if strings.Contains(prompt, kw) {
			foundRules++
		}
	}
	if foundRules < 3 {
		result.Score -= 10
		result.Issues = append(result.Issues, "时间推理规则不完整")
	}

	// 评估5：对话历史检查
	if len(tc.ConversationHistory) > 0 {
		hasHistory := strings.Contains(prompt, "对话历史") || strings.Contains(prompt, "历史")
		if !hasHistory {
			result.Score -= 10
			result.Issues = append(result.Issues, "对话历史未在 Prompt 中体现")
		}
	}

	return result
}

// ========== 测试执行 ==========

func TestFullEvaluation(t *testing.T) {
	s := &VoiceScheduleService{}
	testCases := getTestCases()

	// 按类别统计
	categoryScores := make(map[string][]int)
	categoryIssues := make(map[string][]string)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("语音时刻表智能评估报告")
	fmt.Println(strings.Repeat("=", 80))

	for _, tc := range testCases {
		result := EvaluateTestCase(s, &tc)

		categoryScores[tc.Category] = append(categoryScores[tc.Category], result.Score)
		for _, issue := range result.Issues {
			categoryIssues[tc.Category] = append(categoryIssues[tc.Category], issue)
		}

		// 输出单个测试结果
		status := "✅"
		if result.Score < 80 {
			status = "⚠️"
		}
		if result.Score < 60 {
			status = "❌"
		}

		fmt.Printf("\n%s [%s] %s (得分: %d)\n", status, tc.Category, tc.Name, result.Score)
		fmt.Printf("   输入: %s\n", tc.Input)
		fmt.Printf("   人类预期: %s\n", tc.HumanExpectation)
		fmt.Printf("   深度思考: %v\n", result.ThinkingTriggered)

		if len(result.Issues) > 0 {
			fmt.Printf("   问题:\n")
			for _, issue := range result.Issues {
				fmt.Printf("     - %s\n", issue)
			}
		}

		if tc.CommonMistake != "" {
			fmt.Printf("   常见错误: %s\n", tc.CommonMistake)
		}
	}

	// 输出分类统计
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("分类评估汇总")
	fmt.Println(strings.Repeat("=", 80))

	totalScore := 0
	totalCount := 0

	for category, scores := range categoryScores {
		sum := 0
		for _, s := range scores {
			sum += s
		}
		avg := sum / len(scores)
		totalScore += sum
		totalCount += len(scores)

		status := "✅"
		if avg < 80 {
			status = "⚠️"
		}
		if avg < 60 {
			status = "❌"
		}

		fmt.Printf("\n%s %s (平均分: %d, 用例数: %d)\n", status, category, avg, len(scores))

		// 该分类的主要问题
		issueCount := make(map[string]int)
		for _, issue := range categoryIssues[category] {
			issueCount[issue]++
		}
		if len(issueCount) > 0 {
			fmt.Printf("   主要问题:\n")
			for issue, count := range issueCount {
				fmt.Printf("     - [%dx] %s\n", count, issue)
			}
		}
	}

	overallScore := totalScore / totalCount
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("总体评分: %d/100\n", overallScore)
	fmt.Println(strings.Repeat("=", 80))

	// 生成改进建议
	fmt.Println("\n改进建议:")
	suggestions := generateImprovementSuggestions(categoryScores, categoryIssues)
	for i, s := range suggestions {
		fmt.Printf("%d. %s\n", i+1, s)
	}
}

// generateImprovementSuggestions 生成改进建议
func generateImprovementSuggestions(scores map[string][]int, issues map[string][]string) []string {
	suggestions := []string{}

	// 找出低分类别
	for category, scoreList := range scores {
		sum := 0
		for _, s := range scoreList {
			sum += s
		}
		avg := sum / len(scoreList)

		if avg < 80 {
			switch category {
			case "睡眠推理":
				suggestions = append(suggestions, "增强睡眠时间推理能力，特别是入睡时间到起床时间的自动计算")
			case "修改保留":
				suggestions = append(suggestions, "强化修改场景的上下文保持，确保未提及的时段不被丢弃")
			case "活动衔接":
				suggestions = append(suggestions, "改进连续活动的时间衔接逻辑")
			case "时间歧义":
				suggestions = append(suggestions, "增加更多时间歧义处理规则，结合用户画像判断")
			case "多轮对话":
				suggestions = append(suggestions, "优化对话历史的利用，更好地理解指代关系")
			}
		}
	}

	// 通用建议
	suggestions = append(suggestions, "考虑引入 Few-shot 示例到 Prompt 中")
	suggestions = append(suggestions, "对于复杂场景，增加 Chain-of-Thought 推理引导")

	return suggestions
}

// TestPromptQuality 测试 prompt 质量
func TestPromptQuality(t *testing.T) {
	s := &VoiceScheduleService{}

	// 选择几个关键场景，打印完整 prompt 供人工检查
	keyScenarios := []struct {
		name  string
		input string
		schedule []model.ScheduleItem
	}{
		{
			name:  "睡眠推理",
			input: "1-2点睡觉",
		},
		{
			name:  "修改保留上下文",
			input: "把下午改成开会",
			schedule: []model.ScheduleItem{
				{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "工作"},
				{StartTime: "12:00", EndTime: "13:00", Emoji: "🍚", Status: "午餐"},
				{StartTime: "14:00", EndTime: "18:00", Emoji: "💼", Status: "工作"},
			},
		},
		{
			name:  "连续活动",
			input: "6点健身，然后洗澡，然后吃晚饭",
		},
	}

	for _, scenario := range keyScenarios {
		session := &model.VoiceScheduleSession{
			CurrentSchedule: scenario.schedule,
		}
		ctx := &model.CompressedUserContext{}

		prompt := s.buildPlanModePrompt(scenario.input, ctx, session)
		thinkingTriggered := s.shouldUseThinking(scenario.input, ctx, session)
		if thinkingTriggered {
			prompt = s.addDeepThinkingInstructions(prompt, scenario.input, session)
		}

		fmt.Printf("\n%s\n", strings.Repeat("=", 80))
		fmt.Printf("场景: %s\n", scenario.name)
		fmt.Printf("输入: %s\n", scenario.input)
		fmt.Printf("深度思考: %v\n", thinkingTriggered)
		fmt.Printf("%s\n", strings.Repeat("-", 80))
		fmt.Printf("生成的 Prompt:\n%s\n", prompt)
	}
}

// TestExpectedLLMOutput 模拟测试 LLM 预期输出
func TestExpectedLLMOutput(t *testing.T) {
	// 这里定义我们期望 LLM 对特定输入的输出
	// 用于对比实际 LLM 输出是否符合预期

	expectedOutputs := []struct {
		input    string
		expected string
	}{
		{
			input: "1-2点睡觉",
			expected: `{
  "action": "create",
  "schedule": [
    {"start_time": "02:00", "end_time": "09:00", "emoji": "😴", "status": "睡觉"}
  ],
  "reasoning": ["1-2点表示入睡时间范围，取2点作为入睡时间", "凌晨入睡默认睡到早上9点"]
}`,
		},
		{
			input: "把下午改成开会（已有时刻表：9-12工作，12-13午餐，14-18工作）",
			expected: `{
  "action": "create",
  "schedule": [
    {"start_time": "09:00", "end_time": "12:00", "emoji": "💼", "status": "工作"},
    {"start_time": "12:00", "end_time": "13:00", "emoji": "🍚", "status": "午餐"},
    {"start_time": "14:00", "end_time": "18:00", "emoji": "💼", "status": "开会"}
  ],
  "reasoning": ["用户要求修改下午时段", "保留上午工作和午餐不变", "将下午的工作改为开会"]
}`,
		},
	}

	fmt.Println("\n预期 LLM 输出示例（用于对比实际输出）:")
	for _, eo := range expectedOutputs {
		fmt.Printf("\n输入: %s\n", eo.input)
		fmt.Printf("预期输出:\n%s\n", eo.expected)

		// 验证 JSON 格式正确
		var result model.LLMVoiceAnalysisResult
		if err := json.Unmarshal([]byte(eo.expected), &result); err != nil {
			t.Errorf("预期输出 JSON 格式错误: %v", err)
		}
	}
}
