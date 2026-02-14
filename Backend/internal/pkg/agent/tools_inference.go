package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

// LocationClassifier 位置分类接口（避免循环依赖）
type LocationClassifier interface {
	ClassifyLocation(ctx context.Context, userID string, lat, lng float64) interface{}
	RecordVisit(ctx context.Context, userID string, lat, lng float64, placeName string)
}

// InferenceToolDeps 推断工具依赖
type InferenceToolDeps struct {
	RedisClient        *tencent.RedisClient
	MemoryRepo         *repository.MemoryRepository
	ScheduleRepo       *repository.ScheduleRepository
	UserProfileService UserProfileProvider
	LocationService    LocationClassifier // Phase 4: 服务端位置聚类（可选）
	MemoryDocRepo      *repository.UserMemoryDocumentRepository
	UserID             string
}

// UserProfileProvider 用户画像查询接口（避免循环依赖）
type UserProfileProvider interface {
	GetProfile(userID string) (*model.UserProfileData, error)
}

// InferenceTools 返回 V3 推断工具列表（2 个输出工具）
func InferenceTools(deps *InferenceToolDeps) []*Tool {
	return []*Tool{
		finalizeStatusTool(deps),
		askUserChoiceTool(deps),
	}
}

// ========== 工具 1: finalize_status ==========

func finalizeStatusTool(deps *InferenceToolDeps) *Tool {
	return &Tool{
		Name: "finalize_status",
		Description: `Use this tool to output the final status inference. This is the DEFAULT and PREFERRED tool — use it in the vast majority of cases.

Use this tool when:
- You have any real-time sensor signal (screen, location, movement, calendar) to base your inference on.
- Historical records differ from current signals — trust current signals and finalize.
- You are fairly confident, or even if confidence is low — just set confidence="low".

IMPORTANT: Call exactly once per inference. Base your inference primarily on real-time device signals, not historical records.`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"emoji": {
					Type:        "string",
					Description: "1-3个emoji组合",
				},
				"activity": {
					Type:        "string",
					Description: "活动描述，2-6字口语化",
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
				"reasoning": {
					Type:        "string",
					Description: "推理依据，一句话",
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
		reasoning, _ := args["reasoning"].(string)

		if emoji == "" || activity == "" {
			return &ToolResult{Success: false, Error: "emoji 和 activity 不能为空"}, nil
		}
		if confidence == "" {
			confidence = "medium"
		}

		result := &model.CurrentStatusInference{
			Emoji:       emoji,
			Activity:    activity,
			Place:       place,
			IsAvailable: isAvailable,
			Confidence:  confidence,
			Reasoning:   reasoning,
			InferredAt:  time.Now().UnixMilli(),
		}

		return &ToolResult{
			Success: true,
			Data: map[string]interface{}{
				"status":  "finalized",
				"result":  result,
				"message": "推断结果已生成",
			},
			ShouldStop: true,
		}, nil
	}
}

// ========== 工具 2: ask_user_choice ==========

func askUserChoiceTool(deps *InferenceToolDeps) *Tool {
	return &Tool{
		Name: "ask_user_choice",
		Description: `Use this tool ONLY when real-time sensor signals directly contradict each other and there is genuinely no way to determine the most likely state.

Do NOT use when:
- You can make a reasonable inference from the signals — use finalize_status instead.
- Historical records conflict with current signals — trust current signals and use finalize_status.
- You have any screen or location data — that is usually enough to finalize.
- You are uncertain but one option is more likely than others — just finalize with lower confidence.

IMPORTANT: This tool should be used in less than 10% of cases. When in doubt, use finalize_status.`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"question": {
					Type:        "string",
					Description: "向用户提问的简短问题，如'你现在在做什么？'",
				},
				"options": {
					Type:        "array",
					Description: "2-3个选项，每个选项是 {emoji, activity, reason} 对象",
					Items: &ToolParam{
						Type:        "object",
						Description: "选项对象，含 emoji(string)、activity(string)、reason(string)",
					},
				},
				"default_index": {
					Type:        "integer",
					Description: "默认选中的选项索引(0-based)，最可能的排第一",
				},
			},
			Required: []string{"question", "options"},
		},
		Handler: createAskUserChoiceHandler(deps),
	}
}

