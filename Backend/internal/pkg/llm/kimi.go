package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// Kimi/Moonshot API
	kimiAPIURL       = "https://api.moonshot.cn/v1/chat/completions"
	defaultKimiModel = "kimi-k2.5-preview"
)

// KimiClient Kimi/Moonshot API 客户端
type KimiClient struct {
	apiKey     string
	apiURL     string
	model      string
	httpClient *http.Client
}

// NewKimiClient 创建 Kimi 客户端
func NewKimiClient(apiKey string, model string) *KimiClient {
	if model == "" {
		model = defaultKimiModel
	}

	return &KimiClient{
		apiKey: apiKey,
		apiURL: kimiAPIURL,
		model:  model,
		httpClient: &http.Client{
			Timeout: 180 * time.Second, // 匹配 Qwen 的超时设置
		},
	}
}

// KimiMessage Kimi 消息格式
type KimiMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []KimiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Name       string         `json:"name,omitempty"`
}

// KimiToolCall Kimi 工具调用格式
type KimiToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "function"
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// KimiTool Kimi 工具定义格式
type KimiTool struct {
	Type     string        `json:"type"` // "function"
	Function KimiFunction  `json:"function"`
}

// KimiFunction Kimi 函数定义
type KimiFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// KimiRequest Kimi API 请求
type KimiRequest struct {
	Model       string        `json:"model"`
	Messages    []KimiMessage `json:"messages"`
	Tools       []KimiTool    `json:"tools,omitempty"`
	ToolChoice  interface{}   `json:"tool_choice,omitempty"` // "auto", "none", "required" 或具体函数
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// KimiResponse Kimi API 响应
type KimiResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      KimiMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// KimiToolResponse 工具调用响应（标准化格式）
type KimiToolResponse struct {
	Content   string           `json:"content"`
	ToolCalls []KimiToolCall   `json:"tool_calls,omitempty"`
	Reasoning string           `json:"reasoning,omitempty"`
}

// Chat 发送简单聊天请求
func (c *KimiClient) Chat(ctx context.Context, prompt string) (string, error) {
	messages := []KimiMessage{
		{Role: "user", Content: prompt},
	}
	return c.ChatWithMessages(ctx, messages)
}

// ChatWithMessages 多轮对话
func (c *KimiClient) ChatWithMessages(ctx context.Context, messages []KimiMessage) (string, error) {
	req := KimiRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.6, // Kimi 推荐温度
	}

	resp, err := c.doRequest(ctx, &req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices")
	}

	return resp.Choices[0].Message.Content, nil
}

// ChatWithTools 带工具调用的聊天
func (c *KimiClient) ChatWithTools(ctx context.Context, messages []KimiMessage, tools []KimiTool, temperature float64) (*KimiToolResponse, error) {
	if temperature <= 0 {
		temperature = 0.6
	}

	req := KimiRequest{
		Model:       c.model,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto",
		Temperature: temperature,
	}

	resp, err := c.doRequest(ctx, &req)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices")
	}

	choice := resp.Choices[0]
	return &KimiToolResponse{
		Content:   choice.Message.Content,
		ToolCalls: choice.Message.ToolCalls,
	}, nil
}

// ChatWithToolsStream 流式带工具调用的聊天
func (c *KimiClient) ChatWithToolsStream(ctx context.Context, messages []KimiMessage, tools []KimiTool, temperature float64, callback func(chunk string)) (*KimiToolResponse, error) {
	if temperature <= 0 {
		temperature = 0.6
	}

	req := KimiRequest{
		Model:       c.model,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto",
		Temperature: temperature,
		Stream:      true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// 解析流式响应
	var contentBuilder strings.Builder
	var toolCalls []KimiToolCall
	toolCallsMap := make(map[int]*KimiToolCall) // 用于累积 tool_calls

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)

		if data == "[DONE]" {
			break
		}

		var event struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int `json:"index"`
						ID       string `json:"id,omitempty"`
						Type     string `json:"type,omitempty"`
						Function struct {
							Name      string `json:"name,omitempty"`
							Arguments string `json:"arguments,omitempty"`
						} `json:"function,omitempty"`
					} `json:"tool_calls,omitempty"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if len(event.Choices) > 0 {
			delta := event.Choices[0].Delta

			if delta.Content != "" {
				contentBuilder.WriteString(delta.Content)
				if callback != nil {
					callback(delta.Content)
				}
			}

			// 累积 tool_calls
			for _, tc := range delta.ToolCalls {
				if _, exists := toolCallsMap[tc.Index]; !exists {
					toolCallsMap[tc.Index] = &KimiToolCall{
						ID:   tc.ID,
						Type: tc.Type,
					}
				}
				existing := toolCallsMap[tc.Index]
				if tc.ID != "" {
					existing.ID = tc.ID
				}
				if tc.Type != "" {
					existing.Type = tc.Type
				}
				if tc.Function.Name != "" {
					existing.Function.Name = tc.Function.Name
				}
				existing.Function.Arguments += tc.Function.Arguments
			}
		}
	}

	// 转换 toolCallsMap 为数组
	for i := 0; i < len(toolCallsMap); i++ {
		if tc, ok := toolCallsMap[i]; ok {
			toolCalls = append(toolCalls, *tc)
		}
	}

	return &KimiToolResponse{
		Content:   contentBuilder.String(),
		ToolCalls: toolCalls,
	}, nil
}

// doRequest 发送 HTTP 请求
func (c *KimiClient) doRequest(ctx context.Context, req *KimiRequest) (*KimiResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var kimiResp KimiResponse
	if err := json.Unmarshal(respBody, &kimiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w, body: %s", err, string(respBody))
	}

	if kimiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s (code: %s, type: %s)",
			kimiResp.Error.Message, kimiResp.Error.Code, kimiResp.Error.Type)
	}

	return &kimiResp, nil
}

// GetModel 获取当前模型名称
func (c *KimiClient) GetModel() string {
	return c.model
}

// SetModel 设置模型名称
func (c *KimiClient) SetModel(model string) {
	c.model = model
}

// TestConnection 测试连接
func (c *KimiClient) TestConnection(ctx context.Context) error {
	_, err := c.Chat(ctx, "你好")
	return err
}

// ========== 工具格式转换辅助函数 ==========

// ConvertToolToKimiFormat 将通用工具格式转换为 Kimi 格式
func ConvertToolToKimiFormat(name, description string, parameters map[string]interface{}) KimiTool {
	return KimiTool{
		Type: "function",
		Function: KimiFunction{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

// ConvertKimiToolCallToGeneric 将 Kimi 工具调用转换为通用格式
func ConvertKimiToolCallToGeneric(tc KimiToolCall) map[string]interface{} {
	return map[string]interface{}{
		"id":   tc.ID,
		"type": tc.Type,
		"function": map[string]interface{}{
			"name":      tc.Function.Name,
			"arguments": tc.Function.Arguments,
		},
	}
}
