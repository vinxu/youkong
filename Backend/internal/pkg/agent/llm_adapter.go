package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LLMAdapter LLM 工具调用适配器
type LLMAdapter struct {
	apiKey     string
	apiURL     string
	model      string
	httpClient *http.Client
}

// LLMAdapterConfig LLM 适配器配置
type LLMAdapterConfig struct {
	APIKey  string
	APIURL  string
	Model   string
	Timeout time.Duration
}

// NewLLMAdapter 创建 LLM 适配器
func NewLLMAdapter(config *LLMAdapterConfig) *LLMAdapter {
	apiURL := config.APIURL
	if apiURL == "" {
		// 根据 API Key 前缀判断使用哪个 API
		if strings.HasPrefix(config.APIKey, "sk-or-") {
			apiURL = "https://openrouter.ai/api/v1/chat/completions"
		} else {
			apiURL = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
		}
	}

	model := config.Model
	if model == "" {
		model = "qwen3-max-2026-01-23"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 180 * time.Second
	}

	return &LLMAdapter{
		apiKey: config.APIKey,
		apiURL: apiURL,
		model:  model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// LLMRequest LLM 请求
type LLMRequest struct {
	Messages    []AgentMessage `json:"messages"`
	Tools       []*Tool        `json:"tools,omitempty"`
	ToolChoice  string         `json:"tool_choice,omitempty"` // "auto", "none", "required"
	Temperature float64        `json:"temperature,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
}

// LLMResponse LLM 响应
type LLMResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Reasoning string     `json:"reasoning,omitempty"` // 思考过程
}

// ChatWithTools 带工具的聊天请求
func (a *LLMAdapter) ChatWithTools(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	// 构建请求体
	requestBody := map[string]interface{}{
		"model":    a.model,
		"messages": MessagesToLLMFormat(req.Messages),
	}

	// 添加工具定义
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = tool.ToOpenAIFormat()
		}
		requestBody["tools"] = tools

		if req.ToolChoice != "" {
			requestBody["tool_choice"] = req.ToolChoice
		} else {
			requestBody["tool_choice"] = "auto"
		}
	}

	// 添加温度
	if req.Temperature > 0 {
		requestBody["temperature"] = req.Temperature
	} else {
		requestBody["temperature"] = 0.7
	}

	// 添加最大 Token
	if req.MaxTokens > 0 {
		requestBody["max_tokens"] = req.MaxTokens
	}

	// 序列化请求
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	// 发送请求
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var apiResp struct {
		Choices []struct {
			Message struct {
				Role             string     `json:"role"`
				Content          string     `json:"content"`
				ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
				ReasoningContent string     `json:"reasoning_content,omitempty"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, body: %s", err, string(respBody))
	}

	// 检查错误
	if apiResp.Error != nil {
		return nil, fmt.Errorf("API 错误: %s (code: %s)", apiResp.Error.Message, apiResp.Error.Code)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("无响应内容")
	}

	choice := apiResp.Choices[0]
	return &LLMResponse{
		Content:   choice.Message.Content,
		ToolCalls: choice.Message.ToolCalls,
		Reasoning: choice.Message.ReasoningContent,
	}, nil
}

// Chat 简单聊天请求（不带工具）
func (a *LLMAdapter) Chat(ctx context.Context, prompt string) (string, error) {
	resp, err := a.ChatWithTools(ctx, &LLMRequest{
		Messages: []AgentMessage{
			NewUserMessage(prompt),
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// ChatWithMessages 多轮对话
func (a *LLMAdapter) ChatWithMessages(ctx context.Context, messages []AgentMessage) (string, error) {
	resp, err := a.ChatWithTools(ctx, &LLMRequest{
		Messages: messages,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// GenerateSummary 生成对话摘要（实现 SummaryGenerator 接口）
func (a *LLMAdapter) GenerateSummary(ctx context.Context, messages []AgentMessage, previousSummary string) (string, error) {
	generator := NewDefaultSummaryGenerator(func(ctx context.Context, prompt string) (string, error) {
		return a.Chat(ctx, prompt)
	})
	return generator.GenerateSummary(ctx, messages, previousSummary)
}

// StreamChatWithTools 流式带工具的聊天请求
func (a *LLMAdapter) StreamChatWithTools(ctx context.Context, req *LLMRequest, callback func(chunk string)) (*LLMResponse, error) {
	// 构建请求体
	requestBody := map[string]interface{}{
		"model":    a.model,
		"messages": MessagesToLLMFormat(req.Messages),
		"stream":   true,
	}

	// 添加工具定义
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = tool.ToOpenAIFormat()
		}
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}

	if req.Temperature > 0 {
		requestBody["temperature"] = req.Temperature
	} else {
		requestBody["temperature"] = 0.7
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析流式响应
	var fullContent strings.Builder
	var toolCalls []ToolCall
	var reasoning string

	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string     `json:"content"`
					ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
					ReasoningContent string     `json:"reasoning_content,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			// 跳过无法解析的行（如 "data: [DONE]"）
			continue
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta

			if delta.Content != "" {
				fullContent.WriteString(delta.Content)
				if callback != nil {
					callback(delta.Content)
				}
			}

			if delta.ReasoningContent != "" {
				reasoning += delta.ReasoningContent
			}

			if len(delta.ToolCalls) > 0 {
				toolCalls = append(toolCalls, delta.ToolCalls...)
			}
		}
	}

	return &LLMResponse{
		Content:   fullContent.String(),
		ToolCalls: toolCalls,
		Reasoning: reasoning,
	}, nil
}
