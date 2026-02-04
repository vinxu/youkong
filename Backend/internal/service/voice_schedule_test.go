package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"youkong/internal/model"
)

// ============ 新增：时间格式验证测试 ============

// TestTimeFormatValidation 测试时间格式验证
func TestTimeFormatValidation(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
		reason   string
	}{
		// 正确格式
		{"09:00", true, "标准格式"},
		{"23:59", true, "接近午夜"},
		{"00:00", true, "午夜"},
		{"12:30", true, "中午"},
		{"01:05", true, "凌晨带前导零"},

		// 错误格式 - 缺少前导零
		{"9:00", false, "小时缺少前导零"},
		{"12:5", false, "分钟缺少前导零"},
		{"9:5", false, "小时和分钟都缺少前导零"},

		// 错误格式 - 超出范围
		{"19:60", false, "分钟超出范围（60）"},
		{"24:00", false, "小时超出范围（24）"},
		{"25:30", false, "小时超出范围（25）"},
		{"12:99", false, "分钟超出范围（99）"},

		// 错误格式 - 格式错误
		{"12:00:00", false, "包含秒"},
		{"1200", false, "缺少冒号"},
		{"12：00", false, "中文冒号"},
		{"", false, "空字符串"},
		{"abc", false, "非数字"},
		{"12:0a", false, "分钟包含字母"},
	}

	for _, tc := range testCases {
		t.Run(tc.input+"_"+tc.reason, func(t *testing.T) {
			valid, errMsg := model.ValidateTimeHHMM(tc.input)
			if valid != tc.expected {
				t.Errorf("ValidateTimeHHMM(%q) = %v, 期望 %v, 原因: %s, 错误: %s",
					tc.input, valid, tc.expected, tc.reason, errMsg)
			}
		})
	}
}

// TestValidateScheduleItems 测试时刻表项目验证
func TestValidateScheduleItems(t *testing.T) {
	testCases := []struct {
		name     string
		items    []model.ScheduleItem
		expected bool
	}{
		{
			name: "正确的时刻表",
			items: []model.ScheduleItem{
				{StartTime: "09:00", EndTime: "12:00", Status: "开会", Emoji: "💼"},
				{StartTime: "12:00", EndTime: "13:00", Status: "午餐", Emoji: "🍚"},
			},
			expected: true,
		},
		{
			name: "时间格式错误（缺少前导零）",
			items: []model.ScheduleItem{
				{StartTime: "9:00", EndTime: "12:00", Status: "开会", Emoji: "💼"},
			},
			expected: false,
		},
		{
			name: "分钟超出范围",
			items: []model.ScheduleItem{
				{StartTime: "19:60", EndTime: "20:00", Status: "健身", Emoji: "🏃"},
			},
			expected: false,
		},
		{
			name: "状态为空",
			items: []model.ScheduleItem{
				{StartTime: "09:00", EndTime: "12:00", Status: "", Emoji: "💼"},
			},
			expected: false,
		},
		{
			name: "Emoji为空",
			items: []model.ScheduleItem{
				{StartTime: "09:00", EndTime: "12:00", Status: "开会", Emoji: ""},
			},
			expected: false,
		},
		{
			name:     "空时刻表",
			items:    []model.ScheduleItem{},
			expected: true, // 空时刻表是合法的
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := model.ValidateScheduleItems(tc.items)
			if result.Valid != tc.expected {
				t.Errorf("ValidateScheduleItems() = %v, 期望 %v, 错误: %s",
					result.Valid, tc.expected, result.Error())
			}
		})
	}
}

// ============ 新增：多轮对话状态累积测试 ============

// TestMultiTurnScheduleAccumulation 测试多轮对话状态累积
func TestMultiTurnScheduleAccumulation(t *testing.T) {
	s := &VoiceScheduleService{}

	// 模拟多轮对话
	session := &model.VoiceScheduleSession{
		UserID:    "test-user",
		SessionID: "test-session",
		CurrentSchedule: []model.ScheduleItem{
			{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "开会"},
		},
		ConversationHistory: []model.ConversationTurn{
			{Role: "user", Content: "上午开会"},
			{Role: "assistant", Content: "好的，已安排09:00-12:00开会"},
		},
	}
	ctx := &model.CompressedUserContext{}

	// 用户继续添加新的安排
	transcript := "下午3点健身"

	// 生成 Prompt
	prompt := s.buildPlanModePrompt(transcript, ctx, session)

	// 验证 Prompt 包含之前的安排
	if !strings.Contains(prompt, "09:00-12:00") {
		t.Error("Prompt 应该包含之前的安排（09:00-12:00 开会）")
	}
	if !strings.Contains(prompt, "开会") {
		t.Error("Prompt 应该包含之前的活动描述（开会）")
	}

	// 验证对话历史被包含
	if !strings.Contains(prompt, "上午开会") || !strings.Contains(prompt, "对话历史") {
		t.Error("Prompt 应该包含对话历史")
	}

	fmt.Println("✅ 多轮对话状态累积测试通过")
}

