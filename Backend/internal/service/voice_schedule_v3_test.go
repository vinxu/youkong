package service

import (
	"strings"
	"testing"
	"time"

	"youkong/internal/model"
)

// ========================================
// 语音时刻表系统 v3 迭代测试
// 测试目标：验证确认逻辑、撤销机制、状态机、UX增强
// ========================================

// ========== 测试场景1：ApprovalHandler LLM化 - 误判修复测试 ==========

// TestApprovalHandler_MisjudgmentFix 测试审批处理器误判修复
// 核心问题：之前"这个时间好像不对"会被误判为确认
func TestApprovalHandler_MisjudgmentFix(t *testing.T) {
	handler := NewApprovalHandler() // 不带LLM的版本，测试模式匹配改进

	testCases := []struct {
		name           string
		input          string
		expectedResult string
		description    string
	}{
		// ===== 容易误判的场景（v3重点修复）=====
		{
			name:           "假确认1_好像不对",
			input:          "这个时间好像不对",
			expectedResult: "modify_request",
			description:    "用户说'好像不对'，不应被识别为确认",
		},
		{
			name:           "假确认2_有问题",
			input:          "好的，但是有点问题",
			expectedResult: "modify_request",
			description:    "用户说'有问题'，应识别为修改请求",
		},
		{
			name:           "假确认3_不太对",
			input:          "不太对吧",
			expectedResult: "modify_request",
			description:    "用户说'不太对'，应识别为修改请求",
		},
		{
			name:           "假确认4_好像错了",
			input:          "好像错了",
			expectedResult: "modify_request",
			description:    "用户说'好像错了'，应识别为修改请求",
		},
		{
			name:           "条件确认1",
			input:          "好的，但是把开会改到3点",
			expectedResult: "modify_request",
			description:    "带条件的确认应识别为修改请求",
		},
		{
			name:           "条件确认2",
			input:          "行，不过时间能调整一下吗",
			expectedResult: "modify_request",
			description:    "'不过'是条件词，即使有疑问句式也应识别为修改请求",
		},

		// ===== 真正的确认场景 =====
		{
			name:           "真确认1_好的",
			input:          "好的",
			expectedResult: "approve",
			description:    "纯粹的'好的'应识别为确认",
		},
		{
			name:           "真确认2_确认",
			input:          "确认",
			expectedResult: "approve",
			description:    "纯粹的'确认'应识别为确认",
		},
		{
			name:           "真确认3_没问题",
			input:          "没问题",
			expectedResult: "approve",
			description:    "纯粹的'没问题'应识别为确认",
		},
		{
			name:           "真确认4_就这样",
			input:          "就这样吧",
			expectedResult: "approve",
			description:    "'就这样'应识别为确认",
		},
		{
			name:           "真确认5_OK",
			input:          "OK",
			expectedResult: "approve",
			description:    "'OK'应识别为确认",
		},

		// ===== 拒绝场景 =====
		{
			name:           "拒绝1_取消",
			input:          "取消吧",
			expectedResult: "reject",
			description:    "'取消'应识别为拒绝",
		},
		{
			name:           "拒绝2_不要了",
			input:          "不要了",
			expectedResult: "reject",
			description:    "'不要了'应识别为拒绝",
		},
		{
			name:           "拒绝3_算了",
			input:          "算了算了",
			expectedResult: "reject",
			description:    "'算了'应识别为拒绝",
		},

		// ===== 修改场景 =====
		{
			name:           "修改1_把时间改一下",
			input:          "把时间改一下",
			expectedResult: "modify",
			description:    "明确的修改请求",
		},
		{
			name:           "修改2_换成",
			input:          "换成下午4点",
			expectedResult: "modify",
			description:    "明确的修改请求",
		},

		// ===== 需要澄清场景 =====
		{
			name:           "澄清1_疑问",
			input:          "这是什么意思？",
			expectedResult: "need_clarification",
			description:    "疑问句应识别为需要澄清",
		},
		{
			name:           "澄清2_不确定",
			input:          "下午3点是指哪个活动？",
			expectedResult: "need_clarification",
			description:    "疑问句应识别为需要澄清",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := handler.analyzeUserDecisionByPattern(tc.input)

			// 检查结果是否符合预期
			if result != tc.expectedResult {
				t.Errorf("【%s】\n输入: %q\n期望: %s\n实际: %s\n说明: %s",
					tc.name, tc.input, tc.expectedResult, result, tc.description)
			} else {
				t.Logf("✓ %s: %q → %s", tc.name, tc.input, result)
			}
		})
	}
}

