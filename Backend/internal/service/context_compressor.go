package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
)

// ContextCompressor 上下文压缩服务
// 负责将长对话历史压缩为摘要，减少 token 消耗
type ContextCompressor struct {
	llmAdapter *agent.LLMAdapter
}

// NewContextCompressor 创建上下文压缩服务
func NewContextCompressor(llmAdapter *agent.LLMAdapter) *ContextCompressor {
	return &ContextCompressor{
		llmAdapter: llmAdapter,
	}
}

// ShouldCompress 判断是否需要压缩
func (c *ContextCompressor) ShouldCompress(session *model.V4Session) bool {
	// 条件1：消息数量超过阈值
	if len(session.Messages) >= model.MessageThreshold {
		return true
	}

	// 条件2：估算 token 数超过阈值
	estimatedTokens := c.estimateTokens(session.Messages)
	if estimatedTokens >= model.TokenThreshold {
		return true
	}

	return false
}

// estimateTokens 估算消息的 token 数量
// 简单估算：中文约 1.5 字符/token，英文约 4 字符/token
func (c *ContextCompressor) estimateTokens(messages []model.V4Message) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
	}
	// 简单估算：平均 2 字符/token
	return totalChars / 2
}

// CompressSession 压缩会话，生成摘要
func (c *ContextCompressor) CompressSession(ctx context.Context, session *model.V4Session) (*model.SessionSummary, error) {
	if len(session.Messages) < 2 {
		return nil, nil // 消息太少，无需压缩
	}

	// 1. 提取关键信息
	keyPoints := c.extractKeyPoints(session.Messages)

	// 2. 使用 LLM 生成摘要
	summary, err := c.generateSummary(ctx, session.Messages)
	if err != nil {
		// 降级：使用简单摘要
		summary = c.generateSimpleSummary(session.Messages)
	}

	return &model.SessionSummary{
		SessionID: session.SessionID,
		Date:      time.Now().Format("2006-01-02"),
		Summary:   summary,
		KeyPoints: keyPoints,
	}, nil
}

// extractKeyPoints 提取关键决策点
func (c *ContextCompressor) extractKeyPoints(messages []model.V4Message) []string {
	keyPoints := make([]string, 0)

	// 关键词匹配
	rememberKeywords := []string{
		"记住", "以后", "每次", "总是", "固定", "习惯",
		"周一", "周二", "周三", "周四", "周五", "周六", "周日",
		"工作日", "周末",
	}

	decisionKeywords := []string{
		"好的", "确认", "保存", "设置", "修改", "删除", "取消",
	}

	for _, msg := range messages {
		if msg.Role != "user" {
			continue
		}

		content := msg.Content

		// 检查是否包含"记住"类关键词
		for _, keyword := range rememberKeywords {
			if strings.Contains(content, keyword) {
				// 截取关键内容
				if len(content) > 50 {
					content = content[:50] + "..."
				}
				keyPoints = append(keyPoints, "用户说: "+content)
				break
			}
		}

		// 检查是否是决策确认
		for _, keyword := range decisionKeywords {
			if strings.Contains(content, keyword) {
				keyPoints = append(keyPoints, "用户确认: "+content)
				break
			}
		}
	}

	// 限制数量
	if len(keyPoints) > 5 {
		keyPoints = keyPoints[:5]
	}

	return keyPoints
}

// generateSummary 使用 LLM 生成摘要
func (c *ContextCompressor) generateSummary(ctx context.Context, messages []model.V4Message) (string, error) {
	if c.llmAdapter == nil {
		return c.generateSimpleSummary(messages), nil
	}

	// 构建对话文本
	var dialogBuilder strings.Builder
	for _, msg := range messages {
		role := "用户"
		if msg.Role == "assistant" {
			role = "助手"
		}
		dialogBuilder.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}

	prompt := fmt.Sprintf(`请将以下对话压缩成一句话摘要（不超过50字）：

%s

要求：
1. 概括对话主题和结果
2. 保留关键决策信息
3. 语言简洁自然`, dialogBuilder.String())

	// 调用 LLM
	response, err := c.llmAdapter.Chat(ctx, prompt)
	if err != nil {
		return "", err
	}

	// 清理响应
	summary := strings.TrimSpace(response)
	if len(summary) > 100 {
		summary = summary[:100]
	}

	return summary, nil
}

