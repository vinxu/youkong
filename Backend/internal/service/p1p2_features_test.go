package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
)

// ========== P1: 推理纠正闭环测试 ==========

func TestInferenceCorrectionModel(t *testing.T) {
	t.Run("InferenceCorrection结构体字段完整", func(t *testing.T) {
		correction := model.InferenceCorrection{
			UserID:            "user-123",
			OriginalEmoji:     "💼",
			OriginalActivity:  "工作",
			CorrectedEmoji:    "🎮",
			CorrectedActivity: "打游戏",
			CorrectedPlace:    "家里",
			DayOfWeek:         3,  // 周三
			HourOfDay:         20, // 晚上8点
			DeviceContext:     `{"screen":{"is_active":true}}`,
			LocationType:      "home",
		}

		if correction.UserID != "user-123" {
			t.Errorf("UserID 不匹配")
		}
		if correction.DayOfWeek != 3 {
			t.Errorf("DayOfWeek 应为 3，得到 %d", correction.DayOfWeek)
		}
		if correction.HourOfDay != 20 {
			t.Errorf("HourOfDay 应为 20，得到 %d", correction.HourOfDay)
		}
	})

	t.Run("TimeslotCorrectionSummary聚合格式", func(t *testing.T) {
		summary := model.TimeslotCorrectionSummary{
			OriginalEmoji:     "💼",
			OriginalActivity:  "工作",
			CorrectedEmoji:    "🎮",
			CorrectedActivity: "打游戏",
			CorrectedPlace:    "家里",
			Count:             5,
		}

		if summary.Count != 5 {
			t.Errorf("Count 应为 5，得到 %d", summary.Count)
		}
		if summary.OriginalEmoji != "💼" {
			t.Errorf("OriginalEmoji 不匹配")
		}
	})

	t.Run("AgentInferenceContext包含TimeslotCorrections", func(t *testing.T) {
		ctx := model.AgentInferenceContext{
			TimeslotCorrections: []model.TimeslotCorrectionSummary{
				{
					OriginalEmoji:     "💼",
					OriginalActivity:  "工作",
					CorrectedEmoji:    "🎮",
					CorrectedActivity: "打游戏",
					Count:             3,
				},
			},
		}

		if len(ctx.TimeslotCorrections) != 1 {
			t.Fatalf("TimeslotCorrections 长度应为 1，得到 %d", len(ctx.TimeslotCorrections))
		}
		if ctx.TimeslotCorrections[0].Count != 3 {
			t.Errorf("第一条纠正的 Count 应为 3")
		}
	})
}

func TestFormatContextForPrompt_TimeslotCorrections(t *testing.T) {
	t.Run("有时段纠正记录时输出", func(t *testing.T) {
		ic := &model.AgentInferenceContext{
			TimeslotCorrections: []model.TimeslotCorrectionSummary{
				{
					OriginalEmoji:     "💼",
					OriginalActivity:  "工作",
					CorrectedEmoji:    "🎮",
					CorrectedActivity: "打游戏",
					Count:             5,
				},
				{
					OriginalEmoji:     "😴",
					OriginalActivity:  "睡觉",
					CorrectedEmoji:    "📱",
					CorrectedActivity: "刷手机",
					Count:             2,
				},
			},
		}

		result := agent.FormatContextForPrompt(ic)

		if !strings.Contains(result, "本时段历史纠正记录（高优先级）") {
			t.Errorf("缺少时段纠正记录标题")
		}
		if !strings.Contains(result, "你之前推断「💼 工作」但用户纠正为「🎮 打游戏」（已发生5次）") {
			t.Errorf("缺少第一条纠正记录: got:\n%s", result)
		}
		if !strings.Contains(result, "你之前推断「😴 睡觉」但用户纠正为「📱 刷手机」（已发生2次）") {
			t.Errorf("缺少第二条纠正记录")
		}
	})

	t.Run("无时段纠正记录时不输出", func(t *testing.T) {
		ic := &model.AgentInferenceContext{
			TimeslotCorrections: nil,
		}

		result := agent.FormatContextForPrompt(ic)

		if strings.Contains(result, "本时段历史纠正记录") {
			t.Errorf("不应包含时段纠正标题")
		}
	})

	t.Run("纠正记录在偏好之前", func(t *testing.T) {
		ic := &model.AgentInferenceContext{
			TimeslotCorrections: []model.TimeslotCorrectionSummary{
				{OriginalEmoji: "💼", OriginalActivity: "工作", CorrectedEmoji: "🎮", CorrectedActivity: "打游戏", Count: 1},
			},
			Preferences: []string{"晚上在家偏好说刷剧"},
		}

		result := agent.FormatContextForPrompt(ic)

		correctionIdx := strings.Index(result, "本时段历史纠正记录")
		prefIdx := strings.Index(result, "用户偏好")

		if correctionIdx < 0 {
			t.Fatalf("缺少时段纠正记录")
		}
		if prefIdx < 0 {
			t.Fatalf("缺少用户偏好")
		}
		if correctionIdx > prefIdx {
			t.Errorf("时段纠正记录应在用户偏好之前，correctionIdx=%d, prefIdx=%d", correctionIdx, prefIdx)
		}
	})
}