func createAskUserChoiceHandler(deps *InferenceToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		question, _ := args["question"].(string)
		if question == "" {
			question = "你现在在做什么？"
		}

		defaultIdx := 0
		if idx, ok := args["default_index"].(float64); ok {
			defaultIdx = int(idx)
		}

		// 解析 options 数组
		optionsRaw, _ := args["options"].([]interface{})
		if len(optionsRaw) < 2 {
			return &ToolResult{Success: false, Error: "至少需要2个选项"}, nil
		}

		options := make([]model.InferenceOption, 0, len(optionsRaw))
		for i, opt := range optionsRaw {
			optMap, ok := opt.(map[string]interface{})
			if !ok {
				continue
			}
			emoji, _ := optMap["emoji"].(string)
			activity, _ := optMap["activity"].(string)
			reason, _ := optMap["reason"].(string)
			if emoji == "" || activity == "" {
				continue
			}
			options = append(options, model.InferenceOption{
				Index:    i,
				Emoji:    emoji,
				Activity: activity,
				Reason:   reason,
			})
		}

		if len(options) < 2 {
			return &ToolResult{Success: false, Error: "有效选项不足2个"}, nil
		}

		return &ToolResult{
			Success: true,
			Data: map[string]interface{}{
				"status":        "awaiting_choice",
				"question":      question,
				"options":       options,
				"default_index": defaultIdx,
			},
			ShouldStop: true,
		}, nil
	}
}

// ========== 预聚合: Go 代码收集所有数据源 ==========

// PreGatherContext 预聚合所有推断数据源（Go 代码执行，~50ms）
func PreGatherContext(ctx context.Context, deps *InferenceToolDeps) *model.AgentInferenceContext {
	ic := &model.AgentInferenceContext{}

	// 1. 设备信号（Redis 缓存）
	ic.DeviceSignals = gatherDeviceSignals(ctx, deps)

	// 2. 今日时刻表 + 修正记录 + 最近状态
	gatherHistoryData(ctx, deps, ic)

	// 3. 对话上下文
	ic.Conversation = gatherConversation(ctx, deps)

	// 4. 用户画像 + 核心记忆
	gatherProfileData(ctx, deps, ic)

	// 5. 上次推断结果
	ic.PrevInference = gatherPrevInference(ctx, deps)

	// 6. 用户偏好
	ic.Preferences = gatherPreferences(ctx, deps)

	return ic
}

