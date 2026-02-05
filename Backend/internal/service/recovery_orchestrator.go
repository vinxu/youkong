package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
)

// RecoveryOrchestrator 恢复协调器
// 当系统遇到无法处理的情况时，协调恢复流程
type RecoveryOrchestrator struct {
	llmAdapter *agent.LLMAdapter
	classifier *ErrorClassifier
}

// NewRecoveryOrchestrator 创建恢复协调器
func NewRecoveryOrchestrator(llmAdapter *agent.LLMAdapter) *RecoveryOrchestrator {
	return &RecoveryOrchestrator{
		llmAdapter: llmAdapter,
		classifier: NewErrorClassifier(),
	}
}

// 注意：EventCallback 类型定义在 voice_schedule_phases.go 中

// Handle 处理需要恢复的情况
// 这是恢复协调器的核心方法，根据错误类型选择合适的恢复策略
func (r *RecoveryOrchestrator) Handle(
	ctx context.Context,
	outcome *model.OperationOutcome,
	session *model.VoiceScheduleSession,
	userInput string,
	callback EventCallback,
) error {
	// 1. 分类错误
	classification := r.classifier.Classify(outcome)

	fmt.Printf("[RecoveryOrchestrator] Handling outcome: code=%s, severity=%s, strategy=%s\n",
		outcome.Code, outcome.Severity.String(), classification.Strategy)

	// 2. 根据策略处理
	switch classification.Strategy {
	case model.StrategyRetry:
		// 重试策略由调用方处理，这里只处理重试失败后的情况
		if outcome.IsRecoverable() {
			// 还可以重试，不需要通知用户
			return nil
		}
		// 重试次数用尽，降级到解释
		return r.explainToUser(ctx, outcome, session, userInput, callback)

	case model.StrategyFallback:
		// 降级策略：使用默认消息
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventChat,
			Message: classification.GetDefaultMessage(),
			AvailableActions: r.suggestionsToActions(classification.GetDefaultSuggestions()),
		})
		return nil

	case model.StrategyExplain:
		// 解释策略：让 LLM 生成解释
		return r.explainToUser(ctx, outcome, session, userInput, callback)

	case model.StrategyIgnore:
		// 忽略策略：不做任何处理
		return nil

	default:
		// 未知策略，降级到硬编码兜底
		return r.hardcodedFallback(outcome, callback)
	}
}

// explainToUser 使用 LLM 向用户解释发生了什么
func (r *RecoveryOrchestrator) explainToUser(
	ctx context.Context,
	outcome *model.OperationOutcome,
	session *model.VoiceScheduleSession,
	userInput string,
	callback EventCallback,
) error {
	// 如果没有 LLM adapter，使用硬编码兜底
	if r.llmAdapter == nil {
		return r.hardcodedFallback(outcome, callback)
	}

	// 构建上下文
	contextInfo := r.buildContextInfo(outcome, session)

	prompt := fmt.Sprintf(`用户说："%s"

系统遇到了问题：
- 错误类型：%s
- 详情：%s
- 当前状态：%s

%s

请调用 explain_to_user 工具，用通俗易懂的语言向用户解释发生了什么，并提供下一步建议。`,
		userInput, outcome.Code, outcome.Message, session.State, contextInfo)

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 调用 LLM
	response, err := r.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
		Messages: []agent.AgentMessage{
			agent.NewSystemMessage(agent.SystemPromptExplainError),
			agent.NewUserMessage(prompt),
		},
		Tools:      []*agent.Tool{agent.ExplainToUserTool()},
		ToolChoice: "required",
	})

	if err != nil {
		fmt.Printf("[RecoveryOrchestrator] LLM call failed: %v, using fallback\n", err)
		return r.hardcodedFallback(outcome, callback)
	}

	// 解析响应
	explanation, err := r.parseExplanationResponse(response)
	if err != nil {
		fmt.Printf("[RecoveryOrchestrator] Parse response failed: %v, using fallback\n", err)
		return r.hardcodedFallback(outcome, callback)
	}

	// 发送给用户
	callback(&model.VoiceScheduleEvent{
		Type:             model.VSEventChat,
		Message:          explanation.UserMessage,
		AvailableActions: r.suggestionsToActions(explanation.SuggestedActions),
	})

	return nil
}

// hardcodedFallback 硬编码兜底
// 当 LLM 也失败时，使用预定义的消息
func (r *RecoveryOrchestrator) hardcodedFallback(
	outcome *model.OperationOutcome,
	callback EventCallback,
) error {
	// 优先使用已有的用户消息
	message := outcome.UserMessage
	if message == "" {
		// 根据错误代码选择默认消息
		pattern := r.classifier.GetPattern(outcome.Code)
		if pattern != nil {
			message = pattern.DefaultUserMessage
		} else {
			message = "遇到了一些问题，请稍后再试。"
		}
	}

	suggestions := outcome.Suggestions
	if len(suggestions) == 0 {
		pattern := r.classifier.GetPattern(outcome.Code)
		if pattern != nil {
			suggestions = pattern.DefaultSuggestions
		}
	}

	callback(&model.VoiceScheduleEvent{
		Type:             model.VSEventChat,
		Message:          message,
		AvailableActions: r.suggestionsToActions(suggestions),
	})

	return nil
}

