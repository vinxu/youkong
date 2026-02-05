package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
)

// StatusInferrer 当下状态推理器
type StatusInferrer struct {
	client *OpenRouterClient
}

// NewStatusInferrer 创建状态推理器
func NewStatusInferrer(client *OpenRouterClient) *StatusInferrer {
	return &StatusInferrer{client: client}
}

// StatusInferInput 推理输入
type StatusInferInput struct {
	SensorData   *model.ExtendedStatusReportRequest `json:"sensor_data"`   // 传感器数据
	UserProfile  *model.UserProfileData             `json:"user_profile"`  // 用户画像
	CoreMemory   *model.CoreMemory                  `json:"core_memory"`   // 核心记忆
	StatusMemory []*model.StatusMemoryEntry         `json:"status_memory"` // 状态修正记忆
	Timestamp    time.Time                          `json:"timestamp"`
	Weekday      string                             `json:"weekday"`
	Hour         int                                `json:"hour"`
}

// LLMStatusInferResponse LLM 推理结果
type LLMStatusInferResponse struct {
	Emoji        string `json:"emoji"`
	Place        string `json:"place,omitempty"`
	Activity     string `json:"activity"`
	IsAvailable  bool   `json:"is_available"`
	DurationHint string `json:"duration_hint"`
	Confidence   string `json:"confidence"`
	Reasoning    string `json:"reasoning"`
}

// Infer 推理当下状态
func (s *StatusInferrer) Infer(ctx context.Context, input *StatusInferInput) (*model.CurrentStatusInference, error) {
	if s.client == nil {
		return s.inferWithRules(input), nil
	}

	prompt := s.buildPrompt(input)
	response, err := s.client.Chat(ctx, prompt)
	if err != nil {
		return s.inferWithRules(input), nil
	}

	result, err := s.parseResponse(response)
	if err != nil {
		return s.inferWithRules(input), nil
	}

	return result, nil
}

// buildPrompt 构建推理 Prompt
func (s *StatusInferrer) buildPrompt(input *StatusInferInput) string {
	// 格式化传感器数据
	sensorDesc := s.formatSensorData(input.SensorData)

	// 格式化用户画像
	profileDesc := "暂无用户画像"
	if input.UserProfile != nil {
		profileDesc = fmt.Sprintf("- 职业：%s\n- 工作时间：%s\n- 生活方式：%s",
			model.GetProfileTypeName(string(input.UserProfile.OccupationType)),
			formatWorkSchedule(input.UserProfile.WorkSchedule),
			formatLifestyle(input.UserProfile.LifestyleType),
		)
	}

	// 格式化核心记忆
	memoryDesc := "暂无历史记忆"
	if input.CoreMemory != nil && input.CoreMemory.SampleCount > 0 {
		memoryDesc = fmt.Sprintf(`- 行为模式: %s
- 时间规律: %s
- 地点偏好: %s`,
			nvl(input.CoreMemory.BehaviorInsights, "暂无"),
			nvl(input.CoreMemory.TimePatterns, "暂无"),
			nvl(input.CoreMemory.LocationPreferences, "暂无"),
		)
	}

	// 格式化状态修正记忆（用户曾经修正过的状态）
	correctionDesc := "暂无修正记录"
	if len(input.StatusMemory) > 0 {
		var corrections []string
		for i, m := range input.StatusMemory {
			if i >= 5 { // 最多显示5条
				break
			}
			corrections = append(corrections, fmt.Sprintf("- %s %s → %s %s",
				m.OriginalEmoji, m.OriginalActivity,
				m.CorrectedEmoji, m.CorrectedActivity))
		}
		correctionDesc = strings.Join(corrections, "\n")
	}

	prompt := fmt.Sprintf(`你是一个状态推理助手。根据以下信息推断用户当前的状态。

## 当前时间
%s（%s，%d点）

## 传感器数据
%s

## 用户画像
%s

## 历史规律
%s

## 用户曾修正过的状态（参考）
%s

## 输出要求
请推断用户当前状态，返回 JSON：
{
  "emoji": "1-3个emoji表达状态",
  "place": "场所（可选，没有明确场所时返回空字符串）",
  "activity": "简短活动描述（2-6字）",
  "is_available": true/false,
  "duration_hint": "预计持续时长",
  "confidence": "high/medium/low",
  "reasoning": "推理依据（简短）"
}

## 注意事项
1. emoji 可以是1-3个，组合表达更丰富（如"🎮🎧"表示戴耳机打游戏）
2. place 和 activity 要分开：有些状态不适合带场所（如"睡觉"）
3. is_available=true 仅当用户明确处于"有空等待约"的状态
   - 在家休闲、刷手机、无事可做 → 可能有空
   - 工作、开会、运动、睡觉、专注 → 没空
4. duration_hint 是估算值，如"约1小时"、"下午"、"一整天"
5. 参考用户曾修正过的状态，类似情况应给出类似结果

只输出 JSON，不要有任何前缀或解释。`,
		input.Timestamp.Format("2006-01-02 15:04"),
		input.Weekday,
		input.Hour,
		sensorDesc,
		profileDesc,
		memoryDesc,
		correctionDesc,
	)

	return prompt
}

