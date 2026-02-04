package agent

import (
	"context"
	"fmt"
	"strings"
)

// ContextManager 上下文管理器
type ContextManager struct {
	maxTokens     int // 最大 Token 数量
	summaryTarget int // 总结后目标 Token 数
	keepLastN     int // 保留最近 N 轮对话
}

// NewContextManager 创建上下文管理器
func NewContextManager(maxTokens, summaryTarget, keepLastN int) *ContextManager {
	if maxTokens <= 0 {
		maxTokens = 8000
	}
	if summaryTarget <= 0 {
		summaryTarget = 4000
	}
	if keepLastN <= 0 {
		keepLastN = 4
	}

	return &ContextManager{
		maxTokens:     maxTokens,
		summaryTarget: summaryTarget,
		keepLastN:     keepLastN,
	}
}

// SummaryGenerator 总结生成器接口
type SummaryGenerator interface {
	GenerateSummary(ctx context.Context, messages []AgentMessage, previousSummary string) (string, error)
}

// ManageContext 管理会话上下文
func (m *ContextManager) ManageContext(ctx context.Context, session *AgentSession, generator SummaryGenerator, callback StreamCallback) error {
	if session.TokenCount <= m.maxTokens {
		return nil // 预算内，无需处理
	}

	// 发送总结事件
	if callback != nil {
		callback(NewSummarizingEvent())
	}

	return m.compressHistory(ctx, session, generator)
}

// compressHistory 压缩历史对话
func (m *ContextManager) compressHistory(ctx context.Context, session *AgentSession, generator SummaryGenerator) error {
	if len(session.Messages) <= m.keepLastN+1 {
		return nil // 消息太少，无需压缩
	}

	var systemMsg *AgentMessage
	startIdx := 0

	// 保留系统消息
	if len(session.Messages) > 0 && session.Messages[0].Role == "system" {
		systemMsg = &session.Messages[0]
		startIdx = 1
	}

	// 计算需要总结的消息范围
	summaryEnd := len(session.Messages) - m.keepLastN
	if summaryEnd <= startIdx {
		return nil
	}

	toSummarize := session.Messages[startIdx:summaryEnd]
	recentMsgs := session.Messages[summaryEnd:]

	// 生成摘要
	summary, err := generator.GenerateSummary(ctx, toSummarize, session.Summary)
	if err != nil {
		return fmt.Errorf("生成摘要失败: %w", err)
	}

	// 重建消息列表
	newMessages := []AgentMessage{}

	if systemMsg != nil {
		newMessages = append(newMessages, *systemMsg)
	}

	// 添加摘要消息
	summaryContent := "[历史对话摘要]\n" + summary
	summaryMsg := NewSystemMessage(summaryContent)
	newMessages = append(newMessages, summaryMsg)

	// 添加最近消息
	newMessages = append(newMessages, recentMsgs...)

	// 更新会话
	session.Messages = newMessages
	session.Summary = summary
	session.RecalculateTokens()

	return nil
}

// NeedsCompression 检查是否需要压缩
func (m *ContextManager) NeedsCompression(session *AgentSession) bool {
	return session.TokenCount > m.maxTokens
}

// EstimateTokensAfterCompression 估算压缩后的 Token 数
func (m *ContextManager) EstimateTokensAfterCompression(session *AgentSession) int {
	// 估算：保留的消息 + 系统消息 + 摘要（约 500 token）
	tokens := 500 // 摘要估算

	// 系统消息
	if len(session.Messages) > 0 && session.Messages[0].Role == "system" {
		tokens += session.Messages[0].EstimateTokens()
	}

	// 最近消息
	if len(session.Messages) > m.keepLastN {
		for _, msg := range session.Messages[len(session.Messages)-m.keepLastN:] {
			tokens += msg.EstimateTokens()
		}
	}

	return tokens
}

// DefaultSummaryGenerator 默认摘要生成器（使用 LLM）
type DefaultSummaryGenerator struct {
	llmFunc func(ctx context.Context, prompt string) (string, error)
}

// NewDefaultSummaryGenerator 创建默认摘要生成器
func NewDefaultSummaryGenerator(llmFunc func(ctx context.Context, prompt string) (string, error)) *DefaultSummaryGenerator {
	return &DefaultSummaryGenerator{
		llmFunc: llmFunc,
	}
}

// GenerateSummary 生成对话摘要
func (g *DefaultSummaryGenerator) GenerateSummary(ctx context.Context, messages []AgentMessage, previousSummary string) (string, error) {
	// 构建对话文本
	var conversationBuilder strings.Builder
	for _, msg := range messages {
		role := msg.Role
		if role == "assistant" {
			role = "AI"
		} else if role == "user" {
			role = "用户"
		} else if role == "tool" {
			role = "工具[" + msg.Name + "]"
		}
		conversationBuilder.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))

		// 工具调用
		for _, tc := range msg.ToolCalls {
			conversationBuilder.WriteString(fmt.Sprintf("(调用工具: %s)\n", tc.Function.Name))
		}
	}

	prompt := fmt.Sprintf(`请将以下对话内容压缩为简洁的摘要，保留关键信息和决策。

%s上下文摘要:
%s

当前对话:
%s

请生成摘要（控制在 300 字以内）:`,
		func() string {
			if previousSummary != "" {
				return "之前的"
			}
			return ""
		}(),
		func() string {
			if previousSummary != "" {
				return previousSummary
			}
			return "(无)"
		}(),
		conversationBuilder.String(),
	)

	return g.llmFunc(ctx, prompt)
}

// SimpleSummaryGenerator 简单摘要生成器（不使用 LLM）
type SimpleSummaryGenerator struct{}

// NewSimpleSummaryGenerator 创建简单摘要生成器
func NewSimpleSummaryGenerator() *SimpleSummaryGenerator {
	return &SimpleSummaryGenerator{}
}

// GenerateSummary 生成简单摘要（截取关键内容）
func (g *SimpleSummaryGenerator) GenerateSummary(ctx context.Context, messages []AgentMessage, previousSummary string) (string, error) {
	var summary strings.Builder

	if previousSummary != "" {
		summary.WriteString(previousSummary)
		summary.WriteString("\n---\n")
	}

	for _, msg := range messages {
		if msg.Role == "user" {
			// 保留用户消息的前 100 字符
			content := msg.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			summary.WriteString("用户: " + content + "\n")
		} else if msg.Role == "assistant" && msg.Content != "" {
			// 保留助手消息的前 100 字符
			content := msg.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			summary.WriteString("AI: " + content + "\n")
		}

		// 记录工具使用
		for _, tc := range msg.ToolCalls {
			summary.WriteString("(使用工具: " + tc.Function.Name + ")\n")
		}
	}

	return summary.String(), nil
}