// ========== 测试场景2：撤销机制测试 ==========

// TestExecutionHistory_UndoWindow 测试执行历史和撤销窗口
func TestExecutionHistory_UndoWindow(t *testing.T) {
	userID := "test-user-001"
	sessionID := "test-session-001"

	t.Run("创建执行历史", func(t *testing.T) {
		beforeState := []model.ScheduleItem{
			{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "工作"},
		}
		afterState := []model.ScheduleItem{
			{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "工作"},
			{StartTime: "14:00", EndTime: "16:00", Emoji: "📚", Status: "开会"},
		}
		targetDate := time.Now()

		history := model.NewExecutionHistory(userID, sessionID, "save", beforeState, afterState, targetDate)

		if history.UserID != userID {
			t.Errorf("UserID 不匹配: expected %s, got %s", userID, history.UserID)
		}
		if !history.CanUndo {
			t.Error("新创建的执行历史应该可撤销")
		}
		if history.UndoDeadline.Before(time.Now()) {
			t.Error("撤销截止时间应该在未来")
		}
		t.Logf("✓ 执行历史创建成功: ID=%s, CanUndo=%v", history.ID, history.CanUndo)
	})

	t.Run("撤销窗口内可撤销", func(t *testing.T) {
		history := model.NewExecutionHistory(userID, sessionID, "save", nil, nil, time.Now())

		if !history.IsUndoable() {
			t.Error("刚创建的执行历史应该可撤销")
		}

		remaining := history.RemainingUndoTime()
		if remaining <= 0 || remaining > 300 {
			t.Errorf("剩余时间应该在 0-300 秒之间，实际: %d", remaining)
		}
		t.Logf("✓ 撤销窗口内: RemainingTime=%d秒", remaining)
	})

	t.Run("撤销窗口外不可撤销", func(t *testing.T) {
		history := &model.ExecutionHistory{
			ID:           "test-exec",
			UserID:       userID,
			CanUndo:      true,
			UndoDeadline: time.Now().Add(-1 * time.Minute), // 已过期
		}

		if history.IsUndoable() {
			t.Error("过期的执行历史不应该可撤销")
		}

		remaining := history.RemainingUndoTime()
		if remaining != 0 {
			t.Errorf("过期后剩余时间应该为0，实际: %d", remaining)
		}
		t.Logf("✓ 撤销窗口外: IsUndoable=%v, RemainingTime=%d", history.IsUndoable(), remaining)
	})

	t.Run("手动禁用撤销", func(t *testing.T) {
		history := model.NewExecutionHistory(userID, sessionID, "save", nil, nil, time.Now())
		history.CanUndo = false
		history.UndoReason = "用户已撤销"

		if history.IsUndoable() {
			t.Error("手动禁用后不应该可撤销")
		}
		t.Logf("✓ 手动禁用撤销: CanUndo=%v, Reason=%s", history.CanUndo, history.UndoReason)
	})
}

// ========== 测试场景3：Phase 状态机测试 ==========

