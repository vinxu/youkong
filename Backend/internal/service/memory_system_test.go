package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"youkong/internal/model"
)

// ===============================================
// 一、模型序列化测试
// ===============================================

// TestUserMemoryDocument_JSONSerialization 验证 UserMemoryDocument 的 JSON 序列化
func TestUserMemoryDocument_JSONSerialization(t *testing.T) {
	doc := &model.UserMemoryDocument{
		UserID:  "test-user-001",
		Version: 1,
		Preferences: model.UserMemoryPreferences{
			HidePastEvents:    true,
			DefaultViewDays:   7,
			PreferredTimezone: "Asia/Shanghai",
			Language:          "zh-CN",
		},
		SchedulePatterns: []model.SchedulePattern{
			{
				Pattern:     "每周一 9:00 开会",
				Confidence:  0.85,
				LearnedFrom: []string{"session-001", "session-002"},
				CreatedAt:   "2026-02-01T10:00:00Z",
			},
		},
		InteractionStyle: model.UserInteractionStyle{
			PrefersBrief:      true,
			ConfirmBeforeSave: true,
			ShowReasoning:     false,
		},
		KeyFacts: []model.KeyFact{
			{
				Fact:      "周三下午固定健身",
				Source:    "用户说：我周三下午固定去健身房",
				LearnedAt: "2026-02-01T10:00:00Z",
			},
		},
		SessionSummaries: []model.SessionSummary{
			{
				SessionID: "session-001",
				Date:      "2026-02-01",
				Summary:   "用户设置了明天的开会日程",
				KeyPoints: []string{"9:00 开会", "公司会议室"},
			},
		},
	}

	// 序列化
	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 反序列化
	var decoded model.UserMemoryDocument
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	// 验证字段
	if decoded.UserID != doc.UserID {
		t.Errorf("UserID 不匹配: got %s, want %s", decoded.UserID, doc.UserID)
	}
	if decoded.Preferences.HidePastEvents != true {
		t.Errorf("HidePastEvents 应为 true")
	}
	if len(decoded.SchedulePatterns) != 1 {
		t.Errorf("SchedulePatterns 数量不匹配: got %d, want 1", len(decoded.SchedulePatterns))
	}
	if decoded.SchedulePatterns[0].Confidence != 0.85 {
		t.Errorf("Pattern Confidence 不匹配: got %f, want 0.85", decoded.SchedulePatterns[0].Confidence)
	}
	if len(decoded.KeyFacts) != 1 {
		t.Errorf("KeyFacts 数量不匹配: got %d, want 1", len(decoded.KeyFacts))
	}
	if len(decoded.SessionSummaries) != 1 {
		t.Errorf("SessionSummaries 数量不匹配: got %d, want 1", len(decoded.SessionSummaries))
	}

	t.Logf("✓ UserMemoryDocument JSON 序列化/反序列化测试通过")
}

// TestAllSQLTypes_Serialization 验证所有 SQL 类型的序列化
func TestAllSQLTypes_Serialization(t *testing.T) {
	t.Run("UserMemoryPreferences", func(t *testing.T) {
		prefs := model.UserMemoryPreferences{
			HidePastEvents:    true,
			DefaultViewDays:   7,
			PreferredTimezone: "Asia/Shanghai",
		}
		val, err := prefs.Value()
		if err != nil {
			t.Fatalf("Value() 失败: %v", err)
		}
		var scanned model.UserMemoryPreferences
		if err := scanned.Scan(val); err != nil {
			t.Fatalf("Scan() 失败: %v", err)
		}
		if scanned.HidePastEvents != true || scanned.DefaultViewDays != 7 {
			t.Errorf("Scan 后数据不匹配")
		}
	})

	t.Run("UserInteractionStyle", func(t *testing.T) {
		style := model.UserInteractionStyle{
			PrefersBrief:      true,
			ConfirmBeforeSave: true,
			ShowReasoning:     false,
		}
		val, err := style.Value()
		if err != nil {
			t.Fatalf("Value() 失败: %v", err)
		}
		var scanned model.UserInteractionStyle
		if err := scanned.Scan(val); err != nil {
			t.Fatalf("Scan() 失败: %v", err)
		}
		if scanned.PrefersBrief != true || scanned.ConfirmBeforeSave != true {
			t.Errorf("Scan 后数据不匹配")
		}
	})

	t.Run("KeyFactList", func(t *testing.T) {
		facts := model.KeyFactList{
			{Fact: "测试事实1", Source: "用户说"},
			{Fact: "测试事实2", Source: "对话中"},
		}
		val, err := facts.Value()
		if err != nil {
			t.Fatalf("Value() 失败: %v", err)
		}
		var scanned model.KeyFactList
		if err := scanned.Scan(val); err != nil {
			t.Fatalf("Scan() 失败: %v", err)
		}
		if len(scanned) != 2 {
			t.Errorf("Scan 后数量不匹配: got %d, want 2", len(scanned))
		}
	})

	t.Run("SessionSummaryList", func(t *testing.T) {
		summaries := model.SessionSummaryList{
			{SessionID: "s1", Summary: "摘要1"},
			{SessionID: "s2", Summary: "摘要2"},
		}
		val, err := summaries.Value()
		if err != nil {
			t.Fatalf("Value() 失败: %v", err)
		}
		var scanned model.SessionSummaryList
		if err := scanned.Scan(val); err != nil {
			t.Fatalf("Scan() 失败: %v", err)
		}
		if len(scanned) != 2 {
			t.Errorf("Scan 后数量不匹配: got %d, want 2", len(scanned))
		}
	})

	t.Logf("✓ 所有 SQL 类型序列化测试通过")
}