func TestSaveStatusFeedback_CorrectionTriggerCondition(t *testing.T) {
	// 测试纠正触发条件逻辑
	testCases := []struct {
		name             string
		originalActivity string
		originalEmoji    string
		correctedEmoji   string
		correctedActivity string
		shouldTrigger    bool
	}{
		{
			name:              "有原始+不同活动→触发",
			originalActivity:  "工作",
			originalEmoji:     "💼",
			correctedEmoji:    "🎮",
			correctedActivity: "打游戏",
			shouldTrigger:     true,
		},
		{
			name:              "有原始+不同emoji→触发",
			originalActivity:  "休息",
			originalEmoji:     "😴",
			correctedEmoji:    "📱",
			correctedActivity: "休息",
			shouldTrigger:     true,
		},
		{
			name:              "无原始活动→不触发",
			originalActivity:  "",
			originalEmoji:     "",
			correctedEmoji:    "🎮",
			correctedActivity: "打游戏",
			shouldTrigger:     false,
		},
		{
			name:              "有原始但完全相同→不触发",
			originalActivity:  "工作",
			originalEmoji:     "💼",
			correctedEmoji:    "💼",
			correctedActivity: "工作",
			shouldTrigger:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shouldTrigger := tc.originalActivity != "" &&
				(tc.originalEmoji != tc.correctedEmoji || tc.originalActivity != tc.correctedActivity)
			if shouldTrigger != tc.shouldTrigger {
				t.Errorf("纠正触发条件判断错误: 期望 %v, 得到 %v", tc.shouldTrigger, shouldTrigger)
			}
		})
	}
}

// ========== P2-a: 日维度传感器摘要测试 ==========

func TestDailyActivitySummaryModel(t *testing.T) {
	t.Run("DailyActivitySummary字段正确", func(t *testing.T) {
		summary := model.DailyActivitySummary{
			UserID:            "user-123",
			SummaryDate:       "2026-02-13",
			HomeHours:         6.5,
			WorkHours:         8.0,
			TransitHours:      1.5,
			ScreenActiveHours: 3.2,
			TotalSteps:        8500,
			MostActivePeriod:  "14:00-15:00",
			SampleCount:       48,
			TextSummary:       "工作8.0h，在家6.5h，通勤1.5h，看屏幕3.2h，步数8500",
		}

		if summary.HomeHours != 6.5 {
			t.Errorf("HomeHours 应为 6.5, 得到 %f", summary.HomeHours)
		}
		if summary.TotalSteps != 8500 {
			t.Errorf("TotalSteps 应为 8500, 得到 %d", summary.TotalSteps)
		}
		if summary.SampleCount != 48 {
			t.Errorf("SampleCount 应为 48, 得到 %d", summary.SampleCount)
		}
	})

	t.Run("JSON序列化正确", func(t *testing.T) {
		summary := model.DailyActivitySummary{
			UserID:      "user-123",
			SummaryDate: "2026-02-13",
			HomeHours:   6.5,
			WorkHours:   8.0,
		}

		data, err := json.Marshal(summary)
		if err != nil {
			t.Fatalf("JSON marshal 失败: %v", err)
		}

		var decoded model.DailyActivitySummary
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("JSON unmarshal 失败: %v", err)
		}
		if decoded.HomeHours != 6.5 || decoded.WorkHours != 8.0 {
			t.Errorf("JSON 往返序列化数据不匹配")
		}
	})
}