// TestPhaseStateMachine_ValidTransitions 测试状态转换有效性
func TestPhaseStateMachine_ValidTransitions(t *testing.T) {
	testCases := []struct {
		name     string
		from     model.ConversationPhase
		to       model.ConversationPhase
		expected bool
	}{
		// 有效转换
		{"Idle→Understanding", model.PhaseIdle, model.PhaseUnderstanding, true},
		{"Understanding→Planning", model.PhaseUnderstanding, model.PhasePlanning, true},
		{"Understanding→Discussing", model.PhaseUnderstanding, model.PhaseDiscussing, true},
		{"Planning→AwaitingApproval", model.PhasePlanning, model.PhaseAwaitingApproval, true},
		{"Planning→Approval(兼容)", model.PhasePlanning, model.PhaseApproval, true},
		{"Execution→ConfirmedUndoable", model.PhaseExecution, model.PhaseConfirmedUndoable, true},
		{"ConfirmedUndoable→Completed", model.PhaseConfirmedUndoable, model.PhaseCompleted, true},
		{"ConfirmedUndoable→Planning(撤销)", model.PhaseConfirmedUndoable, model.PhasePlanning, true},

		// 无效转换
		{"Idle→Execution(跳跃)", model.PhaseIdle, model.PhaseExecution, false},
		{"Completed→Planning(回退)", model.PhaseCompleted, model.PhasePlanning, false},
		{"Understanding→Execution(跳跃)", model.PhaseUnderstanding, model.PhaseExecution, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := model.IsValidTransition(tc.from, tc.to)
			if result != tc.expected {
				t.Errorf("转换 %s → %s: 期望 %v, 实际 %v",
					tc.from, tc.to, tc.expected, result)
			} else {
				symbol := "✓"
				if !tc.expected {
					symbol = "✗"
				}
				t.Logf("%s %s → %s = %v", symbol, tc.from, tc.to, result)
			}
		})
	}
}

// ========== 测试场景4：意图分发器 undo 测试 ==========

// TestIntentDispatcher_UndoIntent 测试撤销意图识别
func TestIntentDispatcher_UndoIntent(t *testing.T) {
	// 测试撤销关键词在 Prompt 中的存在
	dispatcher := &IntentDispatcher{}
	session := &model.VoiceScheduleSession{
		Phase: model.PhaseConfirmedUndoable,
		State: "confirmed",
	}

	prompt := dispatcher.buildSystemPrompt(session)

	// 检查 Prompt 是否包含 undo 相关内容
	if !strings.Contains(prompt, "undo") {
		t.Error("系统 Prompt 应该包含 undo 意图说明")
	}

	t.Logf("✓ IntentDispatcher Prompt 包含 undo 意图")
}

// ========== 测试场景5：简单意图识别测试（含撤销）==========

// TestIsSimpleIntent_WithUndo 测试简单意图识别（含撤销关键词）
func TestIsSimpleIntent_WithUndo(t *testing.T) {
	svc := &VoiceScheduleService{}

	testCases := []struct {
		input    string
		expected bool
		reason   string
	}{
		// 确认类
		{"确认", true, "纯确认词"},
		{"好的", true, "纯确认词"},
		{"OK", true, "纯确认词"},

		// 取消类
		{"取消", true, "纯取消词"},
		{"不要了", true, "纯取消词"},

		// 撤销类（v3新增）
		{"撤销", true, "撤销关键词"},
		{"撤回", true, "撤销关键词"},
		{"后悔了", true, "撤销关键词"},
		{"恢复", true, "撤销关键词"},
		{"回退", true, "撤销关键词"},

		// 复杂句子（不应匹配）
		{"帮我把明天下午的会议改到3点", false, "复杂修改请求"},
		{"我想撤销刚才的操作然后重新安排一下", false, "超过10字的撤销"},
		{"确认一下这个时间对不对", false, "疑问句"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := svc.isSimpleIntent(tc.input)
			if result != tc.expected {
				t.Errorf("输入: %q\n期望: %v, 实际: %v\n原因: %s",
					tc.input, tc.expected, result, tc.reason)
			} else {
				symbol := "✓"
				if !tc.expected {
					symbol = "✗"
				}
				t.Logf("%s %q → isSimple=%v (%s)", symbol, tc.input, result, tc.reason)
			}
		})
	}
}

// ========== 测试场景6：VoiceScheduleEvent 增强字段测试 ==========

