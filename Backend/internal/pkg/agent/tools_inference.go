package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

// InferenceToolDeps 推断工具依赖
type InferenceToolDeps struct {
	RedisClient  *tencent.RedisClient
	MemoryRepo   *repository.MemoryRepository
	ScheduleRepo *repository.ScheduleRepository
	UserID       string

	// 用户确认回调：发送问题到前端，等待用户回答
	// 返回用户的回答字符串，超时返回空字符串
	AskUserFunc func(ctx context.Context, question string, options []string, contextMsg string) (string, error)

	// 保存最终结果的回调
	SaveResultFunc func(ctx context.Context, result *model.CurrentStatusInference) error
}

// InferenceTools 返回推断专用工具列表
func InferenceTools(deps *InferenceToolDeps) []*Tool {
	return []*Tool{
		inferQuerySensorDataTool(deps),
		inferQueryStatusHistoryTool(deps),
		inferQueryScheduleContextTool(deps),
		inferAskUserConfirmationTool(deps),
		inferFinalizeStatusTool(deps),
	}
}

// ========== 工具 1: query_sensor_data ==========

func inferQuerySensorDataTool(deps *InferenceToolDeps) *Tool {
	return &Tool{
		Name: "query_sensor_data",
		Description: `查询用户当前的传感器数据（结构化 JSON 输出）。

返回数据包含：
- screen: 屏幕使用数据（is_active, activity_type, session_duration_minutes, last_active_minutes_ago）
- location: 位置数据（place_type, place_name, at_place_since_minutes, city）
- calendar: 日历数据（has_current_event, current_event_title, event_end_minutes, next_event_in_minutes, today_remaining_count）
- movement: 运动数据（is_moving, movement_type, steps_today, stationary_minutes）
- device: 设备状态（battery_level, is_charging, is_low_power_mode, is_focus_mode_on, is_headphones_connected, network_type）

data_type 参数可以过滤返回的数据维度。`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"data_type": {
					Type:        "string",
					Description: "数据类型：all(全部)、location(位置)、screen(屏幕)、calendar(日历)、movement(运动)、device(设备)",
					Enum:        []string{"all", "location", "screen", "calendar", "movement", "device"},
				},
			},
			Required: []string{"data_type"},
		},
		Handler: createQuerySensorDataHandler(deps),
	}
}

func createQuerySensorDataHandler(deps *InferenceToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		dataType, _ := args["data_type"].(string)
		if dataType == "" {
			dataType = "all"
		}

		// 从 Redis 获取传感器数据
		sensorData := getCachedSensorForInference(ctx, deps.RedisClient, deps.UserID)
		if sensorData == nil {
			return &ToolResult{
				Success: true,
				Data:    map[string]string{"message": "无传感器数据，用户可能未上报或数据已过期"},
			}, nil
		}

		// 根据 data_type 过滤
		result := make(map[string]interface{})

		if dataType == "all" || dataType == "screen" {
			if sensorData.Screen != nil {
				result["screen"] = sensorData.Screen
			}
		}
		if dataType == "all" || dataType == "location" {
			loc := map[string]interface{}{}
			if sensorData.Location != nil {
				loc["place_type"] = sensorData.Location.PlaceType
				loc["at_place_since_minutes"] = sensorData.Location.AtPlaceSinceMinutes
				if sensorData.Location.City != "" {
					loc["city"] = sensorData.Location.City
				}
			}
			if sensorData.ExtendedLocation != nil {
				loc["place_type"] = sensorData.ExtendedLocation.PlaceType
				loc["at_place_since_minutes"] = sensorData.ExtendedLocation.AtPlaceSinceMinutes
				if sensorData.ExtendedLocation.PlaceName != "" {
					loc["place_name"] = sensorData.ExtendedLocation.PlaceName
				}
			}
			if len(loc) > 0 {
				result["location"] = loc
			}
		}
		if dataType == "all" || dataType == "calendar" {
			if sensorData.Calendar != nil {
				result["calendar"] = sensorData.Calendar
			}
		}
		if dataType == "all" || dataType == "movement" {
			if sensorData.Movement != nil {
				result["movement"] = sensorData.Movement
			}
		}
		if dataType == "all" || dataType == "device" {
			device := map[string]interface{}{}
			if sensorData.Battery != nil {
				device["battery_level"] = sensorData.Battery.BatteryLevel
				device["is_charging"] = sensorData.Battery.IsCharging
			}
			if sensorData.Mode != nil {
				device["is_low_power_mode"] = sensorData.Mode.IsLowPowerMode
				device["is_focus_mode_on"] = sensorData.Mode.IsFocusModeOn
			}
			if sensorData.Connection != nil {
				device["is_headphones_connected"] = sensorData.Connection.IsHeadphonesConnected
				device["network_type"] = sensorData.Connection.NetworkType
			}
			if len(device) > 0 {
				result["device"] = device
			}
		}

		if len(result) == 0 {
			return &ToolResult{
				Success: true,
				Data:    map[string]string{"message": "请求的数据维度无数据"},
			}, nil
		}

		return &ToolResult{
			Success:       true,
			Data:          result,
			TokenEstimate: estimateTokens(result),
		}, nil
	}
}

