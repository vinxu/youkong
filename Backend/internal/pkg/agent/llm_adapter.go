package agent

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

// LLMProvider 模型提供商
type LLMProvider string

const (
	ProviderQwen   LLMProvider = "qwen"
	ProviderKimi   LLMProvider = "kimi"
	ProviderClaude LLMProvider = "claude"
	ProviderAuto   LLMProvider = "auto" // 自动检测
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
		} else if strings.HasPrefix(config.APIKey, "sk-ant-") {
			// Claude/Anthropic API Key 前缀检测
			provider = ProviderClaude
		} else {
			provider = ProviderQwen
		}
	}

	// 根据 Provider 设置默认 API URL 和模型
	if apiURL == "" {
		switch provider {
		case ProviderKimi:
			apiURL = "https://api.moonshot.cn/v1/chat/completions"
		case ProviderClaude:
			apiURL = "https://api.anthropic.com/v1/messages"
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
		case ProviderClaude:
			model = "claude-sonnet-4-20250514" // Claude Sonnet 4
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
		if a.model == "" || a.model == "qwen3-max-2026-01-23" || a.model == "qwen-max" || strings.HasPrefix(a.model, "claude-") {
			a.model = "kimi-k2.5"
		}
	case ProviderClaude:
		a.apiURL = "https://api.anthropic.com/v1/messages"
		if a.model == "" || a.model == "qwen-max" || a.model == "kimi-k2.5" {
			a.model = "claude-sonnet-4-20250514"
		}
	case ProviderQwen:
		a.apiURL = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
		if a.model == "" || a.model == "kimi-k2.5" || strings.HasPrefix(a.model, "moonshot-") || strings.HasPrefix(a.model, "claude-") {
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
	// Claude API 使用不同的格式
	if a.provider == ProviderClaude {
		return a.chatWithToolsClaude(ctx, req)
	}

	// OpenAI 兼容格式（Qwen、Kimi）
	return a.chatWithToolsOpenAI(ctx, req)
}

// chatWithToolsOpenAI OpenAI 兼容格式的 API 调用
func (a *LLMAdapter) chatWithToolsOpenAI(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
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
		requestBody["temperature"] = 0.3
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

// chatWithToolsClaude Claude API 格式的调用
func (a *LLMAdapter) chatWithToolsClaude(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	// 提取 system message 和 user/assistant messages
	var systemPrompt string
	var messages []map[string]interface{}

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			messages = append(messages, map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	// 构建 Claude 格式的请求体
	requestBody := map[string]interface{}{
		"model":      a.model,
		"max_tokens": 1024,
		"messages":   messages,
	}

	if systemPrompt != "" {
		requestBody["system"] = systemPrompt
	}

	// 添加工具定义（Claude 格式）
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = tool.ToClaudeFormat()
		}
		requestBody["tools"] = tools
	}

	// 添加温度
	if req.Temperature > 0 {
		requestBody["temperature"] = req.Temperature
	} else {
		requestBody["temperature"] = 0.3
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
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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

	// 解析 Claude 响应
	var claudeResp struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text,omitempty"`
			ID    string `json:"id,omitempty"`
			Name  string `json:"name,omitempty"`
			Input json.RawMessage `json:"input,omitempty"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Error      *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, body: %s", err, string(respBody))
	}

	// 检查错误
	if claudeResp.Error != nil {
		return nil, fmt.Errorf("Claude API 错误: %s (%s)", claudeResp.Error.Message, claudeResp.Error.Type)
	}

	// 解析响应内容
	result := &LLMResponse{}
	for _, content := range claudeResp.Content {
		switch content.Type {
		case "text":
			result.Content += content.Text
		case "tool_use":
			// 转换为统一的 ToolCall 格式
			tc := ToolCall{
				ID:   content.ID,
				Type: "function",
				Function: ToolCallFunction{
					Name:      content.Name,
					Arguments: string(content.Input),
				},
			}
			result.ToolCalls = append(result.ToolCalls, tc)
		}
	}

	return result, nil
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
		requestBody["temperature"] = 0.3
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

	// 解析流式 SSE 响应
	// SSE 格式: "data: {JSON}\n" 或 "data: [DONE]\n"
	var fullContent strings.Builder
	var reasoning string

	// 流式 tool_calls 需要按 index 增量合并
	type streamToolCall struct {
		Index    int    `json:"index"`
		ID       string `json:"id,omitempty"`
		Type     string `json:"type,omitempty"`
		Function struct {
			Name      string `json:"name,omitempty"`
			Arguments string `json:"arguments,omitempty"`
		} `json:"function,omitempty"`
	}
	toolCallMap := make(map[int]*ToolCall) // index → merged tool call

	scanner := bufio.NewScanner(resp.Body)
	// 增大缓冲区以处理大型 SSE 行
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行和非 data 行
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		// 去掉 "data: " 前缀
		data := strings.TrimPrefix(line, "data: ")

		// 检测流结束标记
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string           `json:"content"`
					ToolCalls        []streamToolCall `json:"tool_calls,omitempty"`
					ReasoningContent string           `json:"reasoning_content,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 跳过无法解析的行
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

			// 按 index 增量合并 tool_calls
			for _, tc := range delta.ToolCalls {
				existing, ok := toolCallMap[tc.Index]
				if !ok {
					existing = &ToolCall{
						Type: "function",
					}
					toolCallMap[tc.Index] = existing
				}
				if tc.ID != "" {
					existing.ID = tc.ID
				}
				if tc.Function.Name != "" {
					existing.Function.Name += tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					existing.Function.Arguments += tc.Function.Arguments
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取流式响应失败: %w", err)
	}

	// 将 toolCallMap 转换为有序切片
	var toolCalls []ToolCall
	for i := 0; i < len(toolCallMap); i++ {
		if tc, ok := toolCallMap[i]; ok {
			toolCalls = append(toolCalls, *tc)
		}
	}

	return &LLMResponse{
		Content:   fullContent.String(),
		ToolCalls: toolCalls,
		Reasoning: reasoning,
	}, nil
}