// TestVoiceScheduleEvent_V3Fields 测试事件结构体的 v3 新增字段
func TestVoiceScheduleEvent_V3Fields(t *testing.T) {
	t.Run("撤销相关字段", func(t *testing.T) {
		event := model.VoiceScheduleEvent{
			Type:          model.VSEventConfirmed,
			Message:       "时刻表已保存。5分钟内可以说「撤销」来取消。",
			Phase:         model.PhaseConfirmedUndoable,
			CanUndo:       true,
			UndoDeadline:  time.Now().Add(5 * time.Minute).UnixMilli(),
			UndoRemaining: 300,
			AvailableActions: []model.ActionHint{
				{Action: "undo", Label: "撤销", Shortcut: "撤销"},
			},
		}

		if !event.CanUndo {
			t.Error("CanUndo 应该为 true")
		}
		if event.UndoRemaining != 300 {
			t.Errorf("UndoRemaining 期望 300, 实际 %d", event.UndoRemaining)
		}
		if len(event.AvailableActions) != 1 {
			t.Errorf("AvailableActions 应该有 1 个动作, 实际 %d", len(event.AvailableActions))
		}
		if event.AvailableActions[0].Action != "undo" {
			t.Errorf("第一个动作应该是 undo, 实际 %s", event.AvailableActions[0].Action)
		}

		t.Logf("✓ 撤销相关字段正确: CanUndo=%v, UndoRemaining=%d, Actions=%v",
			event.CanUndo, event.UndoRemaining, event.AvailableActions)
	})

	t.Run("AwaitingAction字段", func(t *testing.T) {
		event := model.VoiceScheduleEvent{
			Type:           model.VSEventApprovalPrompt,
			AwaitingAction: "approval",
			AvailableActions: []model.ActionHint{
				{Action: "confirm", Label: "确认", Shortcut: "好的"},
				{Action: "reject", Label: "取消", Shortcut: "取消"},
				{Action: "modify", Label: "修改", Shortcut: "改一下"},
			},
		}

		if event.AwaitingAction != "approval" {
			t.Errorf("AwaitingAction 期望 approval, 实际 %s", event.AwaitingAction)
		}
		if len(event.AvailableActions) != 3 {
			t.Errorf("AvailableActions 应该有 3 个动作, 实际 %d", len(event.AvailableActions))
		}

		t.Logf("✓ AwaitingAction 字段正确: %s, Actions=%d", event.AwaitingAction, len(event.AvailableActions))
	})
}

// ========== 测试场景7：ActionHint 结构测试 ==========

// TestActionHint_Structure 测试操作提示结构体
func TestActionHint_Structure(t *testing.T) {
	hints := []model.ActionHint{
		{Action: "confirm", Label: "确认保存", Shortcut: "好的", Description: "保存当前时刻表"},
		{Action: "reject", Label: "取消", Shortcut: "取消", Description: "放弃当前编辑"},
		{Action: "modify", Label: "修改", Shortcut: "改一下", Description: "返回继续编辑"},
		{Action: "undo", Label: "撤销", Shortcut: "撤销", Description: "恢复到保存前的状态"},
	}

	for _, hint := range hints {
		if hint.Action == "" {
			t.Error("Action 不应为空")
		}
		if hint.Label == "" {
			t.Error("Label 不应为空")
		}
		t.Logf("✓ ActionHint: Action=%s, Label=%s, Shortcut=%s",
			hint.Action, hint.Label, hint.Shortcut)
	}
}

// ========== 测试场景8：VSActionUndo 动作类型测试 ==========

// TestVSActionUndo 测试撤销动作类型
func TestVSActionUndo(t *testing.T) {
	if model.VSActionUndo != "undo" {
		t.Errorf("VSActionUndo 期望 'undo', 实际 %s", model.VSActionUndo)
	}

	// 验证所有动作类型
	actions := []model.VoiceScheduleAction{
		model.VSActionAnswer,
		model.VSActionVoice,
		model.VSActionSupplement,
		model.VSActionConfirm,
		model.VSActionCancel,
		model.VSActionUndo,
	}

	for _, action := range actions {
		if action == "" {
			t.Error("动作类型不应为空字符串")
		}
		t.Logf("✓ VoiceScheduleAction: %s", action)
	}
}

