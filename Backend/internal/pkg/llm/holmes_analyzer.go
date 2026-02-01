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

现在需要判断一个人此刻是否有空约朋友。

## 线索（原始数据）

%s

## 特征（提取的关键信息）

%s

## 历史规律

%s

## 推理要求

1. 像侦探一样分析每个线索的含义
2. 综合所有线索进行推断
3. 考虑时间、地点、活动的关联性
4. 给出结论和置信度

## 输出格式（严格 JSON，不要包含任何其他内容）

{
    "reasoning": "你的完整推理过程（不限长度，详细分析）",
    "available": true,
    "probability": 88,
    "confidence": "high",
    "summary": "看起来在咖啡厅边喝咖啡边刷手机摸鱼，应该有空约",
    "emoji": "☕"
}

## 输出要求
- summary: 用自然口语化描述当前状态（20-40字），可以使用"摸鱼"、"刷手机"、"追剧"等真实表达
- emoji: 可以自由选择任何 Unicode emoji，不限于预设列表
- reasoning: 详细的推理过程，不限长度

## 置信度说明
- high: 线索充分，推断可靠
- medium: 部分线索缺失，但主要判断可信
- low: 线索不足，存在较大不确定性

## Emoji 建议（可自由选择任何 Unicode emoji）
🎮 游戏 | 📺 追剧 | 💼 工作 | ☕ 咖啡厅/摸鱼
🍜 吃饭 | 🛋️ 躺平 | 🚶 外出 | 😴 睡觉
📱 刷手机 | 💬 聊天 | 🎧 听歌 | 🏃 运动
🍻 聚会 | 🔕 勿扰 | 📊 开会 | 🚇 通勤
🛍️ 购物 | 🎬 看电影 | 📚 看书 | 🎨 创作
🏋️ 健身 | 🍕 美食 | 🎉 娱乐 | 😊 休闲

你也可以根据具体情况选择其他更合适的 emoji。

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

	// 屏幕线索
	if clue.ScreenActive {
		lines = append(lines, fmt.Sprintf("屏幕：活跃，%s（已使用 %d 分钟）", clue.ActivityType, clue.ScreenDurationMins))
	} else {
		lines = append(lines, fmt.Sprintf("屏幕：闲置（%d 分钟前活跃）", clue.LastActiveMinutesAgo))
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
