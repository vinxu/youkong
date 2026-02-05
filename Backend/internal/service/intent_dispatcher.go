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

// IntentDispatcher 意图分发器
// v2 架构改进：使用 Tool Calling + Chain of Thought 混合机制
// 让 LLM 通过调用 decide_action 工具自主判断用户意图，解决 update_status vs create 混淆问题
type IntentDispatcher struct {
	llmAdapter   *agent.LLMAdapter
	toolRegistry *agent.ToolRegistry
}

// NewIntentDispatcher 创建意图分发器
func NewIntentDispatcher(llmAdapter *agent.LLMAdapter, toolRegistry *agent.ToolRegistry) *IntentDispatcher {
	return &IntentDispatcher{
		llmAdapter:   llmAdapter,
		toolRegistry: toolRegistry,
	}
}

// Dispatch 分发用户意图
// 返回解析后的意图结果，包含 action、thinking、confidence、entities
func (d *IntentDispatcher) Dispatch(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	input string,
) (*model.IntentResult, error) {
	if d.llmAdapter == nil {
		return nil, fmt.Errorf("LLM adapter not configured")
	}

	// 1. 构建上下文感知的系统 Prompt
	systemPrompt := d.buildSystemPrompt(session)

	// 2. 构建 decide_action 工具定义
	decideActionTool := d.buildDecideActionTool()

	// 3. 构建消息
	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
		agent.NewUserMessage(input),
	}

	// 4. 调用 LLM + Tool Calling
	response, err := d.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
		Messages:   messages,
		Tools:      []*agent.Tool{decideActionTool},
		ToolChoice: "required", // 强制必须调用工具
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// 5. 解析 Tool Call 结果
	if len(response.ToolCalls) == 0 {
		// 如果 LLM 没有调用工具，返回默认的 chat 意图
		fmt.Printf("[IntentDispatcher] LLM did not call decide_action, defaulting to chat\n")
		return &model.IntentResult{
			Action:     "chat",
			Thinking:   []string{"LLM 未调用 decide_action 工具", "可能是闲聊"},
			Confidence: 0.3,
		}, nil
	}

	// 解析第一个 Tool Call
	toolCall := response.ToolCalls[0]
	if toolCall.Function.Name != "decide_action" {
		fmt.Printf("[IntentDispatcher] Unexpected tool call: %s\n", toolCall.Function.Name)
		return nil, fmt.Errorf("unexpected tool call: %s", toolCall.Function.Name)
	}

	// 解析参数
	result, err := d.parseToolCallResult(toolCall.Function.Arguments)
	if err != nil {
		return nil, fmt.Errorf("parse tool call result failed: %w", err)
	}

	fmt.Printf("[IntentDispatcher] Dispatch result: action=%s, confidence=%.2f, thinking=%v\n",
		result.Action, result.Confidence, result.Thinking)

	return result, nil
}