// ========== 测试场景9：模拟完整对话流程 ==========

// TestFullConversationFlow_WithUndo 模拟包含撤销的完整对话流程
func TestFullConversationFlow_WithUndo(t *testing.T) {
	t.Log("========== 模拟完整对话流程 ==========")

	// 初始化会话
	session := &model.VoiceScheduleSession{
		UserID:    "test-user",
		SessionID: "test-session",
		Phase:     model.PhaseUnderstanding,
		State:     "initial",
	}

	t.Log("【第1轮】用户: 明天下午2点到4点开会")
	t.Log("  → 系统理解意图，进入规划阶段")
	session.Phase = model.PhasePlanning
	session.CurrentSchedule = []model.ScheduleItem{
		{StartTime: "14:00", EndTime: "16:00", Emoji: "💼", Status: "开会"},
	}

	t.Log("【第2轮】系统生成草案，等待审批")
	session.Phase = model.PhaseApproval
	t.Logf("  → 当前时刻表: %v", session.CurrentSchedule)

	t.Log("【第3轮】用户: 好的")
	handler := NewApprovalHandler()
	decision := handler.analyzeUserDecisionByPattern("好的")
	if decision != "approve" {
		t.Errorf("  ✗ 期望 approve, 实际 %s", decision)
	} else {
		t.Log("  ✓ 识别为 approve")
	}

	t.Log("【第4轮】系统执行保存")
	session.Phase = model.PhaseConfirmedUndoable
	execHistory := model.NewExecutionHistory(
		session.UserID, session.SessionID, "save",
		nil, session.CurrentSchedule, time.Now(),
	)
	t.Logf("  → 记录执行历史: ID=%s, CanUndo=%v", execHistory.ID, execHistory.CanUndo)

	t.Log("【第5轮】用户: 撤销")
	t.Log("  → 检查撤销窗口...")
	if execHistory.IsUndoable() {
		t.Log("  ✓ 在撤销窗口内，可以撤销")
		session.Phase = model.PhasePlanning
		session.CurrentSchedule = execHistory.BeforeState
		t.Log("  → 恢复到保存前状态，回到规划阶段")
	} else {
		t.Log("  ✗ 撤销窗口已过期")
	}

	t.Log("========== 对话流程测试完成 ==========")
}

// ========== 测试场景10：边界条件测试 ==========

// TestEdgeCases 测试边界条件
func TestEdgeCases(t *testing.T) {
	t.Run("空输入处理", func(t *testing.T) {
		handler := NewApprovalHandler()
		result := handler.analyzeUserDecisionByPattern("")
		t.Logf("空输入结果: %s", result)
		// 空输入应返回 unknown
		if result != "unknown" {
			t.Errorf("空输入期望 unknown, 实际 %s", result)
		}
	})

	t.Run("超长输入处理", func(t *testing.T) {
		svc := &VoiceScheduleService{}
		longInput := strings.Repeat("好的", 100)
		result := svc.isSimpleIntent(longInput)
		// 超长输入不应该是简单意图
		if result {
			t.Error("超长输入不应该被识别为简单意图")
		}
		t.Logf("超长输入(%d字): isSimple=%v", len([]rune(longInput)), result)
	})

	t.Run("混合语言输入", func(t *testing.T) {
		handler := NewApprovalHandler()
		testCases := []struct {
			input    string
			expected string
		}{
			{"ok啦", "approve"},
			{"OK没问题", "approve"},
			{"cancel掉", "reject"},
		}
		for _, tc := range testCases {
			result := handler.analyzeUserDecisionByPattern(tc.input)
			t.Logf("混合语言 %q → %s (期望 %s)", tc.input, result, tc.expected)
		}
	})

	t.Run("特殊字符输入", func(t *testing.T) {
		handler := NewApprovalHandler()
		inputs := []string{
			"好的！！！",
			"确认～～",
			"取消。。。",
			"???",
		}
		for _, input := range inputs {
			result := handler.analyzeUserDecisionByPattern(input)
			t.Logf("特殊字符 %q → %s", input, result)
		}
	})
}

