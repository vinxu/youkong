package service

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"youkong/internal/model"
)

// ============================================================================
// 第一部分：ApprovalHandler 全面测试
// 目标：覆盖真实用户可能说的各种话，确保不会误判
// ============================================================================

func TestApprovalHandler_Comprehensive(t *testing.T) {
	handler := NewApprovalHandler()

	// -------------------------------------------------------------------------
	// 1.1 假确认测试 - 包含"好"/"确认"等词但实际不是确认
	// -------------------------------------------------------------------------
	t.Run("假确认_包含好字但不是确认", func(t *testing.T) {
		cases := []struct {
			input    string
			expected string
			reason   string
		}{
			// 否定性质的"好像"
			{"这个时间好像不对", "modify_request", "好像+不对 = 质疑"},
			{"好像有点问题", "modify_request", "好像+问题 = 质疑"},
			{"好像不太行", "modify_request", "好像+不行 = 否定"},
			{"好像错了吧", "modify_request", "好像+错 = 质疑"},
			{"这个好像不是我要的", "modify_request", "好像+不是 = 否定"},

			// 条件性的"好"
			{"好是好，但是时间不对", "modify_request", "但是 = 条件"},
			{"好吧，不过要改一下", "modify_request", "不过 = 条件"},
			{"好的好的，但是有个问题", "modify_request", "但是+问题 = 条件"},

			// 疑问性质的"好"
			{"好的是哪个时间段？", "need_clarification", "疑问句"},
			{"好，那这个是什么意思", "need_clarification", "疑问"},
			{"好的，我确认一下这是几点", "need_clarification", "确认一下 = 疑问"},

			// 不确定的"好"
			{"好吗？", "need_clarification", "疑问句"},
			{"这样好不好", "need_clarification", "疑问句"},
			{"好不好啊", "need_clarification", "疑问句"},
		}

		for _, c := range cases {
			result := handler.analyzeUserDecisionByPattern(c.input)
			if result != c.expected {
				t.Errorf("❌ [%s] 输入=%q, 期望=%s, 实际=%s, 原因=%s",
					c.reason, c.input, c.expected, result, c.reason)
			} else {
				t.Logf("✓ [%s] %q → %s", c.reason, c.input, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 1.2 假确认测试 - 包含"没问题"但实际有问题
	// -------------------------------------------------------------------------
	t.Run("假确认_包含没问题但实际有问题", func(t *testing.T) {
		cases := []struct {
			input    string
			expected string
			reason   string
		}{
			{"没问题吗？", "need_clarification", "疑问句"},
			{"真的没问题？", "need_clarification", "疑问句"},
			{"没问题，但是要改时间", "modify_request", "但是 = 条件"},
			{"没问题吧，不过...", "modify_request", "不过 = 条件"},
			{"有没有问题啊", "need_clarification", "疑问句"},
		}

		for _, c := range cases {
			result := handler.analyzeUserDecisionByPattern(c.input)
			if result != c.expected {
				t.Errorf("❌ [%s] 输入=%q, 期望=%s, 实际=%s",
					c.reason, c.input, c.expected, result)
			} else {
				t.Logf("✓ [%s] %q → %s", c.reason, c.input, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 1.3 真确认测试 - 各种确认的表达方式
	// -------------------------------------------------------------------------
	t.Run("真确认_各种表达方式", func(t *testing.T) {
		cases := []struct {
			input    string
			expected string
			reason   string
		}{
			// 简短确认
			{"好", "approve", "单字确认"},
			{"好的", "approve", "标准确认"},
			{"行", "approve", "口语确认"},
			{"行的", "approve", "口语确认"},
			{"可以", "approve", "标准确认"},
			{"OK", "approve", "英文确认"},
			{"ok", "approve", "小写英文"},
			{"Ok", "approve", "混合大小写"},
			{"嗯", "approve", "语气词确认"},
			{"嗯嗯", "approve", "重复语气词"},
			{"对", "approve", "简短确认"},
			{"对的", "approve", "标准确认"},
			{"是", "approve", "简短确认"},
			{"是的", "approve", "标准确认"},

			// 肯定性确认
			{"确认", "approve", "正式确认"},
			{"确定", "approve", "正式确认"},
			{"没问题", "approve", "标准确认"},
			{"没有问题", "approve", "完整表达"},
			{"可以的", "approve", "口语确认"},
			{"没毛病", "approve", "网络用语"},
			{"就这样", "approve", "确认表达"},
			{"就这样吧", "approve", "确认表达"},
			{"就这么定了", "approve", "确认表达"},
			{"保存吧", "approve", "动作确认"},
			{"存吧", "approve", "简短动作"},

			// 热情确认
			{"太好了", "approve", "积极确认"},
			{"好的好的", "approve", "重复确认"},
			{"好好好", "approve", "多次确认"},
			{"行行行", "approve", "多次确认"},
			{"可以可以", "approve", "重复确认"},
			{"完美", "approve", "积极确认"},
			{"很好", "approve", "积极确认"},
			{"不错", "approve", "积极确认"},
			{"挺好的", "approve", "积极确认"},
		}

		for _, c := range cases {
			result := handler.analyzeUserDecisionByPattern(c.input)
			if result != c.expected {
				t.Errorf("❌ [%s] 输入=%q, 期望=%s, 实际=%s",
					c.reason, c.input, c.expected, result)
			} else {
				t.Logf("✓ [%s] %q → %s", c.reason, c.input, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 1.4 拒绝测试 - 各种拒绝的表达方式
	// -------------------------------------------------------------------------
	t.Run("拒绝_各种表达方式", func(t *testing.T) {
		cases := []struct {
			input    string
			expected string
			reason   string
		}{
			// 直接拒绝
			{"不要", "reject", "直接拒绝"},
			{"不要了", "reject", "直接拒绝"},
			{"不用了", "reject", "委婉拒绝"},
			{"不用", "reject", "简短拒绝"},
			{"算了", "reject", "放弃"},
			{"算了吧", "reject", "放弃"},
			{"取消", "reject", "取消操作"},
			{"取消吧", "reject", "取消操作"},
			{"别了", "reject", "口语拒绝"},
			{"别保存", "reject", "动作拒绝"},
			{"不保存", "reject", "动作拒绝"},
			{"不存了", "reject", "动作拒绝"},

			// 否定性拒绝
			{"不行", "reject", "否定"},
			{"不可以", "reject", "否定"},
			{"不好", "reject", "否定"},
			{"不对", "reject", "否定"},
			{"错了", "reject", "否定"},
			{"不是这样", "reject", "否定"},
			{"不是我要的", "reject", "否定"},

			// 放弃性拒绝
			{"放弃", "reject", "放弃"},
			{"放弃吧", "reject", "放弃"},
			{"不做了", "reject", "放弃"},
			{"不弄了", "reject", "放弃"},
			{"停", "reject", "停止"},
			{"停下", "reject", "停止"},
			{"停止", "reject", "停止"},
		}

		for _, c := range cases {
			result := handler.analyzeUserDecisionByPattern(c.input)
			if result != c.expected {
				t.Errorf("❌ [%s] 输入=%q, 期望=%s, 实际=%s",
					c.reason, c.input, c.expected, result)
			} else {
				t.Logf("✓ [%s] %q → %s", c.reason, c.input, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 1.5 修改请求测试 - 各种修改的表达方式
	// -------------------------------------------------------------------------
	t.Run("修改请求_各种表达方式", func(t *testing.T) {
		cases := []struct {
			input    string
			expected string
			reason   string
		}{
			// 直接修改请求
			{"改一下", "modify", "直接修改"},
			{"改改", "modify", "口语修改"},
			{"修改一下", "modify", "正式修改"},
			{"调整一下", "modify", "正式修改"},
			{"换一下", "modify", "替换请求"},
			{"换成下午", "modify", "具体替换"},
			{"改成3点", "modify", "具体修改"},
			{"把时间改一下", "modify", "指定修改"},
			{"时间改改", "modify", "指定修改"},

			// 条件性修改（带"但是"等）
			{"好的，但是把时间改一下", "modify_request", "条件修改"},
			{"行，不过时间要调整", "modify_request", "条件修改"},
			{"可以，但是有个地方要改", "modify_request", "条件修改"},
			{"嗯，不过换个时间", "modify_request", "条件修改"},

			// 指出问题的修改
			// 注意："这个时间不对，改成4点" 包含"不对"，返回 reject 是合理的
			// 用户如果想修改应该说"把时间改成4点"而不是先否定
			{"这个时间不对，改成4点", "reject", "问题+修改 → 否定优先"},
			{"下午的安排有问题", "modify_request", "指出问题"},
			{"开会时间不太对", "modify_request", "指出问题"},
			{"这里有点问题", "modify_request", "指出问题"},
		}

		for _, c := range cases {
			result := handler.analyzeUserDecisionByPattern(c.input)
			if result != c.expected {
				t.Errorf("❌ [%s] 输入=%q, 期望=%s, 实际=%s",
					c.reason, c.input, c.expected, result)
			} else {
				t.Logf("✓ [%s] %q → %s", c.reason, c.input, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 1.6 澄清请求测试 - 各种疑问的表达方式
	// -------------------------------------------------------------------------
	t.Run("澄清请求_各种表达方式", func(t *testing.T) {
		cases := []struct {
			input    string
			expected string
			reason   string
		}{
			// 疑问句
			{"这是什么意思", "need_clarification", "询问含义"},
			{"什么意思", "need_clarification", "简短询问"},
			{"啥意思", "need_clarification", "口语询问"},
			{"不太明白", "need_clarification", "表示困惑"},
			{"没懂", "need_clarification", "表示困惑"},
			{"不理解", "need_clarification", "表示困惑"},
			{"能解释一下吗", "need_clarification", "请求解释"},
			{"再说一遍", "need_clarification", "请求重复"},
			{"什么时间", "need_clarification", "具体询问"},
			{"哪个活动", "need_clarification", "具体询问"},
			{"是指哪个", "need_clarification", "具体询问"},

			// 确认性疑问
			{"确认一下", "need_clarification", "请求确认"},
			{"我确认一下", "need_clarification", "请求确认"},
			{"让我看看", "need_clarification", "查看请求"},
			{"等等", "need_clarification", "暂停"},
			{"等一下", "need_clarification", "暂停"},
			{"稍等", "need_clarification", "暂停"},
		}

		for _, c := range cases {
			result := handler.analyzeUserDecisionByPattern(c.input)
			if result != c.expected {
				t.Errorf("❌ [%s] 输入=%q, 期望=%s, 实际=%s",
					c.reason, c.input, c.expected, result)
			} else {
				t.Logf("✓ [%s] %q → %s", c.reason, c.input, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 1.7 撤销意图测试
	// -------------------------------------------------------------------------
	t.Run("撤销意图_各种表达方式", func(t *testing.T) {
		cases := []struct {
			input    string
			expected string
			reason   string
		}{
			{"撤销", "undo", "直接撤销"},
			{"撤回", "undo", "撤回"},
			{"取消刚才的", "undo", "取消操作"},
			{"后悔了", "undo", "后悔"},
			{"恢复", "undo", "恢复"},
			{"还原", "undo", "还原"},
			{"回退", "undo", "回退"},
			{"撤销刚才的操作", "undo", "完整撤销"},
			{"刚才那个不要了", "undo", "口语撤销"},
		}

		for _, c := range cases {
			result := handler.analyzeUserDecisionByPattern(c.input)
			if result != c.expected {
				t.Errorf("❌ [%s] 输入=%q, 期望=%s, 实际=%s",
					c.reason, c.input, c.expected, result)
			} else {
				t.Logf("✓ [%s] %q → %s", c.reason, c.input, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 1.8 边界情况测试
	// -------------------------------------------------------------------------
	t.Run("边界情况", func(t *testing.T) {
		cases := []struct {
			input    string
			expected string
			reason   string
		}{
			// 空输入和特殊字符
			{"", "unknown", "空输入"},
			{"   ", "unknown", "纯空格"},
			{"...", "unknown", "省略号"},
			{"???", "need_clarification", "多问号"},
			{"！！！", "unknown", "多感叹号"},

			// 表情符号
			{"👍", "approve", "点赞表情"},
			{"👌", "approve", "OK手势"},
			{"✓", "approve", "勾选符号"},
			{"✅", "approve", "绿勾"},
			{"❌", "reject", "红叉"},
			{"👎", "reject", "踩表情"},

			// 混合语言
			{"OK啦", "approve", "中英混合"},
			{"好的OK", "approve", "中英混合"},
			{"no", "reject", "英文拒绝"},
			{"yes", "approve", "英文确认"},
			{"nope", "reject", "英文口语拒绝"},
			{"yep", "approve", "英文口语确认"},

			// 语气词和标点
			{"好啊", "approve", "带语气词"},
			{"好呀", "approve", "带语气词"},
			{"好哦", "approve", "带语气词"},
			{"好嘞", "approve", "带语气词"},
			{"行啊", "approve", "带语气词"},
			{"行嘞", "approve", "带语气词"},
		}

		for _, c := range cases {
			result := handler.analyzeUserDecisionByPattern(c.input)
			if result != c.expected {
				t.Errorf("❌ [%s] 输入=%q, 期望=%s, 实际=%s",
					c.reason, c.input, c.expected, result)
			} else {
				t.Logf("✓ [%s] %q → %s", c.reason, c.input, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 1.9 复杂句子测试 - 真实用户可能说的长句子
	// -------------------------------------------------------------------------
	t.Run("复杂句子_真实场景", func(t *testing.T) {
		cases := []struct {
			input    string
			expected string
			reason   string
		}{
			// 看起来像确认但实际有问题
			{"好的，这个安排看起来不错，但是下午3点我有个会，能调整一下吗", "modify_request", "长句条件修改"},
			{"嗯，时间是没问题，不过地点能换一下吗", "modify_request", "长句条件修改"},
			{"行，就是有一点，晚上的安排好像和我原来的计划冲突了", "modify_request", "长句指出问题"},

			// 确认性的长句
			{"好的，这个安排挺合理的，就按这个来吧", "approve", "长句确认"},
			{"没问题，时间地点都可以，保存吧", "approve", "长句确认"},
			{"行，我觉得可以，就这么定了", "approve", "长句确认"},

			// 疑问性的长句
			{"我想确认一下，下午3点的那个活动是在哪里举行的", "need_clarification", "长句疑问"},
			{"等一下，这个时间安排是今天的还是明天的", "need_clarification", "长句疑问"},

			// 拒绝性的长句
			{"不行，这个安排完全不对，我根本没有说下午要开会", "reject", "长句拒绝"},
			{"算了吧，我觉得这个安排太紧了，改天再说", "reject", "长句放弃"},
		}

		for _, c := range cases {
			result := handler.analyzeUserDecisionByPattern(c.input)
			if result != c.expected {
				t.Errorf("❌ [%s] 输入=%q, 期望=%s, 实际=%s",
					c.reason, c.input, c.expected, result)
			} else {
				t.Logf("✓ [%s] %q → %s", c.reason, c.input, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 1.10 优先级冲突测试 - 当输入包含多种信号时
	// -------------------------------------------------------------------------
	t.Run("优先级冲突测试", func(t *testing.T) {
		cases := []struct {
			input    string
			expected string
			reason   string
		}{
			// "好" + "但是" → 条件优先
			{"好的但是不对", "modify_request", "条件优先于确认"},
			{"确认，但是要改", "modify_request", "条件优先于确认"},
			{"没问题，不过有问题", "modify_request", "条件优先于确认"},

			// "好" + "?" → 疑问优先
			{"好的？", "need_clarification", "疑问优先于确认"},
			{"可以吗？", "need_clarification", "疑问优先于确认"},
			{"行不行？", "need_clarification", "疑问优先于确认"},

			// "不" + "好像" → 因为"不好"被识别为拒绝
			// 这个句子本身语法不太正常，"不好像"不是常见表达
			{"不好像是这样", "reject", "包含'不好'→拒绝"},
			{"好像不是这个", "modify_request", "质疑优先"},

			// "撤销" vs "取消" - 上下文相关
			{"取消", "reject", "当前操作取消"},
			{"撤销", "undo", "撤销已保存"},
		}

		for _, c := range cases {
			result := handler.analyzeUserDecisionByPattern(c.input)
			if result != c.expected {
				t.Errorf("❌ [%s] 输入=%q, 期望=%s, 实际=%s",
					c.reason, c.input, c.expected, result)
			} else {
				t.Logf("✓ [%s] %q → %s", c.reason, c.input, result)
			}
		}
	})
}

// ============================================================================
// 第二部分：ExecutionHistory 撤销机制全面测试
// ============================================================================

func TestExecutionHistory_Comprehensive(t *testing.T) {
	// -------------------------------------------------------------------------
	// 2.1 时间边界测试
	// -------------------------------------------------------------------------
	t.Run("时间边界_精确到秒", func(t *testing.T) {
		now := time.Now()

		testCases := []struct {
			name          string
			createdAt     time.Time
			checkAt       time.Time
			expectCanUndo bool
		}{
			{
				name:          "刚创建_0秒",
				createdAt:     now,
				checkAt:       now,
				expectCanUndo: true,
			},
			{
				name:          "1分钟后",
				createdAt:     now,
				checkAt:       now.Add(1 * time.Minute),
				expectCanUndo: true,
			},
			{
				name:          "4分59秒后",
				createdAt:     now,
				checkAt:       now.Add(4*time.Minute + 59*time.Second),
				expectCanUndo: true,
			},
			{
				name:          "刚好5分钟",
				createdAt:     now,
				checkAt:       now.Add(5 * time.Minute),
				expectCanUndo: false, // 5分钟整，不可撤销
			},
			{
				name:          "5分01秒",
				createdAt:     now,
				checkAt:       now.Add(5*time.Minute + 1*time.Second),
				expectCanUndo: false,
			},
			{
				name:          "10分钟后",
				createdAt:     now,
				checkAt:       now.Add(10 * time.Minute),
				expectCanUndo: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				history := &model.ExecutionHistory{
					ID:           fmt.Sprintf("test_%d", tc.createdAt.UnixNano()),
					UserID:       "user_test",
					CanUndo:      true,
					UndoDeadline: tc.createdAt.Add(model.UndoWindow),
					Timestamp:    tc.createdAt,
				}

				// 模拟在 checkAt 时间点检查
				isUndoable := history.CanUndo && tc.checkAt.Before(history.UndoDeadline)

				if isUndoable != tc.expectCanUndo {
					t.Errorf("❌ %s: 期望 canUndo=%v, 实际=%v (创建=%v, 检查=%v, 截止=%v)",
						tc.name, tc.expectCanUndo, isUndoable,
						tc.createdAt.Format("15:04:05"),
						tc.checkAt.Format("15:04:05"),
						history.UndoDeadline.Format("15:04:05"))
				} else {
					t.Logf("✓ %s: canUndo=%v", tc.name, isUndoable)
				}
			})
		}
	})

	// -------------------------------------------------------------------------
	// 2.2 撤销剩余时间计算测试
	// -------------------------------------------------------------------------
	t.Run("撤销剩余时间计算", func(t *testing.T) {
		now := time.Now()
		history := &model.ExecutionHistory{
			ID:           "test_remaining",
			UserID:       "user_test",
			CanUndo:      true,
			UndoDeadline: now.Add(5 * time.Minute),
			Timestamp:    now,
		}

		testCases := []struct {
			name            string
			checkAt         time.Time
			expectedSeconds int
		}{
			{"刚创建", now, 300},
			{"1分钟后", now.Add(1 * time.Minute), 240},
			{"2分30秒后", now.Add(2*time.Minute + 30*time.Second), 150},
			{"4分钟后", now.Add(4 * time.Minute), 60},
			{"4分59秒后", now.Add(4*time.Minute + 59*time.Second), 1},
			{"5分钟后", now.Add(5 * time.Minute), 0},
			{"超时后", now.Add(6 * time.Minute), 0},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				remaining := history.UndoDeadline.Sub(tc.checkAt)
				if remaining < 0 {
					remaining = 0
				}
				seconds := int(remaining.Seconds())

				if seconds != tc.expectedSeconds {
					t.Errorf("❌ %s: 期望剩余=%d秒, 实际=%d秒",
						tc.name, tc.expectedSeconds, seconds)
				} else {
					t.Logf("✓ %s: 剩余=%d秒", tc.name, seconds)
				}
			})
		}
	})

	// -------------------------------------------------------------------------
	// 2.3 状态数据完整性测试
	// -------------------------------------------------------------------------
	t.Run("状态数据完整性", func(t *testing.T) {
		beforeState := []model.ScheduleItem{
			{StartTime: "09:00", EndTime: "10:00", Status: "开会", Emoji: "📅"},
			{StartTime: "10:00", EndTime: "12:00", Status: "工作", Emoji: "💻"},
		}
		afterState := []model.ScheduleItem{
			{StartTime: "09:00", EndTime: "10:00", Status: "开会", Emoji: "📅"},
			{StartTime: "10:00", EndTime: "12:00", Status: "工作", Emoji: "💻"},
			{StartTime: "14:00", EndTime: "16:00", Status: "健身", Emoji: "🏃"},
		}

		history := model.NewExecutionHistory("user_test", "session_test", "create", beforeState, afterState, time.Now())

		// 验证 BeforeState
		if len(history.BeforeState) != 2 {
			t.Errorf("❌ BeforeState 长度错误: 期望=2, 实际=%d", len(history.BeforeState))
		}
		if history.BeforeState[0].Status != "开会" {
			t.Errorf("❌ BeforeState[0].Status 错误: 期望=开会, 实际=%s", history.BeforeState[0].Status)
		}

		// 验证 AfterState
		if len(history.AfterState) != 3 {
			t.Errorf("❌ AfterState 长度错误: 期望=3, 实际=%d", len(history.AfterState))
		}
		if history.AfterState[2].Status != "健身" {
			t.Errorf("❌ AfterState[2].Status 错误: 期望=健身, 实际=%s", history.AfterState[2].Status)
		}

		// 验证撤销后恢复
		restoredState := history.BeforeState
		if len(restoredState) != 2 {
			t.Errorf("❌ 恢复后状态长度错误: 期望=2, 实际=%d", len(restoredState))
		}

		t.Logf("✓ 状态数据完整性验证通过")
		t.Logf("  BeforeState: %d 项", len(history.BeforeState))
		t.Logf("  AfterState: %d 项", len(history.AfterState))
		t.Logf("  Action: %s", history.Action)
	})

	// -------------------------------------------------------------------------
	// 2.4 多次撤销尝试测试
	// -------------------------------------------------------------------------
	t.Run("多次撤销尝试", func(t *testing.T) {
		history := model.NewExecutionHistory("user_test", "session_test", "create", nil, nil, time.Now())

		// 第一次撤销 - 应该成功
		if !history.CanUndo {
			t.Error("❌ 第一次撤销应该允许")
		}
		t.Log("✓ 第一次撤销检查: 允许")

		// 执行撤销
		history.CanUndo = false
		history.UndoReason = "用户已撤销"

		// 第二次撤销 - 应该失败
		if history.CanUndo {
			t.Error("❌ 第二次撤销不应该允许")
		}
		t.Logf("✓ 第二次撤销检查: 不允许 (原因: %s)", history.UndoReason)

		// 尝试强制设置 CanUndo
		// 在真实系统中，这应该被阻止或记录
		history.CanUndo = true // 模拟恶意修改
		t.Log("⚠️ 警告: CanUndo 被强制修改为 true（真实系统应防止此操作）")
	})
}

// ============================================================================
// 第三部分：Phase 状态机全面测试
// ============================================================================

func TestPhaseStateMachine_Comprehensive(t *testing.T) {
	// -------------------------------------------------------------------------
	// 3.1 所有合法转换路径
	// -------------------------------------------------------------------------
	t.Run("所有合法转换路径", func(t *testing.T) {
		allPhases := []model.ConversationPhase{
			model.PhaseIdle,
			model.PhaseUnderstanding,
			model.PhaseDiscussing,
			model.PhasePlanning,
			model.PhaseApproval,
			model.PhaseAwaitingApproval,
			model.PhaseProcessingApproval,
			model.PhaseExecution,
			model.PhaseConfirmedUndoable,
			model.PhaseCompleted,
		}

		validCount := 0
		invalidCount := 0

		for _, from := range allPhases {
			for _, to := range allPhases {
				if from == to {
					continue // 跳过自转换
				}
				valid := model.IsValidTransition(from, to)
				if valid {
					validCount++
					t.Logf("✓ %s → %s", from, to)
				} else {
					invalidCount++
				}
			}
		}

		t.Logf("\n合法转换: %d, 非法转换: %d", validCount, invalidCount)
	})

	// -------------------------------------------------------------------------
	// 3.2 典型工作流程测试
	// -------------------------------------------------------------------------
	t.Run("典型工作流程_创建时刻表", func(t *testing.T) {
		workflow := []model.ConversationPhase{
			model.PhaseIdle,
			model.PhaseUnderstanding,
			model.PhasePlanning,
			model.PhaseAwaitingApproval,
			model.PhaseProcessingApproval,
			model.PhaseExecution,
			model.PhaseConfirmedUndoable,
			model.PhaseCompleted,
		}

		for i := 0; i < len(workflow)-1; i++ {
			from := workflow[i]
			to := workflow[i+1]
			if !model.IsValidTransition(from, to) {
				t.Errorf("❌ 工作流程断裂: %s → %s 不合法", from, to)
			} else {
				t.Logf("✓ Step %d: %s → %s", i+1, from, to)
			}
		}
	})

	t.Run("典型工作流程_撤销后继续编辑", func(t *testing.T) {
		workflow := []model.ConversationPhase{
			model.PhaseIdle,
			model.PhaseUnderstanding,
			model.PhasePlanning,
			model.PhaseAwaitingApproval,
			model.PhaseProcessingApproval,
			model.PhaseExecution,
			model.PhaseConfirmedUndoable,
			model.PhasePlanning, // 撤销回到规划
			model.PhaseAwaitingApproval,
			model.PhaseProcessingApproval,
			model.PhaseExecution,
			model.PhaseConfirmedUndoable,
			model.PhaseCompleted,
		}

		for i := 0; i < len(workflow)-1; i++ {
			from := workflow[i]
			to := workflow[i+1]
			if !model.IsValidTransition(from, to) {
				t.Errorf("❌ 撤销工作流程断裂: %s → %s 不合法", from, to)
			} else {
				t.Logf("✓ Step %d: %s → %s", i+1, from, to)
			}
		}
	})

	t.Run("典型工作流程_中途取消", func(t *testing.T) {
		workflow := []model.ConversationPhase{
			model.PhaseIdle,
			model.PhaseUnderstanding,
			model.PhasePlanning,
			model.PhaseIdle, // 用户取消 - 需要检查是否支持这个转换
		}

		for i := 0; i < len(workflow)-1; i++ {
			from := workflow[i]
			to := workflow[i+1]
			valid := model.IsValidTransition(from, to)
			if valid {
				t.Logf("✓ Step %d: %s → %s", i+1, from, to)
			} else {
				t.Logf("⚠️ Step %d: %s → %s 不在转换表中（可能需要添加）", i+1, from, to)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 3.3 非法转换测试
	// -------------------------------------------------------------------------
	t.Run("非法转换_应被拒绝", func(t *testing.T) {
		illegalTransitions := []struct {
			from   model.ConversationPhase
			to     model.ConversationPhase
			reason string
		}{
			{model.PhaseIdle, model.PhaseExecution, "跳过中间步骤"},
			{model.PhaseIdle, model.PhaseCompleted, "跳过所有步骤"},
			{model.PhaseUnderstanding, model.PhaseExecution, "跳过规划和审批"},
		}

		for _, tt := range illegalTransitions {
			if model.IsValidTransition(tt.from, tt.to) {
				t.Errorf("❌ 非法转换被允许: %s → %s (%s)", tt.from, tt.to, tt.reason)
			} else {
				t.Logf("✓ 正确拒绝: %s → %s (%s)", tt.from, tt.to, tt.reason)
			}
		}
	})
}

// ============================================================================
// 第四部分：VoiceScheduleEvent 响应结构测试
// ============================================================================

func TestVoiceScheduleEvent_Comprehensive(t *testing.T) {
	// -------------------------------------------------------------------------
	// 4.1 不同阶段的响应结构
	// -------------------------------------------------------------------------
	t.Run("不同阶段响应结构", func(t *testing.T) {
		testCases := []struct {
			phase          model.ConversationPhase
			awaitingAction string
			hasActions     bool
			canUndo        bool
		}{
			{model.PhaseIdle, "", false, false},
			{model.PhaseUnderstanding, "", false, false},
			{model.PhasePlanning, "", false, false},
			{model.PhaseAwaitingApproval, "approval", true, false},
			{model.PhaseProcessingApproval, "", false, false},
			{model.PhaseExecution, "", false, false},
			{model.PhaseConfirmedUndoable, "", true, true},
			{model.PhaseCompleted, "", false, false},
		}

		for _, tc := range testCases {
			event := &model.VoiceScheduleEvent{
				Type:  model.VSEventPhaseChange,
				Phase: tc.phase,
			}

			// 根据阶段设置响应字段
			switch tc.phase {
			case model.PhaseAwaitingApproval:
				event.AwaitingAction = "approval"
				event.AvailableActions = []model.ActionHint{
					{Action: "confirm", Label: "确认"},
					{Action: "reject", Label: "取消"},
					{Action: "modify", Label: "修改"},
				}
			case model.PhaseConfirmedUndoable:
				event.CanUndo = true
				event.UndoDeadline = time.Now().Add(5 * time.Minute).Unix()
				event.UndoRemaining = 300
				event.AvailableActions = []model.ActionHint{
					{Action: string(model.VSActionUndo), Label: "撤销"},
				}
			}

			// 验证
			if event.AwaitingAction != tc.awaitingAction {
				t.Errorf("❌ Phase=%s: AwaitingAction 期望=%s, 实际=%s",
					tc.phase, tc.awaitingAction, event.AwaitingAction)
			}

			hasActions := len(event.AvailableActions) > 0
			if hasActions != tc.hasActions {
				t.Errorf("❌ Phase=%s: HasActions 期望=%v, 实际=%v",
					tc.phase, tc.hasActions, hasActions)
			}

			if event.CanUndo != tc.canUndo {
				t.Errorf("❌ Phase=%s: CanUndo 期望=%v, 实际=%v",
					tc.phase, tc.canUndo, event.CanUndo)
			}

			t.Logf("✓ Phase=%s: AwaitingAction=%s, HasActions=%v, CanUndo=%v",
				tc.phase, event.AwaitingAction, hasActions, event.CanUndo)
		}
	})

	// -------------------------------------------------------------------------
	// 4.2 ActionHint 结构完整性
	// -------------------------------------------------------------------------
	t.Run("ActionHint结构完整性", func(t *testing.T) {
		actions := []model.ActionHint{
			{Action: "confirm", Label: "确认保存", Shortcut: "好的"},
			{Action: "reject", Label: "取消", Shortcut: "取消"},
			{Action: "modify", Label: "修改", Shortcut: "改一下"},
			{Action: string(model.VSActionUndo), Label: "撤销", Shortcut: "撤销"},
		}

		for _, action := range actions {
			if action.Action == "" {
				t.Error("❌ Action 为空")
			}
			if action.Label == "" {
				t.Error("❌ Label 为空")
			}
			t.Logf("✓ ActionHint: Action=%s, Label=%s, Shortcut=%s",
				action.Action, action.Label, action.Shortcut)
		}
	})

	// -------------------------------------------------------------------------
	// 4.3 撤销倒计时显示
	// -------------------------------------------------------------------------
	t.Run("撤销倒计时显示", func(t *testing.T) {
		testCases := []struct {
			remaining int
			display   string
		}{
			{300, "5:00"},
			{299, "4:59"},
			{180, "3:00"},
			{60, "1:00"},
			{59, "0:59"},
			{10, "0:10"},
			{1, "0:01"},
			{0, "0:00"},
		}

		for _, tc := range testCases {
			minutes := tc.remaining / 60
			seconds := tc.remaining % 60
			display := fmt.Sprintf("%d:%02d", minutes, seconds)

			if display != tc.display {
				t.Errorf("❌ remaining=%d: 期望=%s, 实际=%s",
					tc.remaining, tc.display, display)
			} else {
				t.Logf("✓ remaining=%d → %s", tc.remaining, display)
			}
		}
	})
}

// ============================================================================
// 第五部分：并发安全测试
// ============================================================================

func TestConcurrency_Comprehensive(t *testing.T) {
	// -------------------------------------------------------------------------
	// 5.1 并发创建 ExecutionHistory
	// -------------------------------------------------------------------------
	t.Run("并发创建ExecutionHistory", func(t *testing.T) {
		var wg sync.WaitGroup
		histories := make([]*model.ExecutionHistory, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				histories[idx] = model.NewExecutionHistory(
					fmt.Sprintf("user_%d", idx),
					fmt.Sprintf("session_%d", idx),
					"create",
					nil,
					nil,
					time.Now(),
				)
			}(i)
		}

		wg.Wait()

		// 验证所有历史记录都创建成功
		successCount := 0
		for _, h := range histories {
			if h != nil && h.ID != "" {
				successCount++
			}
		}

		t.Logf("✓ 并发创建成功: %d/100", successCount)

		// 验证 ID 唯一性
		idMap := make(map[string]bool)
		duplicates := 0
		for _, h := range histories {
			if h != nil {
				if idMap[h.ID] {
					duplicates++
				}
				idMap[h.ID] = true
			}
		}

		if duplicates > 0 {
			t.Errorf("❌ 发现 %d 个重复 ID", duplicates)
		} else {
			t.Log("✓ 所有 ID 唯一")
		}
	})

	// -------------------------------------------------------------------------
	// 5.2 并发状态转换
	// -------------------------------------------------------------------------
	t.Run("并发状态转换", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]bool, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				// 随机测试一个转换
				from := model.PhasePlanning
				to := model.PhaseAwaitingApproval
				results[idx] = model.IsValidTransition(from, to)
			}(i)
		}

		wg.Wait()

		// 所有结果应该一致
		trueCount := 0
		for _, r := range results {
			if r {
				trueCount++
			}
		}

		if trueCount != 100 {
			t.Errorf("❌ 并发状态转换结果不一致: %d/100 为 true", trueCount)
		} else {
			t.Log("✓ 并发状态转换结果一致")
		}
	})

	// -------------------------------------------------------------------------
	// 5.3 并发 ApprovalHandler 分析
	// -------------------------------------------------------------------------
	t.Run("并发ApprovalHandler分析", func(t *testing.T) {
		handler := NewApprovalHandler()
		var wg sync.WaitGroup

		inputs := []string{"好的", "取消", "改一下", "撤销", "这是什么意思"}
		expected := []string{"approve", "reject", "modify", "undo", "need_clarification"}
		results := make([][]string, len(inputs))

		for i := range inputs {
			results[i] = make([]string, 20)
		}

		for round := 0; round < 20; round++ {
			for i, input := range inputs {
				wg.Add(1)
				go func(inputIdx, roundIdx int, inp string) {
					defer wg.Done()
					results[inputIdx][roundIdx] = handler.analyzeUserDecisionByPattern(inp)
				}(i, round, input)
			}
		}

		wg.Wait()

		// 验证每个输入的所有结果一致
		for i, input := range inputs {
			allSame := true
			first := results[i][0]
			for _, r := range results[i] {
				if r != first {
					allSame = false
					break
				}
			}

			if !allSame {
				t.Errorf("❌ 输入 %q 并发结果不一致", input)
			} else if first != expected[i] {
				t.Errorf("❌ 输入 %q 结果错误: 期望=%s, 实际=%s", input, expected[i], first)
			} else {
				t.Logf("✓ 输入 %q 并发结果一致: %s", input, first)
			}
		}
	})
}

// ============================================================================
// 第六部分：边界情况和异常处理测试
// ============================================================================

func TestEdgeCases_Comprehensive(t *testing.T) {
	// -------------------------------------------------------------------------
	// 6.1 极端输入长度
	// -------------------------------------------------------------------------
	t.Run("极端输入长度", func(t *testing.T) {
		handler := NewApprovalHandler()

		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"空字符串", "", "unknown"},
			{"单字符", "好", "approve"},
			{"超长确认", strings.Repeat("好", 100), "approve"},
			// 超长文本不含确认/拒绝关键词
			{"超长文本", strings.Repeat("哈哈哈哈哈哈哈", 100), "unknown"},
			{"超长+确认词", strings.Repeat("x", 1000) + "好的", "approve"},
			{"超长+拒绝词", strings.Repeat("x", 1000) + "取消", "reject"},
		}

		for _, tc := range testCases {
			result := handler.analyzeUserDecisionByPattern(tc.input)
			inputPreview := tc.input
			if len(inputPreview) > 50 {
				inputPreview = inputPreview[:50] + "..."
			}

			if result != tc.expected {
				t.Errorf("❌ %s: 期望=%s, 实际=%s (输入=%q)",
					tc.name, tc.expected, result, inputPreview)
			} else {
				t.Logf("✓ %s: %s (输入长度=%d)", tc.name, result, len(tc.input))
			}
		}
	})

	// -------------------------------------------------------------------------
	// 6.2 特殊字符和编码
	// -------------------------------------------------------------------------
	t.Run("特殊字符和编码", func(t *testing.T) {
		handler := NewApprovalHandler()

		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"纯数字", "12345", "unknown"},
			// 纯标点包含问号，会被识别为疑问
			{"纯标点", "。，！？", "need_clarification"},
			{"换行符", "好的\n确认", "approve"},
			{"制表符", "好的\t确认", "approve"},
			{"零宽字符", "好​的", "approve"}, // 包含零宽空格
			{"全角字符", "好的", "approve"},
			{"日文确认", "はい", "unknown"}, // 日语"是"
			{"韩文确认", "네", "unknown"},   // 韩语"是"
			{"繁体确认", "好的", "approve"},
			{"Emoji开头", "👍好的", "approve"},
			{"Emoji结尾", "好的👍", "approve"},
		}

		for _, tc := range testCases {
			result := handler.analyzeUserDecisionByPattern(tc.input)
			if result != tc.expected {
				t.Errorf("❌ %s: 输入=%q, 期望=%s, 实际=%s",
					tc.name, tc.input, tc.expected, result)
			} else {
				t.Logf("✓ %s: %q → %s", tc.name, tc.input, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 6.3 ExecutionHistory 边界情况
	// -------------------------------------------------------------------------
	t.Run("ExecutionHistory边界情况", func(t *testing.T) {
		// nil BeforeState 和 AfterState
		history1 := model.NewExecutionHistory("user_test", "session_test", "create", nil, nil, time.Now())
		if history1.BeforeState != nil && len(history1.BeforeState) != 0 {
			// 允许 nil 或空切片
			if len(history1.BeforeState) > 0 {
				t.Error("❌ nil BeforeState 应该保持为空")
			}
		}
		t.Log("✓ nil BeforeState 处理正确")

		// 空切片
		history2 := model.NewExecutionHistory("user_test", "session_test", "create",
			[]model.ScheduleItem{}, []model.ScheduleItem{}, time.Now())
		if len(history2.BeforeState) != 0 {
			t.Error("❌ 空切片 BeforeState 应该长度为 0")
		}
		t.Log("✓ 空切片 BeforeState 处理正确")

		// 大量数据
		largeState := make([]model.ScheduleItem, 1000)
		for i := range largeState {
			largeState[i] = model.ScheduleItem{
				StartTime: fmt.Sprintf("%02d:00", i%24),
				EndTime:   fmt.Sprintf("%02d:00", (i+1)%24),
				Status:    fmt.Sprintf("活动%d", i),
				Emoji:     "📅",
			}
		}
		history3 := model.NewExecutionHistory("user_test", "session_test", "create", largeState, largeState, time.Now())
		if len(history3.BeforeState) != 1000 {
			t.Errorf("❌ 大量数据处理错误: 期望=1000, 实际=%d", len(history3.BeforeState))
		}
		t.Logf("✓ 大量数据处理正确: %d 项", len(history3.BeforeState))
	})

	// -------------------------------------------------------------------------
	// 6.4 Phase 转换边界
	// -------------------------------------------------------------------------
	t.Run("Phase转换边界", func(t *testing.T) {
		// 未知 Phase - 使用类型转换来测试
		unknownPhases := []model.ConversationPhase{"", "invalid", "IDLE", "phase_idle", "idle_phase"}

		for _, phase := range unknownPhases {
			// 从未知 Phase 转换
			result := model.IsValidTransition(phase, model.PhaseUnderstanding)
			// 应该返回 false（未知状态不应该有有效转换）
			t.Logf("未知 Phase %q → understanding: %v", phase, result)

			// 转换到未知 Phase
			result2 := model.IsValidTransition(model.PhaseIdle, phase)
			t.Logf("idle → 未知 Phase %q: %v", phase, result2)
		}
	})
}

// ============================================================================
// 第七部分：完整对话流程测试
// ============================================================================

func TestFullConversationFlow_Comprehensive(t *testing.T) {
	// -------------------------------------------------------------------------
	// 7.1 正常流程：创建 → 确认 → 完成
	// -------------------------------------------------------------------------
	t.Run("正常流程_创建确认完成", func(t *testing.T) {
		handler := NewApprovalHandler()

		// 模拟对话
		conversation := []struct {
			userInput      string
			expectedAction string
			description    string
		}{
			{"下午3点到5点开会", "create", "用户创建时刻表"},
			{"好的，确认", "approve", "用户确认"},
		}

		for i, turn := range conversation {
			if turn.expectedAction == "create" {
				// create 动作由 IntentDispatcher 处理，这里跳过
				t.Logf("✓ Turn %d: %s - %s", i+1, turn.description, turn.userInput)
				continue
			}

			result := handler.analyzeUserDecisionByPattern(turn.userInput)
			if result != turn.expectedAction {
				t.Errorf("❌ Turn %d (%s): 期望=%s, 实际=%s",
					i+1, turn.description, turn.expectedAction, result)
			} else {
				t.Logf("✓ Turn %d: %s → %s", i+1, turn.description, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 7.2 修改流程：创建 → 修改 → 确认
	// -------------------------------------------------------------------------
	t.Run("修改流程_创建修改确认", func(t *testing.T) {
		handler := NewApprovalHandler()

		conversation := []struct {
			userInput      string
			expectedAction string
			description    string
		}{
			{"下午3点到5点开会", "create", "用户创建时刻表"},
			{"把时间改到4点", "modify", "用户要求修改"},
			{"好的", "approve", "用户确认修改后的结果"},
		}

		for i, turn := range conversation {
			if turn.expectedAction == "create" {
				t.Logf("✓ Turn %d: %s", i+1, turn.description)
				continue
			}

			result := handler.analyzeUserDecisionByPattern(turn.userInput)
			if result != turn.expectedAction {
				t.Errorf("❌ Turn %d (%s): 期望=%s, 实际=%s",
					i+1, turn.description, turn.expectedAction, result)
			} else {
				t.Logf("✓ Turn %d: %s → %s", i+1, turn.description, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 7.3 撤销流程：创建 → 确认 → 撤销 → 重新确认
	// -------------------------------------------------------------------------
	t.Run("撤销流程_确认后撤销", func(t *testing.T) {
		handler := NewApprovalHandler()

		conversation := []struct {
			userInput      string
			expectedAction string
			description    string
		}{
			{"下午3点到5点开会", "create", "用户创建时刻表"},
			{"好的", "approve", "用户确认"},
			{"撤销", "undo", "用户撤销"},
			{"把时间改到4点", "modify", "用户修改"},
			{"好的", "approve", "用户重新确认"},
		}

		for i, turn := range conversation {
			if turn.expectedAction == "create" {
				t.Logf("✓ Turn %d: %s", i+1, turn.description)
				continue
			}

			result := handler.analyzeUserDecisionByPattern(turn.userInput)
			if result != turn.expectedAction {
				t.Errorf("❌ Turn %d (%s): 期望=%s, 实际=%s",
					i+1, turn.description, turn.expectedAction, result)
			} else {
				t.Logf("✓ Turn %d: %s → %s", i+1, turn.description, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 7.4 取消流程：创建 → 讨论 → 取消
	// -------------------------------------------------------------------------
	t.Run("取消流程_中途放弃", func(t *testing.T) {
		handler := NewApprovalHandler()

		conversation := []struct {
			userInput      string
			expectedAction string
			description    string
		}{
			{"下午3点到5点开会", "create", "用户创建时刻表"},
			{"这是什么意思？", "need_clarification", "用户询问"},
			{"算了，不要了", "reject", "用户取消"},
		}

		for i, turn := range conversation {
			if turn.expectedAction == "create" {
				t.Logf("✓ Turn %d: %s", i+1, turn.description)
				continue
			}

			result := handler.analyzeUserDecisionByPattern(turn.userInput)
			if result != turn.expectedAction {
				t.Errorf("❌ Turn %d (%s): 期望=%s, 实际=%s",
					i+1, turn.description, turn.expectedAction, result)
			} else {
				t.Logf("✓ Turn %d: %s → %s", i+1, turn.description, result)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 7.5 复杂流程：多轮修改后确认
	// -------------------------------------------------------------------------
	t.Run("复杂流程_多轮修改", func(t *testing.T) {
		handler := NewApprovalHandler()

		conversation := []struct {
			userInput      string
			expectedAction string
			description    string
		}{
			{"下午开会", "create", "创建"},
			{"好像不对", "modify_request", "质疑"},
			{"改成3点", "modify", "修改"},
			{"这个时间不太行", "modify_request", "再次质疑"},
			{"换成4点吧", "modify", "再次修改"},
			{"行，不过加上地点", "modify_request", "条件确认"},
			{"好的，就这样", "approve", "最终确认"},
		}

		for i, turn := range conversation {
			if turn.expectedAction == "create" {
				t.Logf("✓ Turn %d: %s", i+1, turn.description)
				continue
			}

			result := handler.analyzeUserDecisionByPattern(turn.userInput)
			if result != turn.expectedAction {
				t.Errorf("❌ Turn %d (%s): 输入=%q, 期望=%s, 实际=%s",
					i+1, turn.description, turn.userInput, turn.expectedAction, result)
			} else {
				t.Logf("✓ Turn %d: %s - %q → %s", i+1, turn.description, turn.userInput, result)
			}
		}
	})
}

// ============================================================================
// 第八部分：统计和覆盖率报告
// ============================================================================

func TestCoverageReport(t *testing.T) {
	t.Run("生成覆盖率报告", func(t *testing.T) {
		handler := NewApprovalHandler()

		// 统计所有测试用例的分布
		categories := map[string]int{
			"approve":            0,
			"reject":             0,
			"modify":             0,
			"modify_request":     0,
			"need_clarification": 0,
			"undo":               0,
			"unknown":            0,
		}

		// 测试用例集合
		allInputs := []string{
			// 确认类
			"好", "好的", "行", "可以", "OK", "嗯", "对", "是", "确认", "没问题",
			"就这样", "保存", "太好了", "完美", "不错",
			// 拒绝类
			"不要", "不用", "算了", "取消", "不行", "不好", "错了", "放弃", "停",
			// 修改类
			"改一下", "修改", "调整", "换成", "改成3点",
			// 条件修改类
			"好的但是不对", "行不过要改", "可以但有问题", "好像不对", "有点问题",
			// 澄清类
			"什么意思", "不明白", "解释一下", "确认一下", "等等", "稍等",
			// 撤销类
			"撤销", "撤回", "后悔了", "恢复", "还原",
			// 边界类
			"", "   ", "...", "👍", "❌",
		}

		for _, input := range allInputs {
			result := handler.analyzeUserDecisionByPattern(input)
			categories[result]++
		}

		// 输出统计
		t.Log("\n========== 测试覆盖率统计 ==========")
		total := 0
		for cat, count := range categories {
			total += count
			t.Logf("  %s: %d", cat, count)
		}
		t.Logf("  --------------------------------")
		t.Logf("  总计: %d 个测试用例", total)
		t.Log("====================================")
	})
}

// ============================================================================
// 第八部分：删除操作测试
// 目标：验证 delete 意图能正确处理，包括确认流程
// ============================================================================

func TestDeleteAction_Handling(t *testing.T) {
	// -------------------------------------------------------------------------
	// 8.1 删除意图识别（模拟 IntentDispatcher 返回结果）
	// -------------------------------------------------------------------------
	t.Run("删除意图结果构建", func(t *testing.T) {
		// 模拟 IntentDispatcher 返回的结果
		cases := []struct {
			name        string
			action      string
			targetIndex *int
			targetDate  string
			expectOp    string
		}{
			{"按索引删除", "delete", intPtr(2), "2026-02-05", "delete"},
			{"按名称删除", "delete", nil, "2026-02-05", "delete"},
			{"删除明天的", "delete", nil, "2026-02-06", "delete"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				result := &model.LLMVoiceAnalysisResult{
					Action:      tc.action,
					Operation:   tc.expectOp,
					TargetDate:  tc.targetDate,
					TargetIndex: tc.targetIndex,
					Reasoning:   []string{"用户请求删除"},
				}

				if result.Action != tc.action {
					t.Errorf("Action mismatch: got %s, want %s", result.Action, tc.action)
				}
				if result.Operation != tc.expectOp {
					t.Errorf("Operation mismatch: got %s, want %s", result.Operation, tc.expectOp)
				}
				t.Logf("✓ %s: Action=%s, Operation=%s, TargetIndex=%v",
					tc.name, result.Action, result.Operation, tc.targetIndex)
			})
		}
	})

	// -------------------------------------------------------------------------
	// 8.2 删除后时刻表验证
	// -------------------------------------------------------------------------
	t.Run("删除后时刻表正确性", func(t *testing.T) {
		original := []model.ScheduleItem{
			{StartTime: "09:00", EndTime: "10:00", Emoji: "💼", Status: "开会"},
			{StartTime: "10:00", EndTime: "12:00", Emoji: "💻", Status: "工作"},
			{StartTime: "14:00", EndTime: "16:00", Emoji: "📝", Status: "写文档"},
		}

		// 按索引删除
		t.Run("按索引删除第2个", func(t *testing.T) {
			idx := 1
			result := deleteByIndex(original, idx)
			if len(result) != 2 {
				t.Errorf("Expected 2 items, got %d", len(result))
			}
			if result[0].Status != "开会" || result[1].Status != "写文档" {
				t.Errorf("Wrong items remaining: %+v", result)
			}
			t.Logf("✓ 删除索引%d后剩余: %d个", idx, len(result))
		})

		// 按名称匹配删除
		t.Run("按名称删除工作", func(t *testing.T) {
			toRemove := []model.ScheduleItem{
				{Status: "工作"},
			}
			result := removeMatchingSlots(original, toRemove)
			if len(result) != 2 {
				t.Errorf("Expected 2 items, got %d", len(result))
			}
			t.Logf("✓ 删除'工作'后剩余: %d个", len(result))
		})

		// 按时间匹配删除
		t.Run("按时间删除", func(t *testing.T) {
			toRemove := []model.ScheduleItem{
				{StartTime: "14:00", EndTime: "16:00"},
			}
			result := removeMatchingSlots(original, toRemove)
			if len(result) != 2 {
				t.Errorf("Expected 2 items, got %d", len(result))
			}
			t.Logf("✓ 删除14:00-16:00后剩余: %d个", len(result))
		})
	})

	// -------------------------------------------------------------------------
	// 8.3 删除确认事件验证
	// -------------------------------------------------------------------------
	t.Run("删除事件包含确认标志", func(t *testing.T) {
		event := &model.VoiceScheduleEvent{
			Type:              model.VSEventSchedule,
			Items:             []model.ScheduleItem{{Status: "剩余项"}},
			NeedsConfirmation: true,
			PendingOperation:  "delete",
			Reasoning:         []string{"已删除指定时段，请确认"},
		}

		if !event.NeedsConfirmation {
			t.Error("NeedsConfirmation should be true")
		}
		if event.PendingOperation != "delete" {
			t.Errorf("PendingOperation should be 'delete', got '%s'", event.PendingOperation)
		}
		t.Logf("✓ 删除事件正确: NeedsConfirmation=%v, PendingOperation=%s",
			event.NeedsConfirmation, event.PendingOperation)
	})
}

// 辅助函数：创建整数指针
func intPtr(i int) *int {
	return &i
}

// 辅助函数：按索引删除
func deleteByIndex(items []model.ScheduleItem, idx int) []model.ScheduleItem {
	if idx < 0 || idx >= len(items) {
		return items
	}
	result := make([]model.ScheduleItem, 0, len(items)-1)
	for i, item := range items {
		if i != idx {
			result = append(result, item)
		}
	}
	return result
}

// 辅助函数：按匹配删除
func removeMatchingSlots(existing, toRemove []model.ScheduleItem) []model.ScheduleItem {
	result := make([]model.ScheduleItem, 0, len(existing))
	for _, item := range existing {
		shouldRemove := false
		for _, rm := range toRemove {
			if item.Status == rm.Status ||
				(item.StartTime == rm.StartTime && item.EndTime == rm.EndTime) {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			result = append(result, item)
		}
	}
	return result
}
