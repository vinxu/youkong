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

// LLMProvider 模型提供商
type LLMProvider string

const (
	ProviderQwen LLMProvider = "qwen"
	ProviderKimi LLMProvider = "kimi"
	ProviderAuto LLMProvider = "auto" // 自动检测
)

// LLMAdapter LLM 工具调用适配器
type LLMAdapter struct {
	apiKey     string
	apiURL     string
	model      string
	provider   LLMProvider
	httpClient *http.Client
}

// LLMAdapterConfig LLM 适配器配置
type LLMAdapterConfig struct {
	APIKey   string
	APIURL   string
	Model    string
	Provider LLMProvider // 新增：模型提供商
	Timeout  time.Duration
}

// NewLLMAdapter 创建 LLM 适配器
func NewLLMAdapter(config *LLMAdapterConfig) *LLMAdapter {
	provider := config.Provider
	apiURL := config.APIURL
	model := config.Model

	// 自动检测或根据 Provider 配置
	if provider == "" || provider == ProviderAuto {
		if strings.HasPrefix(config.APIKey, "sk-or-") {
			provider = ProviderQwen // OpenRouter 默认用于 Qwen
			apiURL = "https://openrouter.ai/api/v1/chat/completions"
		} else if strings.HasPrefix(config.APIKey, "sk-Kua") || strings.Contains(config.APIKey, "moonshot") {
			// Kimi API Key 前缀检测
			provider = ProviderKimi
		} else {
			provider = ProviderQwen
		}
	}

	// 根据 Provider 设置默认 API URL 和模型
	if apiURL == "" {
		switch provider {
		case ProviderKimi:
			apiURL = "https://api.moonshot.cn/v1/chat/completions"
		case ProviderQwen:
			fallthrough
		default:
			apiURL = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
		}
	}

	if model == "" {
		switch provider {
		case ProviderKimi:
			model = "kimi-k2.5" // Kimi 2.5 模型
		case ProviderQwen:
			fallthrough
		default:
			model = "qwen3-max-2026-01-23"
		}
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 180 * time.Second
	}

	return &LLMAdapter{
		apiKey:   config.APIKey,
		apiURL:   apiURL,
		model:    model,
		provider: provider,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetProvider 获取当前提供商
func (a *LLMAdapter) GetProvider() LLMProvider {
	return a.provider
}

// GetModel 获取当前模型
func (a *LLMAdapter) GetModel() string {
	return a.model
}

// SetModel 动态设置模型
func (a *LLMAdapter) SetModel(model string) {
	a.model = model
}

// SetProvider 动态切换提供商（同时更新 API URL）
func (a *LLMAdapter) SetProvider(provider LLMProvider, apiKey string) {
	a.provider = provider
	a.apiKey = apiKey

	switch provider {
	case ProviderKimi:
		a.apiURL = "https://api.moonshot.cn/v1/chat/completions"
		if a.model == "" || a.model == "qwen3-max-2026-01-23" || a.model == "qwen-max" {
			a.model = "kimi-k2.5"
		}
	case ProviderQwen:
		a.apiURL = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
		if a.model == "" || a.model == "kimi-k2.5" || strings.HasPrefix(a.model, "moonshot-") {
			a.model = "qwen-max"
		}
	}
}

// LLMRequest LLM 请求
type LLMRequest struct {
	Messages       []AgentMessage `json:"messages"`
	Tools          []*Tool        `json:"tools,omitempty"`
	ToolChoice     string         `json:"tool_choice,omitempty"` // "auto", "none", "required"
	Temperature    float64        `json:"temperature,omitempty"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	EnableThinking bool           `json:"enable_thinking,omitempty"` // 是否启用思考模式
	ThinkingBudget int            `json:"thinking_budget,omitempty"` // 思考预算（Token 数）
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

	// 添加温度（kimi-k2.5 模型要求 temperature 必须为 1）
	if strings.HasPrefix(a.model, "kimi-k2") {
		requestBody["temperature"] = 1.0
	} else if req.Temperature > 0 {
		requestBody["temperature"] = req.Temperature
	} else {
		requestBody["temperature"] = 0.7
	}

	// 添加最大 Token
	if req.MaxTokens > 0 {
		requestBody["max_tokens"] = req.MaxTokens
	}

	// 添加思考模式（仅 Qwen 支持）
	if req.EnableThinking && a.provider == ProviderQwen {
		requestBody["enable_thinking"] = true
		if req.ThinkingBudget > 0 {
			requestBody["thinking_budget"] = req.ThinkingBudget
		}
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

// ChatWithAdaptiveThinking 自适应思考模式的聊天请求
// 根据输入复杂度自动决定是否启用思考模式和思考预算
func (a *LLMAdapter) ChatWithAdaptiveThinking(ctx context.Context, req *LLMRequest, userInput string, dataSize int) (*LLMResponse, *AdaptiveThinkingResult, error) {
	// 创建复杂度分析器
	analyzer := NewComplexityAnalyzer()
	score := analyzer.Analyze(userInput, dataSize)

	// 根据复杂度配置思考模式
	req.EnableThinking = score.EnableThinking
	req.ThinkingBudget = score.ThinkingBudget

	// 调用 LLM
	resp, err := a.ChatWithTools(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return resp, &AdaptiveThinkingResult{
		ComplexityScore:   score,
		ThinkingEnabled:   score.EnableThinking,
		ThinkingBudget:    score.ThinkingBudget,
		ComplexityLevel:   string(score.Level),
		EstimatedTimeMs:   score.EstimatedTimeMs,
		ReasoningContent:  resp.Reasoning,
	}, nil
}

// AdaptiveThinkingResult 自适应思考结果
type AdaptiveThinkingResult struct {
	ComplexityScore   *ComplexityScore `json:"complexity_score"`
	ThinkingEnabled   bool             `json:"thinking_enabled"`
	ThinkingBudget    int              `json:"thinking_budget"`
	ComplexityLevel   string           `json:"complexity_level"`
	EstimatedTimeMs   int              `json:"estimated_time_ms"`
	ReasoningContent  string           `json:"reasoning_content,omitempty"`
}

// ComplexityScore 复杂度评分（从 llm 包引用）
type ComplexityScore struct {
	Score           int      `json:"score"`
	Level           string   `json:"level"`
	Factors         []string `json:"factors"`
	EnableThinking  bool     `json:"enable_thinking"`
	ThinkingBudget  int      `json:"thinking_budget"`
	EstimatedTimeMs int      `json:"estimated_time_ms"`
}

// ComplexityAnalyzer 复杂度分析器（简化版，避免循环依赖）
type ComplexityAnalyzer struct {
	deepKeywords  []string
	quickKeywords []string
}

// NewComplexityAnalyzer 创建复杂度分析器
func NewComplexityAnalyzer() *ComplexityAnalyzer {
	return &ComplexityAnalyzer{
		deepKeywords: []string{
			"为什么", "分析", "对比", "比较", "解释", "推理",
			"规律", "总结", "评估", "预测", "建议",
		},
		quickKeywords: []string{
			"快点", "简单说", "一句话", "直接",
			"好的", "确认", "保存", "没问题",
		},
	}
}

// Analyze 分析输入复杂度
func (a *ComplexityAnalyzer) Analyze(input string, dataSize int) *ComplexityScore {
	score := 0
	factors := make([]string, 0)

	// 深度问题检测
	for _, keyword := range a.deepKeywords {
		if strings.Contains(input, keyword) {
			score += 30
			factors = append(factors, "深度问题")
			break
		}
	}

	// 输入长度
	if len(input) > 200 {
		score += 20
		factors = append(factors, "长输入")
	}

	// 数据量
	if dataSize > 100 {
		score += 25
		factors = append(factors, "大数据量")
	}

	// 快速请求减分
	for _, keyword := range a.quickKeywords {
		if strings.Contains(input, keyword) {
			score -= 15
			factors = append(factors, "快速请求")
			break
		}
	}

	// 简单确认减分
	inputLower := strings.ToLower(strings.TrimSpace(input))
	simplePatterns := []string{"好", "行", "ok", "可以", "谢谢", "你好"}
	for _, pattern := range simplePatterns {
		if inputLower == pattern {
			score -= 20
			factors = append(factors, "简单确认")
			break
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// 分类
	result := &ComplexityScore{Score: score, Factors: factors}
	switch {
	case score < 30:
		result.Level = "simple"
		result.EnableThinking = false
		result.ThinkingBudget = 0
		result.EstimatedTimeMs = 1000
	case score < 50:
		result.Level = "medium"
		result.EnableThinking = true
		result.ThinkingBudget = 4096
		result.EstimatedTimeMs = 5000
	case score < 70:
		result.Level = "complex"
		result.EnableThinking = true
		result.ThinkingBudget = 8192
		result.EstimatedTimeMs = 10000
	default:
		result.Level = "very_complex"
		result.EnableThinking = true
		result.ThinkingBudget = 16384
		result.EstimatedTimeMs = 20000
	}
	return result
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

	// 添加温度（kimi-k2.5 模型要求 temperature 必须为 1）
	if strings.HasPrefix(a.model, "kimi-k2") {
		requestBody["temperature"] = 1.0
	} else if req.Temperature > 0 {
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
