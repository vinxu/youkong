package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"youkong/internal/model"
)

// 福尔摩斯推理模型配置
const (
	// 使用通义千问思考模型
	holmesModel = "qwen3-max-2026-01-23"
)

// HolmesAnalyzer 福尔摩斯推理分析器
type HolmesAnalyzer struct {
	client *OpenRouterClient
}

// NewHolmesAnalyzer 创建福尔摩斯推理分析器
func NewHolmesAnalyzer(client *OpenRouterClient) *HolmesAnalyzer {
	return &HolmesAnalyzer{client: client}
}

// HolmesInput 福尔摩斯分析输入
type HolmesInput struct {
	Status    *model.ExtendedStatusReportRequest
	Timestamp time.Time
	// 历史数据（用于 Few-shot，可选）
	RecentHistory []*model.StatusHistory
	// 用户核心记忆（可选）
	CoreMemory *model.CoreMemory
	// 用户角色画像（可选）
	ProfileType string
}

// HolmesLLMResponse LLM 返回的福尔摩斯推理结果
type HolmesLLMResponse struct {
	Reasoning  string `json:"reasoning"`   // 推理过程
	Available  bool   `json:"available"`   // 是否有空
	Probability int   `json:"probability"` // 有空概率 0-100
	Confidence string `json:"confidence"`  // 置信度 high/medium/low
	Summary    string `json:"summary"`     // 简短总结
	Emoji      string `json:"emoji"`       // 状态 Emoji
}

// Analyze 执行福尔摩斯式推理分析
func (h *HolmesAnalyzer) Analyze(ctx context.Context, input *HolmesInput) (*model.HolmesResult, error) {
	// Layer 1: 收集线索
	clue := h.collectClues(input)

	// Layer 2: 提取特征
	features := h.extractFeatures(clue)

	// Layer 3: 调用 LLM 进行推理
	reasoning, err := h.callLLM(ctx, clue, features, input.CoreMemory)
	if err != nil {
		// LLM 调用失败，使用规则引擎降级
		return h.analyzeWithRules(clue, features), nil
	}

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
	result.Result.Emoji = reasoning.Emoji

	return result, nil
}

// collectClues 收集福尔摩斯线索（Layer 1）
func (h *HolmesAnalyzer) collectClues(input *HolmesInput) *model.HolmesClue {
	clue := &model.HolmesClue{
		Timestamp:  input.Timestamp,
		Weekday:    getChineseWeekday(input.Timestamp),
		TimePeriod: formatTimePeriod(input.Timestamp),
		IsWeekend:  isWeekend(input.Timestamp),
	}

	status := input.Status
	if status == nil {
		return clue
	}

	// 位置线索
	if status.Location != nil {
		clue.PlaceType = string(status.Location.PlaceType)
		clue.AtPlaceSinceMinutes = status.Location.AtPlaceSinceMinutes
	}
	if status.ExtendedLocation != nil {
		clue.PlaceName = status.ExtendedLocation.PlaceName
		clue.PlaceType = string(status.ExtendedLocation.PlaceType)
		clue.AtPlaceSinceMinutes = status.ExtendedLocation.AtPlaceSinceMinutes
	}

	// 海拔线索
	if status.Altitude != nil {
		clue.Altitude = status.Altitude.Altitude
		clue.Floor = status.Altitude.Floor
	}

	// 移动线索
	if status.Movement != nil {
		clue.IsMoving = status.Movement.IsMoving
		clue.MovementType = status.Movement.MovementType
		clue.StepsToday = status.Movement.StepsToday
		clue.StepsLastHour = status.Movement.StepsLastHour
		clue.StationaryMinutes = status.Movement.StationaryMinutes
	}

	// 屏幕线索
	if status.Screen != nil {
		clue.ScreenActive = status.Screen.IsActive
		clue.ScreenDurationMins = status.Screen.SessionDurationMinutes
		clue.ActivityType = string(status.Screen.ActivityType)
		clue.LastActiveMinutesAgo = status.Screen.LastActiveMinutesAgo
	}

	// 日历线索
	if status.Calendar != nil {
		clue.HasCalendarEvent = status.Calendar.HasCurrentEvent
		clue.CalendarEventTitle = sanitizeCalendarTitle(status.Calendar.CurrentEventTitle)
		clue.EventEndMinutes = status.Calendar.EventEndMinutes
		clue.TodayRemainingEvents = status.Calendar.TodayRemainingCount
	}

	// 设备线索
	if status.Connection != nil {
		clue.HeadphonesConnected = status.Connection.IsHeadphonesConnected
	}
	if status.Mode != nil {
		clue.FocusModeOn = status.Mode.IsFocusModeOn
		clue.LowBatteryMode = status.Mode.IsLowPowerMode
	}
	if status.Battery != nil {
		clue.BatteryLevel = status.Battery.BatteryLevel
	}

	return clue
}

// extractFeatures 提取特征（Layer 2）
func (h *HolmesAnalyzer) extractFeatures(clue *model.HolmesClue) *model.HolmesFeatures {
	features := &model.HolmesFeatures{}

	// 位置类型特征
	features.LocationType = h.inferLocationType(clue)

	// 移动状态特征
	features.MovementState = h.inferMovementState(clue)

	// 时间段特征
	features.TimePeriod = h.inferTimePeriod(clue)

	// 活动特征
	features.Activity = h.inferActivity(clue)

	// 日程特征
	features.Schedule = h.inferSchedule(clue)

	// 设备状态特征
	features.DeviceState = h.inferDeviceState(clue)

	return features
}