// FormatContextForPrompt 将预聚合的上下文格式化为 prompt 注入文本
// 数据段按三级可信度排列：实时信号(高) → 用户主动设置(最高) → AI推测(低)
func FormatContextForPrompt(ic *model.AgentInferenceContext) string {
	var sb strings.Builder

	// === 高可信度: 实时设备信号 ===
	if len(ic.DeviceSignals) > 0 {
		sb.WriteString("# 实时设备信号（推断的主要依据）\n")
		signalOrder := []string{"screen", "location", "movement", "calendar", "battery", "mode", "connection"}
		for _, k := range signalOrder {
			if v, ok := ic.DeviceSignals[k]; ok {
				data, _ := json.Marshal(v)
				sb.WriteString(fmt.Sprintf("- %s: %s\n", k, string(data)))
			}
		}
	} else {
		sb.WriteString("# 实时设备信号\n无（数据未上报）\n")
	}

	// === 最高可信度: 用户修正历史 ===
	if len(ic.Corrections) > 0 {
		sb.WriteString("\n# 用户修正历史（绝对不能再犯）\n")
		for _, c := range ic.Corrections {
			if c["original_emoji"] != "" {
				sb.WriteString(fmt.Sprintf("- %s %s → 修正为 %s %s\n",
					c["original_emoji"], c["original_activity"],
					c["corrected_emoji"], c["corrected_activity"]))
			} else {
				sb.WriteString(fmt.Sprintf("- 用户主动设为 %s %s\n", c["corrected_emoji"], c["corrected_activity"]))
			}
		}
	}

	// P1: 本时段历史纠正记录（高优先级）
	if len(ic.TimeslotCorrections) > 0 {
		sb.WriteString("\n# 本时段历史纠正记录（高优先级）\n")
		for _, tc := range ic.TimeslotCorrections {
			if tc.OriginalEmoji != "" || tc.OriginalActivity != "" {
				sb.WriteString(fmt.Sprintf("- 你之前推断「%s %s」但用户纠正为「%s %s」（已发生%d次）\n",
					tc.OriginalEmoji, tc.OriginalActivity,
					tc.CorrectedEmoji, tc.CorrectedActivity, tc.Count))
			}
		}
	}

	// 偏好
	if len(ic.Preferences) > 0 {
		sb.WriteString("\n# 用户偏好\n")
		for _, p := range ic.Preferences {
			sb.WriteString(fmt.Sprintf("- %s\n", p))
		}
	}

	// === 最高可信度: 用户主动设置的时刻表 ===
	if len(ic.UserScheduleItems) > 0 {
		sb.WriteString("\n# 用户主动设置的时刻表（高可信度）\n")
		for _, item := range ic.UserScheduleItems {
			marker := ""
			if m, ok := item["current_marker"].(string); ok {
				marker = " " + m
			}
			sb.WriteString(fmt.Sprintf("- %s %s (%s-%s)%s\n",
				item["emoji"], item["status"], item["start_time"], item["end_time"], marker))
		}
	}

	// === 用户画像 ===
	if len(ic.Profile) > 0 {
		sb.WriteString("\n# 用户画像\n")
		for k, v := range ic.Profile {
			sb.WriteString(fmt.Sprintf("- %s: %v\n", k, v))
		}
	}

	// === 最近状态记录（按来源标注可信度）===
	if len(ic.RecentStatuses) > 0 {
		sb.WriteString("\n# 最近状态记录（按来源标注可信度）\n")
		limit := len(ic.RecentStatuses)
		if limit > 5 {
			limit = 5
		}
		for _, s := range ic.RecentStatuses[:limit] {
			sourceLabel := sourceToLabel(s["source"])
			sb.WriteString(fmt.Sprintf("- %s %s (%s) [%s]\n", s["emoji"], s["status"], s["created_at"], sourceLabel))
		}
	}

	// === 上次推断（标注来源）===
	if ic.PrevInference != nil {
		sourceLabel := sourceToLabel(ic.PrevInference.Source)
		sb.WriteString(fmt.Sprintf("\n# 上次推断（来源: %s）\n", sourceLabel))
		if ic.PrevInference.Source == model.InferenceSourceAIAuto || ic.PrevInference.Source == model.InferenceSourceAIOneclick {
			sb.WriteString("# ⚠️ 此条来自AI推测，可信度低\n")
		}
		sb.WriteString(fmt.Sprintf("- %s %s (置信度: %s)\n",
			ic.PrevInference.Emoji, ic.PrevInference.Activity, ic.PrevInference.Confidence))
		if ic.PrevInference.Place != "" {
			sb.WriteString(fmt.Sprintf("- 地点: %s\n", ic.PrevInference.Place))
		}
	}

	// === 低可信度: AI 自动推测的时刻表 ===
	if len(ic.AIGuessScheduleItems) > 0 {
		sb.WriteString("\n# AI 自动推测的时刻表（低可信度，仅弱参考）\n")
		sb.WriteString("# ⚠️ 这些是 AI 之前的猜测，当实时信号矛盾时必须忽略\n")
		for _, item := range ic.AIGuessScheduleItems {
			marker := ""
			if m, ok := item["current_marker"].(string); ok {
				marker = " " + m
			}
			sb.WriteString(fmt.Sprintf("- %s %s (%s-%s)%s\n",
				item["emoji"], item["status"], item["start_time"], item["end_time"], marker))
		}
	}

	// 对话上下文
	if ic.Conversation != "" {
		sb.WriteString("\n# 最近对话\n")
		sb.WriteString(ic.Conversation)
		sb.WriteString("\n")
	}

	// 核心记忆
	if len(ic.CoreMemory) > 0 {
		sb.WriteString("\n# 行为记忆\n")
		for k, v := range ic.CoreMemory {
			sb.WriteString(fmt.Sprintf("- %s: %v\n", k, v))
		}
	}

	return sb.String()
}