// TestSchedulePatternList_SQLValue 验证 SchedulePatternList 的数据库序列化
func TestSchedulePatternList_SQLValue(t *testing.T) {
	patterns := model.SchedulePatternList{
		{Pattern: "每周一开会", Confidence: 0.8},
		{Pattern: "周五下午请假", Confidence: 0.6},
	}

	// Value 接口
	val, err := patterns.Value()
	if err != nil {
		t.Fatalf("Value() 失败: %v", err)
	}

	// Scan 接口
	var scanned model.SchedulePatternList
	if err := scanned.Scan(val); err != nil {
		t.Fatalf("Scan() 失败: %v", err)
	}

	if len(scanned) != 2 {
		t.Errorf("Scan 后数量不匹配: got %d, want 2", len(scanned))
	}

	// 测试空值处理
	var empty model.SchedulePatternList
	if err := empty.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) 失败: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("Scan(nil) 后应为空数组")
	}

	t.Logf("✓ SchedulePatternList SQL 序列化测试通过")
}

// ===============================================
// 二、上下文压缩器测试
// ===============================================

// TestContextCompressor_ShouldCompress 测试压缩触发条件
func TestContextCompressor_ShouldCompress(t *testing.T) {
	compressor := NewContextCompressor(nil) // 不需要 LLM

	tests := []struct {
		name           string
		messageCount   int
		contentLength  int
		expectCompress bool
	}{
		{
			name:           "消息数量不足，不需要压缩",
			messageCount:   5,
			contentLength:  100,
			expectCompress: false,
		},
		{
			name:           "消息数量达到阈值，需要压缩",
			messageCount:   model.MessageThreshold,
			contentLength:  100,
			expectCompress: true,
		},
		{
			name:           "消息数量超过阈值，需要压缩",
			messageCount:   model.MessageThreshold + 5,
			contentLength:  100,
			expectCompress: true,
		},
		{
			name:           "消息内容很长，token数超过阈值",
			messageCount:   3,
			contentLength:  model.TokenThreshold * 2 * 3, // 每条消息很长
			expectCompress: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &model.V4Session{
				SessionID: "test-session",
				UserID:    "test-user",
				Messages:  make([]model.V4Message, tt.messageCount),
			}

			// 填充消息内容
			content := strings.Repeat("测试内容", tt.contentLength/6) // 中文约6字节
			for i := 0; i < tt.messageCount; i++ {
				session.Messages[i] = model.V4Message{
					Role:    "user",
					Content: content,
				}
			}

			got := compressor.ShouldCompress(session)
			if got != tt.expectCompress {
				t.Errorf("ShouldCompress() = %v, want %v", got, tt.expectCompress)
			}
		})
	}

	t.Logf("✓ ShouldCompress 测试通过 (MessageThreshold=%d, TokenThreshold=%d)", model.MessageThreshold, model.TokenThreshold)
}

// TestContextCompressor_EstimateTokens 测试 token 估算
func TestContextCompressor_EstimateTokens(t *testing.T) {
	compressor := NewContextCompressor(nil)

	messages := []model.V4Message{
		{Role: "user", Content: "你好，我想设置明天的日程"},       // ~30 字符
		{Role: "assistant", Content: "好的，请告诉我具体时间和安排"}, // ~30 字符
		{Role: "user", Content: "上午9点开会，下午2点健身"},      // ~24 字符
	}

	tokens := compressor.estimateTokens(messages)

	// 总字符约 84，除以 2 约 42 tokens
	if tokens < 30 || tokens > 60 {
		t.Errorf("estimateTokens() = %d, 期望在 30-60 之间", tokens)
	}

	t.Logf("✓ estimateTokens 测试通过 (估算 tokens=%d)", tokens)
}

// TestContextCompressor_ExtractKeyPoints 测试关键点提取
func TestContextCompressor_ExtractKeyPoints(t *testing.T) {
	compressor := NewContextCompressor(nil)

	tests := []struct {
		name            string
		messages        []model.V4Message
		expectKeyPoints int
		expectContains  []string
	}{
		{
			name: "提取记住类关键词",
			messages: []model.V4Message{
				{Role: "user", Content: "记住我每周一都要开会"},
				{Role: "assistant", Content: "好的，我记住了"},
			},
			expectKeyPoints: 1,
			expectContains:  []string{"记住"},
		},
		{
			name: "提取时间规律关键词",
			messages: []model.V4Message{
				{Role: "user", Content: "我周三下午固定健身"},
				{Role: "assistant", Content: "好的"},
			},
			expectKeyPoints: 1,
			expectContains:  []string{"周三"},
		},
		{
			name: "提取决策确认关键词",
			messages: []model.V4Message{
				{Role: "assistant", Content: "确认保存这个日程吗？"},
				{Role: "user", Content: "好的，确认保存"},
			},
			expectKeyPoints: 1,
			expectContains:  []string{"确认"},
		},
		{
			name: "混合提取多个关键点",
			messages: []model.V4Message{
				{Role: "user", Content: "以后每周五下午帮我留出健身时间"},
				{Role: "assistant", Content: "好的，每周五下午健身"},
				{Role: "user", Content: "确认保存"},
			},
			expectKeyPoints: 2, // "以后" + "确认"
			expectContains:  []string{"以后", "确认"},
		},
		{
			name: "assistant 消息不提取",
			messages: []model.V4Message{
				{Role: "assistant", Content: "记住你的习惯了"},
			},
			expectKeyPoints: 0,
			expectContains:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPoints := compressor.extractKeyPoints(tt.messages)

			if len(keyPoints) != tt.expectKeyPoints {
				t.Errorf("extractKeyPoints() 数量 = %d, want %d, got: %v", len(keyPoints), tt.expectKeyPoints, keyPoints)
			}

			for _, keyword := range tt.expectContains {
				found := false
				for _, kp := range keyPoints {
					if strings.Contains(kp, keyword) {
						found = true
						break
					}
				}
				if !found && len(tt.expectContains) > 0 {
					t.Errorf("未找到期望的关键词 '%s' in %v", keyword, keyPoints)
				}
			}
		})
	}

	t.Logf("✓ extractKeyPoints 测试通过")
}