// inferLocationType 推断位置类型
func (h *HolmesAnalyzer) inferLocationType(clue *model.HolmesClue) string {
	if clue.PlaceName != "" {
		// 如果有地点名称，直接使用
		return clue.PlaceName
	}

	switch clue.PlaceType {
	case "home":
		return "家"
	case "work":
		if clue.Floor > 0 {
			return fmt.Sprintf("办公室（%d楼）", clue.Floor)
		}
		return "办公室"
	case "leisure":
		return "休闲场所"
	case "transit":
		return "路上"
	default:
		return "未知"
	}
}

// inferMovementState 推断移动状态
func (h *HolmesAnalyzer) inferMovementState(clue *model.HolmesClue) string {
	if clue.IsMoving {
		switch clue.MovementType {
		case "walking":
			return fmt.Sprintf("步行中（今日%d步）", clue.StepsToday)
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

	if clue.StationaryMinutes > 0 {
		return fmt.Sprintf("静止（已%d分钟）", clue.StationaryMinutes)
	}
	return "静止"
}

// inferTimePeriod 推断时间段
func (h *HolmesAnalyzer) inferTimePeriod(clue *model.HolmesClue) string {
	hour := clue.Timestamp.Hour()
	weekdayStr := ""
	if clue.IsWeekend {
		weekdayStr = "周末"
	} else {
		weekdayStr = "工作日"
	}

	switch {
	case hour >= 6 && hour < 9:
		return weekdayStr + "早晨"
	case hour >= 9 && hour < 12:
		return weekdayStr + "上午"
	case hour >= 12 && hour < 14:
		return weekdayStr + "午间"
	case hour >= 14 && hour < 18:
		return weekdayStr + "下午"
	case hour >= 18 && hour < 22:
		return weekdayStr + "晚上"
	case hour >= 22 || hour < 6:
		return "深夜"
	default:
		return weekdayStr
	}
}

// inferActivity 推断活动
func (h *HolmesAnalyzer) inferActivity(clue *model.HolmesClue) string {
	if !clue.ScreenActive {
		if clue.LastActiveMinutesAgo > 60 {
			return "长时间未使用手机"
		}
		return "手机闲置"
	}

	duration := ""
	if clue.ScreenDurationMins > 0 {
		duration = fmt.Sprintf("（已%d分钟）", clue.ScreenDurationMins)
	}

	switch clue.ActivityType {
	case "entertainment":
		return "娱乐内容" + duration
	case "productivity":
		return "工作内容" + duration
	case "communication":
		return "通讯中" + duration
	case "idle":
		return "闲置"
	default:
		return "使用中" + duration
	}
}

// inferSchedule 推断日程
func (h *HolmesAnalyzer) inferSchedule(clue *model.HolmesClue) string {
	if !clue.HasCalendarEvent {
		if clue.TodayRemainingEvents > 0 {
			return fmt.Sprintf("无当前日程（今日还有%d个日程）", clue.TodayRemainingEvents)
		}
		return "无日程"
	}

	if clue.EventEndMinutes > 0 {
		if clue.CalendarEventTitle != "" {
			return fmt.Sprintf("%s（还有%d分钟结束）", clue.CalendarEventTitle, clue.EventEndMinutes)
		}
		return fmt.Sprintf("有日程（还有%d分钟结束）", clue.EventEndMinutes)
	}

	if clue.CalendarEventTitle != "" {
		return clue.CalendarEventTitle
	}
	return "有日程"
}

// inferDeviceState 推断设备状态
func (h *HolmesAnalyzer) inferDeviceState(clue *model.HolmesClue) string {
	states := []string{}

	if clue.FocusModeOn {
		states = append(states, "专注模式开启")
	}
	if clue.LowBatteryMode {
		states = append(states, "低电量模式")
	}
	if clue.HeadphonesConnected {
		states = append(states, "戴着耳机")
	}
	if clue.BatteryLevel > 0 && clue.BatteryLevel < 20 {
		states = append(states, fmt.Sprintf("电量%d%%", clue.BatteryLevel))
	}

	if len(states) == 0 {
		return "正常"
	}
	return strings.Join(states, "、")
}

// callLLM 调用 LLM 进行福尔摩斯推理
func (h *HolmesAnalyzer) callLLM(ctx context.Context, clue *model.HolmesClue, features *model.HolmesFeatures, memory *model.CoreMemory) (*HolmesLLMResponse, error) {
	if h.client == nil {
		return nil, fmt.Errorf("LLM client not available")
	}

	prompt := h.buildPrompt(clue, features, memory)

	// 构建请求，启用思考模式
	requestBody := map[string]interface{}{
		"model": holmesModel,
		"messages": []ChatMessage{
			{Role: "user", Content: prompt},
		},
		"enable_thinking": true, // 启用思考模式
	}

	// 使用思考模式 API，获取内容和思考过程
	thinkingResp, err := h.client.ChatWithThinking(ctx, requestBody)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	return h.parseThinkingResponse(thinkingResp)
}

// buildPrompt 构建福尔摩斯推理 Prompt
func (h *HolmesAnalyzer) buildPrompt(clue *model.HolmesClue, features *model.HolmesFeatures, memory *model.CoreMemory) string {
	// 格式化线索
	clueText := h.formatClues(clue)

	// 格式化特征
	featuresText := h.formatFeatures(features)

	// 格式化历史规律（如果有）
	memoryText := "暂无历史规律数据"
	if memory != nil && memory.SampleCount > 0 {
		memoryText = fmt.Sprintf(`- 行为模式: %s
- 时间规律: %s
- 地点偏好: %s
- 社交倾向: %s
（基于 %d 个历史样本）`,
			nvl(memory.BehaviorInsights, "未知"),
			nvl(memory.TimePatterns, "未知"),
			nvl(memory.LocationPreferences, "未知"),
			nvl(memory.SocialTendency, "未知"),
			memory.SampleCount,
		)
	}

	prompt := fmt.Sprintf(`你是福尔摩斯，擅长从细节推断真相。

现在需要推测一个人**此刻正在做什么**、**心情如何**。

## 线索（原始数据）

%s

## 特征（提取的关键信息）

%s

## 历史规律

%s

## 推理要求

1. 像侦探一样分析每个线索的含义
2. 综合所有线索推断此刻的状态和心情
3. 考虑时间、地点、活动的关联性
4. 给出生动的状态描述

## 输出格式（严格 JSON，不要包含任何其他内容）

{
    "reasoning": "你的完整推理过程（不限长度，详细分析每个线索）",
    "available": true,
    "probability": 88,
    "confidence": "high",
    "summary": "周日下午在家躺着刷手机，看起来很悠闲放松",
    "emoji": "🛋️"
}

## 输出要求
- reasoning: 详细的推理过程，像侦探一样分析每个线索，推理出此刻的状态
- summary: 用自然口语化描述**此刻在做什么**（15-30字），要生动形象，如"在家躺平刷手机"、"上班摸鱼中"、"通勤路上听歌"
- emoji: 选择最能代表当前状态的 emoji
- available/probability: 顺便判断是否方便被打扰

## 心情/状态参考
- 放松：在家休息、刷手机、追剧
- 专注：工作中、学习中、开会中
- 运动：跑步、健身、散步
- 社交：聚会、聊天、约会
- 通勤：上班路上、下班路上
- 疲惫：深夜、加班、长时间工作

## Emoji 建议（可自由选择）
🎮 游戏 | 📺 追剧 | 💼 工作 | ☕ 摸鱼
🍜 吃饭 | 🛋️ 躺平 | 🚶 外出 | 😴 睡觉
📱 刷手机 | 💬 聊天 | 🎧 听歌 | 🏃 运动
🍻 聚会 | 🔕 勿扰 | 📊 开会 | 🚇 通勤
🛍️ 购物 | 🎬 看电影 | 📚 看书 | 🎨 创作
🏋️ 健身 | 🍕 美食 | 🎉 娱乐 | 😊 休闲

只输出 JSON，不要有任何前缀或解释。`,
		clueText,
		featuresText,
		memoryText,
	)

	return prompt
}

// formatClues 格式化线索
func (h *HolmesAnalyzer) formatClues(clue *model.HolmesClue) string {
	lines := []string{}

	// 时间线索
	lines = append(lines, fmt.Sprintf("时间：%s %s", clue.Weekday, clue.TimePeriod))

	// 位置线索
	if clue.PlaceName != "" {
		lines = append(lines, fmt.Sprintf("位置：%s", clue.PlaceName))
	} else if clue.PlaceType != "" {
		lines = append(lines, fmt.Sprintf("位置：%s", clue.PlaceType))
	}
	if clue.AtPlaceSinceMinutes > 0 {
		lines = append(lines, fmt.Sprintf("在此地：%d 分钟", clue.AtPlaceSinceMinutes))
	}

	// 海拔/楼层线索
	if clue.Floor > 0 {
		lines = append(lines, fmt.Sprintf("楼层：%d 楼", clue.Floor))
	} else if clue.Altitude > 0 {
		lines = append(lines, fmt.Sprintf("海拔：%.0f 米", clue.Altitude))
	}

	// 移动线索
	if clue.IsMoving {
		lines = append(lines, fmt.Sprintf("移动状态：%s", clue.MovementType))
	} else if clue.StationaryMinutes > 0 {
		lines = append(lines, fmt.Sprintf("静止时长：%d 分钟", clue.StationaryMinutes))
	}
	if clue.StepsToday > 0 {
		lines = append(lines, fmt.Sprintf("今日步数：%d 步（最近1小时 %d 步）", clue.StepsToday, clue.StepsLastHour))
	}

	// 屏幕线索（只有当有实际屏幕数据时才输出）
	hasScreenData := clue.ActivityType != "" || clue.ScreenDurationMins > 0 || clue.LastActiveMinutesAgo > 0
	if hasScreenData {
		if clue.ScreenActive {
			lines = append(lines, fmt.Sprintf("屏幕：活跃，%s（已使用 %d 分钟）", clue.ActivityType, clue.ScreenDurationMins))
		} else if clue.LastActiveMinutesAgo > 0 {
			lines = append(lines, fmt.Sprintf("屏幕：闲置（%d 分钟前活跃）", clue.LastActiveMinutesAgo))
		}
	}

	// 日历线索
	if clue.HasCalendarEvent {
		if clue.CalendarEventTitle != "" {
			lines = append(lines, fmt.Sprintf("日历：%s（还有 %d 分钟结束）", clue.CalendarEventTitle, clue.EventEndMinutes))
		} else {
			lines = append(lines, fmt.Sprintf("日历：有日程（还有 %d 分钟结束）", clue.EventEndMinutes))
		}
	} else {
		lines = append(lines, fmt.Sprintf("日历：无日程（今日还剩 %d 个）", clue.TodayRemainingEvents))
	}

	// 设备线索
	deviceLines := []string{}
	if clue.FocusModeOn {
		deviceLines = append(deviceLines, "专注模式开启")
	}
	if clue.LowBatteryMode {
		deviceLines = append(deviceLines, "低电量模式")
	}
	if clue.HeadphonesConnected {
		deviceLines = append(deviceLines, "耳机已连接")
	}
	if clue.BatteryLevel > 0 {
		deviceLines = append(deviceLines, fmt.Sprintf("电量 %d%%", clue.BatteryLevel))
	}
	if len(deviceLines) > 0 {
		lines = append(lines, fmt.Sprintf("设备：%s", strings.Join(deviceLines, "、")))
	}

	return strings.Join(lines, "\n")
}

// formatFeatures 格式化特征
func (h *HolmesAnalyzer) formatFeatures(features *model.HolmesFeatures) string {
	return fmt.Sprintf(`- 位置类型：%s
- 移动状态：%s
- 时间段：%s
- 活动：%s
- 日程：%s
- 设备状态：%s`,
		features.LocationType,
		features.MovementState,
		features.TimePeriod,
		features.Activity,
		features.Schedule,
		features.DeviceState,
	)
}

// parseThinkingResponse 解析通义千问思考模式响应
func (h *HolmesAnalyzer) parseThinkingResponse(resp *ThinkingResponse) (*HolmesLLMResponse, error) {
	// 清理响应内容（移除可能的 markdown 代码块）
	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result HolmesLLMResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w, content: %s", err, content)
	}

	// 使用 reasoning_content 作为思考过程
	if resp.ReasoningContent != "" {
		result.Reasoning = resp.ReasoningContent
	}

	// 验证和修正数据
	if result.Probability < 0 {
		result.Probability = 0
	}
	if result.Probability > 100 {
		result.Probability = 100
	}
	if result.Confidence == "" {
		result.Confidence = "medium"
	}
	if result.Summary == "" {
		result.Summary = "状态未知"
	}
	if result.Emoji == "" {
		result.Emoji = "🤔"
	}

	return &result, nil
}

// parseResponse 解析 LLM 响应（兼容旧格式）
func (h *HolmesAnalyzer) parseResponse(response string) (*HolmesLLMResponse, error) {
	// 提取 <think> 标签中的思考内容（如果有）
	thinkContent := ""
	thinkRegex := regexp.MustCompile(`<think>([\s\S]*?)</think>`)
	if matches := thinkRegex.FindStringSubmatch(response); len(matches) > 1 {
		thinkContent = strings.TrimSpace(matches[1])
	}

	// 移除 <think> 标签
	response = thinkRegex.ReplaceAllString(response, "")

	// 清理响应（移除可能的 markdown 代码块）
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var result HolmesLLMResponse
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}

	// 如果有思考内容，添加到推理中
	if thinkContent != "" && result.Reasoning == "" {
		result.Reasoning = thinkContent
	}

	// 验证和修正数据
	if result.Probability < 0 {
		result.Probability = 0
	}
	if result.Probability > 100 {
		result.Probability = 100
	}
	if result.Confidence == "" {
		result.Confidence = "medium"
	}
	if result.Summary == "" {
		result.Summary = "状态未知"
	}
	if result.Emoji == "" {
		result.Emoji = "🤔"
	}

	return &result, nil
}

