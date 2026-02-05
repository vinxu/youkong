package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/repository"
)

// MemoryLearningService 记忆学习服务
// 负责从会话中学习用户偏好，并持久化到记忆文档
type MemoryLearningService struct {
	memoryDocRepo *repository.UserMemoryDocumentRepository
	compressor    *ContextCompressor
	llmAdapter    *agent.LLMAdapter
}

// NewMemoryLearningService 创建记忆学习服务
func NewMemoryLearningService(
	memoryDocRepo *repository.UserMemoryDocumentRepository,
	llmAdapter *agent.LLMAdapter,
) *MemoryLearningService {
	return &MemoryLearningService{
		memoryDocRepo: memoryDocRepo,
		compressor:    NewContextCompressor(llmAdapter),
		llmAdapter:    llmAdapter,
	}
}

// LearnFromSession 会话结束时触发学习
func (s *MemoryLearningService) LearnFromSession(ctx context.Context, session *model.V4Session) error {
	if len(session.Messages) < 2 {
		return nil // 消息太少，不学习
	}

	// 1. 从会话中提取洞察
	insights, err := s.compressor.LearnFromSession(ctx, session)
	if err != nil {
		fmt.Printf("[MemoryLearning] 提取洞察失败: %v\n", err)
		// 不中断，继续尝试保存基本摘要
	}

	// 2. 获取或创建用户记忆文档
	doc, err := s.memoryDocRepo.GetByUserID(ctx, session.UserID)
	if err != nil {
		return fmt.Errorf("获取记忆文档失败: %w", err)
	}

	if doc == nil {
		doc = &model.UserMemoryDocument{
			UserID: session.UserID,
		}
	}

	// 3. 合并洞察到记忆文档
	if insights != nil {
		s.mergeInsights(doc, insights)
	}

	// 4. 生成并保存会话摘要
	summary, err := s.compressor.CompressSession(ctx, session)
	if err == nil && summary != nil {
		s.addSessionSummary(doc, summary)
	}

	// 5. 保存记忆文档
	if err := s.memoryDocRepo.Upsert(ctx, doc); err != nil {
		return fmt.Errorf("保存记忆文档失败: %w", err)
	}

	fmt.Printf("[MemoryLearning] 用户 %s 的记忆已更新\n", session.UserID)
	return nil
}

// mergeInsights 合并洞察到记忆文档
func (s *MemoryLearningService) mergeInsights(doc *model.UserMemoryDocument, insights *model.SessionInsights) {
	// 1. 合并日程模式
	for _, pattern := range insights.Patterns {
		if pattern.Confidence < 0.5 {
			continue // 忽略低置信度的模式
		}

		// 检查是否已存在
		found := false
		for i, existing := range doc.SchedulePatterns {
			if s.isSimilarPattern(existing.Pattern, pattern.Pattern) {
				// 更新置信度
				doc.SchedulePatterns[i].Confidence = (existing.Confidence + pattern.Confidence) / 2
				found = true
				break
			}
		}

		if !found && len(doc.SchedulePatterns) < model.MaxSchedulePatterns {
			doc.SchedulePatterns = append(doc.SchedulePatterns, pattern)
		}
	}

	// 2. 合并偏好设置
	if insights.PreferenceUpdates != nil {
		if hidePast, ok := insights.PreferenceUpdates["hide_past_events"].(bool); ok {
			doc.Preferences.HidePastEvents = hidePast
		}
		if timezone, ok := insights.PreferenceUpdates["preferred_timezone"].(string); ok {
			doc.Preferences.PreferredTimezone = timezone
		}
	}

	// 3. 合并关键事实
	for _, fact := range insights.KeyFacts {
		// 检查是否已存在
		found := false
		for _, existing := range doc.KeyFacts {
			if existing.Fact == fact.Fact {
				found = true
				break
			}
		}

		if !found && len(doc.KeyFacts) < model.MaxKeyFacts {
			doc.KeyFacts = append(doc.KeyFacts, fact)
		}
	}
}

// isSimilarPattern 判断两个模式是否相似
func (s *MemoryLearningService) isSimilarPattern(a, b string) bool {
	// 简单实现：检查是否包含相同的时间关键词
	timeKeywords := []string{
		"周一", "周二", "周三", "周四", "周五", "周六", "周日",
		"工作日", "周末", "每天", "上午", "下午", "晚上",
	}

	aKeywords := make(map[string]bool)
	bKeywords := make(map[string]bool)

	for _, keyword := range timeKeywords {
		if strings.Contains(a, keyword) {
			aKeywords[keyword] = true
		}
		if strings.Contains(b, keyword) {
			bKeywords[keyword] = true
		}
	}

	// 如果有相同的时间关键词，认为是相似模式
	for k := range aKeywords {
		if bKeywords[k] {
			return true
		}
	}

	return false
}