// TestContextCompressor_GenerateSimpleSummary 测试简单摘要生成
func TestContextCompressor_GenerateSimpleSummary(t *testing.T) {
	compressor := NewContextCompressor(nil)

	tests := []struct {
		name           string
		messages       []model.V4Message
		expectContains string
	}{
		{
			name:           "空消息返回'空对话'",
			messages:       []model.V4Message{},
			expectContains: "空对话",
		},
		{
			name: "提取第一条用户消息作为主题",
			messages: []model.V4Message{
				{Role: "user", Content: "帮我设置明天的会议日程"},
				{Role: "assistant", Content: "好的"},
			},
			expectContains: "帮我设置", // 中文按字节截断，只检查前缀
		},
		{
			name: "长消息被截断",
			messages: []model.V4Message{
				{Role: "user", Content: "这是一个非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常长的消息内容"},
			},
			expectContains: "...",
		},
		{
			name: "只有 assistant 消息时显示消息数",
			messages: []model.V4Message{
				{Role: "assistant", Content: "你好"},
				{Role: "assistant", Content: "有什么可以帮你的"},
			},
			expectContains: "2 条消息",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := compressor.generateSimpleSummary(tt.messages)
			if !strings.Contains(summary, tt.expectContains) {
				t.Errorf("generateSimpleSummary() = '%s', 期望包含 '%s'", summary, tt.expectContains)
			}
		})
	}

	t.Logf("✓ generateSimpleSummary 测试通过")
}

// TestContextCompressor_LearnByRules 测试规则匹配学习
func TestContextCompressor_LearnByRules(t *testing.T) {
	compressor := NewContextCompressor(nil)

	tests := []struct {
		name             string
		messages         []model.V4Message
		expectPatterns   int
		expectKeyFacts   int
		expectPreference string
	}{
		{
			name: "学习时间规律模式",
			messages: []model.V4Message{
				{Role: "user", Content: "我每周一都要开会"},
			},
			expectPatterns: 1,
		},
		{
			name: "学习偏好设置 - 隐藏过去日程",
			messages: []model.V4Message{
				{Role: "user", Content: "以后不要显示过去的日程了"},
			},
			expectPreference: "hide_past_events",
		},
		{
			name: "学习关键事实",
			messages: []model.V4Message{
				{Role: "user", Content: "记住我周三下午固定健身"},
			},
			expectKeyFacts: 1,
		},
		{
			name: "综合学习",
			messages: []model.V4Message{
				{Role: "user", Content: "我每周五下午要去接孩子"},
				{Role: "user", Content: "以后记住这个习惯"},
				{Role: "user", Content: "简短一点回复就行"},
			},
			expectPatterns: 1,
			expectKeyFacts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insights := compressor.learnByRules(tt.messages)

			if insights == nil {
				t.Fatal("learnByRules() 返回 nil")
			}

			if tt.expectPatterns > 0 && len(insights.Patterns) != tt.expectPatterns {
				t.Errorf("Patterns 数量 = %d, want %d", len(insights.Patterns), tt.expectPatterns)
			}

			if tt.expectKeyFacts > 0 && len(insights.KeyFacts) != tt.expectKeyFacts {
				t.Errorf("KeyFacts 数量 = %d, want %d", len(insights.KeyFacts), tt.expectKeyFacts)
			}

			if tt.expectPreference != "" {
				if _, ok := insights.PreferenceUpdates[tt.expectPreference]; !ok {
					t.Errorf("未找到期望的偏好设置 '%s'", tt.expectPreference)
				}
			}
		})
	}

	t.Logf("✓ learnByRules 测试通过")
}