// analyzeWithRules 基于规则的分析（LLM 不可用时的降级方案）
func (h *HolmesAnalyzer) analyzeWithRules(clue *model.HolmesClue, features *model.HolmesFeatures) *model.HolmesResult {
	score := 50
	emoji := "🤔"
	summary := "状态未知"
	reasoning := "基于规则分析"

	// 时间因素
	if clue.IsWeekend {
		score += 15
		reasoning += "，周末时间"
	} else {
		hour := clue.Timestamp.Hour()
		if hour >= 18 && hour < 22 {
			score += 15
			reasoning += "，下班时间"
		} else if hour >= 9 && hour < 18 {
			score -= 15
			reasoning += "，工作时间"
		}
	}

	// 位置因素
	switch clue.PlaceType {
	case "home":
		score += 15
		emoji = "🛋️"
		summary = "在家"
		reasoning += "，在家"
	case "work":
		hour := clue.Timestamp.Hour()
		if hour >= 9 && hour < 18 && !clue.IsWeekend {
			score -= 20
			emoji = "💼"
			summary = "在工作"
			reasoning += "，工作时间在公司"
		} else {
			emoji = "☕"
			summary = "可能在摸鱼"
		}
	case "leisure":
		score += 20
		emoji = "🚶"
		summary = "在外面"
		reasoning += "，在休闲场所"
	}

	// 日历因素
	if clue.HasCalendarEvent {
		score -= 25
		emoji = "📊"
		summary = "有日程"
		reasoning += "，有日程安排"
	}

	// 屏幕因素
	if clue.ScreenActive {
		switch clue.ActivityType {
		case "entertainment":
			score += 15
			if emoji == "🤔" {
				emoji = "📱"
				summary = "在刷手机"
			}
			reasoning += "，在看娱乐内容"
		case "productivity":
			score -= 10
			emoji = "💼"
			summary = "在工作"
			reasoning += "，在处理工作"
		case "communication":
			score += 5
			emoji = "💬"
			summary = "在聊天"
			reasoning += "，在通讯"
		}
	} else if clue.LastActiveMinutesAgo > 60 {
		score -= 15
		hour := clue.Timestamp.Hour()
		if hour >= 23 || hour < 7 {
			emoji = "😴"
			summary = "可能在睡觉"
			reasoning += "，长时间未使用手机，可能在睡觉"
		} else {
			reasoning += "，长时间未使用手机"
		}
	}

	// 专注模式
	if clue.FocusModeOn {
		score -= 20
		emoji = "🔕"
		summary = "不想被打扰"
		reasoning += "，专注模式开启"
	}

	// 边界处理
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	confidence := "medium"
	if clue.HasCalendarEvent || clue.FocusModeOn {
		confidence = "high"
	} else if clue.PlaceType == "" || clue.PlaceType == "unknown" {
		confidence = "low"
	}

	result := &model.HolmesResult{
		RawData:  clue,
		Features: features,
		Reasoning: &model.HolmesReasoning{
			Model:      "rules",
			Thinking:   "",
			Conclusion: reasoning,
		},
		GeneratedAt: time.Now(),
	}
	result.Result.Available = score >= 50
	result.Result.Probability = score
	result.Result.Confidence = confidence
	result.Result.Summary = summary
	result.Result.Emoji = emoji

	return result
}