// ========== 工具 2: query_status_history ==========

func inferQueryStatusHistoryTool(deps *InferenceToolDeps) *Tool {
	return &Tool{
		Name: "query_status_history",
		Description: `查询用户的历史状态规律，用于辅助推断。

query_type 选项：
- recent: 最近的状态记录（用户选择或修正过的状态）
- corrections: 用户修正过的状态映射（AI 推断 → 用户修正，最重要的参考）`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"query_type": {
					Type:        "string",
					Description: "查询类型：recent(最近状态)、corrections(修正记录)",
					Enum:        []string{"recent", "corrections"},
				},
				"limit": {
					Type:        "number",
					Description: "返回数量限制，默认 10",
					Default:     10,
				},
			},
			Required: []string{"query_type"},
		},
		Handler: createQueryStatusHistoryHandler(deps),
	}
}

func createQueryStatusHistoryHandler(deps *InferenceToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		queryType, _ := args["query_type"].(string)
		limit := 10
		if l, ok := args["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}

		switch queryType {
		case "recent":
			// 查询最近的用户选择状态记忆
			if deps.MemoryRepo == nil {
				return &ToolResult{Success: true, Data: map[string]string{"message": "记忆服务未配置"}}, nil
			}
			memories, err := deps.MemoryRepo.GetRecentUserStatusMemory(ctx, deps.UserID, limit)
			if err != nil {
				return &ToolResult{Success: false, Error: fmt.Sprintf("查询失败: %v", err)}, nil
			}
			if len(memories) == 0 {
				return &ToolResult{Success: true, Data: map[string]string{"message": "无历史状态记录（新用户）"}}, nil
			}
			// 格式化输出
			entries := make([]map[string]string, 0, len(memories))
			for _, m := range memories {
				entries = append(entries, map[string]string{
					"emoji":      m.Emoji,
					"status":     m.Status,
					"created_at": m.CreatedAt.Format("01-02 15:04"),
				})
			}
			return &ToolResult{Success: true, Data: entries, TokenEstimate: estimateTokens(entries)}, nil

		case "corrections":
			// 查询用户修正记录
			key := fmt.Sprintf("agent:status_feedback:%s", deps.UserID)
			data, err := deps.RedisClient.GetBytes(ctx, key)
			if err != nil || len(data) == 0 {
				return &ToolResult{Success: true, Data: map[string]string{"message": "无修正记录"}}, nil
			}
			var corrections []*model.StatusMemoryEntry
			if err := json.Unmarshal(data, &corrections); err != nil {
				return &ToolResult{Success: false, Error: "解析修正记录失败"}, nil
			}
			if len(corrections) > limit {
				corrections = corrections[:limit]
			}
			// 格式化
			entries := make([]map[string]string, 0, len(corrections))
			for _, c := range corrections {
				entry := map[string]string{
					"corrected_emoji":    c.CorrectedEmoji,
					"corrected_activity": c.CorrectedActivity,
					"created_at":         c.CreatedAt,
				}
				if c.OriginalEmoji != "" {
					entry["original_emoji"] = c.OriginalEmoji
					entry["original_activity"] = c.OriginalActivity
				}
				if c.CorrectedPlace != "" {
					entry["corrected_place"] = c.CorrectedPlace
				}
				entries = append(entries, entry)
			}
			return &ToolResult{Success: true, Data: entries, TokenEstimate: estimateTokens(entries)}, nil

		default:
			return &ToolResult{Success: false, Error: "无效的 query_type"}, nil
		}
	}
}

// ========== 工具 3: query_schedule_context ==========