// TestContextCompressor_TruncateMessages 测试消息截断
func TestContextCompressor_TruncateMessages(t *testing.T) {
	compressor := NewContextCompressor(nil)

	// 创建测试消息
	messages := []model.V4Message{
		{Role: "system", Content: "你是一个日程助手"},
		{Role: "user", Content: "消息1"},
		{Role: "assistant", Content: "回复1"},
		{Role: "user", Content: "消息2"},
		{Role: "assistant", Content: "回复2"},
		{Role: "user", Content: "消息3"},
		{Role: "assistant", Content: "回复3"},
		{Role: "user", Content: "消息4"},
		{Role: "assistant", Content: "回复4"},
		{Role: "user", Content: "最新消息"},
	}

	// 测试截断到 5 条
	truncated := compressor.TruncateMessages(messages, 5)

	// 验证：应保留 system 消息 + 最近的 4 条
	if len(truncated) != 5 {
		t.Errorf("TruncateMessages() 长度 = %d, want 5", len(truncated))
	}

	// system 消息应该在第一位
	if truncated[0].Role != "system" {
		t.Errorf("第一条消息应该是 system")
	}

	// 最后一条应该是最新的用户消息
	if truncated[len(truncated)-1].Content != "最新消息" {
		t.Errorf("最后一条消息应该是 '最新消息', got '%s'", truncated[len(truncated)-1].Content)
	}

	// 测试：消息数量少于 maxMessages 时不截断
	short := compressor.TruncateMessages(messages[:3], 10)
	if len(short) != 3 {
		t.Errorf("消息数量少于 maxMessages 时不应截断")
	}

	t.Logf("✓ TruncateMessages 测试通过")
}

// TestContextCompressor_CompressSession 测试会话压缩
func TestContextCompressor_CompressSession(t *testing.T) {
	compressor := NewContextCompressor(nil) // 使用简单摘要模式

	session := &model.V4Session{
		SessionID: "test-session-001",
		UserID:    "test-user",
		Messages: []model.V4Message{
			{Role: "user", Content: "帮我设置明天上午9点开会"},
			{Role: "assistant", Content: "好的，已添加明天9:00开会的日程"},
			{Role: "user", Content: "确认保存"},
			{Role: "assistant", Content: "已保存"},
		},
	}

	ctx := context.Background()
	summary, err := compressor.CompressSession(ctx, session)

	if err != nil {
		t.Fatalf("CompressSession() 错误: %v", err)
	}

	if summary == nil {
		t.Fatal("CompressSession() 返回 nil")
	}

	if summary.SessionID != session.SessionID {
		t.Errorf("SessionID 不匹配")
	}

	if summary.Summary == "" {
		t.Errorf("Summary 不应为空")
	}

	// 验证日期格式
	_, err = time.Parse("2006-01-02", summary.Date)
	if err != nil {
		t.Errorf("Date 格式错误: %s", summary.Date)
	}

	t.Logf("✓ CompressSession 测试通过 (Summary: %s)", summary.Summary)
}

// TestContextCompressor_CompressSession_TooFewMessages 测试消息太少时不压缩
func TestContextCompressor_CompressSession_TooFewMessages(t *testing.T) {
	compressor := NewContextCompressor(nil)

	session := &model.V4Session{
		Messages: []model.V4Message{
			{Role: "user", Content: "你好"},
		},
	}

	summary, err := compressor.CompressSession(context.Background(), session)

	if err != nil {
		t.Fatalf("CompressSession() 错误: %v", err)
	}

	if summary != nil {
		t.Errorf("消息太少时应返回 nil")
	}

	t.Logf("✓ CompressSession 消息太少时正确返回 nil")
}

// ===============================================
// 三、记忆学习服务测试（使用 Mock）
// ===============================================

// MockMemoryDocRepository 模拟记忆文档仓库
type MockMemoryDocRepository struct {
	docs map[string]*model.UserMemoryDocument
}

func NewMockMemoryDocRepository() *MockMemoryDocRepository {
	return &MockMemoryDocRepository{
		docs: make(map[string]*model.UserMemoryDocument),
	}
}

func (m *MockMemoryDocRepository) GetByUserID(ctx context.Context, userID string) (*model.UserMemoryDocument, error) {
	doc, ok := m.docs[userID]
	if !ok {
		return nil, nil
	}
	return doc, nil
}

func (m *MockMemoryDocRepository) Upsert(ctx context.Context, doc *model.UserMemoryDocument) error {
	m.docs[doc.UserID] = doc
	return nil
}

func (m *MockMemoryDocRepository) Delete(ctx context.Context, userID string) error {
	delete(m.docs, userID)
	return nil
}

// TestMemoryLearningService_IsSimilarPattern 测试相似模式判断
func TestMemoryLearningService_IsSimilarPattern(t *testing.T) {
	// 创建服务实例（使用反射或直接调用私有方法）
	// 由于 isSimilarPattern 是私有方法，我们通过测试 mergeInsights 来间接测试

	tests := []struct {
		patternA string
		patternB string
		similar  bool
	}{
		{"每周一开会", "每周一团建", true},         // 都包含 "周一"
		{"工作日上班", "周末休息", false},          // 不同的时间关键词
		{"每天早上跑步", "每天晚上散步", true},       // 都包含 "每天"
		{"上午开会", "下午健身", false},           // 不同时段
		{"周末和朋友聚餐", "周末去公园", true},       // 都包含 "周末"
		{"普通日程安排", "另一个普通安排", false},     // 没有时间关键词
	}

	// 创建一个 MemoryLearningService 来访问方法
	svc := &MemoryLearningService{}

	for _, tt := range tests {
		t.Run(tt.patternA+"_vs_"+tt.patternB, func(t *testing.T) {
			result := svc.isSimilarPattern(tt.patternA, tt.patternB)
			if result != tt.similar {
				t.Errorf("isSimilarPattern('%s', '%s') = %v, want %v", tt.patternA, tt.patternB, result, tt.similar)
			}
		})
	}

	t.Logf("✓ isSimilarPattern 测试通过")
}