// 辅助函数

func getChineseWeekday(t time.Time) string {
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return weekdays[t.Weekday()]
}

func formatTimePeriod(t time.Time) string {
	hour := t.Hour()
	minute := t.Minute()

	var period string
	switch {
	case hour >= 5 && hour < 9:
		period = "早上"
	case hour >= 9 && hour < 12:
		period = "上午"
	case hour >= 12 && hour < 14:
		period = "中午"
	case hour >= 14 && hour < 18:
		period = "下午"
	case hour >= 18 && hour < 22:
		period = "晚上"
	default:
		period = "深夜"
	}

	return fmt.Sprintf("%s%d点%d分", period, hour, minute)
}

func isWeekend(t time.Time) bool {
	day := t.Weekday()
	return day == time.Saturday || day == time.Sunday
}

func sanitizeCalendarTitle(title string) string {
	if title == "" {
		return ""
	}
	// 脱敏处理：移除敏感词汇
	sensitiveWords := []string{"密码", "账号", "薪资", "工资", "面试"}
	for _, word := range sensitiveWords {
		if strings.Contains(title, word) {
			return "私人事务"
		}
	}
	// 截断过长标题
	if len([]rune(title)) > 20 {
		return string([]rune(title)[:20]) + "..."
	}
	return title
}

