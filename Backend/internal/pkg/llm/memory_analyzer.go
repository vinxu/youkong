package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
)

// MemoryAnalyzer 记忆分析器
type MemoryAnalyzer struct {
	client *OpenRouterClient
}

// NewMemoryAnalyzer 创建记忆分析器
func NewMemoryAnalyzer(client *OpenRouterClient) *MemoryAnalyzer {
	return &MemoryAnalyzer{client: client}
}

// AnalysisInput 分析输入
type AnalysisInput struct {
	CurrentStatus  *model.ExtendedStatusReportRequest `json:"current_status"`
	CurrentMemory  *model.CoreMemory                  `json:"current_memory"`
	RecentHistory  []*model.StatusHistory             `json:"recent_history"`
	Timestamp      time.Time                          `json:"timestamp"`
	Weekday        string                             `json:"weekday"`
	Hour           int                                `json:"hour"`
}

// LLMAnalysisResponse LLM 分析结果（JSON 格式）
type LLMAnalysisResponse struct {
	Availability struct {
		Status      string `json:"status"`
		Probability int    `json:"probability"`
		Reason      string `json:"reason"`
		Confidence  string `json:"confidence"`
	} `json:"availability"`
	LifeStatus struct {
		Emoji       string `json:"emoji"`
		Label       string `json:"label"`
		Description string `json:"description"`
	} `json:"life_status"`
	MemoryUpdates struct {
		BehaviorInsights    string `json:"behavior_insights"`
		TimePatterns        string `json:"time_patterns"`
		LocationPreferences string `json:"location_preferences"`
		SocialTendency      string `json:"social_tendency"`
	} `json:"memory_updates"`
	ShouldUpdateMemory bool `json:"should_update_memory"`
}

// Analyze 分析用户状态并返回结果
func (a *MemoryAnalyzer) Analyze(ctx context.Context, input *AnalysisInput) (*model.AnalysisResult, error) {
	if a.client == nil {
		// 无 LLM 客户端，返回基于规则的默认结果
		return a.analyzeWithRules(input), nil
	}

	prompt := a.buildPrompt(input)
	response, err := a.client.Chat(ctx, prompt)
	if err != nil {
		// LLM 调用失败，降级到规则分析
		return a.analyzeWithRules(input), nil
	}

	// 解析 LLM 响应
	result, err := a.parseResponse(response)
	if err != nil {
		// 解析失败，降级到规则分析
		return a.analyzeWithRules(input), nil
	}

	return result, nil
}

// buildPrompt 构建分析 Prompt
func (a *MemoryAnalyzer) buildPrompt(input *AnalysisInput) string {
	// 格式化当前状态
	statusDesc := a.formatStatus(input.CurrentStatus)

	// 格式化历史记忆
	memoryDesc := "暂无历史记忆"
	if input.CurrentMemory != nil && input.CurrentMemory.SampleCount > 0 {
		memoryDesc = fmt.Sprintf(`- 行为模式: %s
- 时间规律: %s
- 地点偏好: %s
- 社交倾向: %s
- 数据置信度: %d%% (基于 %d 个样本)`,
			nvl(input.CurrentMemory.BehaviorInsights, "暂无"),
			nvl(input.CurrentMemory.TimePatterns, "暂无"),
			nvl(input.CurrentMemory.LocationPreferences, "暂无"),
			nvl(input.CurrentMemory.SocialTendency, "暂无"),
			input.CurrentMemory.ConfidenceScore,
			input.CurrentMemory.SampleCount,
		)
	}

	// 格式化最近数据摘要
	recentSummary := "暂无最近数据"
	if len(input.RecentHistory) > 0 {
		recentSummary = fmt.Sprintf("最近 %d 条上报记录", len(input.RecentHistory))
	}

	prompt := fmt.Sprintf(`你是一个用户行为分析专家。根据用户的实时状态数据和历史记忆，分析用户当前状态并更新记忆。

## 当前数据
- 时间：%s（%s，%d点）
%s

## 历史核心记忆
%s

## 最近数据摘要
%s

## 任务
1. **有空分析**：判断用户当前是否有空（有空/忙碌/可能有空），预测概率（0-100）
2. **生活状态**：用一个 emoji 描述用户当前在做什么，要贴近真实生活
3. **记忆更新**：如果发现新的规律或洞察，更新核心记忆

## 输出格式（严格 JSON，不要包含任何其他内容）
{
    "availability": {
        "status": "有空|忙碌|可能有空",
        "probability": 75,
        "reason": "简短理由（15字以内）",
        "confidence": "high|medium|low"
    },
    "life_status": {
        "emoji": "🛋️",
        "label": "在家躺着",
        "description": "看起来很悠闲"
    },
    "memory_updates": {
        "behavior_insights": "更新的行为洞察（如果有，否则为空字符串）",
        "time_patterns": "更新的时间规律（如果有，否则为空字符串）",
        "location_preferences": "更新的地点偏好（如果有，否则为空字符串）",
        "social_tendency": "更新的社交倾向（如果有，否则为空字符串）"
    },
    "should_update_memory": true
}

## Emoji 选择参考
- 🎮 在玩游戏（娱乐+长会话）
- 📺 在追剧（娱乐+在家）
- 💼 在工作（工作类+公司）
- ☕ 在摸鱼（娱乐+公司）
- 🍜 在吃饭（餐点时间+外出）
- 🛋️ 在家躺着（闲置+在家）
- 🚶 在外面逛（移动中）
- 😴 可能在睡觉（不活跃+深夜）
- 📱 在刷手机（娱乐+短会话）
- 💬 在聊天（通讯类）
- 🎧 在听音乐（耳机连接+娱乐）
- 🏃 在运动（移动中+不活跃）
- 🍻 可能在聚会（周末晚+外出）
- 🔕 不想被打扰（专注模式）
- 🪫 电量告急（低电量模式）
- 🤔 状态未知（数据不足）

注意：
- 记忆更新应该是增量式的，保留有价值的历史洞察
- 只有发现新规律时才更新记忆（should_update_memory=true）
- 保持记忆简洁，每个字段不超过100字
- reason 必须简短口语化，不超过15字
- 只输出 JSON，不要有任何前缀或解释`,
		input.Timestamp.Format("2006-01-02 15:04"),
		input.Weekday,
		input.Hour,
		statusDesc,
		memoryDesc,
		recentSummary,
	)

	return prompt
}

