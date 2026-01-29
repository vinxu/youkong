package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

import "strings"

const (
	// OpenRouter API
	openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	// 阿里云通义千问 API
	qwenAPIURL   = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
	defaultModel = "qwen-max"
)

// OpenRouterClient LLM API 客户端（支持 OpenRouter 和阿里云通义千问）
type OpenRouterClient struct {
	apiKey     string
	apiURL     string // API 地址
	model      string
	httpClient *http.Client
}

// NewOpenRouterClient 创建 LLM 客户端
func NewOpenRouterClient(apiKey string, model string) *OpenRouterClient {
	// 根据 API Key 前缀判断使用哪个 API
	apiURL := qwenAPIURL
	if strings.HasPrefix(apiKey, "sk-or-") {
		// OpenRouter API Key 以 sk-or- 开头
		apiURL = openRouterAPIURL
		if model == "" {
			model = "google/gemini-2.5-pro-preview-06-05" // OpenRouter 默认模型
		}
	} else if model == "" {
		model = defaultModel
	}

	return &OpenRouterClient{
		apiKey: apiKey,
		apiURL: apiURL,
		model:  model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // 增加超时时间
		},
	}
}

// ChatRequest 请求结构
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatMessage 消息结构
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse 响应结构
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"` // 通义千问思考模式返回的思考过程
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Chat 发送聊天请求
func (c *OpenRouterClient) Chat(ctx context.Context, prompt string) (string, error) {
	req := ChatRequest{
		Model: c.model,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("api error: %s (code: %s)", chatResp.Error.Message, chatResp.Error.Code)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// ActivityLevel 活跃度级别（脱敏后的数据）
type ActivityLevel string

const (
	ActivityLevelHigh   ActivityLevel = "高"
	ActivityLevelMedium ActivityLevel = "中"
	ActivityLevelLow    ActivityLevel = "低"
	ActivityLevelNone   ActivityLevel = "无"
)

// ActivityCategory 活动类别（脱敏后的数据）
type ActivityCategory string

const (
	ActivityCategoryLeisure ActivityCategory = "休闲"
	ActivityCategoryWork    ActivityCategory = "工作"
	ActivityCategorySocial  ActivityCategory = "社交"
	ActivityCategoryIdle    ActivityCategory = "闲置"
)

// LocationCategory 位置类别（脱敏后的数据）
type LocationCategory string

const (
	LocationCategoryHome    LocationCategory = "家"
	LocationCategoryWork    LocationCategory = "公司"
	LocationCategoryOutside LocationCategory = "外出"
	LocationCategoryUnknown LocationCategory = "未知"
)

// TimePeriod 时间段
type TimePeriod string

const (
	TimePeriodWorkHours    TimePeriod = "工作时间"
	TimePeriodAfterWork    TimePeriod = "下班后"
	TimePeriodLateNight    TimePeriod = "深夜"
	TimePeriodWeekend      TimePeriod = "周末"
	TimePeriodLunchBreak   TimePeriod = "午休"
	TimePeriodEarlyMorning TimePeriod = "清晨"
)

// SanitizedUserState 脱敏后的用户状态
type SanitizedUserState struct {
	ActivityLevel    ActivityLevel    // 活跃度
	ActivityCategory ActivityCategory // 活动类别
	LocationCategory LocationCategory // 位置类别
	TimePeriod       TimePeriod       // 时间段
	Probability      int              // 有空概率 0-100
}

// ChatWithMessages 多轮对话
func (c *OpenRouterClient) ChatWithMessages(ctx context.Context, messages []ChatMessage) (string, error) {
	// 使用 map 构建请求，以便添加多样性控制参数
	requestBody := map[string]interface{}{
		"model":              c.model,
		"messages":           messages,
		"temperature":        0.85,  // 提高创造性
		"top_p":              0.9,
		"frequency_penalty":  0.5,   // 惩罚重复词汇
		"presence_penalty":   0.3,   // 惩罚重复话题
		"repetition_penalty": 1.1,   // 通义千问专用参数
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("api error: %s (code: %s)", chatResp.Error.Message, chatResp.Error.Code)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// chatWithRequestBody 使用自定义请求体发送请求
func (c *OpenRouterClient) chatWithRequestBody(ctx context.Context, requestBody map[string]interface{}) (string, error) {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("api error: %s (code: %s)", chatResp.Error.Message, chatResp.Error.Code)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// ThinkingResponse 思考模式响应
type ThinkingResponse struct {
	Content          string // 最终回复内容
	ReasoningContent string // 思考过程
}

// ChatWithThinking 使用思考模式发送请求，返回内容和思考过程
func (c *OpenRouterClient) ChatWithThinking(ctx context.Context, requestBody map[string]interface{}) (*ThinkingResponse, error) {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", qwenAPIURL, bytes.NewReader(body))
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

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("api error: %s (code: %s)", chatResp.Error.Message, chatResp.Error.Code)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices")
	}

	return &ThinkingResponse{
		Content:          chatResp.Choices[0].Message.Content,
		ReasoningContent: chatResp.Choices[0].Message.ReasoningContent,
	}, nil
}

// GenerateFreeReason 生成隐私安全的有空理由
func (c *OpenRouterClient) GenerateFreeReason(ctx context.Context, state SanitizedUserState) (string, error) {
	prompt := fmt.Sprintf(`你是一个帮助用户判断朋友是否有空的助手。

根据以下状态信息，生成一句自然的口语化描述，说明这个人可能在做什么。

状态信息：
- 手机活跃度: %s
- 活动类别: %s
- 位置: %s
- 时间段: %s
- 系统预估有空概率: %d%%

要求：
1. 使用自然口语化表达，像朋友间聊天
2. 可以使用"在摸鱼"、"在刷手机"、"在看剧"等真实描述
3. 不限制长度，表达清楚即可（建议20-30字）
4. 语气轻松自然，带点推测性

好的示例：
- "看起来在公司摸鱼刷手机"
- "可能在家躺着追剧"
- "应该在咖啡厅工作"
- "深夜了还在赶项目，估计没空"
- "周末在外面逛街购物"

直接输出结果：`,
		state.ActivityLevel,
		state.ActivityCategory,
		state.LocationCategory,
		state.TimePeriod,
		state.Probability,
	)

	return c.Chat(ctx, prompt)
}