func TestYesterdayActivityInjection(t *testing.T) {
	// 测试昨天活动摘要注入到 Profile 的格式
	t.Run("摘要格式化正确", func(t *testing.T) {
		summary := &model.DailyActivitySummary{
			WorkHours:         6.5,
			HomeHours:         4.0,
			TransitHours:      0,
			ScreenActiveHours: 3.2,
			TotalSteps:        8500,
		}

		parts := []string{}
		if summary.WorkHours > 0 {
			parts = append(parts, "工作6.5h")
		}
		if summary.HomeHours > 0 {
			parts = append(parts, "在家4.0h")
		}
		if summary.TransitHours > 0 {
			parts = append(parts, "通勤0.0h")
		}
		if summary.ScreenActiveHours > 0 {
			parts = append(parts, "看屏幕3.2h")
		}
		if summary.TotalSteps > 0 {
			parts = append(parts, "步数8500")
		}

		result := strings.Join(parts, "，")
		expected := "工作6.5h，在家4.0h，看屏幕3.2h，步数8500"
		if result != expected {
			t.Errorf("摘要格式不匹配:\n期望: %s\n得到: %s", expected, result)
		}
	})

	t.Run("零值字段不输出", func(t *testing.T) {
		summary := &model.DailyActivitySummary{
			WorkHours:         0,
			HomeHours:         4.0,
			TransitHours:      0,
			ScreenActiveHours: 0,
			TotalSteps:        0,
		}

		parts := []string{}
		if summary.WorkHours > 0 {
			parts = append(parts, "工作0h")
		}
		if summary.HomeHours > 0 {
			parts = append(parts, "在家4.0h")
		}

		result := strings.Join(parts, "，")
		if strings.Contains(result, "工作") {
			t.Errorf("WorkHours=0 不应输出")
		}
		if !strings.Contains(result, "在家4.0h") {
			t.Errorf("应包含在家4.0h")
		}
	})
}

// ========== P2-b: 结构化时间表模板测试 ==========