// sourceToLabel 将 source 转为中文标签
func sourceToLabel(source string) string {
	switch source {
	case model.InferenceSourceUserConfirmed:
		return "用户确认"
	case model.InferenceSourceAIAuto:
		return "AI推测"
	case model.InferenceSourceAIOneclick:
		return "一键推断"
	default:
		return "未知来源"
	}
}

// ========== 数据收集子函数 ==========

func gatherDeviceSignals(ctx context.Context, deps *InferenceToolDeps) map[string]interface{} {
	sensorData := getCachedSensorForInference(ctx, deps.RedisClient, deps.UserID)
	if sensorData == nil {
		return nil
	}

	result := make(map[string]interface{})

	if sensorData.ExtendedLocation != nil {
		loc := map[string]interface{}{
			"place_type":             sensorData.ExtendedLocation.PlaceType,
			"place_type_source":      "client_guess",
			"at_place_since_minutes": sensorData.ExtendedLocation.AtPlaceSinceMinutes,
		}
		if sensorData.ExtendedLocation.PlaceName != "" {
			loc["place_name"] = sensorData.ExtendedLocation.PlaceName
		}
		lat := sensorData.ExtendedLocation.Latitude
		lng := sensorData.ExtendedLocation.Longitude
		enrichLocationWithConfidence(ctx, deps, loc, lat, lng, string(sensorData.ExtendedLocation.PlaceType))
		result["location"] = loc
	} else if sensorData.Location != nil {
		loc := map[string]interface{}{
			"place_type":             sensorData.Location.PlaceType,
			"place_type_source":      "client_guess",
			"at_place_since_minutes": sensorData.Location.AtPlaceSinceMinutes,
		}
		if sensorData.Location.City != "" {
			loc["city"] = sensorData.Location.City
		}
		lat := sensorData.Location.Latitude
		lng := sensorData.Location.Longitude
		if lat != 0 || lng != 0 {
			enrichLocationWithConfidence(ctx, deps, loc, lat, lng, string(sensorData.Location.PlaceType))
		}
		result["location"] = loc
	}

	if sensorData.Movement != nil {
		mv := sensorData.Movement
		// 当客户端运动类型为 unknown 时，从位置历史计算速度做兜底
		if mv.MovementType == "" || mv.MovementType == "unknown" {
			speed := estimateSpeedFromLocationHistory(ctx, deps.RedisClient, deps.UserID)
			if speed > 0 {
				switch {
				case speed > 10.0: // >10 km/h → 开车/乘车
					mv.MovementType = "driving"
					mv.IsMoving = true
					fmt.Printf("[运动兜底] user=%s speed=%.1fkm/h → driving\n", deps.UserID, speed)
				case speed > 3.0: // >3 km/h → 步行/骑行
					mv.MovementType = "walking"
					mv.IsMoving = true
					fmt.Printf("[运动兜底] user=%s speed=%.1fkm/h → walking\n", deps.UserID, speed)
				}
			}
		}
		result["movement"] = mv
	}
	if sensorData.Calendar != nil {
		result["calendar"] = sensorData.Calendar
	}
	if sensorData.Screen != nil {
		result["screen"] = sensorData.Screen
	}
	if sensorData.Battery != nil {
		result["battery"] = map[string]interface{}{
			"battery_level": sensorData.Battery.BatteryLevel,
			"is_charging":   sensorData.Battery.IsCharging,
		}
	}
	if sensorData.Mode != nil {
		result["mode"] = map[string]interface{}{
			"is_low_power_mode": sensorData.Mode.IsLowPowerMode,
			"is_focus_mode_on":  sensorData.Mode.IsFocusModeOn,
		}
	}
	if sensorData.Connection != nil {
		result["connection"] = map[string]interface{}{
			"is_headphones_connected": sensorData.Connection.IsHeadphonesConnected,
			"network_type":            sensorData.Connection.NetworkType,
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func gatherHistoryData(ctx context.Context, deps *InferenceToolDeps, ic *model.AgentInferenceContext) {
	now := time.Now()
	currentTime := now.Format("15:04")

	// 今日时刻表：按 IsAIGuess 分组
	if deps.ScheduleRepo != nil {
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		schedule, err := deps.ScheduleRepo.GetActiveByUserAndDate(ctx, deps.UserID, today)
		if err == nil && schedule != nil && len(schedule.Items) > 0 {
			scheduleData := map[string]interface{}{
				"total_items": len(schedule.Items),
			}

			items := make([]map[string]interface{}, 0, len(schedule.Items))
			for i, item := range schedule.Items {
				itemData := map[string]interface{}{
					"start_time":  item.StartTime,
					"end_time":    item.EndTime,
					"emoji":       item.Emoji,
					"status":      item.Status,
					"is_ai_guess": item.IsAIGuess,
				}

				inRange := false
				if item.EndTime < item.StartTime {
					inRange = currentTime >= item.StartTime || currentTime < item.EndTime
				} else {
					inRange = currentTime >= item.StartTime && currentTime < item.EndTime
				}
				if inRange {
					if item.IsAIGuess {
						itemData["current_marker"] = "← 当前时段(AI猜测)"
					} else {
						itemData["current_marker"] = "← 当前时段"
					}
					scheduleData["current_item"] = itemData
					if i+1 < len(schedule.Items) {
						next := schedule.Items[i+1]
						scheduleData["next_item"] = map[string]interface{}{
							"start_time": next.StartTime,
							"end_time":   next.EndTime,
							"emoji":      next.Emoji,
							"status":     next.Status,
						}
					}
				}

				items = append(items, itemData)

				// 按来源分组
				if item.IsAIGuess {
					ic.AIGuessScheduleItems = append(ic.AIGuessScheduleItems, itemData)
				} else {
					ic.UserScheduleItems = append(ic.UserScheduleItems, itemData)
				}
			}
			scheduleData["items"] = items
			ic.TodaySchedule = scheduleData
		}
	}

	// 最近状态：附带 source 标注
	if deps.MemoryRepo != nil {
		memories, err := deps.MemoryRepo.GetRecentUserStatusMemory(ctx, deps.UserID, 10)
		if err == nil && len(memories) > 0 {
			entries := make([]map[string]string, 0, len(memories))
			for _, m := range memories {
				source := m.Source
				if source == "" {
					source = "unknown"
				}
				entries = append(entries, map[string]string{
					"emoji":      m.Emoji,
					"status":     m.Status,
					"source":     source,
					"created_at": m.CreatedAt.Format("01-02 15:04"),
				})
			}
			ic.RecentStatuses = entries
		}
	}

	// 修正记录
	if deps.RedisClient != nil {
		feedbackKey := fmt.Sprintf("agent:feedback:%s", deps.UserID)
		data, err := deps.RedisClient.GetBytes(ctx, feedbackKey)
		if err == nil && len(data) > 0 {
			var corrections []*model.StatusMemoryEntry
			if json.Unmarshal(data, &corrections) == nil && len(corrections) > 0 {
				limit := len(corrections)
				if limit > 5 {
					limit = 5
				}
				entries := make([]map[string]string, 0, limit)
				for _, c := range corrections[:limit] {
					entry := map[string]string{
						"corrected_emoji":    c.CorrectedEmoji,
						"corrected_activity": c.CorrectedActivity,
					}
					if c.OriginalEmoji != "" {
						entry["original_emoji"] = c.OriginalEmoji
						entry["original_activity"] = c.OriginalActivity
					}
					entries = append(entries, entry)
				}
				ic.Corrections = entries
			}
		}
	}

	// P1: 查询本时段历史纠正记录（MySQL 持久化）
	if deps.MemoryRepo != nil {
		now := time.Now()
		corrections, err := deps.MemoryRepo.GetTimeslotCorrections(ctx, deps.UserID, int(now.Weekday()), now.Hour())
		if err == nil && len(corrections) > 0 {
			ic.TimeslotCorrections = corrections
		}
	}
}

func gatherConversation(ctx context.Context, deps *InferenceToolDeps) string {
	if deps.RedisClient == nil {
		return ""
	}
	key := fmt.Sprintf("v4:user_context:%s", deps.UserID)
	summary, err := deps.RedisClient.Get(ctx, key)
	if err != nil || summary == "" {
		return ""
	}
	return summary
}

func gatherProfileData(ctx context.Context, deps *InferenceToolDeps, ic *model.AgentInferenceContext) {
	// 用户画像
	if deps.UserProfileService != nil {
		profile, err := deps.UserProfileService.GetProfile(deps.UserID)
		if err == nil && profile != nil {
			profileData := make(map[string]interface{})
			if profile.OccupationType != "" {
				profileData["occupation_type"] = string(profile.OccupationType)
				profileData["occupation_name"] = model.GetProfileTypeName(string(profile.OccupationType))
			}
			if profile.WorkSchedule != "" {
				profileData["work_schedule"] = string(profile.WorkSchedule)
			}
			if profile.LifestyleType != "" {
				lifestyleNames := map[model.LifestyleType]string{
					model.LifestyleEarlyBird: "早起型",
					model.LifestyleNightOwl:  "夜猫子",
					model.LifestyleBalanced:  "规律型",
				}
				profileData["lifestyle_type"] = lifestyleNames[profile.LifestyleType]
			}
			if len(profileData) > 0 {
				ic.Profile = profileData
			}
		}
	}

	// 核心记忆 + persona + 时段分布
	if deps.MemoryRepo != nil {
		coreMemory, err := deps.MemoryRepo.GetCoreMemory(ctx, deps.UserID)
		if err == nil && coreMemory != nil {
			memoryData := make(map[string]interface{})
			if coreMemory.BehaviorInsights != "" {
				memoryData["behavior_insights"] = coreMemory.BehaviorInsights
			}
			if coreMemory.TimePatterns != "" {
				memoryData["time_patterns"] = coreMemory.TimePatterns
			}
			if coreMemory.LocationPreferences != "" {
				memoryData["location_preferences"] = coreMemory.LocationPreferences
			}
			if len(memoryData) > 0 {
				ic.CoreMemory = memoryData
			}

			// 注入个性化 persona 描述
			if coreMemory.PersonaText != "" {
				if ic.Profile == nil {
					ic.Profile = make(map[string]interface{})
				}
				ic.Profile["persona"] = coreMemory.PersonaText
			}

			// 注入当前时段的历史分布
			if coreMemory.TimePatternStats != "" {
				slotDist := extractCurrentSlotDistribution(coreMemory.TimePatternStats, time.Now())
				if slotDist != "" {
					if ic.Profile == nil {
						ic.Profile = make(map[string]interface{})
					}
					ic.Profile["current_slot_history"] = slotDist
				}
			}
		}
	}

	// 注入当前时段的历史可用率（user_time_slot_stats）
	if deps.MemoryRepo != nil {
		now := time.Now()
		sampleCount, availableRate, err := deps.MemoryRepo.GetTimeSlotStats(
			ctx, deps.UserID, int(now.Weekday()), now.Hour())
		if err == nil && sampleCount >= 3 {
			if ic.Profile == nil {
				ic.Profile = make(map[string]interface{})
			}
			weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
			ic.Profile["current_slot_availability"] = fmt.Sprintf(
				"%s %d:00-%d:00 历史可用率: %d%% (样本%d次)",
				weekdays[now.Weekday()], now.Hour(), now.Hour()+1,
				availableRate, sampleCount)
		}

		// P2-a: 注入昨天活动摘要
		yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
		summary, err := deps.MemoryRepo.GetDailyActivitySummary(ctx, deps.UserID, yesterday)
		if err == nil && summary != nil {
			if ic.Profile == nil {
				ic.Profile = make(map[string]interface{})
			}
			parts := []string{}
			if summary.WorkHours > 0 {
				parts = append(parts, fmt.Sprintf("工作%.1fh", summary.WorkHours))
			}
			if summary.HomeHours > 0 {
				parts = append(parts, fmt.Sprintf("在家%.1fh", summary.HomeHours))
			}
			if summary.TransitHours > 0 {
				parts = append(parts, fmt.Sprintf("通勤%.1fh", summary.TransitHours))
			}
			if summary.ScreenActiveHours > 0 {
				parts = append(parts, fmt.Sprintf("看屏幕%.1fh", summary.ScreenActiveHours))
			}
			if summary.TotalSteps > 0 {
				parts = append(parts, fmt.Sprintf("步数%d", summary.TotalSteps))
			}
			if len(parts) > 0 {
				ic.Profile["yesterday_activity"] = strings.Join(parts, "，")
			}
		}
	}

	// P2-b: 注入结构化时间表模板
	if deps.MemoryDocRepo != nil {
		doc, err := deps.MemoryDocRepo.GetByUserID(ctx, deps.UserID)
		if err == nil && doc != nil && doc.StructuredSchedule.StructuredScheduleTemplate != nil {
			now := time.Now()
			isWeekend := now.Weekday() == time.Saturday || now.Weekday() == time.Sunday
			var slots []model.TimeSlotTemplate
			if isWeekend {
				slots = doc.StructuredSchedule.WeekendTemplate
			} else {
				slots = doc.StructuredSchedule.WeekdayTemplate
			}
			// 找到匹配当前时段的模板
			for _, slot := range slots {
				if now.Hour() >= slot.StartHour && now.Hour() < slot.EndHour {
					if ic.Profile == nil {
						ic.Profile = make(map[string]interface{})
					}
					ic.Profile["structured_schedule_hint"] = fmt.Sprintf(
						"历史规律: %s %s (%02d:00-%02d:00, %.0f%%置信, %d次)",
						slot.Emoji, slot.Activity,
						slot.StartHour, slot.EndHour,
						slot.Confidence*100, slot.Samples)
					break
				}
			}
		}
	}
}

func gatherPrevInference(ctx context.Context, deps *InferenceToolDeps) *model.CurrentStatusInference {
	if deps.RedisClient == nil {
		return nil
	}
	key := fmt.Sprintf("infer:prev:%s", deps.UserID)
	data, err := deps.RedisClient.GetBytes(ctx, key)
	if err != nil || len(data) == 0 {
		return nil
	}
	var prev model.CurrentStatusInference
	if json.Unmarshal(data, &prev) != nil {
		return nil
	}
	return &prev
}

func gatherPreferences(ctx context.Context, deps *InferenceToolDeps) []string {
	if deps.RedisClient == nil {
		return nil
	}
	key := fmt.Sprintf("infer:pref:%s", deps.UserID)
	data, err := deps.RedisClient.GetBytes(ctx, key)
	if err != nil || len(data) == 0 {
		return nil
	}
	var pref model.UserInferencePreference
	if json.Unmarshal(data, &pref) != nil {
		return nil
	}
	return pref.Preferences
}

// enrichLocationWithConfidence 为 location map 注入置信度和服务端分类
// 优先使用 Phase 4 MySQL 聚类（更精确），回退到 Phase 2 Redis 历史
func enrichLocationWithConfidence(ctx context.Context, deps *InferenceToolDeps, loc map[string]interface{}, lat, lng float64, clientPlaceType string) {
	if lat == 0 && lng == 0 {
		return
	}

	// Phase 4: 尝试 MySQL 聚类分类（如果 LocationService 可用）
	if deps.LocationService != nil {
		classification := deps.LocationService.ClassifyLocation(ctx, deps.UserID, lat, lng)
		if classification != nil {
			// 通过类型断言获取字段
			if classMap, ok := toMap(classification); ok {
				if pt, ok := classMap["place_type"].(string); ok && pt != "" {
					loc["place_type"] = pt
				}
				if pts, ok := classMap["place_type_source"].(string); ok && pts != "" {
					loc["place_type_source"] = pts
				}
				if conf, ok := classMap["confidence"].(string); ok && conf != "" {
					loc["location_confidence"] = conf
				}
				if isNew, ok := classMap["is_new_location"].(bool); ok {
					loc["is_new_location"] = isNew
				}
				return // MySQL 分类成功，不需要 Redis 回退
			}
		}
	}

	// Phase 2: 回退到 Redis 历史置信度
	conf := EvaluateLocationConfidence(ctx, deps.RedisClient, deps.UserID, lat, lng, clientPlaceType)
	if conf != nil {
		loc["location_confidence"] = conf.Confidence
		loc["night_stays_7d"] = conf.NightStays7d
		loc["is_new_location"] = conf.IsNewLocation
		if conf.OverrideType != "" {
			loc["place_type"] = conf.OverrideType
			loc["place_type_source"] = "server_history"
		}
	}
}

// toMap 将 struct/interface 转为 map（通过 JSON 序列化）
func toMap(v interface{}) (map[string]interface{}, bool) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	var m map[string]interface{}
	if json.Unmarshal(data, &m) != nil {
		return nil, false
	}
	return m, true
}

// ========== 辅助函数 ==========

// extractCurrentSlotDistribution 从 time_pattern_stats JSON 提取当前时段历史分布
func extractCurrentSlotDistribution(statsJSON string, now time.Time) string {
	if statsJSON == "" {
		return ""
	}

	type slotStats struct {
		Weekday map[string]map[string]int `json:"weekday"`
		Weekend map[string]map[string]int `json:"weekend"`
	}

	var stats slotStats
	if json.Unmarshal([]byte(statsJSON), &stats) != nil {
		return ""
	}

	weekday := now.Weekday()
	hour := now.Hour()
	hourStr := fmt.Sprintf("%d", hour)
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	isWeekend := weekday == time.Saturday || weekday == time.Sunday
	var dist map[string]int
	if isWeekend {
		dist = stats.Weekend[hourStr]
	} else {
		dist = stats.Weekday[hourStr]
	}

	if len(dist) == 0 {
		return ""
	}

	var parts []string
	for activity, count := range dist {
		parts = append(parts, fmt.Sprintf("%s(%d次)", activity, count))
	}

	return fmt.Sprintf("%s %d:00-%d:00 历史分布: %s",
		weekdays[weekday], hour, hour+1, strings.Join(parts, ", "))
}

// getCachedSensorForInference 从 Redis 获取推断用的传感器数据
func getCachedSensorForInference(ctx context.Context, redisClient *tencent.RedisClient, userID string) *model.ExtendedStatusReportRequest {
	if redisClient == nil {
		return nil
	}

	extKey := fmt.Sprintf("agent:extended:%s", userID)
	data, err := redisClient.GetBytes(ctx, extKey)
	if err == nil && len(data) > 0 {
		var req model.ExtendedStatusReportRequest
		if json.Unmarshal(data, &req) == nil {
			return &req
		}
	}

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
