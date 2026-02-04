package agent

import "encoding/json"

// AgentMessage Agent 消息格式
type AgentMessage struct {
	Role       string     `json:"role"`                   // "system", "user", "assistant", "tool"
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // assistant 返回的工具调用
	ToolCallID string     `json:"tool_call_id,omitempty"` // tool 消息的关联 ID
	Name       string     `json:"name,omitempty"`         // 工具名称（tool 角色使用）
}

// ToLLMFormat 转换为 LLM API 请求格式
func (m *AgentMessage) ToLLMFormat() map[string]interface{} {
	msg := map[string]interface{}{
		"role": m.Role,
	}

	if m.Content != "" {
		msg["content"] = m.Content
	}

	if len(m.ToolCalls) > 0 {
		calls := make([]map[string]interface{}, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			calls[i] = map[string]interface{}{
				"id":   tc.ID,
				"type": tc.Type,
				"function": map[string]interface{}{
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				},
			}
		}
		msg["tool_calls"] = calls
	}

	if m.ToolCallID != "" {
		msg["tool_call_id"] = m.ToolCallID
	}

	if m.Name != "" {
		msg["name"] = m.Name
	}

	return msg
}

// NewUserMessage 创建用户消息
func NewUserMessage(content string) AgentMessage {
	return AgentMessage{
		Role:    "user",
		Content: content,
	}
}

// NewAssistantMessage 创建助手消息
func NewAssistantMessage(content string) AgentMessage {
	return AgentMessage{
		Role:    "assistant",
		Content: content,
	}
}

// NewAssistantToolCallMessage 创建带工具调用的助手消息
func NewAssistantToolCallMessage(content string, toolCalls []ToolCall) AgentMessage {
	return AgentMessage{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	}
}

// NewToolResultMessage 创建工具结果消息
func NewToolResultMessage(toolCallID, toolName, content string) AgentMessage {
	return AgentMessage{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
		Name:       toolName,
	}
}

// NewSystemMessage 创建系统消息
func NewSystemMessage(content string) AgentMessage {
	return AgentMessage{
		Role:    "system",
		Content: content,
	}
}

// EstimateTokens 估算消息的 Token 数量（简单估算）
func (m *AgentMessage) EstimateTokens() int {
	// 简单估算：平均每个字符约 0.3 token（中英文混合）
	tokens := int(float64(len(m.Content)) * 0.3)

	// 工具调用额外估算
	for _, tc := range m.ToolCalls {
		tokens += 20 // 工具调用基础开销
		tokens += int(float64(len(tc.Function.Arguments)) * 0.3)
	}

	return tokens
}

// MessagesToLLMFormat 批量转换消息为 LLM 格式
func MessagesToLLMFormat(messages []AgentMessage) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, m := range messages {
		result[i] = m.ToLLMFormat()
	}
	return result
}

// LLMResponseMessage LLM 响应消息
type LLMResponseMessage struct {
	Role         string     `json:"role"`
	Content      string     `json:"content,omitempty"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Reasoning    string     `json:"reasoning_content,omitempty"` // 思考内容（通义千问）
}

// ParseLLMResponse 解析 LLM 响应
func ParseLLMResponse(data []byte) (*LLMResponseMessage, error) {
	var response struct {
		Choices []struct {
			Message LLMResponseMessage `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return nil, nil
	}

	return &response.Choices[0].Message, nil
}
