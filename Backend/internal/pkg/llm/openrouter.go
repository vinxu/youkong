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
	// OpenRouter API
	openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	// 阿里云通义千问 API
	qwenAPIURL   = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
	defaultModel = "qwen3-max-2026-01-23"
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
	}

	// 统一使用千问模型
	if model == "" {
		model = defaultModel
	}

	return &OpenRouterClient{
		apiKey: apiKey,
		apiURL: apiURL,
		model:  model,
		httpClient: &http.Client{
			Timeout: 180 * time.Second, // 思考模式需要更长时间
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
// 注意：思考模型（如 qwq-plus-latest）只支持流式模式
func (c *OpenRouterClient) ChatWithThinking(ctx context.Context, requestBody map[string]interface{}) (*ThinkingResponse, error) {
	// 强制启用流式模式（思考模型只支持流式）
	requestBody["stream"] = true
	// 启用增量输出
	requestBody["stream_options"] = map[string]bool{"include_usage": true}

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
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// 解析流式响应
	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder

	scanner := bufio.NewScanner(resp.Body)
	// 增大 buffer 以处理长行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行
		if line == "" {
			continue
		}

		// 跳过非数据行
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		// 提取 JSON 数据
		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)

		// 检查是否结束
		if data == "[DONE]" {
			break
		}

		// 解析 SSE 事件
		var event struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			// 忽略解析失败的行
			continue
		}

		if event.Error != nil {
			return nil, fmt.Errorf("api error: %s (code: %s)", event.Error.Message, event.Error.Code)
		}

		if len(event.Choices) > 0 {
			contentBuilder.WriteString(event.Choices[0].Delta.Content)
			reasoningBuilder.WriteString(event.Choices[0].Delta.ReasoningContent)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stream: %w", err)
	}

	return &ThinkingResponse{
		Content:          contentBuilder.String(),
		ReasoningContent: reasoningBuilder.String(),
	}, nil
}

// ChatOptions LLM 调用选项
type ChatOptions struct {
	EnableJSONMode bool    // 启用 JSON Object 模式（确保输出有效 JSON）
	Temperature    float64 // 温度参数（默认 0.85）
}

// ChatWithOptions 带选项的聊天方法
func (c *OpenRouterClient) ChatWithOptions(ctx context.Context, messages []ChatMessage, opts *ChatOptions) (string, error) {
	if opts == nil {
		opts = &ChatOptions{}
	}

	temperature := opts.Temperature
	if temperature == 0 {
		temperature = 0.85
	}

	requestBody := map[string]interface{}{
		"model":              c.model,
		"messages":           messages,
		"temperature":        temperature,
		"top_p":              0.9,
		"frequency_penalty":  0.5,
		"presence_penalty":   0.3,
		"repetition_penalty": 1.1,
	}

	// 启用 JSON Object 模式（阿里云通义千问支持）
	// 注意：提示词中必须包含 "JSON" 关键词
	if opts.EnableJSONMode {
		requestBody["response_format"] = map[string]string{
			"type": "json_object",
		}
	}

	return c.chatWithRequestBody(ctx, requestBody)
}

// PredictionResult AI 推测结果
type PredictionResult struct {
	Schedule         []ScheduleItemResult `json:"schedule"`          // 推测的时刻表
	ReasoningContent string               `json:"reasoning_content"` // 思考过程
	ReasoningSteps   []string             `json:"reasoning_steps"`   // 推理步骤
}

// ScheduleItemResult 推测的时刻表项
type ScheduleItemResult struct {
	StartTime string `json:"start_time"` // HH:MM
	EndTime   string `json:"end_time"`   // HH:MM
	Emoji     string `json:"emoji"`
	Status    string `json:"status"`
}

// PredictSchedule 使用思考模型推测用户未来24小时的状态
func (c *OpenRouterClient) PredictSchedule(ctx context.Context, userContext string, existingSchedule string, startTime string, endTime string) (*PredictionResult, error) {
	prompt := fmt.Sprintf(`你是一个智能助手，需要根据用户的画像和历史行为规律，推测用户在指定时间段内可能的状态。

## 用户上下文
%s

## 已有安排（请勿覆盖）
%s

## 推测时间范围
从 %s 到 %s

## 任务要求
1. **只推测空缺时段**：不要覆盖用户已有的安排
2. **基于用户画像推理**：考虑用户的职业、作息习惯、历史行为规律
3. **合理推测**：给出符合逻辑的状态推测，使用合适的 emoji
4. **时间连贯**：确保时段之间无缝衔接，无重叠

## 输出格式（JSON）
{
  "schedule": [
    {
      "start_time": "09:00",
      "end_time": "12:00",
      "emoji": "💼",
      "status": "上班工作"
    }
  ],
  "reasoning_steps": [
    "分析用户画像：用户是上班族，通常朝九晚五",
    "检查历史规律：工作日上午通常在工作",
    "推测结果：09:00-12:00 应该在上班"
  ]
}

请严格按照 JSON 格式输出，不要包含其他内容。`, userContext, existingSchedule, startTime, endTime)

	// 使用思考模型
	thinkingModel := "qwq-plus-latest" // 通义千问思考模型

	requestBody := map[string]interface{}{
		"model": thinkingModel,
		"messages": []ChatMessage{
			{Role: "user", Content: prompt},
		},
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	resp, err := c.ChatWithThinking(ctx, requestBody)
	if err != nil {
		return nil, fmt.Errorf("思考模型调用失败: %w", err)
	}

	// 清理可能的 markdown 代码块标记
	content := resp.Content
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
	}
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
	}
	if strings.HasSuffix(content, "```") {
		content = strings.TrimSuffix(content, "```")
	}
	content = strings.TrimSpace(content)

	// 解析 JSON 结果
	var result struct {
		Schedule       []ScheduleItemResult `json:"schedule"`
		ReasoningSteps []string             `json:"reasoning_steps"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("解析推测结果失败: %w, 原始内容: %s", err, content)
	}

	return &PredictionResult{
		Schedule:         result.Schedule,
		ReasoningContent: resp.ReasoningContent,
		ReasoningSteps:   result.ReasoningSteps,
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