// generateSimpleSummary 生成简单摘要（不依赖 LLM）
func (c *ContextCompressor) generateSimpleSummary(messages []model.V4Message) string {
	if len(messages) == 0 {
		return "空对话"
	}

	// 提取第一条用户消息作为主题
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != "" {
			topic := msg.Content
			if len(topic) > 30 {
				topic = topic[:30] + "..."
			}
			return fmt.Sprintf("用户咨询: %s", topic)
		}
	}

	return fmt.Sprintf("对话共 %d 条消息", len(messages))
}

// LearnFromSession 从会话中学习用户偏好和模式
func (c *ContextCompressor) LearnFromSession(ctx context.Context, session *model.V4Session) (*model.SessionInsights, error) {
	if len(session.Messages) < 2 {
		return nil, nil
	}

	// 如果没有 LLM，使用规则匹配
	if c.llmAdapter == nil {
		return c.learnByRules(session.Messages), nil
	}

	// 使用 LLM 分析
	return c.learnByLLM(ctx, session.Messages)
}

// learnByRules 规则匹配学习
func (c *ContextCompressor) learnByRules(messages []model.V4Message) *model.SessionInsights {
	insights := &model.SessionInsights{
		Patterns:          make([]model.SchedulePattern, 0),
		PreferenceUpdates: make(map[string]interface{}),
		KeyFacts:          make([]model.KeyFact, 0),
		KeyPoints:         make([]string, 0),
		Decisions:         make([]string, 0),
		Remembers:         make([]string, 0),
	}

	// 日程模式关键词
	schedulePatterns := map[string]string{
		"每周一":  "每周一",
		"每周二":  "每周二",
		"每周三":  "每周三",
		"每周四":  "每周四",
		"每周五":  "每周五",
		"周末":   "周末",
		"工作日":  "工作日",
		"每天早上": "每天早上",
		"每天晚上": "每天晚上",
	}

	// 偏好关键词
	preferenceKeywords := map[string]string{
		"不要显示过去":  "hide_past_events",
		"隐藏过去的日程": "hide_past_events",
		"简短一点":    "prefers_brief",
		"简洁一点":    "prefers_brief",
		"保存前确认":   "confirm_before_save",
	}

	// 记住关键词
	rememberTriggers := []string{"记住", "以后", "每次"}

	for _, msg := range messages {
		if msg.Role != "user" {
			continue
		}

		content := msg.Content

		// 检查日程模式
		for keyword, pattern := range schedulePatterns {
			if strings.Contains(content, keyword) {
				insights.Patterns = append(insights.Patterns, model.SchedulePattern{
					Pattern:    fmt.Sprintf("%s: %s", pattern, content),
					Confidence: 0.6,
					CreatedAt:  time.Now().Format(time.RFC3339),
				})
				break
			}
		}

		// 检查偏好设置
		for keyword, pref := range preferenceKeywords {
			if strings.Contains(content, keyword) {
				if pref == "hide_past_events" {
					insights.PreferenceUpdates[pref] = true
				} else if pref == "prefers_brief" {
					insights.PreferenceUpdates[pref] = true
				} else if pref == "confirm_before_save" {
					insights.PreferenceUpdates[pref] = true
				}
			}
		}

		// 检查需要记住的信息
		for _, trigger := range rememberTriggers {
			if strings.Contains(content, trigger) {
				fact := model.KeyFact{
					Fact:      content,
					Source:    content,
					LearnedAt: time.Now().Format(time.RFC3339),
				}
				insights.KeyFacts = append(insights.KeyFacts, fact)
				insights.Remembers = append(insights.Remembers, content)
				break
			}
		}
	}

	return insights
}