// formatStatus 格式化状态数据
func (a *MemoryAnalyzer) formatStatus(status *model.ExtendedStatusReportRequest) string {
	if status == nil {
		return "- 无状态数据"
	}

	var parts []string

	if status.Screen != nil {
		activeStr := "不活跃"
		if status.Screen.IsActive {
			activeStr = "活跃中"
		}
		parts = append(parts, fmt.Sprintf("- 屏幕：%s，%s，会话%d分钟，上次活跃%d分钟前",
			activeStr,
			formatActivityType(status.Screen.ActivityType),
			status.Screen.SessionDurationMinutes,
			status.Screen.LastActiveMinutesAgo,
		))
	}

	if status.Location != nil {
		parts = append(parts, fmt.Sprintf("- 位置：%s，已待%d分钟",
			formatPlaceType(status.Location.PlaceType),
			status.Location.AtPlaceSinceMinutes,
		))
	}

	if status.Battery != nil {
		chargingStr := "未充电"
		if status.Battery.IsCharging {
			chargingStr = "充电中"
		}
		parts = append(parts, fmt.Sprintf("- 电池：%d%%，%s",
			status.Battery.BatteryLevel,
			chargingStr,
		))
	}

	if status.Mode != nil {
		var modes []string
		if status.Mode.IsLowPowerMode {
			modes = append(modes, "低电量模式")
		}
		if status.Mode.IsFocusModeOn {
			modes = append(modes, "专注模式")
		}
		if len(modes) > 0 {
			parts = append(parts, fmt.Sprintf("- 模式：%s", strings.Join(modes, "、")))
		}
	}

	if status.Connection != nil {
		var conns []string
		if status.Connection.IsHeadphonesConnected {
			conns = append(conns, "耳机已连接")
		}
		if status.Connection.NetworkType != "" {
			conns = append(conns, fmt.Sprintf("网络：%s", status.Connection.NetworkType))
		}
		if len(conns) > 0 {
			parts = append(parts, fmt.Sprintf("- 连接：%s", strings.Join(conns, "、")))
		}
	}

	if status.Display != nil {
		parts = append(parts, fmt.Sprintf("- 亮度：%.0f%%", status.Display.ScreenBrightness*100))
	}

	if len(parts) == 0 {
		return "- 数据不完整"
	}

	return strings.Join(parts, "\n")
}

// parseResponse 解析 LLM 响应
func (a *MemoryAnalyzer) parseResponse(response string) (*model.AnalysisResult, error) {
	// 清理响应（移除可能的 markdown 代码块）
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var llmResult LLMAnalysisResponse
	if err := json.Unmarshal([]byte(response), &llmResult); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}

	result := &model.AnalysisResult{
		Availability: model.AvailabilityAnalysis{
			Status:      llmResult.Availability.Status,
			Probability: llmResult.Availability.Probability,
			Reason:      truncateString(llmResult.Availability.Reason, 15),
			Confidence:  llmResult.Availability.Confidence,
		},
		LifeStatus: model.LifeStatus{
			Emoji:       llmResult.LifeStatus.Emoji,
			Label:       llmResult.LifeStatus.Label,
			Description: llmResult.LifeStatus.Description,
		},
	}

	if llmResult.ShouldUpdateMemory {
		result.MemoryUpdate = &model.MemoryUpdate{
			BehaviorInsights:    truncateString(llmResult.MemoryUpdates.BehaviorInsights, 100),
			TimePatterns:        truncateString(llmResult.MemoryUpdates.TimePatterns, 100),
			LocationPreferences: truncateString(llmResult.MemoryUpdates.LocationPreferences, 100),
			SocialTendency:      truncateString(llmResult.MemoryUpdates.SocialTendency, 100),
			ShouldUpdate:        true,
		}
	}

	return result, nil
}