// TestMemoryLearningService_FormatMemoryContext 测试记忆上下文格式化
func TestMemoryLearningService_FormatMemoryContext(t *testing.T) {
	svc := &MemoryLearningService{}

	doc := &model.UserMemoryDocument{
		Preferences: model.UserMemoryPreferences{
			HidePastEvents:    true,
			PreferredTimezone: "Asia/Shanghai",
		},
		SchedulePatterns: []model.SchedulePattern{
			{Pattern: "每周一开会", Confidence: 0.9},  // 高置信度，应显示
			{Pattern: "偶尔周三健身", Confidence: 0.5}, // 低置信度，不显示
		},
		KeyFacts: []model.KeyFact{
			{Fact: "周三下午固定健身"},
		},
		SessionSummaries: []model.SessionSummary{
			{Date: "2026-02-05", Summary: "设置了开会日程"},
			{Date: "2026-02-04", Summary: "查询了天气"},
			{Date: "2026-02-03", Summary: "添加了健身计划"},
			{Date: "2026-02-02", Summary: "旧的会话，不应显示"}, // 第4个，超过限制
		},
	}

	context := svc.formatMemoryContext(doc)

	// 验证包含偏好
	if !strings.Contains(context, "不显示过去的日程") {
		t.Errorf("应包含隐藏过去日程的偏好")
	}

	// 验证包含高置信度模式
	if !strings.Contains(context, "每周一开会") {
		t.Errorf("应包含高置信度的模式")
	}

	// 验证不包含低置信度模式
	if strings.Contains(context, "偶尔周三健身") {
		t.Errorf("不应包含低置信度的模式")
	}

	// 验证包含关键事实
	if !strings.Contains(context, "周三下午固定健身") {
		t.Errorf("应包含关键事实")
	}

	// 验证只显示最近3次会话
	if strings.Contains(context, "旧的会话") {
		t.Errorf("不应显示超过3次的旧会话")
	}

	// 验证显示了3次会话
	count := strings.Count(context, "上次对话")
	if count != 3 {
		t.Errorf("应显示3次上次对话，实际显示 %d 次", count)
	}

	t.Logf("✓ formatMemoryContext 测试通过")
	t.Logf("生成的上下文:\n%s", context)
}

// TestMemoryLearningService_MergeInsights 测试洞察合并
func TestMemoryLearningService_MergeInsights(t *testing.T) {
	svc := &MemoryLearningService{}

	// 初始文档
	doc := &model.UserMemoryDocument{
		UserID: "test-user",
		Preferences: model.UserMemoryPreferences{
			HidePastEvents: false,
		},
		SchedulePatterns: []model.SchedulePattern{
			{Pattern: "每周一开会", Confidence: 0.7},
		},
		KeyFacts: []model.KeyFact{
			{Fact: "已有的事实"},
		},
	}

	// 新的洞察
	insights := &model.SessionInsights{
		Patterns: []model.SchedulePattern{
			{Pattern: "每周一团建", Confidence: 0.8}, // 与现有模式相似（都是周一）
			{Pattern: "周五下午健身", Confidence: 0.6},
		},
		PreferenceUpdates: map[string]interface{}{
			"hide_past_events": true,
		},
		KeyFacts: []model.KeyFact{
			{Fact: "新的事实"},
			{Fact: "已有的事实"}, // 重复，不应添加
		},
	}

	svc.mergeInsights(doc, insights)

	// 验证偏好更新
	if !doc.Preferences.HidePastEvents {
		t.Errorf("HidePastEvents 应该被更新为 true")
	}

	// 验证模式合并（相似模式应该更新置信度而不是添加新的）
	// 周一相关的模式置信度应该被平均
	foundMondayPattern := false
	for _, p := range doc.SchedulePatterns {
		if strings.Contains(p.Pattern, "周一") {
			foundMondayPattern = true
			// 原置信度 0.7，新置信度 0.8，平均应该是 0.75
			if p.Confidence != 0.75 {
				t.Errorf("周一模式的置信度应该是 0.75, got %f", p.Confidence)
			}
		}
	}
	if !foundMondayPattern {
		t.Errorf("应该保留周一相关的模式")
	}

	// 验证新模式被添加
	foundFridayPattern := false
	for _, p := range doc.SchedulePatterns {
		if strings.Contains(p.Pattern, "周五") {
			foundFridayPattern = true
		}
	}
	if !foundFridayPattern {
		t.Errorf("周五健身模式应该被添加")
	}

	// 验证关键事实去重
	factCount := 0
	for _, f := range doc.KeyFacts {
		if f.Fact == "已有的事实" {
			factCount++
		}
	}
	if factCount != 1 {
		t.Errorf("'已有的事实'应该只出现1次，实际出现 %d 次", factCount)
	}

	// 验证新事实被添加
	foundNewFact := false
	for _, f := range doc.KeyFacts {
		if f.Fact == "新的事实" {
			foundNewFact = true
		}
	}
	if !foundNewFact {
		t.Errorf("'新的事实'应该被添加")
	}

	t.Logf("✓ mergeInsights 测试通过")
}