// NotifyAsync 异步通知用户（不阻塞主流程）
// 用于非致命错误的通知，如会话保存失败等
func (r *RecoveryOrchestrator) NotifyAsync(
	ctx context.Context,
	outcome *model.OperationOutcome,
	callback EventCallback,
) {
	// 异步执行，不阻塞主流程
	go func() {
		// 使用新的上下文，防止原上下文已取消
		asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 简化的通知：直接使用硬编码消息
		pattern := r.classifier.GetPattern(outcome.Code)
		if pattern != nil && pattern.DefaultUserMessage != "" {
			callback(&model.VoiceScheduleEvent{
				Type:    model.VSEventProgress,
				Message: pattern.DefaultUserMessage,
				Action:  "warning",
			})
		}
		_ = asyncCtx // 预留用于未来异步 LLM 调用
	}()
}

// buildContextInfo 构建上下文信息
func (r *RecoveryOrchestrator) buildContextInfo(
	outcome *model.OperationOutcome,
	session *model.VoiceScheduleSession,
) string {
	var info string

	// 添加会话上下文
	if session != nil {
		if len(session.CurrentSchedule) > 0 {
			info += fmt.Sprintf("用户当前有 %d 个时刻表项。", len(session.CurrentSchedule))
		}
		if session.Phase != "" {
			info += fmt.Sprintf("当前对话阶段：%s。", session.Phase)
		}
	}

	// 添加错误上下文
	if outcome.Context != nil {
		if action, ok := outcome.Context["action"].(string); ok {
			info += fmt.Sprintf("用户尝试执行的操作：%s。", action)
		}
	}

	return info
}

// parseExplanationResponse 解析 LLM 解释响应
func (r *RecoveryOrchestrator) parseExplanationResponse(response *agent.LLMResponse) (*model.LLMExplanation, error) {
	if len(response.ToolCalls) == 0 {
		return nil, fmt.Errorf("no tool calls in response")
	}

	toolCall := response.ToolCalls[0]
	if toolCall.Function.Name != "explain_to_user" {
		return nil, fmt.Errorf("unexpected tool call: %s", toolCall.Function.Name)
	}

	var explanation model.LLMExplanation
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &explanation); err != nil {
		return nil, fmt.Errorf("parse arguments failed: %w", err)
	}

	return &explanation, nil
}

// suggestionsToActions 将建议转换为操作提示
func (r *RecoveryOrchestrator) suggestionsToActions(suggestions []string) []model.ActionHint {
	if len(suggestions) == 0 {
		return nil
	}

	actions := make([]model.ActionHint, 0, len(suggestions))
	for _, s := range suggestions {
		actions = append(actions, model.ActionHint{
			Action:   "suggestion",
			Label:    s,
			Shortcut: s,
		})
	}
	return actions
}

// ExecuteWithRecovery 包装函数：执行操作并在失败时自动恢复
// 这是一个通用的包装函数，可以用于任何需要优雅降级的操作
func (r *RecoveryOrchestrator) ExecuteWithRecovery(
	ctx context.Context,
	operation func() (*model.OperationOutcome, error),
	session *model.VoiceScheduleSession,
	userInput string,
	callback EventCallback,
	maxRetries int,
) error {
	var lastOutcome *model.OperationOutcome

	for attempt := 0; attempt <= maxRetries; attempt++ {
		outcome, err := operation()

		if err != nil {
			// 操作返回了 error，创建一个 Fatal outcome
			lastOutcome = model.NewFatalOutcome("OPERATION_ERROR", err.Error(), err)
			break
		}

		if outcome.Success {
			// 成功，直接返回
			return nil
		}

		lastOutcome = outcome

		// 检查是否可以重试
		if !outcome.IsRecoverable() {
			break
		}

		// 重试
		outcome.IncrementRetry()
		fmt.Printf("[RecoveryOrchestrator] Retrying operation, attempt %d/%d\n", attempt+1, maxRetries)
		time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond) // 递增延迟
	}

	// 所有重试都失败了，进行恢复处理
	if lastOutcome != nil {
		return r.Handle(ctx, lastOutcome, session, userInput, callback)
	}

	return nil
}

// CreateOutcomeFromError 从普通 error 创建 OperationOutcome
func CreateOutcomeFromError(code string, err error) *model.OperationOutcome {
	if err == nil {
		return model.NewSuccessOutcome()
	}
	return model.NewRecoverableOutcome(code, err.Error()).WithContext("original_error", err.Error())
}