// ========== Holmes 2.0 语义上下文建模 ==========

// buildSemanticContext 从原始线索构建语义上下文 (Layer 2)
func (h *HolmesAnalyzer) buildSemanticContext(clue *model.HolmesClue) *model.SemanticContext {
	return &model.SemanticContext{
		Space:    h.inferSpaceSemantic(clue),
		Time:     h.inferTimeSemantic(clue),
		Activity: h.inferActivitySemantic(clue),
		Energy:   h.inferEnergyLevel(clue),
	}
}

// inferSpaceSemantic 推断空间语义
func (h *HolmesAnalyzer) inferSpaceSemantic(clue *model.HolmesClue) *model.SpaceSemantic {
	space := &model.SpaceSemantic{
		Nature: "未知空间",
		Vibe:   "未知",
		Social: "未知",
	}

	// 根据移动状态判断
	if clue.IsMoving {
		space.Nature = "移动中"
		if clue.MovementType == "walking" {
			space.Vibe = "户外"
			space.Social = "可能有他人"
		} else if clue.MovementType == "driving" {
			space.Vibe = "封闭"
			space.Social = "独处或少数人"
		}
		return space
	}

	// 根据位置类型判断
	switch clue.PlaceType {
	case "home":
		space.Nature = "私密空间"
		space.Vibe = "安静"
		space.Social = "独处"
	case "work":
		space.Nature = "专业空间"
		space.Vibe = "专业"
		space.Social = "可能有同事"
	case "leisure":
		space.Nature = "公共空间"
		space.Vibe = "休闲"
		space.Social = "社交场合"
	case "transit":
		space.Nature = "移动中"
		space.Vibe = "嘈杂"
		space.Social = "陌生人群"
	}

	// 根据耳机连接判断
	if clue.HeadphonesConnected {
		space.Social = "可能独处（戴耳机）"
	}

	// 根据专注模式判断
	if clue.FocusModeOn {
		space.Vibe = "安静（专注模式）"
	}

	return space
}

// inferTimeSemantic 推断时间语义
func (h *HolmesAnalyzer) inferTimeSemantic(clue *model.HolmesClue) *model.TimeSemantic {
	time := &model.TimeSemantic{
		Phase:      "未知",
		Rhythm:     "未知",
		Continuity: "未知",
	}

	hour := clue.Timestamp.Hour()

	// 推断时间阶段
	switch {
	case hour >= 5 && hour < 9:
		time.Phase = "苏醒期"
	case hour >= 9 && hour < 12:
		time.Phase = "高效期（上午）"
	case hour >= 12 && hour < 14:
		time.Phase = "休整期（午间）"
	case hour >= 14 && hour < 18:
		time.Phase = "高效期（下午）"
	case hour >= 18 && hour < 22:
		time.Phase = "放松期"
	case hour >= 22 || hour < 5:
		time.Phase = "入睡期"
	}

	// 推断节奏
	if clue.IsWeekend {
		time.Rhythm = "休闲节奏"
	} else {
		if hour >= 9 && hour < 18 {
			time.Rhythm = "工作节奏"
		} else if hour >= 7 && hour < 9 || hour >= 18 && hour < 20 {
			time.Rhythm = "过渡期（通勤时段）"
		} else {
			time.Rhythm = "个人时间"
		}
	}

	// 推断持续性
	if clue.AtPlaceSinceMinutes < 15 {
		time.Continuity = "刚开始"
	} else if clue.AtPlaceSinceMinutes < 120 {
		time.Continuity = "进行中"
	} else {
		time.Continuity = "持续较久"
	}

	return time
}