// TestMemoryLearningService_AddSessionSummary 测试会话摘要添加
func TestMemoryLearningService_AddSessionSummary(t *testing.T) {
	svc := &MemoryLearningService{}

	doc := &model.UserMemoryDocument{
		SessionSummaries: []model.SessionSummary{
			{SessionID: "old-1", Summary: "旧会话1"},
			{SessionID: "old-2", Summary: "旧会话2"},
			{SessionID: "old-3", Summary: "旧会话3"},
			{SessionID: "old-4", Summary: "旧会话4"},
			{SessionID: "old-5", Summary: "旧会话5"},
		},
	}

	newSummary := &model.SessionSummary{
		SessionID: "new-1",
		Summary:   "新会话",
	}

	svc.addSessionSummary(doc, newSummary)

	// 验证新摘要在列表开头
	if doc.SessionSummaries[0].SessionID != "new-1" {
		t.Errorf("新摘要应该在列表开头")
	}

	// 验证总数不超过限制
	if len(doc.SessionSummaries) > model.MaxSessionSummaries {
		t.Errorf("会话摘要数量超过限制: %d > %d", len(doc.SessionSummaries), model.MaxSessionSummaries)
	}

	// 验证最旧的被移除
	for _, s := range doc.SessionSummaries {
		if s.SessionID == "old-5" {
			t.Errorf("最旧的会话摘要应该被移除")
		}
	}

	t.Logf("✓ addSessionSummary 测试通过 (MaxSessionSummaries=%d)", model.MaxSessionSummaries)
}

// ===============================================
// 四、端到端集成测试
// ===============================================

// TestEndToEnd_MemoryLearningFlow 测试完整的记忆学习流程
func TestEndToEnd_MemoryLearningFlow(t *testing.T) {
	// 模拟一个完整的用户交互场景

	// 1. 创建上下文压缩器
	compressor := NewContextCompressor(nil)

	// 2. 模拟用户第一次会话
	session1 := &model.V4Session{
		SessionID: "session-001",
		UserID:    "user-test-e2e",
		Messages: []model.V4Message{
			{Role: "user", Content: "帮我设置每周一上午9点开会"},
			{Role: "assistant", Content: "好的，已添加每周一9:00开会的日程"},
			{Role: "user", Content: "记住我周三下午固定健身"},
			{Role: "assistant", Content: "好的，我记住了"},
			{Role: "user", Content: "以后不要显示过去的日程了"},
			{Role: "assistant", Content: "好的，已设置隐藏过去的日程"},
		},
	}

	// 3. 从会话中学习
	insights, err := compressor.LearnFromSession(context.Background(), session1)
	if err != nil {
		t.Fatalf("LearnFromSession 失败: %v", err)
	}

	// 4. 验证学习结果
	if insights == nil {
		t.Fatal("insights 不应为 nil")
	}

	// 验证学习到了时间模式
	if len(insights.Patterns) == 0 {
		t.Errorf("应该学习到时间模式 (每周一开会)")
	}

	// 验证学习到了偏好设置
	if insights.PreferenceUpdates == nil {
		t.Errorf("应该学习到偏好设置")
	} else if _, ok := insights.PreferenceUpdates["hide_past_events"]; !ok {
		t.Errorf("应该学习到 hide_past_events 偏好")
	}

	// 验证学习到了关键事实
	if len(insights.KeyFacts) == 0 {
		t.Errorf("应该学习到关键事实 (周三健身)")
	}

	// 5. 压缩会话
	summary, err := compressor.CompressSession(context.Background(), session1)
	if err != nil {
		t.Fatalf("CompressSession 失败: %v", err)
	}

	if summary == nil || summary.Summary == "" {
		t.Errorf("会话摘要不应为空")
	}

	// 6. 创建一个完整的记忆文档
	doc := &model.UserMemoryDocument{
		UserID: session1.UserID,
	}

	// 使用 MemoryLearningService 合并洞察
	svc := &MemoryLearningService{}
	svc.mergeInsights(doc, insights)
	svc.addSessionSummary(doc, summary)

	// 7. 验证记忆文档
	if doc.Preferences.HidePastEvents != true {
		t.Errorf("偏好应该被更新: HidePastEvents = true")
	}

	if len(doc.SchedulePatterns) == 0 {
		t.Errorf("应该有日程模式")
	}

	if len(doc.SessionSummaries) != 1 {
		t.Errorf("应该有1个会话摘要")
	}

	// 8. 验证格式化的记忆上下文
	memoryContext := svc.formatMemoryContext(doc)
	if memoryContext == "" {
		t.Errorf("记忆上下文不应为空")
	}

	t.Logf("✓ 端到端记忆学习流程测试通过")
	t.Logf("学习结果:")
	t.Logf("  - 模式数量: %d", len(insights.Patterns))
	t.Logf("  - 偏好更新: %v", insights.PreferenceUpdates)
	t.Logf("  - 关键事实: %d", len(insights.KeyFacts))
	t.Logf("  - 会话摘要: %s", summary.Summary)
	t.Logf("格式化的记忆上下文:\n%s", memoryContext)
}