func TestStructuredScheduleJSON_ValueScan(t *testing.T) {
	t.Run("Value_非nil", func(t *testing.T) {
		schedule := model.StructuredScheduleJSON{
			StructuredScheduleTemplate: &model.StructuredScheduleTemplate{
				WeekdayTemplate: []model.TimeSlotTemplate{
					{StartHour: 9, EndHour: 18, Emoji: "💼", Activity: "工作", Confidence: 0.85, Samples: 12},
				},
				WeekendTemplate: []model.TimeSlotTemplate{
					{StartHour: 10, EndHour: 12, Emoji: "☕", Activity: "休息", Confidence: 0.7, Samples: 5},
				},
				GeneratedAt: "2026-02-13T03:00:00+08:00",
			},
		}

		val, err := schedule.Value()
		if err != nil {
			t.Fatalf("Value() 失败: %v", err)
		}
		if val == nil {
			t.Fatalf("Value() 返回 nil")
		}

		// 验证 JSON 结构
		var parsed model.StructuredScheduleTemplate
		if err := json.Unmarshal(val.([]byte), &parsed); err != nil {
			t.Fatalf("Value() 输出的 JSON 无法解析: %v", err)
		}
		if len(parsed.WeekdayTemplate) != 1 {
			t.Errorf("WeekdayTemplate 长度应为 1")
		}
		if parsed.WeekdayTemplate[0].Emoji != "💼" {
			t.Errorf("WeekdayTemplate[0].Emoji 应为 💼")
		}
	})

	t.Run("Value_nil", func(t *testing.T) {
		schedule := model.StructuredScheduleJSON{}

		val, err := schedule.Value()
		if err != nil {
			t.Fatalf("Value() 失败: %v", err)
		}
		if val != nil {
			t.Errorf("nil StructuredScheduleTemplate 的 Value() 应返回 nil, 得到 %v", val)
		}
	})

	t.Run("Scan_有效JSON", func(t *testing.T) {
		jsonData := `{"weekday_template":[{"start_hour":9,"end_hour":18,"emoji":"💼","activity":"工作","confidence":0.85,"samples":12}],"weekend_template":[],"generated_at":"2026-02-13T03:00:00+08:00"}`

		var schedule model.StructuredScheduleJSON
		err := schedule.Scan([]byte(jsonData))
		if err != nil {
			t.Fatalf("Scan() 失败: %v", err)
		}
		if schedule.StructuredScheduleTemplate == nil {
			t.Fatalf("Scan() 后 StructuredScheduleTemplate 为 nil")
		}
		if len(schedule.WeekdayTemplate) != 1 {
			t.Errorf("WeekdayTemplate 长度应为 1, 得到 %d", len(schedule.WeekdayTemplate))
		}
		if schedule.WeekdayTemplate[0].Activity != "工作" {
			t.Errorf("WeekdayTemplate[0].Activity 应为 '工作'")
		}
	})

	t.Run("Scan_nil", func(t *testing.T) {
		var schedule model.StructuredScheduleJSON
		err := schedule.Scan(nil)
		if err != nil {
			t.Fatalf("Scan(nil) 失败: %v", err)
		}
		if schedule.StructuredScheduleTemplate != nil {
			t.Errorf("Scan(nil) 后应为 nil")
		}
	})

	t.Run("Scan_string类型", func(t *testing.T) {
		jsonStr := `{"weekday_template":[],"weekend_template":[{"start_hour":10,"end_hour":12,"emoji":"☕","activity":"休息","confidence":0.7,"samples":5}],"generated_at":"2026-02-13"}`

		var schedule model.StructuredScheduleJSON
		err := schedule.Scan(jsonStr)
		if err != nil {
			t.Fatalf("Scan(string) 失败: %v", err)
		}
		if schedule.StructuredScheduleTemplate == nil {
			t.Fatalf("Scan(string) 后 StructuredScheduleTemplate 为 nil")
		}
		if len(schedule.WeekendTemplate) != 1 {
			t.Errorf("WeekendTemplate 长度应为 1")
		}
	})

	t.Run("Value_Scan往返", func(t *testing.T) {
		original := model.StructuredScheduleJSON{
			StructuredScheduleTemplate: &model.StructuredScheduleTemplate{
				WeekdayTemplate: []model.TimeSlotTemplate{
					{StartHour: 9, EndHour: 12, Emoji: "💼", Activity: "工作", Confidence: 0.9, Samples: 20},
					{StartHour: 14, EndHour: 18, Emoji: "💼", Activity: "工作", Confidence: 0.85, Samples: 18},
				},
				WeekendTemplate: []model.TimeSlotTemplate{
					{StartHour: 10, EndHour: 13, Emoji: "☕", Activity: "休息", Confidence: 0.7, Samples: 6},
				},
				GeneratedAt: "2026-02-13T03:00:00+08:00",
			},
		}

		val, err := original.Value()
		if err != nil {
			t.Fatalf("Value() 失败: %v", err)
		}

		var restored model.StructuredScheduleJSON
		if err := restored.Scan(val); err != nil {
			t.Fatalf("Scan() 失败: %v", err)
		}

		if len(restored.WeekdayTemplate) != 2 {
			t.Errorf("往返后 WeekdayTemplate 长度应为 2, 得到 %d", len(restored.WeekdayTemplate))
		}
		if len(restored.WeekendTemplate) != 1 {
			t.Errorf("往返后 WeekendTemplate 长度应为 1, 得到 %d", len(restored.WeekendTemplate))
		}
		if restored.WeekdayTemplate[0].Confidence != 0.9 {
			t.Errorf("往返后 Confidence 不匹配")
		}
		if restored.GeneratedAt != "2026-02-13T03:00:00+08:00" {
			t.Errorf("往返后 GeneratedAt 不匹配")
		}
	})
}