// inferActivitySemantic 推断活动语义
func (h *HolmesAnalyzer) inferActivitySemantic(clue *model.HolmesClue) *model.ActivitySemantic {
	activity := &model.ActivitySemantic{
		BodyState:  "未知",
		MindState:  "未知",
		Engagement: "未知",
	}

	// 推断身体状态
	if clue.IsMoving {
		switch clue.MovementType {
		case "running":
			activity.BodyState = "运动中"
		case "walking":
			activity.BodyState = "轻度活动"
		case "driving", "cycling":
			activity.BodyState = "移动中"
		default:
			activity.BodyState = "移动中"
		}
	} else {
		if clue.StationaryMinutes > 60 {
			activity.BodyState = "长时间静态"
		} else {
			activity.BodyState = "静态"
		}
	}

	// 推断心智状态
	if clue.HasCalendarEvent {
		activity.MindState = "专注（有日程）"
	} else if clue.FocusModeOn {
		activity.MindState = "专注（勿扰模式）"
	} else if clue.ScreenActive {
		switch clue.ActivityType {
		case "entertainment":
			activity.MindState = "消遣"
		case "productivity":
			activity.MindState = "专注"
		case "communication":
			activity.MindState = "社交"
		default:
			activity.MindState = "消遣"
		}
	} else if clue.LastActiveMinutesAgo > 60 {
		activity.MindState = "休息"
	} else {
		activity.MindState = "闲置"
	}

	// 推断投入程度
	if clue.ScreenDurationMins > 30 || clue.StationaryMinutes > 60 {
		activity.Engagement = "深度投入"
	} else if clue.ScreenActive {
		activity.Engagement = "浅层互动"
	} else if clue.LastActiveMinutesAgo < 30 {
		activity.Engagement = "间歇使用"
	} else {
		activity.Engagement = "闲置"
	}

	return activity
}

// inferEnergyLevel 推断能量状态
func (h *HolmesAnalyzer) inferEnergyLevel(clue *model.HolmesClue) *model.EnergyLevel {
	energy := &model.EnergyLevel{
		Physical: "正常",
		Mental:   "平静",
		Social:   "中性",
	}

	hour := clue.Timestamp.Hour()

	// 推断身体能量
	if clue.StepsLastHour > 500 {
		energy.Physical = "活跃"
	} else if clue.LowBatteryMode || clue.BatteryLevel < 20 {
		energy.Physical = "可能疲惫"
	} else if hour >= 23 || hour < 6 {
		energy.Physical = "可能疲惫"
	}

	// 推断精神能量
	if clue.FocusModeOn {
		energy.Mental = "专注"
	} else if hour >= 23 || hour < 6 {
		energy.Mental = "低迷"
	} else if clue.ScreenActive && clue.ActivityType == "entertainment" {
		energy.Mental = "放松"
	} else if clue.HasCalendarEvent {
		energy.Mental = "专注"
	}

	// 推断社交能量
	if clue.FocusModeOn {
		energy.Social = "封闭"
	} else if clue.PlaceType == "leisure" || clue.ActivityType == "communication" {
		energy.Social = "开放"
	} else if clue.PlaceType == "home" && (hour >= 22 || hour < 8) {
		energy.Social = "封闭"
	}

	return energy
}

// ========== 异常检测 ==========

// detectAnomalies 检测行为异常
func (h *HolmesAnalyzer) detectAnomalies(clue *model.HolmesClue, memory *model.CoreMemory) []model.Anomaly {
	anomalies := []model.Anomaly{}

	if memory == nil || memory.SampleCount < 5 {
		return anomalies // 数据不足，无法检测异常
	}

	hour := clue.Timestamp.Hour()

	// 时间异常检测
	if (hour >= 23 || hour < 6) && clue.ScreenActive && clue.ScreenDurationMins > 30 {
		if strings.Contains(memory.TimePatterns, "早睡") {
			anomalies = append(anomalies, model.Anomaly{
				Type:   "unusual_time",
				Detail: "通常这时候已经休息了，今晚还在活跃",
			})
		}
	}

	// 地点异常检测
	if !clue.IsWeekend && hour >= 9 && hour < 18 {
		if clue.PlaceType == "leisure" || clue.PlaceType == "home" {
			if strings.Contains(memory.BehaviorInsights, "工作日") && strings.Contains(memory.BehaviorInsights, "公司") {
				anomalies = append(anomalies, model.Anomaly{
					Type:   "unusual_location",
					Detail: "工作日工作时间不在公司，可能请假或远程",
				})
			}
		}
	}

	// 周末地点异常
	if clue.IsWeekend && clue.PlaceType == "work" {
		if strings.Contains(memory.BehaviorInsights, "周末") && !strings.Contains(memory.BehaviorInsights, "加班") {
			anomalies = append(anomalies, model.Anomaly{
				Type:   "unusual_location",
				Detail: "周末在公司，可能有紧急工作或加班",
			})
		}
	}

	// 活动异常检测
	if clue.StepsToday > 15000 {
		if !strings.Contains(memory.BehaviorInsights, "运动") && !strings.Contains(memory.BehaviorInsights, "健身") {
			anomalies = append(anomalies, model.Anomaly{
				Type:   "behavior_change",
				Detail: "今天活动量异常大，可能在旅行或运动",
			})
		}
	}

	// 长时间静止异常
	if clue.StationaryMinutes > 180 && !clue.HasCalendarEvent && hour >= 9 && hour < 18 {
		anomalies = append(anomalies, model.Anomaly{
			Type:   "behavior_change",
			Detail: "已静止超过3小时，可能在专注做某事或休息",
		})
	}

	return anomalies
}