func inferQueryScheduleContextTool(deps *InferenceToolDeps) *Tool {
	return &Tool{
		Name: "query_schedule_context",
		Description: `查询用户的时刻表上下文。
返回当前时间段是否有时刻表安排、当前项和下一项的内容。
时刻表是用户自己设定或 AI 推测的一天状态时间线。`,
		Parameters: ToolParameters{
			Type:       "object",
			Properties: map[string]ToolParam{},
		},
		Handler: createQueryScheduleContextHandler(deps),
	}
}

func createQueryScheduleContextHandler(deps *InferenceToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		if deps.ScheduleRepo == nil {
			return &ToolResult{Success: true, Data: map[string]string{"message": "时刻表服务未配置"}}, nil
		}

		now := time.Now()
		currentTime := now.Format("15:04")
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		schedule, err := deps.ScheduleRepo.GetActiveByUserAndDate(ctx, deps.UserID, today)
		if err != nil || schedule == nil || len(schedule.Items) == 0 {
			return &ToolResult{
				Success: true,
				Data: map[string]interface{}{
					"has_schedule": false,
					"message":      "今天没有时刻表",
				},
			}, nil
		}

		result := map[string]interface{}{
			"has_schedule": true,
			"total_items":  len(schedule.Items),
		}

		// 找当前时段和下一个时段
		for i, item := range schedule.Items {
			inRange := false
			if item.EndTime < item.StartTime {
				// 跨午夜
				inRange = currentTime >= item.StartTime || currentTime < item.EndTime
			} else {
				inRange = currentTime >= item.StartTime && currentTime < item.EndTime
			}

			if inRange {
				result["current_item"] = map[string]interface{}{
					"start_time":  item.StartTime,
					"end_time":    item.EndTime,
					"emoji":       item.Emoji,
					"status":      item.Status,
					"is_ai_guess": item.IsAIGuess,
				}
				// 下一项
				if i+1 < len(schedule.Items) {
					next := schedule.Items[i+1]
					result["next_item"] = map[string]interface{}{
						"start_time": next.StartTime,
						"end_time":   next.EndTime,
						"emoji":      next.Emoji,
						"status":     next.Status,
					}
				}
				break
			}
		}

		if _, ok := result["current_item"]; !ok {
			result["message"] = "当前时间不在任何时段内"
		}

		return &ToolResult{
			Success:       true,
			Data:          result,
			TokenEstimate: estimateTokens(result),
		}, nil
	}
}

// ========== 工具 4: ask_user_confirmation ==========

func inferAskUserConfirmationTool(deps *InferenceToolDeps) *Tool {
	return &Tool{
		Name: "ask_user_confirmation",
		Description: `向用户提问以确认当前状态。仅在以下情况使用：
- 传感器数据互相矛盾
- 置信度很低（无传感器数据、新用户）
- 一次推断最多问 1 个问题

问题要简短具体，最多给 3 个选项。`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"question": {
					Type:        "string",
					Description: "问用户的问题，简短具体，如「你现在是在工作吗？」",
				},
				"options": {
					Type:        "array",
					Description: "选项列表，最多3个，如[\"是的，在工作\", \"不是，在休息\", \"其他\"]",
					Items:       &ToolParam{Type: "string"},
				},
				"context": {
					Type:        "string",
					Description: "为什么要问（展示给用户看的上下文说明）",
				},
			},
			Required: []string{"question", "options"},
		},
		Handler: createAskUserConfirmationHandler(deps),
	}
}

func createAskUserConfirmationHandler(deps *InferenceToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		question, _ := args["question"].(string)
		contextMsg, _ := args["context"].(string)

		var options []string
		if opts, ok := args["options"].([]interface{}); ok {
			for _, opt := range opts {
				if s, ok := opt.(string); ok {
					options = append(options, s)
				}
			}
		}

		if question == "" {
			return &ToolResult{Success: false, Error: "question 不能为空"}, nil
		}

		// 如果没有配置 AskUserFunc，返回超时提示
		if deps.AskUserFunc == nil {
			return &ToolResult{
				Success: true,
				Data: map[string]interface{}{
					"answer":   "",
					"timeout":  true,
					"message":  "用户确认功能未启用，请基于现有数据做最佳推测",
				},
			}, nil
		}

		// 调用用户确认回调
		answer, err := deps.AskUserFunc(ctx, question, options, contextMsg)
		if err != nil {
			return &ToolResult{
				Success: true,
				Data: map[string]interface{}{
					"answer":  "",
					"timeout": true,
					"message": "等待用户回答超时，请基于现有数据做最佳推测",
				},
			}, nil
		}

		return &ToolResult{
			Success: true,
			Data: map[string]interface{}{
				"answer":  answer,
				"timeout": false,
			},
		}, nil
	}
}