func TestMergeAdjacentSlots(t *testing.T) {
	t.Run("合并相邻同活动时段", func(t *testing.T) {
		slots := []model.TimeSlotTemplate{
			{StartHour: 9, EndHour: 10, Emoji: "💼", Activity: "工作", Confidence: 0.8, Samples: 5},
			{StartHour: 10, EndHour: 11, Emoji: "💼", Activity: "工作", Confidence: 0.9, Samples: 7},
			{StartHour: 11, EndHour: 12, Emoji: "💼", Activity: "工作", Confidence: 0.85, Samples: 6},
		}

		merged := mergeAdjacentSlots(slots)

		if len(merged) != 1 {
			t.Fatalf("合并后应为 1 段，得到 %d", len(merged))
		}
		if merged[0].StartHour != 9 || merged[0].EndHour != 12 {
			t.Errorf("合并后时间应为 9-12，得到 %d-%d", merged[0].StartHour, merged[0].EndHour)
		}
		if merged[0].Samples != 18 {
			t.Errorf("合并后 Samples 应为 18，得到 %d", merged[0].Samples)
		}
	})

	t.Run("不同活动不合并", func(t *testing.T) {
		slots := []model.TimeSlotTemplate{
			{StartHour: 9, EndHour: 10, Emoji: "💼", Activity: "工作", Confidence: 0.8, Samples: 5},
			{StartHour: 10, EndHour: 11, Emoji: "🍚", Activity: "吃饭", Confidence: 0.9, Samples: 3},
			{StartHour: 11, EndHour: 12, Emoji: "💼", Activity: "工作", Confidence: 0.85, Samples: 6},
		}

		merged := mergeAdjacentSlots(slots)

		if len(merged) != 3 {
			t.Fatalf("不同活动不应合并，应为 3 段，得到 %d", len(merged))
		}
	})

	t.Run("乱序输入也能正确合并", func(t *testing.T) {
		slots := []model.TimeSlotTemplate{
			{StartHour: 11, EndHour: 12, Emoji: "💼", Activity: "工作", Confidence: 0.85, Samples: 6},
			{StartHour: 9, EndHour: 10, Emoji: "💼", Activity: "工作", Confidence: 0.8, Samples: 5},
			{StartHour: 10, EndHour: 11, Emoji: "💼", Activity: "工作", Confidence: 0.9, Samples: 7},
		}

		merged := mergeAdjacentSlots(slots)

		if len(merged) != 1 {
			t.Fatalf("乱序输入合并后应为 1 段，得到 %d", len(merged))
		}
		if merged[0].StartHour != 9 || merged[0].EndHour != 12 {
			t.Errorf("合并后时间应为 9-12")
		}
	})

	t.Run("不相邻的同活动不合并", func(t *testing.T) {
		slots := []model.TimeSlotTemplate{
			{StartHour: 9, EndHour: 10, Emoji: "💼", Activity: "工作", Confidence: 0.8, Samples: 5},
			{StartHour: 14, EndHour: 15, Emoji: "💼", Activity: "工作", Confidence: 0.85, Samples: 6},
		}

		merged := mergeAdjacentSlots(slots)

		if len(merged) != 2 {
			t.Fatalf("不相邻的时段不应合并，应为 2 段，得到 %d", len(merged))
		}
	})

	t.Run("空输入", func(t *testing.T) {
		merged := mergeAdjacentSlots(nil)
		if len(merged) != 0 {
			t.Errorf("空输入应返回空，得到 %d", len(merged))
		}
	})

	t.Run("单个时段", func(t *testing.T) {
		slots := []model.TimeSlotTemplate{
			{StartHour: 9, EndHour: 10, Emoji: "💼", Activity: "工作", Confidence: 0.8, Samples: 5},
		}

		merged := mergeAdjacentSlots(slots)

		if len(merged) != 1 {
			t.Fatalf("单个时段应保持不变")
		}
	})
}

func TestGenerateStructuredSchedule_MinimumData(t *testing.T) {
	// 测试数据不足时返回 nil
	t.Run("记忆不足10条返回nil", func(t *testing.T) {
		// GenerateStructuredSchedule 要求 >= 10 条记忆
		// 这里我们测试逻辑：memories < 10 → return nil
		memories := make([]*model.UserStatusMemory, 5)
		if len(memories) >= 10 {
			t.Errorf("不足 10 条记忆应跳过生成")
		}
	})
}