// addSessionSummary 添加会话摘要
func (s *MemoryLearningService) addSessionSummary(doc *model.UserMemoryDocument, summary *model.SessionSummary) {
	// 添加到列表开头
	doc.SessionSummaries = append(model.SessionSummaryList{*summary}, doc.SessionSummaries...)

	// 限制数量
	if len(doc.SessionSummaries) > model.MaxSessionSummaries {
		doc.SessionSummaries = doc.SessionSummaries[:model.MaxSessionSummaries]
	}
}

// GetMemoryContext 获取用户记忆上下文（用于注入到 System Prompt）
func (s *MemoryLearningService) GetMemoryContext(ctx context.Context, userID string) (string, error) {
	doc, err := s.memoryDocRepo.GetByUserID(ctx, userID)
	if err != nil {
		return "", err
	}

	if doc == nil {
		return "", nil // 新用户，无记忆
	}

	return s.formatMemoryContext(doc), nil
}

// formatMemoryContext 格式化记忆上下文
func (s *MemoryLearningService) formatMemoryContext(doc *model.UserMemoryDocument) string {
	var parts []string

	// 用户偏好
	if doc.Preferences.HidePastEvents {
		parts = append(parts, "- 用户偏好：不显示过去的日程")
	}
	if doc.Preferences.PreferredTimezone != "" {
		parts = append(parts, fmt.Sprintf("- 用户时区：%s", doc.Preferences.PreferredTimezone))
	}

	// 日程模式（只显示高置信度的）
	for _, p := range doc.SchedulePatterns {
		if p.Confidence >= 0.7 {
			parts = append(parts, fmt.Sprintf("- 习惯：%s", p.Pattern))
		}
	}

	// 关键事实
	for _, f := range doc.KeyFacts {
		parts = append(parts, fmt.Sprintf("- 记住：%s", f.Fact))
	}

	// 最近会话摘要（只保留最近 3 次）
	maxSummaries := 3
	for i, summary := range doc.SessionSummaries {
		if i >= maxSummaries {
			break
		}
		parts = append(parts, fmt.Sprintf("- 上次对话（%s）：%s", summary.Date, summary.Summary))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n")
}

// UpdatePreference 更新用户偏好
func (s *MemoryLearningService) UpdatePreference(ctx context.Context, userID string, key string, value interface{}) error {
	doc, err := s.memoryDocRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if doc == nil {
		doc = &model.UserMemoryDocument{
			UserID: userID,
		}
	}

	// 更新偏好
	switch key {
	case "hide_past_events":
		if v, ok := value.(bool); ok {
			doc.Preferences.HidePastEvents = v
		}
	case "default_view_days":
		if v, ok := value.(int); ok {
			doc.Preferences.DefaultViewDays = v
		}
	case "preferred_timezone":
		if v, ok := value.(string); ok {
			doc.Preferences.PreferredTimezone = v
		}
	case "language":
		if v, ok := value.(string); ok {
			doc.Preferences.Language = v
		}
	}

	return s.memoryDocRepo.Upsert(ctx, doc)
}

// AddKeyFact 添加关键事实
func (s *MemoryLearningService) AddKeyFact(ctx context.Context, userID string, fact string, source string) error {
	keyFact := model.KeyFact{
		Fact:      fact,
		Source:    source,
		LearnedAt: time.Now().Format(time.RFC3339),
	}

	return s.memoryDocRepo.AddKeyFact(ctx, userID, keyFact)
}

// AddSchedulePattern 添加日程模式
func (s *MemoryLearningService) AddSchedulePattern(ctx context.Context, userID string, pattern string, confidence float64, sessionID string) error {
	p := model.SchedulePattern{
		Pattern:     pattern,
		Confidence:  confidence,
		LearnedFrom: []string{sessionID},
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	return s.memoryDocRepo.AddSchedulePattern(ctx, userID, p)
}

// GetUserMemoryDocument 获取完整的用户记忆文档
func (s *MemoryLearningService) GetUserMemoryDocument(ctx context.Context, userID string) (*model.UserMemoryDocument, error) {
	return s.memoryDocRepo.GetByUserID(ctx, userID)
}

// ClearUserMemory 清除用户记忆（用于测试或用户请求）
func (s *MemoryLearningService) ClearUserMemory(ctx context.Context, userID string) error {
	return s.memoryDocRepo.Delete(ctx, userID)
}