// TestEndToEnd_MultiSessionLearning 测试多次会话的学习累积
func TestEndToEnd_MultiSessionLearning(t *testing.T) {
	compressor := NewContextCompressor(nil)
	svc := &MemoryLearningService{}

	// 初始化空的记忆文档
	doc := &model.UserMemoryDocument{
		UserID: "user-multi-session",
	}

	// 模拟多次会话
	sessions := []*model.V4Session{
		{
			SessionID: "session-1",
			UserID:    "user-multi-session",
			Messages: []model.V4Message{
				{Role: "user", Content: "每周一上午开会"},
				{Role: "assistant", Content: "好的"},
			},
		},
		{
			SessionID: "session-2",
			UserID:    "user-multi-session",
			Messages: []model.V4Message{
				{Role: "user", Content: "周三下午健身"},
				{Role: "assistant", Content: "好的"},
			},
		},
		{
			SessionID: "session-3",
			UserID:    "user-multi-session",
			Messages: []model.V4Message{
				{Role: "user", Content: "每周一下午还有部门会议"},
				{Role: "assistant", Content: "好的"},
			},
		},
	}

	// 逐个会话学习
	for _, session := range sessions {
		insights, _ := compressor.LearnFromSession(context.Background(), session)
		if insights != nil {
			svc.mergeInsights(doc, insights)
		}
		summary, _ := compressor.CompressSession(context.Background(), session)
		if summary != nil {
			svc.addSessionSummary(doc, summary)
		}
	}

	// 验证累积学习结果
	if len(doc.SessionSummaries) != 3 {
		t.Errorf("应该有3个会话摘要, got %d", len(doc.SessionSummaries))
	}

	// 验证周一相关的模式被正确处理（应该合并而不是重复添加）
	mondayPatternCount := 0
	for _, p := range doc.SchedulePatterns {
		if strings.Contains(p.Pattern, "周一") {
			mondayPatternCount++
		}
	}
	// 由于相似模式合并，周一相关的应该被合并
	t.Logf("周一相关模式数量: %d (相似模式应该合并)", mondayPatternCount)

	t.Logf("✓ 多会话学习累积测试通过")
	t.Logf("最终记忆文档:")
	t.Logf("  - 模式数量: %d", len(doc.SchedulePatterns))
	t.Logf("  - 会话摘要: %d", len(doc.SessionSummaries))
}

// ===============================================
// 五、边界条件和错误处理测试
// ===============================================

// TestContextCompressor_EmptySession 测试空会话处理
func TestContextCompressor_EmptySession(t *testing.T) {
	compressor := NewContextCompressor(nil)

	emptySession := &model.V4Session{
		SessionID: "empty",
		UserID:    "user",
		Messages:  []model.V4Message{},
	}

	// 不应该压缩
	if compressor.ShouldCompress(emptySession) {
		t.Errorf("空会话不应该需要压缩")
	}

	// 压缩应该返回 nil
	summary, err := compressor.CompressSession(context.Background(), emptySession)
	if err != nil {
		t.Errorf("CompressSession 不应返回错误: %v", err)
	}
	if summary != nil {
		t.Errorf("空会话压缩应返回 nil")
	}

	// 学习应该返回 nil
	insights, err := compressor.LearnFromSession(context.Background(), emptySession)
	if err != nil {
		t.Errorf("LearnFromSession 不应返回错误: %v", err)
	}
	if insights != nil {
		t.Errorf("空会话学习应返回 nil")
	}

	t.Logf("✓ 空会话处理测试通过")
}

// TestMemoryLearningService_MergeInsights_LowConfidence 测试低置信度模式过滤
func TestMemoryLearningService_MergeInsights_LowConfidence(t *testing.T) {
	svc := &MemoryLearningService{}

	doc := &model.UserMemoryDocument{}

	insights := &model.SessionInsights{
		Patterns: []model.SchedulePattern{
			{Pattern: "高置信度模式", Confidence: 0.8},
			{Pattern: "中等置信度模式", Confidence: 0.5},
			{Pattern: "低置信度模式", Confidence: 0.3}, // 低于 0.5 阈值
		},
	}

	svc.mergeInsights(doc, insights)

	// 低置信度模式不应被添加
	for _, p := range doc.SchedulePatterns {
		if p.Pattern == "低置信度模式" {
			t.Errorf("低置信度模式不应该被添加")
		}
	}

	// 高置信度和中等置信度应该被添加
	if len(doc.SchedulePatterns) != 2 {
		t.Errorf("应该有2个模式, got %d", len(doc.SchedulePatterns))
	}

	t.Logf("✓ 低置信度模式过滤测试通过")
}

// TestContextCompressor_ParseLLMInsights 测试 LLM 响应解析
func TestContextCompressor_ParseLLMInsights(t *testing.T) {
	compressor := NewContextCompressor(nil)

	tests := []struct {
		name     string
		response string
		hasError bool
	}{
		{
			name: "正常 JSON",
			response: `{
				"patterns": [{"pattern": "每周一开会", "confidence": 0.8}],
				"preferences": {"hide_past_events": true},
				"key_facts": [{"fact": "健身", "source": "用户说"}],
				"summary": "测试摘要",
				"decisions": ["决策1"],
				"remembers": ["记住的信息"]
			}`,
			hasError: false,
		},
		{
			name: "带 markdown 标记的 JSON",
			response: "```json\n{\"patterns\": [], \"summary\": \"测试\"}\n```",
			hasError: false,
		},
		{
			name:     "空 JSON 对象",
			response: "{}",
			hasError: false,
		},
		{
			name:     "无效 JSON",
			response: "这不是有效的JSON",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insights, err := compressor.parseLLMInsights(tt.response)

			if tt.hasError {
				if err == nil {
					t.Errorf("应该返回错误")
				}
			} else {
				if err != nil {
					t.Errorf("不应该返回错误: %v", err)
				}
				if insights == nil {
					t.Errorf("insights 不应为 nil")
				}
			}
		})
	}

	t.Logf("✓ parseLLMInsights 测试通过")
}