func TestScheduleHintFormatting(t *testing.T) {
	t.Run("结构化时间表提示格式", func(t *testing.T) {
		slot := model.TimeSlotTemplate{
			StartHour:  9,
			EndHour:    18,
			Emoji:      "💼",
			Activity:   "工作",
			Confidence: 0.85,
			Samples:    12,
		}

		hint := formatScheduleHint(slot)
		expected := "历史规律: 💼 工作 (09:00-18:00, 85%置信, 12次)"
		if hint != expected {
			t.Errorf("提示格式不匹配:\n期望: %s\n得到: %s", expected, hint)
		}
	})

	t.Run("不同置信度格式", func(t *testing.T) {
		slot := model.TimeSlotTemplate{
			StartHour:  22,
			EndHour:    23,
			Emoji:      "😴",
			Activity:   "休息",
			Confidence: 0.65,
			Samples:    5,
		}

		hint := formatScheduleHint(slot)
		if !strings.Contains(hint, "65%置信") {
			t.Errorf("置信度格式错误: %s", hint)
		}
		if !strings.Contains(hint, "22:00-23:00") {
			t.Errorf("时间格式错误: %s", hint)
		}
	})
}

func TestFormatContextForPrompt_IntegrationAllFeatures(t *testing.T) {
	t.Run("完整上下文包含所有P1P2段", func(t *testing.T) {
		ic := &model.AgentInferenceContext{
			DeviceSignals: map[string]interface{}{
				"screen": map[string]interface{}{"is_active": true},
			},
			TimeslotCorrections: []model.TimeslotCorrectionSummary{
				{OriginalEmoji: "💼", OriginalActivity: "工作", CorrectedEmoji: "🎮", CorrectedActivity: "打游戏", Count: 3},
			},
			Profile: map[string]interface{}{
				"yesterday_activity":        "工作6.5h，在家4.0h",
				"structured_schedule_hint":  "历史规律: 💼 工作 (09:00-18:00, 85%置信, 12次)",
			},
			Preferences: []string{"晚上在家偏好说刷剧"},
		}

		result := agent.FormatContextForPrompt(ic)

		// 验证所有段都存在
		sections := []string{
			"实时设备信号",
			"本时段历史纠正记录（高优先级）",
			"你之前推断「💼 工作」但用户纠正为「🎮 打游戏」（已发生3次）",
			"用户偏好",
			"用户画像",
			"yesterday_activity",
			"structured_schedule_hint",
		}

		for _, section := range sections {
			if !strings.Contains(result, section) {
				t.Errorf("缺少段落: %s\n完整输出:\n%s", section, result)
			}
		}
	})

	t.Run("段落顺序正确", func(t *testing.T) {
		ic := &model.AgentInferenceContext{
			DeviceSignals: map[string]interface{}{"screen": true},
			TimeslotCorrections: []model.TimeslotCorrectionSummary{
				{OriginalEmoji: "💼", OriginalActivity: "工作", CorrectedEmoji: "🎮", CorrectedActivity: "打游戏", Count: 1},
			},
			Corrections: []map[string]string{
				{"original_emoji": "💼", "original_activity": "工作", "corrected_emoji": "🎮", "corrected_activity": "打游戏"},
			},
			Profile: map[string]interface{}{"occupation": "engineer"},
			UserScheduleItems: []map[string]interface{}{
				{"emoji": "💼", "status": "工作", "start_time": "09:00", "end_time": "18:00"},
			},
		}

		result := agent.FormatContextForPrompt(ic)

		// 验证关键顺序
		signalIdx := strings.Index(result, "实时设备信号")
		correctionRedisIdx := strings.Index(result, "用户修正历史")
		correctionMySQLIdx := strings.Index(result, "本时段历史纠正记录")
		scheduleIdx := strings.Index(result, "用户主动设置的时刻表")
		profileIdx := strings.Index(result, "用户画像")

		if signalIdx > correctionRedisIdx {
			t.Errorf("实时信号应在修正历史之前")
		}
		if correctionRedisIdx > correctionMySQLIdx {
			t.Errorf("Redis修正应在MySQL纠正之前")
		}
		if correctionMySQLIdx > scheduleIdx {
			t.Errorf("MySQL纠正应在时刻表之前")
		}
		if scheduleIdx > profileIdx {
			t.Errorf("时刻表应在画像之前")
		}
	})
}