// analyzeWithRules 基于规则的分析（LLM 不可用时的降级方案）
func (a *MemoryAnalyzer) analyzeWithRules(input *AnalysisInput) *model.AnalysisResult {
	result := &model.AnalysisResult{
		Availability: model.AvailabilityAnalysis{
			Status:      "可能有空",
			Probability: 50,
			Reason:      "数据分析中",
			Confidence:  "low",
		},
		LifeStatus: model.LifeStatus{
			Emoji: "🤔",
			Label: "状态未知",
		},
	}

	if input.CurrentStatus == nil {
		return result
	}

	score := 50
	emoji := "🤔"
	label := "状态未知"

	// 分析屏幕状态
	if input.CurrentStatus.Screen != nil {
		screen := input.CurrentStatus.Screen
		if screen.IsActive {
			switch screen.ActivityType {
			case model.ActivityEntertainment:
				if screen.SessionDurationMinutes >= 30 {
					score += 30
					emoji = "🎮"
					label = "在玩游戏"
				} else {
					score += 15
					emoji = "📱"
					label = "在刷手机"
				}
			case model.ActivityProductivity:
				score -= 10
				emoji = "💼"
				label = "在工作"
			case model.ActivityCommunication:
				score += 10
				emoji = "💬"
				label = "在聊天"
			}
		} else {
			if screen.LastActiveMinutesAgo > 120 {
				score -= 25
				if input.Hour >= 23 || input.Hour < 7 {
					emoji = "😴"
					label = "可能在睡觉"
				}
			}
		}
	}

	// 分析位置
	if input.CurrentStatus.Location != nil {
		location := input.CurrentStatus.Location
		switch location.PlaceType {
		case model.PlaceHome:
			score += 15
			if emoji == "🤔" {
				emoji = "🛋️"
				label = "在家躺着"
			}
		case model.PlaceWork:
			if input.Hour >= 9 && input.Hour < 18 {
				score -= 20
				emoji = "💼"
				label = "在工作"
			} else {
				if input.CurrentStatus.Screen != nil && input.CurrentStatus.Screen.ActivityType == model.ActivityEntertainment {
					emoji = "☕"
					label = "在摸鱼"
				}
			}
		case model.PlaceLeisure:
			score += 10
			emoji = "🚶"
			label = "在外面逛"
		}
	}

	// 分析模式
	if input.CurrentStatus.Mode != nil {
		if input.CurrentStatus.Mode.IsFocusModeOn {
			score -= 20
			emoji = "🔕"
			label = "不想被打扰"
		}
		if input.CurrentStatus.Mode.IsLowPowerMode {
			emoji = "🪫"
			label = "电量告急"
		}
	}

	// 分析连接
	if input.CurrentStatus.Connection != nil && input.CurrentStatus.Connection.IsHeadphonesConnected {
		if input.CurrentStatus.Screen != nil && input.CurrentStatus.Screen.ActivityType == model.ActivityEntertainment {
			emoji = "🎧"
			label = "在听音乐"
		}
	}

	// 时间因素
	weekday := int(input.Timestamp.Weekday())
	isWeekend := weekday == 0 || weekday == 6
	if isWeekend {
		score += 15
	} else {
		if input.Hour >= 18 && input.Hour < 22 {
			score += 20
		} else if input.Hour >= 9 && input.Hour < 18 {
			score -= 15
		}
	}

	// 边界处理
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	// 确定状态
	status := "可能有空"
	if score >= 70 {
		status = "有空"
	} else if score < 30 {
		status = "忙碌"
	}

	confidence := "medium"
	if input.CurrentMemory != nil && input.CurrentMemory.SampleCount > 50 {
		confidence = "high"
	} else if input.CurrentMemory == nil || input.CurrentMemory.SampleCount < 10 {
		confidence = "low"
	}

	result.Availability = model.AvailabilityAnalysis{
		Status:      status,
		Probability: score,
		Reason:      getDefaultReason(score),
		Confidence:  confidence,
	}
	result.LifeStatus = model.LifeStatus{
		Emoji: emoji,
		Label: label,
	}

	return result
}

// 辅助函数

func formatActivityType(t model.ActivityType) string {
	switch t {
	case model.ActivityEntertainment:
		return "娱乐"
	case model.ActivityProductivity:
		return "工作"
	case model.ActivityCommunication:
		return "通讯"
	case model.ActivityIdle:
		return "闲置"
	default:
		return "未知"
	}
}

func formatPlaceType(t model.PlaceType) string {
	switch t {
	case model.PlaceHome:
		return "家"
	case model.PlaceWork:
		return "公司"
	case model.PlaceLeisure:
		return "休闲场所"
	case model.PlaceTransit:
		return "路上"
	default:
		return "未知"
	}
}

func nvl(s, defaultVal string) string {
	if s == "" {
		return defaultVal
	}
	return s
}

func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// getDefaultReason 获取默认理由
func getDefaultReason(score int) string {
	if score >= 80 {
		return "可能有空"
	}
	if score >= 60 {
		return "应该有空"
	}
	if score >= 40 {
		return "不太确定"
	}
	if score >= 20 {
		return "可能在忙"
	}
	return "应该在忙"
}
