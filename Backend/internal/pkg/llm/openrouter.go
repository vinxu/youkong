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

const (
	openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	defaultModel     = "google/gemini-3-flash-preview"
)

// OpenRouterClient OpenRouter API 客户端
type OpenRouterClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenRouterClient 创建 OpenRouter 客户端
func NewOpenRouterClient(apiKey string, model string) *OpenRouterClient {
	if model == "" {
		model = defaultModel
	}
	return &OpenRouterClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
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
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openRouterAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://youkong.app")
	httpReq.Header.Set("X-Title", "YouKong")

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
		return "", fmt.Errorf("api error: %s", chatResp.Error.Message)
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
	ActivityCategoryLeisure      ActivityCategory = "休闲"
	ActivityCategoryWork         ActivityCategory = "工作"
	ActivityCategorySocial       ActivityCategory = "社交"
	ActivityCategoryIdle         ActivityCategory = "闲置"
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
	TimePeriodWorkHours   TimePeriod = "工作时间"
	TimePeriodAfterWork   TimePeriod = "下班后"
	TimePeriodLateNight   TimePeriod = "深夜"
	TimePeriodWeekend     TimePeriod = "周末"
	TimePeriodLunchBreak  TimePeriod = "午休"
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
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openRouterAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://youkong.app")
	httpReq.Header.Set("X-Title", "YouKong")

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
		return "", fmt.Errorf("api error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// GenerateFreeReason 生成隐私安全的有空理由
func (c *OpenRouterClient) GenerateFreeReason(ctx context.Context, state SanitizedUserState) (string, error) {
	prompt := fmt.Sprintf(`你是一个帮助用户判断朋友是否有空的助手。

根据以下脱敏后的状态信息，生成一句简短的推测性描述（10字以内），说明这个人可能是否有空。

状态信息：
- 手机活跃度: %s
- 活动类别: %s
- 位置: %s
- 时间段: %s
- 系统预估有空概率: %d%%

要求：
1. 只输出一句话，不要有任何解释
2. 使用模糊推测性语言，如"可能"、"应该"、"看起来"
3. 绝对不能提及具体APP名称或具体行为
4. 不能说"刷手机"、"玩手机"等暴露隐私的表述
5. 语气轻松自然，像朋友间的口头表达

好的示例：
- "可能有空"
- "应该在忙"
- "看起来挺闲"
- "估计在休息"
- "好像不太方便"

直接输出结果，不要任何前缀或解释：`,
		state.ActivityLevel,
		state.ActivityCategory,
		state.LocationCategory,
		state.TimePeriod,
		state.Probability,
	)

	return c.Chat(ctx, prompt)
}