// TestConversationHistorySummary 测试对话历史摘要生成
func TestConversationHistorySummary(t *testing.T) {
	s := &VoiceScheduleService{}

	// 创建一个有多轮对话的会话
	session := &model.VoiceScheduleSession{
		UserID:    "test-user",
		SessionID: "test-session",
		ConversationHistory: []model.ConversationTurn{
			{Role: "user", Content: "上午开会"},
			{Role: "assistant", Content: "好的"},
			{Role: "user", Content: "下午健身"},
			{Role: "assistant", Content: "好的"},
			{Role: "user", Content: "晚上看电影"},
		},
	}

	// 生成摘要
	summary := s.generateHistorySummary(session.ConversationHistory)

	// 验证摘要包含用户的请求
	if !strings.Contains(summary, "上午开会") {
		t.Error("摘要应该包含第一个请求")
	}
	if !strings.Contains(summary, "下午健身") {
		t.Error("摘要应该包含第二个请求")
	}
	if !strings.Contains(summary, "晚上看电影") {
		t.Error("摘要应该包含第三个请求")
	}
	if !strings.Contains(summary, "→") {
		t.Error("摘要应该用箭头连接请求")
	}

	fmt.Printf("生成的摘要: %s\n", summary)
	fmt.Println("✅ 对话历史摘要测试通过")
}

// ============ 新增：修改操作测试 ============

// TestModifyOperationWithIndex 测试带索引的修改操作
func TestModifyOperationWithIndex(t *testing.T) {
	s := &VoiceScheduleService{}

	session := &model.VoiceScheduleSession{
		CurrentSchedule: []model.ScheduleItem{
			{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "开会"},
			{StartTime: "12:00", EndTime: "13:00", Emoji: "🍚", Status: "午餐"},
			{StartTime: "14:00", EndTime: "18:00", Emoji: "💻", Status: "工作"},
		},
	}
	ctx := &model.CompressedUserContext{}

	// 用户要修改第一个时段
	transcript := "把开会改到10点"

	prompt := s.buildPlanModePrompt(transcript, ctx, session)

	// 验证 Prompt 包含索引信息
	if !strings.Contains(prompt, "[0]") {
		t.Error("Prompt 应该包含时段索引 [0]")
	}
	if !strings.Contains(prompt, "[1]") {
		t.Error("Prompt 应该包含时段索引 [1]")
	}
	if !strings.Contains(prompt, "[2]") {
		t.Error("Prompt 应该包含时段索引 [2]")
	}

	// 验证修改规则
	if !strings.Contains(prompt, "target_index") {
		t.Error("Prompt 应该包含 target_index 说明")
	}
	if !strings.Contains(prompt, "operation=\"modify\"") {
		t.Error("Prompt 应该包含 operation=\"modify\" 说明")
	}

	fmt.Println("✅ 修改操作索引测试通过")
}

// ============ 新增：时间格式 Few-shot 示例测试 ============

// TestTimeFormatExamplesInPrompt 测试 Prompt 中包含时间格式示例
func TestTimeFormatExamplesInPrompt(t *testing.T) {
	s := &VoiceScheduleService{}
	session := &model.VoiceScheduleSession{}
	ctx := &model.CompressedUserContext{}

	prompt := s.buildPlanModePrompt("下午3点开会", ctx, session)

	// 验证包含时间格式要求
	timeFormatRequired := []string{
		"HH:MM",
		"09:00",      // 正确示例
		"23:59",      // 正确示例
		"9:00",       // 错误示例
		"19:60",      // 错误示例
		"00-23",      // 小时范围
		"00-59",      // 分钟范围
	}

	for _, required := range timeFormatRequired {
		if !strings.Contains(prompt, required) {
			t.Errorf("Prompt 应该包含时间格式说明: %s", required)
		}
	}

	fmt.Println("✅ 时间格式示例测试通过")
}

// ============ 新增：日期格式验证测试 ============

