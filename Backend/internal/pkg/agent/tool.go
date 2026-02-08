package agent

import (
	"context"
	"encoding/json"
)

// Tool 工具定义
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  ToolParameters `json:"parameters"`
	Handler     ToolHandler    `json:"-"`
}

// ToolParameters 工具参数定义 (JSON Schema 格式)
type ToolParameters struct {
	Type       string               `json:"type"` // "object"
	Properties map[string]ToolParam `json:"properties,omitempty"`
	Required   []string             `json:"required,omitempty"`
}

// ToolParam 工具参数
type ToolParam struct {
	Type        string   `json:"type"`                  // string, number, boolean, array, object
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Items       *ToolParam `json:"items,omitempty"`     // for array type
	Default     any      `json:"default,omitempty"`
}

// ToolHandler 工具处理函数
type ToolHandler func(ctx context.Context, args map[string]interface{}) (*ToolResult, error)

// ToolResult 工具执行结果
type ToolResult struct {
	Success       bool        `json:"success"`
	Data          interface{} `json:"data,omitempty"`
	Error         string      `json:"error,omitempty"`
	TokenEstimate int         `json:"token_estimate,omitempty"` // 估算的 token 数
	ShouldStop    bool        `json:"-"`                        // 终止工具：调用后立即结束 Agent 循环
}

// ToolCall 工具调用 (LLM 返回)
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction 工具调用函数信息
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串
}

// ParseArguments 解析工具参数
func (t *ToolCall) ParseArguments() (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(t.Function.Arguments), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// ToOpenAIFormat 转换为 OpenAI/OpenRouter 格式
func (t *Tool) ToOpenAIFormat() map[string]interface{} {
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.Parameters,
		},
	}
}

// ToClaudeFormat 转换为 Claude/Anthropic 格式
func (t *Tool) ToClaudeFormat() map[string]interface{} {
	return map[string]interface{}{
		"name":         t.Name,
		"description":  t.Description,
		"input_schema": t.Parameters,
	}
}

// FormatToolResult 格式化工具结果为字符串（注入到消息）
func FormatToolResult(result *ToolResult, err error) string {
	if err != nil {
		return "工具执行错误: " + err.Error()
	}
	if !result.Success {
		return "执行失败: " + result.Error
	}

	// 序列化数据
	data, _ := json.MarshalIndent(result.Data, "", "  ")
	return string(data)
}