// ========== Holmes 2.0 创意叙事生成 ==========

// Analyze2 执行 Holmes 2.0 推理分析（新版本）
func (h *HolmesAnalyzer) Analyze2(ctx context.Context, input *HolmesInput) (*model.Holmes2Result, error) {
	// Layer 1: 收集线索
	clue := h.collectClues(input)

	// Layer 2: 构建语义上下文
	semanticCtx := h.buildSemanticContext(clue)

	// Layer 3: 异常检测
	anomalies := h.detectAnomalies(clue, input.CoreMemory)

	// Layer 4: 调用 LLM 进行创意叙事生成
	creative, err := h.callCreativeLLM(ctx, clue, semanticCtx, anomalies, input.CoreMemory)
	if err != nil {
		// LLM 调用失败，使用规则引擎降级
		fallbackResult := h.analyzeWithRules(clue, h.extractFeatures(clue))
		return &model.Holmes2Result{
			RawData:   clue,
			Context:   semanticCtx,
			Anomalies: anomalies,
			Creative: &model.HolmesCreativeResult{
				Narrative:   "规则推理：" + fallbackResult.Reasoning.Conclusion,
				Scene:       fallbackResult.Result.Summary,
				Emoji:       fallbackResult.Result.Emoji,
				Mood:        &model.MoodVector{Valence: 0, Arousal: 0.3, Openness: 0.5},
				Confidence:  fallbackResult.Result.Confidence,
				Basis:       []string{"规则推理"},
				GeneratedAt: time.Now().UnixMilli(),
			},
			Result: fallbackResult.Result,
			GeneratedAt: time.Now().UnixMilli(),
		}, nil
	}

	// 构建最终结果
	result := &model.Holmes2Result{
		RawData:     clue,
		Context:     semanticCtx,
		Anomalies:   anomalies,
		Creative:    creative,
		GeneratedAt: time.Now().UnixMilli(),
	}

	// 设置兼容旧版的结果字段
	result.Result.Summary = creative.Scene
	result.Result.Emoji = creative.Emoji
	result.Result.Confidence = creative.Confidence

	// 根据心情向量推算有空概率
	if creative.Mood != nil {
		// 社交开放度越高，有空概率越高
		// 唤醒度中等时有空概率最高
		openness := creative.Mood.Openness
		arousal := creative.Mood.Arousal
		valence := creative.Mood.Valence

		// 基础分数 = 社交开放度 * 60 + 效价 * 20
		probability := int(openness*60 + (valence+1)*10)

		// 唤醒度过高或过低都会降低概率
		if arousal > 0.7 || arousal < 0.2 {
			probability -= 10
		}

		if probability > 100 {
			probability = 100
		}
		if probability < 0 {
			probability = 0
		}

		result.Result.Probability = probability
		result.Result.Available = probability >= 50
	} else {
		result.Result.Probability = 50
		result.Result.Available = true
	}

	return result, nil
}

// HolmesCreativeLLMResponse LLM 返回的创意叙事结果
type HolmesCreativeLLMResponse struct {
	Narrative  string   `json:"narrative"`
	Scene      string   `json:"scene"`
	Emoji      string   `json:"emoji"`
	Mood       struct {
		Valence  float64 `json:"valence"`
		Arousal  float64 `json:"arousal"`
		Openness float64 `json:"openness"`
	} `json:"mood"`
	Confidence string   `json:"confidence"`
	Basis      []string `json:"basis"`
}

// callCreativeLLM 调用 LLM 进行创意叙事生成
func (h *HolmesAnalyzer) callCreativeLLM(ctx context.Context, clue *model.HolmesClue, semanticCtx *model.SemanticContext, anomalies []model.Anomaly, memory *model.CoreMemory) (*model.HolmesCreativeResult, error) {
	if h.client == nil {
		return nil, fmt.Errorf("LLM client not available")
	}

	prompt := h.buildCreativePrompt(clue, semanticCtx, anomalies, memory)

	requestBody := map[string]interface{}{
		"model": holmesModel,
		"messages": []ChatMessage{
			{Role: "user", Content: prompt},
		},
		"enable_thinking": true,
	}

	thinkingResp, err := h.client.ChatWithThinking(ctx, requestBody)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	return h.parseCreativeResponse(thinkingResp)
}

