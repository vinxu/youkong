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

	"youkong/internal/model"
)

// HolmesStreamEvent 流式事件类型
type HolmesStreamEvent struct {
	Type    string      `json:"type"`    // clue, feature, thinking, conclusion, error
	Phase   string      `json:"phase"`   // collecting, extracting, reasoning, done
	Content string      `json:"content"` // 显示内容
	Data    interface{} `json:"data,omitempty"`
}

// HolmesStreamCallback 流式回调函数
type HolmesStreamCallback func(event *HolmesStreamEvent)

// AnalyzeStream 流式执行福尔摩斯推理分析
func (h *HolmesAnalyzer) AnalyzeStream(ctx context.Context, input *HolmesInput, callback HolmesStreamCallback) (*model.HolmesResult, error) {
	// Phase 1: 收集线索
	callback(&HolmesStreamEvent{
		Type:    "phase",
		Phase:   "collecting",
		Content: "📡 收集线索...",
	})
	time.Sleep(100 * time.Millisecond) // 短暂延迟让前端有时间渲染

	clue := h.collectClues(input)
	h.streamClues(clue, callback)

	// Phase 2: 提取特征
	callback(&HolmesStreamEvent{
		Type:    "phase",
		Phase:   "extracting",
		Content: "🧠 提取特征...",
	})
	time.Sleep(100 * time.Millisecond)

	features := h.extractFeatures(clue)
	h.streamFeatures(features, callback)

	// Phase 3: LLM 推理
	callback(&HolmesStreamEvent{
		Type:    "phase",
		Phase:   "reasoning",
		Content: "💭 福尔摩斯推理中...",
	})

	reasoning, err := h.callLLMStream(ctx, clue, features, input.CoreMemory, callback)
	if err != nil {
		// LLM 调用失败，使用规则引擎降级
		callback(&HolmesStreamEvent{
			Type:    "thinking",
			Content: "⚠️ 大模型暂时不可用，使用规则引擎分析...",
		})
		return h.analyzeWithRules(clue, features), nil
	}

	// Phase 4: 输出结论
	callback(&HolmesStreamEvent{
		Type:    "phase",
		Phase:   "done",
		Content: "✅ 结论",
	})

	h.streamConclusion(reasoning, callback)

	// 构建最终结果
	result := &model.HolmesResult{
		RawData:  clue,
		Features: features,
		Reasoning: &model.HolmesReasoning{
			Model:      holmesModel,
			Thinking:   reasoning.Reasoning,
			Conclusion: reasoning.Summary,
		},
		GeneratedAt: time.Now(),
	}
	result.Result.Available = reasoning.Available
	result.Result.Probability = reasoning.Probability
	result.Result.Confidence = reasoning.Confidence
	result.Result.Summary = reasoning.Summary

	return result, nil
}

// streamClues 流式输出线索
func (h *HolmesAnalyzer) streamClues(clue *model.HolmesClue, callback HolmesStreamCallback) {
	clues := []string{}

	// 时间
	clues = append(clues, fmt.Sprintf("时间: %s %s", clue.Weekday, clue.TimePeriod))

	// 位置
	if clue.PlaceName != "" {
		clues = append(clues, fmt.Sprintf("位置: %s [%s]", clue.PlaceName, h.translatePlaceType(clue.PlaceType)))
	} else if clue.PlaceType != "" {
		clues = append(clues, fmt.Sprintf("位置: %s", h.translatePlaceType(clue.PlaceType)))
	}

	// 屏幕状态
	if clue.ScreenActive {
		activity := h.translateActivityType(clue.ActivityType)
		if clue.ScreenDurationMins > 0 {
			clues = append(clues, fmt.Sprintf("屏幕: 活跃 - %s (%d分钟)", activity, clue.ScreenDurationMins))
		} else {
			clues = append(clues, fmt.Sprintf("屏幕: 活跃 - %s", activity))
		}
	} else {
		if clue.LastActiveMinutesAgo > 0 {
			clues = append(clues, fmt.Sprintf("屏幕: 闲置 (%d分钟前活跃)", clue.LastActiveMinutesAgo))
		} else {
			clues = append(clues, "屏幕: 闲置")
		}
	}

	// 日历
	if clue.HasCalendarEvent {
		if clue.CalendarEventTitle != "" {
			clues = append(clues, fmt.Sprintf("日历: %s", clue.CalendarEventTitle))
		} else {
			clues = append(clues, "日历: 有日程")
		}
	} else {
		if clue.TodayRemainingEvents > 0 {
			clues = append(clues, fmt.Sprintf("日历: 无当前日程 (今日还有%d个)", clue.TodayRemainingEvents))
		} else {
			clues = append(clues, "日历: 无日程")
		}
	}

	// 移动状态
	if clue.IsMoving {
		clues = append(clues, fmt.Sprintf("移动: %s", h.translateMovementType(clue.MovementType)))
	} else if clue.StationaryMinutes > 0 {
		clues = append(clues, fmt.Sprintf("移动: 静止 (%d分钟)", clue.StationaryMinutes))
	}

	// 逐条输出
	for i, c := range clues {
		prefix := "├─"
		if i == len(clues)-1 {
			prefix = "└─"
		}
		callback(&HolmesStreamEvent{
			Type:    "clue",
			Content: fmt.Sprintf("%s %s", prefix, c),
		})
		time.Sleep(80 * time.Millisecond)
	}
}