// TestDateFormatValidation 测试日期格式验证
func TestDateFormatValidation(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
		reason   string
	}{
		{"2026-02-05", true, "标准格式"},
		{"2024-12-31", true, "年末"},
		{"2025-01-01", true, "年初"},
		{"2026-2-5", false, "缺少前导零"},
		{"26-02-05", false, "年份不完整"},
		{"2026/02/05", false, "错误的分隔符"},
		{"20260205", false, "缺少分隔符"},
		{"", false, "空字符串"},
		{"2026-13-05", false, "月份超出范围"},
		{"2026-02-32", false, "日期超出范围"},
	}

	for _, tc := range testCases {
		t.Run(tc.input+"_"+tc.reason, func(t *testing.T) {
			valid, errMsg := model.ValidateDateFormat(tc.input)
			if valid != tc.expected {
				t.Errorf("ValidateDateFormat(%q) = %v, 期望 %v, 原因: %s, 错误: %s",
					tc.input, valid, tc.expected, tc.reason, errMsg)
			}
		})
	}
}

// ============ 新增：重试 Prompt 测试 ============

// TestBuildRetryPrompt 测试重试 Prompt 构建
func TestBuildRetryPrompt(t *testing.T) {
	s := &VoiceScheduleService{}

	originalPrompt := "原始 Prompt 内容"
	userInput := "下午3点开会"
	validationError := "schedule[0].start_time: 时间长度必须是 5 字符，当前是 4 字符"

	retryPrompt := s.buildRetryPrompt(originalPrompt, userInput, validationError)

	// 验证重试 Prompt 包含错误信息
	if !strings.Contains(retryPrompt, "格式错误") {
		t.Error("重试 Prompt 应该包含格式错误提示")
	}
	if !strings.Contains(retryPrompt, validationError) {
		t.Error("重试 Prompt 应该包含具体的验证错误")
	}
	if !strings.Contains(retryPrompt, userInput) {
		t.Error("重试 Prompt 应该包含用户原始输入")
	}
	if !strings.Contains(retryPrompt, "HH:MM") {
		t.Error("重试 Prompt 应该强调时间格式要求")
	}

	fmt.Println("✅ 重试 Prompt 测试通过")
}

// ============ 新增：Session HistorySummary 测试 ============

// TestSessionHistorySummaryUpdate 测试会话历史摘要更新
func TestSessionHistorySummaryUpdate(t *testing.T) {
	s := &VoiceScheduleService{}

	session := &model.VoiceScheduleSession{
		UserID:              "test-user",
		SessionID:           "test-session",
		ConversationHistory: []model.ConversationTurn{},
	}

	// 添加多轮对话
	turns := []struct {
		role    string
		content string
	}{
		{"user", "上午9点开会"},
		{"assistant", "好的"},
		{"user", "中午12点吃饭"},
		{"assistant", "好的"},
		{"user", "下午3点健身"},
		{"assistant", "好的"},
		{"user", "晚上看电影"}, // 第4轮用户输入
	}

	for _, turn := range turns {
		s.addConversationTurn(session, turn.role, turn.content)
	}

	// 超过 3 轮后应该生成摘要
	if session.HistorySummary == "" {
		t.Error("超过 3 轮对话后应该生成历史摘要")
	}

	fmt.Printf("会话历史摘要: %s\n", session.HistorySummary)
	fmt.Println("✅ Session HistorySummary 更新测试通过")
}

// ============ 辅助函数 ============

// init 确保测试时使用正确的时区
func init() {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	time.Local = loc
}