// ========== 工具 5: finalize_status ==========

func inferFinalizeStatusTool(deps *InferenceToolDeps) *Tool {
	return &Tool{
		Name: "finalize_status",
		Description: `输出最终推断结果。必须通过此工具输出结果。

参数说明：
- emoji: 1-3个emoji组合，如"🎮"或"🏠🛋️"
- activity: 2-6字活动描述，如"打游戏"、"在家休息"
- place: 可选，地点描述，如"家里"、"公司"
- is_available: 是否有空（true=有空/闲/可约, false=忙碌/不方便打扰）
- confidence: 置信度 high/medium/low
- duration_hint: 可选，预计持续时长，如"约2小时"
- reasoning: 推理依据简述`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"emoji": {
					Type:        "string",
					Description: "1-3个emoji",
				},
				"activity": {
					Type:        "string",
					Description: "活动描述，2-6字",
				},
				"place": {
					Type:        "string",
					Description: "地点（可选）",
				},
				"is_available": {
					Type:        "boolean",
					Description: "是否有空",
				},
				"confidence": {
					Type:        "string",
					Description: "置信度",
					Enum:        []string{"high", "medium", "low"},
				},
				"duration_hint": {
					Type:        "string",
					Description: "预计持续时长（可选）",
				},
				"reasoning": {
					Type:        "string",
					Description: "推理依据",
				},
			},
			Required: []string{"emoji", "activity", "is_available", "confidence"},
		},
		Handler: createFinalizeStatusHandler(deps),
	}
}

func createFinalizeStatusHandler(deps *InferenceToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		emoji, _ := args["emoji"].(string)
		activity, _ := args["activity"].(string)
		place, _ := args["place"].(string)
		isAvailable, _ := args["is_available"].(bool)
		confidence, _ := args["confidence"].(string)
		durationHint, _ := args["duration_hint"].(string)
		reasoning, _ := args["reasoning"].(string)

		if emoji == "" || activity == "" {
			return &ToolResult{Success: false, Error: "emoji 和 activity 不能为空"}, nil
		}
		if confidence == "" {
			confidence = "medium"
		}

		result := &model.CurrentStatusInference{
			Emoji:        emoji,
			Activity:     activity,
			Place:        place,
			IsAvailable:  isAvailable,
			DurationHint: durationHint,
			Confidence:   confidence,
			Reasoning:    reasoning,
			InferredAt:   time.Now().UnixMilli(),
		}

		// 保存结果
		if deps.SaveResultFunc != nil {
			if err := deps.SaveResultFunc(ctx, result); err != nil {
				fmt.Printf("[finalize_status] 保存结果失败: %v\n", err)
			}
		}

		return &ToolResult{
			Success: true,
			Data: map[string]interface{}{
				"status":  "finalized",
				"result":  result,
				"message": "推断结果已保存",
			},
		}, nil
	}
}

// ========== 辅助函数 ==========

// getCachedSensorForInference 从 Redis 获取推断用的传感器数据
func getCachedSensorForInference(ctx context.Context, redisClient *tencent.RedisClient, userID string) *model.ExtendedStatusReportRequest {
	if redisClient == nil {
		return nil
	}

	// 先尝试获取扩展数据
	extKey := fmt.Sprintf("agent:extended:%s", userID)
	data, err := redisClient.GetBytes(ctx, extKey)
	if err == nil && len(data) > 0 {
		var req model.ExtendedStatusReportRequest
		if json.Unmarshal(data, &req) == nil {
			return &req
		}
	}

	// 降级到基础状态数据
	key := fmt.Sprintf("agent:status:%s", userID)
	data, err = redisClient.GetBytes(ctx, key)
	if err != nil || len(data) == 0 {
		return nil
	}

	var status model.UserRealtimeStatus
	if json.Unmarshal(data, &status) != nil {
		return nil
	}

	return &model.ExtendedStatusReportRequest{
		Screen:   &status.Screen,
		Location: &status.Location,
	}
}