// learnByLLM 使用 LLM 分析学习
func (c *ContextCompressor) learnByLLM(ctx context.Context, messages []model.V4Message) (*model.SessionInsights, error) {
	// 构建对话文本
	var dialogBuilder strings.Builder
	for _, msg := range messages {
		role := "用户"
		if msg.Role == "assistant" {
			role = "助手"
		}
		dialogBuilder.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}

	prompt := fmt.Sprintf(`你是一个用户行为分析助手。请分析以下对话，提取用户的习惯和偏好：

【对话内容】
%s

【分析要求】
1. 提取用户表达的时间规律（如"每周一开会"）
2. 提取用户的偏好设置请求（如"不显示过去的日程"）
3. 提取用户希望被记住的信息
4. 忽略一次性的临时安排

【输出格式】
返回 JSON 格式：
{
  "patterns": [
    {"pattern": "规律描述", "confidence": 0.8}
  ],
  "preferences": {
    "hide_past_events": true
  },
  "key_facts": [
    {"fact": "需要记住的事实", "source": "用户原话"}
  ],
  "summary": "一句话摘要",
  "decisions": ["决策1"],
  "remembers": ["需要记住的信息"]
}

如果没有发现任何有价值的信息，返回空的 JSON 对象 {}`, dialogBuilder.String())

	// 调用 LLM
	response, err := c.llmAdapter.Chat(ctx, prompt)
	if err != nil {
		// 降级到规则匹配
		return c.learnByRules(messages), nil
	}

	// 解析 JSON 响应
	insights, err := c.parseLLMInsights(response)
	if err != nil {
		// 降级到规则匹配
		return c.learnByRules(messages), nil
	}

	return insights, nil
}

// parseLLMInsights 解析 LLM 返回的洞察
func (c *ContextCompressor) parseLLMInsights(response string) (*model.SessionInsights, error) {
	// 清理响应中的 markdown 标记
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// 解析 JSON
	var rawInsights struct {
		Patterns []struct {
			Pattern    string  `json:"pattern"`
			Confidence float64 `json:"confidence"`
		} `json:"patterns"`
		Preferences map[string]interface{} `json:"preferences"`
		KeyFacts    []struct {
			Fact   string `json:"fact"`
			Source string `json:"source"`
		} `json:"key_facts"`
		Summary   string   `json:"summary"`
		Decisions []string `json:"decisions"`
		Remembers []string `json:"remembers"`
	}

	if err := json.Unmarshal([]byte(response), &rawInsights); err != nil {
		return nil, err
	}

	// 转换为模型结构
	insights := &model.SessionInsights{
		Patterns:          make([]model.SchedulePattern, 0, len(rawInsights.Patterns)),
		PreferenceUpdates: rawInsights.Preferences,
		KeyFacts:          make([]model.KeyFact, 0, len(rawInsights.KeyFacts)),
		Summary:           rawInsights.Summary,
		KeyPoints:         rawInsights.Decisions, // 复用 decisions 作为 key points
		Decisions:         rawInsights.Decisions,
		Remembers:         rawInsights.Remembers,
	}

	for _, p := range rawInsights.Patterns {
		insights.Patterns = append(insights.Patterns, model.SchedulePattern{
			Pattern:    p.Pattern,
			Confidence: p.Confidence,
			CreatedAt:  time.Now().Format(time.RFC3339),
		})
	}

	for _, f := range rawInsights.KeyFacts {
		insights.KeyFacts = append(insights.KeyFacts, model.KeyFact{
			Fact:      f.Fact,
			Source:    f.Source,
			LearnedAt: time.Now().Format(time.RFC3339),
		})
	}

	return insights, nil
}

// TruncateMessages 截断消息历史，保留最近的消息
func (c *ContextCompressor) TruncateMessages(messages []model.V4Message, maxMessages int) []model.V4Message {
	if len(messages) <= maxMessages {
		return messages
	}

	// 保留 system 消息 + 最近的用户消息
	var systemMessages []model.V4Message
	var recentMessages []model.V4Message

	for _, msg := range messages {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		}
	}

	// 计算需要保留的最近消息数量
	remainingSlots := maxMessages - len(systemMessages)
	if remainingSlots < 2 {
		remainingSlots = 2
	}

	startIdx := len(messages) - remainingSlots
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(messages); i++ {
		if messages[i].Role != "system" {
			recentMessages = append(recentMessages, messages[i])
		}
	}

	// 合并 system 消息和最近消息
	result := make([]model.V4Message, 0, len(systemMessages)+len(recentMessages))
	result = append(result, systemMessages...)
	result = append(result, recentMessages...)

	return result
}