// buildSystemPrompt 构建上下文感知的系统 Prompt
func (d *IntentDispatcher) buildSystemPrompt(session *model.VoiceScheduleSession) string {
	var sb strings.Builder

	sb.WriteString(`你是时刻表助手的意图分析器。你的任务是理解用户的真实意图，然后调用 decide_action 工具返回结果。

## 重要规则

1. **必须调用 decide_action 工具**，不要直接回复文本
2. **先填写 thinking**（至少3条分析），再选择 action
3. **区分 update_status 和 create 是最重要的判断**

## 关键区分规则

### update_status（更新当前状态）
用户只是想更新首页显示的当前状态，**不涉及时间规划**：
- "修改为睡不着" → update_status（没有时间范围）
- "我失眠了" → update_status（描述当前状态）
- "在工作" → update_status（当前状态）
- "帮我更新状态" → update_status
- "我现在正在XX" → update_status
- "改成XX"（没有时间）→ update_status

### create（创建时刻表）
用户要规划未来的时间安排，**有明确的时间段**：
- "下午3点到5点开会" → create（有时间范围）
- "明天上午健身" → create（有时间范围）
- "今晚8点到10点看电影" → create（有时间范围）

### 其他 action
- confirm: 用户表示同意（"好的"/"确认"/"是的"/"没问题"）
- cancel: 用户表示取消（"取消"/"不要了"/"算了"）
- query: 用户查询时刻表（"看看"/"调出"/"查看"）
- chat: 闲聊（"你好"/"谢谢"/"你是谁"）
- modify: 修改已有时段（"把下午的会改到4点"）
- replace: 完全替换时刻表（"重新安排"）
- delete: 删除时段（"删掉下午的会"）
- undo: 撤销操作（"撤销"/"撤回"/"后悔了"/"恢复"）

`)

	// 添加当前时间
	now := time.Now()
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	sb.WriteString(fmt.Sprintf("\n## 当前时间\n%s %s %02d:%02d\n",
		weekdays[now.Weekday()], now.Format("2006-01-02"), now.Hour(), now.Minute()))

	// 添加会话状态
	sb.WriteString(fmt.Sprintf("\n## 当前会话状态\n- Phase: %s\n- State: %s\n",
		session.Phase, session.State))

	// 添加已有时刻表（用于判断 modify/delete）
	if len(session.CurrentSchedule) > 0 {
		sb.WriteString("\n## 已有时刻表\n")
		for i, item := range session.CurrentSchedule {
			sb.WriteString(fmt.Sprintf("[%d] %s-%s %s %s\n",
				i, item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
	} else {
		sb.WriteString("\n## 已有时刻表\n（暂无）\n")
	}

	// 添加对话历史摘要
	if len(session.ConversationHistory) > 0 {
		sb.WriteString("\n## 最近对话\n")
		// 只显示最近 3 轮
		start := 0
		if len(session.ConversationHistory) > 6 {
			start = len(session.ConversationHistory) - 6
		}
		for _, turn := range session.ConversationHistory[start:] {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", turn.Role, truncateForPrompt(turn.Content, 50)))
		}
	}

	return sb.String()
}

// buildDecideActionTool 构建 decide_action 工具定义
func (d *IntentDispatcher) buildDecideActionTool() *agent.Tool {
	return &agent.Tool{
		Name: "decide_action",
		Description: `根据用户输入决定执行什么操作。

调用前必须分析（填写 thinking 字段）：
1. 用户的关键词是什么？
2. 用户是否提到了具体时间范围？
3. 用户的真实意图是什么？

【最重要】区分 update_status 和 create：
- update_status：只是想更新当前状态（首页展示），没有时间范围
- create：要规划未来的时间安排，有明确的时间段`,
		Parameters: agent.ToolParameters{
			Type: "object",
			Properties: map[string]agent.ToolParam{
				"thinking": {
					Type:        "array",
					Description: "分析过程，必须先填写，至少3条",
					Items:       &agent.ToolParam{Type: "string"},
				},
				"action": {
					Type:        "string",
					Description: "决定的操作类型",
					Enum:        []string{"create", "modify", "confirm", "cancel", "update_status", "query", "chat", "replace", "delete", "undo"},
				},
				"confidence": {
					Type:        "number",
					Description: "置信度 0.0-1.0",
				},
				"entities": {
					Type:        "object",
					Description: "提取的实体（时间、状态等）",
				},
				"target_date": {
					Type:        "string",
					Description: "目标日期，YYYY-MM-DD 格式",
				},
			},
			Required: []string{"thinking", "action", "confidence"},
		},
	}
}

// parseToolCallResult 解析 Tool Call 返回结果
func (d *IntentDispatcher) parseToolCallResult(arguments string) (*model.IntentResult, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("parse arguments failed: %w", err)
	}

	result := &model.IntentResult{
		Confidence: 0.5,
		Entities:   make(map[string]interface{}),
	}

	// 解析 action
	if action, ok := args["action"].(string); ok {
		result.Action = action
	} else {
		return nil, fmt.Errorf("action is required")
	}

	// 解析 thinking
	if thinkingRaw, ok := args["thinking"].([]interface{}); ok {
		for _, t := range thinkingRaw {
			if s, ok := t.(string); ok {
				result.Thinking = append(result.Thinking, s)
			}
		}
	}

	// 解析 confidence
	if confidence, ok := args["confidence"].(float64); ok {
		result.Confidence = confidence
	}

	// 解析 entities
	if entities, ok := args["entities"].(map[string]interface{}); ok {
		result.Entities = entities

		// 如果是 update_status，尝试提取状态信息
		if result.Action == "update_status" {
			result.StatusInfo = d.extractStatusInfo(entities)
		}
	}

	// 解析 target_date
	if targetDate, ok := args["target_date"].(string); ok {
		result.TargetDate = targetDate
	}

	return result, nil
}

// extractStatusInfo 从 entities 中提取状态信息
func (d *IntentDispatcher) extractStatusInfo(entities map[string]interface{}) *model.CurrentStatusGuess {
	status := &model.CurrentStatusGuess{}

	if emoji, ok := entities["emoji"].(string); ok {
		status.Emoji = emoji
	}
	if statusText, ok := entities["status"].(string); ok {
		status.Status = statusText
	}

	// 如果没有提取到完整的状态信息，返回 nil
	if status.Emoji == "" && status.Status == "" {
		return nil
	}

	return status
}

// truncateForPrompt 截断字符串用于 Prompt
func truncateForPrompt(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