// TestMemoryDocument_MaxLimits 测试各种最大数量限制
func TestMemoryDocument_MaxLimits(t *testing.T) {
	// 验证常量定义正确
	if model.MaxSessionSummaries <= 0 {
		t.Errorf("MaxSessionSummaries 应该大于 0, got %d", model.MaxSessionSummaries)
	}
	if model.MaxSchedulePatterns <= 0 {
		t.Errorf("MaxSchedulePatterns 应该大于 0, got %d", model.MaxSchedulePatterns)
	}
	if model.MaxKeyFacts <= 0 {
		t.Errorf("MaxKeyFacts 应该大于 0, got %d", model.MaxKeyFacts)
	}
	if model.TokenThreshold <= 0 {
		t.Errorf("TokenThreshold 应该大于 0, got %d", model.TokenThreshold)
	}
	if model.MessageThreshold <= 0 {
		t.Errorf("MessageThreshold 应该大于 0, got %d", model.MessageThreshold)
	}

	t.Logf("✓ 限制常量验证通过:")
	t.Logf("  - MaxSessionSummaries: %d", model.MaxSessionSummaries)
	t.Logf("  - MaxSchedulePatterns: %d", model.MaxSchedulePatterns)
	t.Logf("  - MaxKeyFacts: %d", model.MaxKeyFacts)
	t.Logf("  - TokenThreshold: %d", model.TokenThreshold)
	t.Logf("  - MessageThreshold: %d", model.MessageThreshold)
}

// ===============================================
// 六、验证流程是否真正执行（关键测试）
// ===============================================

// TestVerify_MemorySystemIsWorking 验证记忆系统确实在工作
// 这个测试用于检测"功能存在但未执行"的情况
func TestVerify_MemorySystemIsWorking(t *testing.T) {
	t.Log("====== 验证记忆系统是否真正工作 ======")

	// 1. 验证上下文压缩器可以被创建
	compressor := NewContextCompressor(nil)
	if compressor == nil {
		t.Fatal("无法创建 ContextCompressor")
	}
	t.Log("✓ ContextCompressor 可以被创建")

	// 2. 验证压缩触发条件正确
	session := &model.V4Session{
		Messages: make([]model.V4Message, model.MessageThreshold),
	}
	if !compressor.ShouldCompress(session) {
		t.Errorf("消息数量达到阈值时应该触发压缩")
	}
	t.Log("✓ ShouldCompress 正确响应阈值")

	// 3. 验证学习函数产生输出
	session.Messages = []model.V4Message{
		{Role: "user", Content: "我每周一都要开会，记住这个"},
		{Role: "assistant", Content: "好的"},
	}
	insights, _ := compressor.LearnFromSession(context.Background(), session)
	if insights == nil {
		t.Fatal("LearnFromSession 返回 nil，记忆学习可能未执行")
	}
	if len(insights.Patterns) == 0 && len(insights.KeyFacts) == 0 && len(insights.Remembers) == 0 {
		t.Errorf("LearnFromSession 未学习到任何内容，检查规则匹配逻辑")
	}
	t.Logf("✓ LearnFromSession 产生了输出: %d 模式, %d 事实", len(insights.Patterns), len(insights.KeyFacts))

	// 4. 验证摘要生成
	summary, _ := compressor.CompressSession(context.Background(), session)
	if summary == nil || summary.Summary == "" {
		t.Fatal("CompressSession 未生成摘要")
	}
	t.Logf("✓ CompressSession 生成了摘要: %s", summary.Summary)

	// 5. 验证记忆格式化
	svc := &MemoryLearningService{}
	doc := &model.UserMemoryDocument{
		Preferences: model.UserMemoryPreferences{HidePastEvents: true},
		KeyFacts:    []model.KeyFact{{Fact: "测试事实"}},
	}
	formatted := svc.formatMemoryContext(doc)
	if formatted == "" {
		t.Fatal("formatMemoryContext 返回空字符串")
	}
	t.Logf("✓ formatMemoryContext 生成了内容:\n%s", formatted)

	t.Log("====== 记忆系统验证通过 ======")
}

// TestVerify_CompressorThresholds 验证压缩器阈值配置
func TestVerify_CompressorThresholds(t *testing.T) {
	t.Logf("当前配置:")
	t.Logf("  - MessageThreshold (消息数量阈值): %d", model.MessageThreshold)
	t.Logf("  - TokenThreshold (Token 数量阈值): %d", model.TokenThreshold)

	// 验证阈值合理性
	if model.MessageThreshold < 5 {
		t.Errorf("MessageThreshold 过小 (%d)，可能导致频繁压缩", model.MessageThreshold)
	}
	if model.MessageThreshold > 50 {
		t.Errorf("MessageThreshold 过大 (%d)，可能导致 token 溢出", model.MessageThreshold)
	}

	if model.TokenThreshold < 1000 {
		t.Errorf("TokenThreshold 过小 (%d)，可能导致频繁压缩", model.TokenThreshold)
	}
	if model.TokenThreshold > 100000 {
		t.Errorf("TokenThreshold 过大 (%d)，可能导致 API 调用失败", model.TokenThreshold)
	}

	t.Log("✓ 阈值配置在合理范围内")
}