// buildCreativePrompt 构建创意叙事 Prompt
func (h *HolmesAnalyzer) buildCreativePrompt(clue *model.HolmesClue, ctx *model.SemanticContext, anomalies []model.Anomaly, memory *model.CoreMemory) string {
	// 格式化语义上下文
	spaceText := "未知"
	if ctx.Space != nil {
		spaceText = fmt.Sprintf("空间性质: %s\n氛围: %s\n社交环境: %s", ctx.Space.Nature, ctx.Space.Vibe, ctx.Space.Social)
	}

	timeText := "未知"
	if ctx.Time != nil {
		timeText = fmt.Sprintf("时间阶段: %s\n生活节奏: %s\n状态持续: %s", ctx.Time.Phase, ctx.Time.Rhythm, ctx.Time.Continuity)
	}

	activityText := "未知"
	if ctx.Activity != nil {
		activityText = fmt.Sprintf("身体状态: %s\n心智状态: %s\n投入程度: %s", ctx.Activity.BodyState, ctx.Activity.MindState, ctx.Activity.Engagement)
	}

	energyText := "未知"
	if ctx.Energy != nil {
		energyText = fmt.Sprintf("身体能量: %s\n精神状态: %s\n社交意愿: %s", ctx.Energy.Physical, ctx.Energy.Mental, ctx.Energy.Social)
	}

	// 格式化记忆
	memoryText := "暂无历史数据，这是一个新用户"
	if memory != nil && memory.SampleCount > 0 {
		memoryText = fmt.Sprintf(`这个人的特点（基于 %d 个历史样本）：
- 行为模式: %s
- 时间规律: %s
- 地点偏好: %s
- 社交倾向: %s`,
			memory.SampleCount,
			nvl(memory.BehaviorInsights, "暂无"),
			nvl(memory.TimePatterns, "暂无"),
			nvl(memory.LocationPreferences, "暂无"),
			nvl(memory.SocialTendency, "暂无"),
		)
	}

	// 格式化异常
	anomalyText := "无特殊异常"
	if len(anomalies) > 0 {
		lines := []string{}
		for _, a := range anomalies {
			lines = append(lines, fmt.Sprintf("⚠️ %s: %s", a.Type, a.Detail))
		}
		anomalyText = strings.Join(lines, "\n")
	}

	// 格式化原始线索（补充）
	rawClueText := h.formatClues(clue)

	return fmt.Sprintf(`你是一个洞察人心的叙事者。

## 你的任务
根据下面的线索，用第三人称描述这个人**此刻**正在做什么、心情如何。
要像写小说一样生动，但要简洁。

## 原始线索

%s

## 语义分析

**空间感**
%s

**时间感**
%s

**活动感**
%s

**能量状态**
%s

## 这个人的特点（来自记忆）

%s

## 异常观察

%s

## 创作要求

1. **不要用预设标签**
   - 不要说"工作中"/"休息中"这种无聊的标签
   - 要有画面感，如"窝在沙发里漫无目的地刷着手机"

2. **要有想象力**
   - 根据线索合理推测具体场景
   - 可以推测心情、想法，但要有依据

3. **简洁有力**
   - scene（场景描述）控制在15-25字
   - 用一个最传神的 emoji

4. **考虑记忆**
   - 如果行为符合习惯，可以更自信
   - 如果偏离习惯，要注意可能有特殊情况

## 输出格式（严格JSON）

{
    "narrative": "你的叙事推理过程（100-200字，像侦探推理一样）",
    "scene": "此刻的场景描述（15-25字，要有画面感）",
    "emoji": "最传神的emoji",
    "mood": {
        "valence": 0.7,
        "arousal": 0.3,
        "openness": 0.5
    },
    "confidence": "high",
    "basis": ["依据1", "依据2", "依据3"]
}

## 字段说明
- narrative: 详细的推理过程，像侦探一样分析每个线索
- scene: 用自然口语化描述**此刻在做什么**，要生动形象
- emoji: 选择最能代表当前状态的 emoji
- mood.valence: 效价，-1(消极) 到 1(积极)
- mood.arousal: 唤醒度，0(平静) 到 1(激动)
- mood.openness: 社交开放度，0(封闭) 到 1(开放)
- confidence: 置信度 high/medium/low
- basis: 列出3个主要推断依据

## Emoji 参考（可自由选择）
🎮 游戏 | 📺 追剧 | 💼 工作 | ☕ 摸鱼
🍜 吃饭 | 🛋️ 躺平 | 🚶 外出 | 😴 睡觉
📱 刷手机 | 💬 聊天 | 🎧 听歌 | 🏃 运动
🍻 聚会 | 🔕 勿扰 | 📊 开会 | 🚇 通勤

只输出 JSON，不要有任何前缀或解释。`,
		rawClueText,
		spaceText,
		timeText,
		activityText,
		energyText,
		memoryText,
		anomalyText,
	)
}

// parseCreativeResponse 解析创意叙事响应
func (h *HolmesAnalyzer) parseCreativeResponse(resp *ThinkingResponse) (*model.HolmesCreativeResult, error) {
	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var llmResp HolmesCreativeLLMResponse
	if err := json.Unmarshal([]byte(content), &llmResp); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w, content: %s", err, content)
	}

	// 使用 reasoning_content 补充 narrative
	narrative := llmResp.Narrative
	if resp.ReasoningContent != "" && narrative == "" {
		narrative = resp.ReasoningContent
	}

	// 验证和修正数据
	if llmResp.Scene == "" {
		llmResp.Scene = "状态未知"
	}
	if llmResp.Emoji == "" {
		llmResp.Emoji = "🤔"
	}
	if llmResp.Confidence == "" {
		llmResp.Confidence = "medium"
	}

	// 限制 mood 范围
	valence := llmResp.Mood.Valence
	if valence < -1 {
		valence = -1
	}
	if valence > 1 {
		valence = 1
	}

	arousal := llmResp.Mood.Arousal
	if arousal < 0 {
		arousal = 0
	}
	if arousal > 1 {
		arousal = 1
	}

	openness := llmResp.Mood.Openness
	if openness < 0 {
		openness = 0
	}
	if openness > 1 {
		openness = 1
	}

	return &model.HolmesCreativeResult{
		Narrative:  narrative,
		Scene:      llmResp.Scene,
		Emoji:      llmResp.Emoji,
		Mood: &model.MoodVector{
			Valence:  valence,
			Arousal:  arousal,
			Openness: openness,
		},
		Confidence:  llmResp.Confidence,
		Basis:       llmResp.Basis,
		GeneratedAt: time.Now().UnixMilli(),
	}, nil
}