// TestPromptGeneration 测试提示词生成逻辑
func TestPromptGeneration(t *testing.T) {
	s := &VoiceScheduleService{}

	// 测试场景
	testCases := []struct {
		name           string
		transcript     string
		currentSchedule []model.ScheduleItem
		expectedContains []string
		description    string
	}{
		{
			name:       "睡眠推理场景",
			transcript: "1-2点睡觉",
			expectedContains: []string{
				"睡眠推理",
				"凌晨(0:00-4:00)入睡 → 睡到 09:00",
				"start_time\":\"02:00\"",
				"end_time\":\"09:00\"",
			},
			description: "用户说'1-2点睡觉'，应该推理为02:00-09:00睡觉",
		},
		{
			name:       "修改保留上下文场景",
			transcript: "把下午改成开会",
			currentSchedule: []model.ScheduleItem{
				{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "工作"},
				{StartTime: "12:00", EndTime: "13:00", Emoji: "🍚", Status: "午餐"},
				{StartTime: "14:00", EndTime: "18:00", Emoji: "💼", Status: "工作"},
			},
			expectedContains: []string{
				"当前完整时刻表【修改/删除时必须基于此表】",
				"[0] 09:00-12:00 💼 工作",
				"[1] 12:00-13:00 🍚 午餐",
				"[2] 14:00-18:00 💼 工作",
				"保留未提及的时段",
				"target_index",
			},
			description: "修改时应该显示带索引的完整时刻表并强调保留未提及的时段",
		},
		{
			name:       "时间歧义场景",
			transcript: "下午3点开会",
			expectedContains: []string{
				"时间歧义处理",
				"下午X点",
			},
			description: "下午3点应该被正确理解为15:00",
		},
		{
			name:       "连续活动场景",
			transcript: "开完会去吃饭",
			expectedContains: []string{
				"时间衔接",
				"会议结束时间 = 吃饭开始时间",
			},
			description: "连续活动应该自动衔接时间",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			session := &model.VoiceScheduleSession{
				CurrentSchedule: tc.currentSchedule,
				ConversationHistory: []model.ConversationTurn{},
			}

			compressedCtx := &model.CompressedUserContext{}

			// 生成提示词
			prompt := s.buildPlanModePrompt(tc.transcript, compressedCtx, session)

			// 检查是否需要深度思考
			needThinking := s.shouldUseThinking(tc.transcript, compressedCtx, session)
			if needThinking {
				prompt = s.addDeepThinkingInstructions(prompt, tc.transcript, session)
			}

			// 验证关键内容
			fmt.Printf("\n=== %s ===\n", tc.name)
			fmt.Printf("描述: %s\n", tc.description)
			fmt.Printf("输入: %s\n", tc.transcript)
			fmt.Printf("需要深度思考: %v\n", needThinking)

			allFound := true
			for _, expected := range tc.expectedContains {
				if !strings.Contains(prompt, expected) {
					t.Errorf("提示词缺少预期内容: %s", expected)
					allFound = false
				}
			}

			if allFound {
				fmt.Println("✅ 所有预期内容都包含在提示词中")
			}

			// 打印部分提示词用于人工检查
			if len(prompt) > 500 {
				fmt.Printf("\n提示词片段（前500字符）:\n%s...\n", prompt[:500])
			} else {
				fmt.Printf("\n提示词:\n%s\n", prompt)
			}
		})
	}
}

// TestSleepTimeReasoning 专门测试睡眠时间推理
func TestSleepTimeReasoning(t *testing.T) {
	s := &VoiceScheduleService{}
	session := &model.VoiceScheduleSession{}
	ctx := &model.CompressedUserContext{}

	sleepInputs := []string{
		"1-2点睡觉",
		"12点睡",
		"凌晨1点睡",
		"晚上11点睡觉",
	}

	for _, input := range sleepInputs {
		prompt := s.buildPlanModePrompt(input, ctx, session)
		needThinking := s.shouldUseThinking(input, ctx, session)

		fmt.Printf("\n输入: %s\n", input)
		fmt.Printf("需要深度思考: %v\n", needThinking)

		// 验证提示词包含睡眠推理规则
		if !strings.Contains(prompt, "睡眠推理") {
			t.Errorf("提示词缺少睡眠推理规则，输入: %s", input)
		}
	}
}

// TestModificationPreservation 测试修改时保留上下文
func TestModificationPreservation(t *testing.T) {
	s := &VoiceScheduleService{}

	session := &model.VoiceScheduleSession{
		CurrentSchedule: []model.ScheduleItem{
			{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "工作"},
			{StartTime: "12:00", EndTime: "13:00", Emoji: "🍚", Status: "午餐"},
			{StartTime: "14:00", EndTime: "18:00", Emoji: "💼", Status: "工作"},
		},
	}
	ctx := &model.CompressedUserContext{}

	modifyInputs := []string{
		"把下午改成开会",
		"换成健身",
		"修改午餐时间",
	}

	for _, input := range modifyInputs {
		prompt := s.buildPlanModePrompt(input, ctx, session)
		needThinking := s.shouldUseThinking(input, ctx, session)

		fmt.Printf("\n输入: %s\n", input)
		fmt.Printf("需要深度思考: %v (当前有%d个时段应该为true)\n", needThinking, len(session.CurrentSchedule))

		// 验证当前时刻表在提示词中
		if !strings.Contains(prompt, "09:00-12:00") {
			t.Errorf("提示词缺少现有时刻表信息")
		}

		// 验证强调保留未提及时段的规则
		if !strings.Contains(prompt, "保留未提及的时段") {
			t.Errorf("提示词缺少保留上下文的规则")
		}

		// 修改场景应该触发深度思考
		if !needThinking {
			t.Errorf("修改场景应该触发深度思考")
		}
	}
}