// formatSensorData 格式化传感器数据
func (s *StatusInferrer) formatSensorData(data *model.ExtendedStatusReportRequest) string {
	if data == nil {
		return "- 无传感器数据"
	}

	var parts []string

	if data.Screen != nil {
		activeStr := "不活跃"
		if data.Screen.IsActive {
			activeStr = "活跃中"
		}
		parts = append(parts, fmt.Sprintf("- 屏幕：%s，%s，会话%d分钟，上次活跃%d分钟前",
			activeStr,
			formatActivityType(data.Screen.ActivityType),
			data.Screen.SessionDurationMinutes,
			data.Screen.LastActiveMinutesAgo,
		))
	}

	if data.Location != nil {
		parts = append(parts, fmt.Sprintf("- 位置：%s，已待%d分钟",
			formatPlaceType(data.Location.PlaceType),
			data.Location.AtPlaceSinceMinutes,
		))
	}

	if data.Battery != nil {
		chargingStr := "未充电"
		if data.Battery.IsCharging {
			chargingStr = "充电中"
		}
		parts = append(parts, fmt.Sprintf("- 电池：%d%%，%s",
			data.Battery.BatteryLevel,
			chargingStr,
		))
	}

	if data.Mode != nil {
		var modes []string
		if data.Mode.IsLowPowerMode {
			modes = append(modes, "低电量模式")
		}
		if data.Mode.IsFocusModeOn {
			modes = append(modes, "专注模式")
		}
		if len(modes) > 0 {
			parts = append(parts, fmt.Sprintf("- 模式：%s", strings.Join(modes, "、")))
		}
	}

	if data.Connection != nil {
		var conns []string
		if data.Connection.IsHeadphonesConnected {
			conns = append(conns, "耳机已连接")
		}
		if data.Connection.NetworkType != "" {
			conns = append(conns, fmt.Sprintf("网络：%s", data.Connection.NetworkType))
		}
		if len(conns) > 0 {
			parts = append(parts, fmt.Sprintf("- 连接：%s", strings.Join(conns, "、")))
		}
	}

	if data.Movement != nil {
		moveStr := "静止"
		if data.Movement.IsMoving {
			moveStr = "移动中"
		}
		parts = append(parts, fmt.Sprintf("- 运动：%s，类型：%s",
			moveStr,
			data.Movement.MovementType,
		))
	}

	if data.Calendar != nil {
		if data.Calendar.HasCurrentEvent {
			parts = append(parts, fmt.Sprintf("- 日历：当前有日程，今日还剩%d个",
				data.Calendar.TodayRemainingCount))
		} else {
			parts = append(parts, fmt.Sprintf("- 日历：当前无日程，今日还剩%d个",
				data.Calendar.TodayRemainingCount))
		}
	}

	if len(parts) == 0 {
		return "- 数据不完整"
	}

	return strings.Join(parts, "\n")
}

// parseResponse 解析 LLM 响应
func (s *StatusInferrer) parseResponse(response string) (*model.CurrentStatusInference, error) {
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var llmResult LLMStatusInferResponse
	if err := json.Unmarshal([]byte(response), &llmResult); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}

	return &model.CurrentStatusInference{
		Emoji:        llmResult.Emoji,
		Place:        llmResult.Place,
		Activity:     llmResult.Activity,
		IsAvailable:  llmResult.IsAvailable,
		DurationHint: llmResult.DurationHint,
		Confidence:   llmResult.Confidence,
		InferredAt:   time.Now().UnixMilli(),
		Reasoning:    llmResult.Reasoning,
	}, nil
}