// streamFeatures 流式输出特征
func (h *HolmesAnalyzer) streamFeatures(features *model.HolmesFeatures, callback HolmesStreamCallback) {
	featureList := []string{}

	if features.LocationType != "" {
		featureList = append(featureList, fmt.Sprintf("位置类型: %s", features.LocationType))
	}
	if features.TimePeriod != "" {
		featureList = append(featureList, fmt.Sprintf("时间段: %s", features.TimePeriod))
	}
	if features.Activity != "" {
		featureList = append(featureList, fmt.Sprintf("活动: %s", features.Activity))
	}
	if features.MovementState != "" {
		featureList = append(featureList, fmt.Sprintf("状态: %s", features.MovementState))
	}
	if features.Schedule != "" {
		featureList = append(featureList, fmt.Sprintf("日程: %s", features.Schedule))
	}

	for i, f := range featureList {
		prefix := "├─"
		if i == len(featureList)-1 {
			prefix = "└─"
		}
		callback(&HolmesStreamEvent{
			Type:    "feature",
			Content: fmt.Sprintf("%s %s", prefix, f),
		})
		time.Sleep(80 * time.Millisecond)
	}
}

// streamConclusion 流式输出结论
func (h *HolmesAnalyzer) streamConclusion(result *HolmesLLMResponse, callback HolmesStreamCallback) {
	status := "有空"
	if !result.Available {
		status = "忙碌"
	}

	conclusions := []string{
		fmt.Sprintf("状态: %s %s", status, result.Emoji),
		fmt.Sprintf("概率: %d%%", result.Probability),
		fmt.Sprintf("置信度: %s", h.translateConfidence(result.Confidence)),
		fmt.Sprintf("摘要: %s", result.Summary),
	}

	for i, c := range conclusions {
		prefix := "├─"
		if i == len(conclusions)-1 {
			prefix = "└─"
		}
		callback(&HolmesStreamEvent{
			Type:    "conclusion",
			Content: fmt.Sprintf("%s %s", prefix, c),
		})
		time.Sleep(80 * time.Millisecond)
	}
}

// callLLMStream 流式调用 LLM
func (h *HolmesAnalyzer) callLLMStream(ctx context.Context, clue *model.HolmesClue, features *model.HolmesFeatures, memory *model.CoreMemory, callback HolmesStreamCallback) (*HolmesLLMResponse, error) {
	if h.client == nil {
		return nil, fmt.Errorf("LLM client not available")
	}

	prompt := h.buildPrompt(clue, features, memory)

	// 构建流式请求
	requestBody := map[string]interface{}{
		"model": holmesModel,
		"messages": []ChatMessage{
			{Role: "user", Content: prompt},
		},
		"enable_thinking": true,
		"stream":          true, // 启用流式输出
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", qwenAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+h.client.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := h.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析 SSE 流
	return h.parseSSEStream(resp.Body, callback)
}

// SSE 数据结构
type sseChunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// parseSSEStream 解析 SSE 流
func (h *HolmesAnalyzer) parseSSEStream(reader io.Reader, callback HolmesStreamCallback) (*HolmesLLMResponse, error) {
	scanner := bufio.NewScanner(reader)

	var fullContent strings.Builder
	var fullReasoning strings.Builder
	var lastReasoningLen int

	for scanner.Scan() {
		line := scanner.Text()

		// SSE 格式: data: {...}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk sseChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		// 处理思考内容
		if delta.ReasoningContent != "" {
			fullReasoning.WriteString(delta.ReasoningContent)

			// 每积累一定长度就输出一次（避免太频繁）
			currentLen := fullReasoning.Len()
			if currentLen-lastReasoningLen >= 20 || strings.HasSuffix(delta.ReasoningContent, "。") || strings.HasSuffix(delta.ReasoningContent, "\n") {
				// 获取新增的内容
				newContent := fullReasoning.String()[lastReasoningLen:]
				if strings.TrimSpace(newContent) != "" {
					callback(&HolmesStreamEvent{
						Type:    "thinking",
						Content: strings.TrimSpace(newContent),
					})
				}
				lastReasoningLen = currentLen
			}
		}

		// 处理最终内容
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
		}

		// 检查是否完成
		if chunk.Choices[0].FinishReason == "stop" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stream: %w", err)
	}

	// 解析最终的 JSON 结果
	return h.parseThinkingResponse(&ThinkingResponse{
		Content:          fullContent.String(),
		ReasoningContent: fullReasoning.String(),
	})
}

// 辅助翻译函数
func (h *HolmesAnalyzer) translatePlaceType(placeType string) string {
	switch placeType {
	case "home":
		return "家"
	case "work":
		return "公司"
	case "leisure":
		return "休闲场所"
	case "transit":
		return "路上"
	default:
		return "未知"
	}
}

func (h *HolmesAnalyzer) translateActivityType(activityType string) string {
	switch activityType {
	case "entertainment":
		return "娱乐内容"
	case "productivity":
		return "工作内容"
	case "communication":
		return "通讯"
	case "idle":
		return "闲置"
	default:
		return "其他"
	}
}

func (h *HolmesAnalyzer) translateMovementType(movementType string) string {
	switch movementType {
	case "walking":
		return "步行中"
	case "running":
		return "跑步中"
	case "driving":
		return "乘车中"
	case "cycling":
		return "骑行中"
	default:
		return "移动中"
	}
}

func (h *HolmesAnalyzer) translateConfidence(confidence string) string {
	switch confidence {
	case "high":
		return "高"
	case "medium":
		return "中"
	case "low":
		return "低"
	default:
		return confidence
	}
}