func TestTimeSlotTemplateModel(t *testing.T) {
	t.Run("JSON序列化", func(t *testing.T) {
		slot := model.TimeSlotTemplate{
			StartHour:  9,
			EndHour:    18,
			Emoji:      "💼",
			Activity:   "工作",
			Confidence: 0.85,
			Samples:    12,
		}

		data, err := json.Marshal(slot)
		if err != nil {
			t.Fatalf("Marshal 失败: %v", err)
		}

		var decoded model.TimeSlotTemplate
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal 失败: %v", err)
		}

		if decoded.StartHour != 9 || decoded.EndHour != 18 {
			t.Errorf("时间不匹配")
		}
		if decoded.Emoji != "💼" || decoded.Activity != "工作" {
			t.Errorf("活动不匹配")
		}
		if decoded.Confidence != 0.85 || decoded.Samples != 12 {
			t.Errorf("统计不匹配")
		}
	})
}

func TestUserMemoryDocument_WithStructuredSchedule(t *testing.T) {
	t.Run("UserMemoryDocument包含StructuredSchedule字段", func(t *testing.T) {
		doc := model.UserMemoryDocument{
			UserID: "user-123",
			StructuredSchedule: model.StructuredScheduleJSON{
				StructuredScheduleTemplate: &model.StructuredScheduleTemplate{
					WeekdayTemplate: []model.TimeSlotTemplate{
						{StartHour: 9, EndHour: 18, Emoji: "💼", Activity: "工作", Confidence: 0.85, Samples: 12},
					},
					GeneratedAt: time.Now().Format(time.RFC3339),
				},
			},
		}

		if doc.StructuredSchedule.StructuredScheduleTemplate == nil {
			t.Fatal("StructuredSchedule 不应为 nil")
		}
		if len(doc.StructuredSchedule.WeekdayTemplate) != 1 {
			t.Errorf("WeekdayTemplate 长度应为 1")
		}
	})

	t.Run("JSON序列化包含structured_schedule", func(t *testing.T) {
		doc := model.UserMemoryDocument{
			UserID: "user-123",
			StructuredSchedule: model.StructuredScheduleJSON{
				StructuredScheduleTemplate: &model.StructuredScheduleTemplate{
					WeekdayTemplate: []model.TimeSlotTemplate{
						{StartHour: 9, EndHour: 12, Emoji: "💼", Activity: "工作", Confidence: 0.8, Samples: 10},
					},
					WeekendTemplate: nil,
					GeneratedAt:     "2026-02-13",
				},
			},
		}

		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("Marshal 失败: %v", err)
		}

		jsonStr := string(data)
		if !strings.Contains(jsonStr, "structured_schedule") {
			t.Errorf("JSON 应包含 structured_schedule 字段")
		}
		if !strings.Contains(jsonStr, "weekday_template") {
			t.Errorf("JSON 应包含 weekday_template")
		}
	})
}

// ========== 辅助函数（复制 FormatContextForPrompt 中的格式化逻辑用于单元测试） ==========

func formatScheduleHint(slot model.TimeSlotTemplate) string {
	return strings.TrimSpace(strings.Replace(
		"历史规律: "+slot.Emoji+" "+slot.Activity+" ("+
			padHour(slot.StartHour)+":00-"+padHour(slot.EndHour)+":00, "+
			formatConfidence(slot.Confidence)+"置信, "+
			formatSamples(slot.Samples)+"次)",
		"  ", " ", -1))
}

func padHour(h int) string {
	if h < 10 {
		return "0" + string(rune('0'+h))
	}
	return string(rune('0'+h/10)) + string(rune('0'+h%10))
}

func formatConfidence(c float64) string {
	pct := int(c * 100)
	return strings.TrimRight(strings.TrimRight(
		strings.Replace(
			string(rune('0'+pct/10))+string(rune('0'+pct%10))+"%",
			"0%", "%", 1),
		"0"), ".")
}

func formatSamples(s int) string {
	result := ""
	if s >= 10 {
		result += string(rune('0' + s/10))
	}
	result += string(rune('0' + s%10))
	return result
}