// formatWorkSchedule 格式化工作时间
func formatWorkSchedule(ws model.WorkSchedule) string {
	switch ws {
	case model.WorkScheduleRegular9to5:
		return "朝九晚五"
	case model.WorkScheduleFlexible:
		return "弹性工作"
	case model.WorkScheduleShift:
		return "轮班制"
	case model.WorkScheduleIrregular:
		return "不规律"
	case model.WorkScheduleNone:
		return "无固定工作时间"
	default:
		return "未知"
	}
}

// formatLifestyle 格式化生活方式
func formatLifestyle(lt model.LifestyleType) string {
	switch lt {
	case model.LifestyleEarlyBird:
		return "早起型"
	case model.LifestyleNightOwl:
		return "夜猫子"
	case model.LifestyleBalanced:
		return "平衡型"
	default:
		return "未知"
	}
}

// inferWithRules 基于规则的推理（LLM 不可用时的降级方案）
func (s *StatusInferrer) inferWithRules(input *StatusInferInput) *model.CurrentStatusInference {
	result := &model.CurrentStatusInference{
		Emoji:        "🤔",
		Activity:     "状态未知",
		IsAvailable:  false,
		DurationHint: "未知",
		Confidence:   "low",
		InferredAt:   time.Now().UnixMilli(),
	}

	if input.SensorData == nil {
		return result
	}

	// 基于规则推断
	emoji := "🤔"
	activity := "状态未知"
	place := ""
	isAvailable := false
	durationHint := "一段时间"

	// 分析屏幕状态
	if input.SensorData.Screen != nil {
		screen := input.SensorData.Screen
		if screen.IsActive {
			switch screen.ActivityType {
			case model.ActivityEntertainment:
				if screen.SessionDurationMinutes >= 30 {
					emoji = "🎮"
					activity = "打游戏"
					isAvailable = false
				} else {
					emoji = "📱"
					activity = "刷手机"
					isAvailable = true
				}
			case model.ActivityProductivity:
				emoji = "💼"
				activity = "工作中"
				isAvailable = false
			case model.ActivityCommunication:
				emoji = "💬"
				activity = "聊天中"
				isAvailable = true
			}
		} else {
			if screen.LastActiveMinutesAgo > 60 {
				if input.Hour >= 23 || input.Hour < 7 {
					emoji = "😴"
					activity = "睡觉中"
					isAvailable = false
				} else {
					emoji = "🚶"
					activity = "外出中"
					isAvailable = false
				}
			}
		}
	}

	// 分析位置
	if input.SensorData.Location != nil {
		location := input.SensorData.Location
		switch location.PlaceType {
		case model.PlaceHome:
			place = "家里"
			if emoji == "🤔" {
				emoji = "🛋️"
				activity = "在家休息"
				isAvailable = true
			}
		case model.PlaceWork:
			place = "公司"
			if input.Hour >= 9 && input.Hour < 18 {
				emoji = "💼"
				activity = "上班中"
				isAvailable = false
			}
		case model.PlaceLeisure:
			place = "外面"
			emoji = "🚶"
			activity = "外出中"
		}
	}

	// 分析模式
	if input.SensorData.Mode != nil {
		if input.SensorData.Mode.IsFocusModeOn {
			emoji = "🔕"
			activity = "专注中"
			isAvailable = false
		}
	}

	// 分析连接
	if input.SensorData.Connection != nil && input.SensorData.Connection.IsHeadphonesConnected {
		if input.SensorData.Screen != nil && input.SensorData.Screen.ActivityType == model.ActivityEntertainment {
			emoji = "🎧"
			activity = "听音乐"
		}
	}

	// 估算持续时长
	if input.SensorData.Location != nil {
		mins := input.SensorData.Location.AtPlaceSinceMinutes
		if mins < 30 {
			durationHint = "刚到不久"
		} else if mins < 120 {
			durationHint = "约1-2小时"
		} else {
			durationHint = "较长时间"
		}
	}

	result.Emoji = emoji
	result.Activity = activity
	result.Place = place
	result.IsAvailable = isAvailable
	result.DurationHint = durationHint
	result.Confidence = "medium"

	return result
}