// ========== 测试场景11：并发安全测试 ==========

// TestConcurrentAccess 测试并发访问
func TestConcurrentAccess(t *testing.T) {
	handler := NewApprovalHandler()

	// 并发调用 analyzeUserDecisionByPattern
	inputs := []string{"好的", "取消", "修改", "撤销", "这个不对"}
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			for _, input := range inputs {
				_ = handler.analyzeUserDecisionByPattern(input)
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	t.Log("✓ 并发访问测试通过")
}

// ========== 测试场景12：Phase 转换的实际场景模拟 ==========

// TestPhaseTransitions_RealScenarios 测试真实场景的阶段转换
func TestPhaseTransitions_RealScenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		transitions []struct {
			from model.ConversationPhase
			to   model.ConversationPhase
		}
		description string
	}{
		{
			name: "正常创建流程",
			transitions: []struct {
				from model.ConversationPhase
				to   model.ConversationPhase
			}{
				{model.PhaseIdle, model.PhaseUnderstanding},
				{model.PhaseUnderstanding, model.PhasePlanning},
				{model.PhasePlanning, model.PhaseApproval},
				{model.PhaseApproval, model.PhaseExecution},
				{model.PhaseExecution, model.PhaseConfirmedUndoable},
				{model.PhaseConfirmedUndoable, model.PhaseCompleted},
			},
			description: "用户创建时刻表的标准流程",
		},
		{
			name: "撤销流程",
			transitions: []struct {
				from model.ConversationPhase
				to   model.ConversationPhase
			}{
				{model.PhaseConfirmedUndoable, model.PhasePlanning},
			},
			description: "用户撤销后回到规划阶段",
		},
		{
			name: "需要澄清的流程",
			transitions: []struct {
				from model.ConversationPhase
				to   model.ConversationPhase
			}{
				{model.PhaseIdle, model.PhaseUnderstanding},
				{model.PhaseUnderstanding, model.PhaseDiscussing},
				{model.PhaseDiscussing, model.PhasePlanning},
			},
			description: "用户输入模糊需要澄清",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("场景: %s", scenario.description)
			allValid := true
			for _, trans := range scenario.transitions {
				valid := model.IsValidTransition(trans.from, trans.to)
				if !valid {
					t.Errorf("  ✗ %s → %s 是无效转换", trans.from, trans.to)
					allValid = false
				} else {
					t.Logf("  ✓ %s → %s", trans.from, trans.to)
				}
			}
			if allValid {
				t.Logf("✓ 场景 '%s' 所有转换有效", scenario.name)
			}
		})
	}
}

// ========== 性能基准测试 ==========

// BenchmarkApprovalHandler_Pattern 基准测试：模式匹配性能
func BenchmarkApprovalHandler_Pattern(b *testing.B) {
	handler := NewApprovalHandler()
	inputs := []string{
		"好的",
		"这个时间好像不对",
		"确认",
		"取消吧",
		"把时间改一下",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			handler.analyzeUserDecisionByPattern(input)
		}
	}
}

// BenchmarkIsSimpleIntent 基准测试：简单意图识别性能
func BenchmarkIsSimpleIntent(b *testing.B) {
	svc := &VoiceScheduleService{}
	inputs := []string{
		"确认",
		"撤销",
		"明天下午开会",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			svc.isSimpleIntent(input)
		}
	}
}

// BenchmarkPhaseValidation 基准测试：阶段验证性能
func BenchmarkPhaseValidation(b *testing.B) {
	transitions := []struct {
		from model.ConversationPhase
		to   model.ConversationPhase
	}{
		{model.PhaseIdle, model.PhaseUnderstanding},
		{model.PhasePlanning, model.PhaseApproval},
		{model.PhaseConfirmedUndoable, model.PhasePlanning},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range transitions {
			model.IsValidTransition(t.from, t.to)
		}
	}
}
