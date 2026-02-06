package service

// ========== 推断评估：测试场景定义 ==========
// 复用 v4_eval_judge.go 中的 AutoEvalResult / JudgeScores
// 复用 v4_eval_scenarios.go 中的 EvalScenario 结构（但使用推断专用分类和工具）

// InferenceEvalCategory 推断测试分类
type InferenceEvalCategory string

const (
	InfCatNormal       InferenceEvalCategory = "I1_normal"        // 正常推断（传感器数据充足）
	InfCatPartialData  InferenceEvalCategory = "I2_partial_data"  // 部分数据缺失
	InfCatNoData       InferenceEvalCategory = "I3_no_data"       // 无传感器数据（新用户/冷启动）
	InfCatConflict     InferenceEvalCategory = "I4_conflict"      // 数据矛盾
	InfCatScheduleHint InferenceEvalCategory = "I5_schedule_hint" // 时刻表辅助
	InfCatHistory      InferenceEvalCategory = "I6_history"       // 历史规律利用
	InfCatCorrection   InferenceEvalCategory = "I7_correction"    // 用户修正场景
	InfCatEdgeCase     InferenceEvalCategory = "I8_edge_case"     // 边界场景
)

// InferenceScenario 推断测试场景
type InferenceScenario struct {
	ID          int
	Category    InferenceEvalCategory
	Description string // 场景描述

	// 模拟输入
	SensorData   map[string]interface{} // 注入 Redis 的传感器数据
	TimeContext  string                 // 时间上下文，如 "14:30 周三"
	HasSchedule  bool                   // 是否有时刻表
	ScheduleItem *MockScheduleItem      // 时刻表当前项
	HasHistory   bool                   // 是否有历史数据
	HasMemory    bool                   // 是否有 CoreMemory
	IsNewUser    bool                   // 是否新用户

	// 期望结果
	ExpectedTools      []string // 期望按顺序调用的工具
	MustUseTools       []string // 必须使用的工具（不强制顺序）
	ForbiddenTools     []string // 禁止使用的工具
	ExpectAskUser      bool     // 是否期望向用户确认
	ExpectedActivity   string   // 期望的 activity（模糊匹配）
	ExpectedAvailable  *bool    // 期望的 is_available
	ExpectedConfidence string   // 期望的 confidence (high/medium/low)
	MustFinalize       bool     // 必须调用 finalize_status

	// 回复关键词断言（在 finalize_status 的 reasoning 中）
	ReasoningMustContain    []string
	ReasoningMustNotContain []string
}

// MockScheduleItem 模拟时刻表项
type MockScheduleItem struct {
	StartTime string
	EndTime   string
	Emoji     string
	Status    string
	IsAIGuess bool
}

// Bool 辅助函数
func boolPtr(b bool) *bool {
	return &b
}

// AllInferenceScenarios 返回所有推断测试场景（25+）
func AllInferenceScenarios() []InferenceScenario {
	var scenarios []InferenceScenario
	id := 1

	for _, group := range [][]InferenceScenario{
		infScenariosI1(),
		infScenariosI2(),
		infScenariosI3(),
		infScenariosI4(),
		infScenariosI5(),
		infScenariosI6(),
		infScenariosI7(),
		infScenariosI8(),
	} {
		for _, s := range group {
			s.ID = id
			id++
			scenarios = append(scenarios, s)
		}
	}

	return scenarios
}

// ========== I1: 正常推断（传感器数据充足）（5 个）==========

func infScenariosI1() []InferenceScenario {
	return []InferenceScenario{
		{
			Category:    InfCatNormal,
			Description: "工作日在公司工作-充足数据",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": true, "activity_type": "productivity", "session_duration_minutes": 45},
				"location": map[string]interface{}{"place_type": "work", "at_place_since_minutes": 120},
				"device":   map[string]interface{}{"battery_level": 75, "is_focus_mode_on": false},
			},
			TimeContext:        "14:30 周三",
			MustUseTools:       []string{"query_sensor_data", "finalize_status"},
			ForbiddenTools:     []string{"ask_user_confirmation"},
			MustFinalize:       true,
			ExpectedConfidence: "high",
			ExpectedAvailable:  boolPtr(false),
		},
		{
			Category:    InfCatNormal,
			Description: "晚上在家休息-充足数据",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": true, "activity_type": "entertainment", "session_duration_minutes": 30},
				"location": map[string]interface{}{"place_type": "home", "at_place_since_minutes": 90},
				"device":   map[string]interface{}{"battery_level": 60, "is_charging": true},
			},
			TimeContext:        "21:00 周三",
			MustUseTools:       []string{"query_sensor_data", "finalize_status"},
			ForbiddenTools:     []string{"ask_user_confirmation"},
			MustFinalize:       true,
			ExpectedConfidence: "high",
			ExpectedAvailable:  boolPtr(true),
		},
		{
			Category:    InfCatNormal,
			Description: "通勤中-运动数据+位置",
			SensorData: map[string]interface{}{
				"location": map[string]interface{}{"place_type": "transit", "at_place_since_minutes": 15},
				"movement": map[string]interface{}{"is_moving": true, "movement_type": "automotive"},
				"screen":   map[string]interface{}{"is_active": false, "last_active_minutes_ago": 10},
			},
			TimeContext:        "08:30 周四",
			MustUseTools:       []string{"query_sensor_data", "finalize_status"},
			MustFinalize:       true,
			ExpectedConfidence: "high",
		},
		{
			Category:    InfCatNormal,
			Description: "午休时间-屏幕不活跃+在公司",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": false, "last_active_minutes_ago": 20, "activity_type": "idle"},
				"location": map[string]interface{}{"place_type": "work", "at_place_since_minutes": 240},
			},
			TimeContext:        "12:30 周五",
			MustUseTools:       []string{"query_sensor_data", "finalize_status"},
			MustFinalize:       true,
			ExpectedAvailable:  boolPtr(true),
			ExpectedConfidence: "medium",
		},
		{
			Category:    InfCatNormal,
			Description: "周末在商场-娱乐场所",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": true, "activity_type": "communication", "session_duration_minutes": 5},
				"location": map[string]interface{}{"place_type": "leisure", "place_name": "三里屯", "at_place_since_minutes": 60},
			},
			TimeContext:        "15:00 周六",
			MustUseTools:       []string{"query_sensor_data", "finalize_status"},
			MustFinalize:       true,
			ExpectedAvailable:  boolPtr(true),
		},
	}
}

// ========== I2: 部分数据缺失（4 个）==========

func infScenariosI2() []InferenceScenario {
	return []InferenceScenario{
		{
			Category:    InfCatPartialData,
			Description: "只有位置数据-无屏幕数据",
			SensorData: map[string]interface{}{
				"location": map[string]interface{}{"place_type": "work", "at_place_since_minutes": 180},
			},
			TimeContext:    "15:00 周二",
			MustUseTools:   []string{"query_sensor_data", "finalize_status"},
			HasHistory:     true,
			MustFinalize:   true,
			ExpectAskUser:  false,
			ReasoningMustContain: []string{"位置"},
		},
		{
			Category:    InfCatPartialData,
			Description: "只有屏幕数据-无位置",
			SensorData: map[string]interface{}{
				"screen": map[string]interface{}{"is_active": true, "activity_type": "entertainment", "session_duration_minutes": 60},
			},
			TimeContext:  "20:00 周四",
			MustUseTools: []string{"query_sensor_data", "finalize_status"},
			MustFinalize: true,
		},
		{
			Category:    InfCatPartialData,
			Description: "传感器数据过期-last_active 很久",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": false, "last_active_minutes_ago": 120},
				"location": map[string]interface{}{"place_type": "unknown", "at_place_since_minutes": 0},
			},
			TimeContext:    "16:00 周三",
			MustUseTools:   []string{"query_sensor_data"},
			MustFinalize:   true,
			ExpectedConfidence: "low",
		},
		{
			Category:    InfCatPartialData,
			Description: "只有设备状态-专注模式开启",
			SensorData: map[string]interface{}{
				"device": map[string]interface{}{"is_focus_mode_on": true, "battery_level": 90},
			},
			TimeContext:        "10:00 周一",
			MustUseTools:       []string{"query_sensor_data", "finalize_status"},
			MustFinalize:       true,
			ExpectedAvailable:  boolPtr(false),
		},
	}
}

// ========== I3: 无传感器数据/新用户（3 个）==========

func infScenariosI3() []InferenceScenario {
	return []InferenceScenario{
		{
			Category:       InfCatNoData,
			Description:    "新用户-无任何数据-应主动问用户",
			IsNewUser:      true,
			TimeContext:     "14:00 周三",
			MustUseTools:   []string{"query_sensor_data", "finalize_status"},
			ExpectAskUser:  true,
			MustFinalize:   true,
			ExpectedConfidence: "low",
		},
		{
			Category:       InfCatNoData,
			Description:    "新用户-凌晨-可基于时间推断",
			IsNewUser:      true,
			TimeContext:     "02:30 周四",
			MustUseTools:   []string{"query_sensor_data", "finalize_status"},
			MustFinalize:   true,
			ForbiddenTools: []string{"ask_user_confirmation"},
			ExpectedConfidence: "medium",
		},
		{
			Category:       InfCatNoData,
			Description:    "新用户-周末上午-不确定",
			IsNewUser:      true,
			TimeContext:     "10:00 周六",
			MustUseTools:   []string{"query_sensor_data", "finalize_status"},
			MustFinalize:   true,
			ExpectedConfidence: "low",
		},
	}
}

// ========== I4: 数据矛盾（4 个）==========

func infScenariosI4() []InferenceScenario {
	return []InferenceScenario{
		{
			Category:    InfCatConflict,
			Description: "在公司但已过下班时间-矛盾",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": true, "activity_type": "productivity", "session_duration_minutes": 30},
				"location": map[string]interface{}{"place_type": "work", "at_place_since_minutes": 600},
			},
			TimeContext:   "21:30 周三",
			MustUseTools:  []string{"query_sensor_data"},
			MustFinalize:  true,
			ExpectAskUser: true,
		},
		{
			Category:    InfCatConflict,
			Description: "位置显示在家但日历有外出事件",
			SensorData: map[string]interface{}{
				"location": map[string]interface{}{"place_type": "home", "at_place_since_minutes": 30},
				"calendar": map[string]interface{}{"has_current_event": true, "current_event_title": "客户拜访-国贸"},
			},
			TimeContext:  "14:00 周二",
			MustUseTools: []string{"query_sensor_data"},
			MustFinalize: true,
		},
		{
			Category:    InfCatConflict,
			Description: "屏幕娱乐+工作时间+在公司-摸鱼?",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": true, "activity_type": "entertainment", "session_duration_minutes": 45},
				"location": map[string]interface{}{"place_type": "work", "at_place_since_minutes": 120},
			},
			TimeContext:    "10:30 周二",
			MustUseTools:   []string{"query_sensor_data", "finalize_status"},
			MustFinalize:   true,
			ExpectedAvailable: boolPtr(false),
		},
		{
			Category:    InfCatConflict,
			Description: "运动中但屏幕活跃-乘车看手机?",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": true, "activity_type": "entertainment", "session_duration_minutes": 20},
				"movement": map[string]interface{}{"is_moving": true, "movement_type": "automotive"},
				"location": map[string]interface{}{"place_type": "transit"},
			},
			TimeContext:  "18:00 周五",
			MustUseTools: []string{"query_sensor_data", "finalize_status"},
			MustFinalize: true,
		},
	}
}

// ========== I5: 时刻表辅助（3 个）==========

func infScenariosI5() []InferenceScenario {
	return []InferenceScenario{
		{
			Category:    InfCatScheduleHint,
			Description: "时刻表显示开会-传感器吻合",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": false, "last_active_minutes_ago": 30},
				"location": map[string]interface{}{"place_type": "work", "at_place_since_minutes": 240},
			},
			TimeContext:  "14:30 周三",
			HasSchedule:  true,
			ScheduleItem: &MockScheduleItem{StartTime: "14:00", EndTime: "16:00", Emoji: "💼", Status: "开会", IsAIGuess: false},
			MustUseTools: []string{"query_sensor_data", "query_schedule_context", "finalize_status"},
			MustFinalize: true,
			ExpectedConfidence: "high",
			ExpectedAvailable:  boolPtr(false),
		},
		{
			Category:    InfCatScheduleHint,
			Description: "时刻表是AI猜的-传感器数据优先",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": true, "activity_type": "entertainment", "session_duration_minutes": 30},
				"location": map[string]interface{}{"place_type": "home", "at_place_since_minutes": 60},
			},
			TimeContext:  "15:00 周三",
			HasSchedule:  true,
			ScheduleItem: &MockScheduleItem{StartTime: "14:00", EndTime: "17:00", Emoji: "💻", Status: "写代码", IsAIGuess: true},
			MustUseTools: []string{"query_sensor_data", "query_schedule_context", "finalize_status"},
			MustFinalize: true,
			ReasoningMustContain: []string{"传感器"},
		},
		{
			Category:    InfCatScheduleHint,
			Description: "无传感器但有用户设定时刻表",
			TimeContext:  "10:00 周一",
			HasSchedule:  true,
			ScheduleItem: &MockScheduleItem{StartTime: "09:00", EndTime: "12:00", Emoji: "💼", Status: "工作", IsAIGuess: false},
			MustUseTools: []string{"query_sensor_data", "query_schedule_context", "finalize_status"},
			MustFinalize: true,
			ExpectedConfidence: "high",
			ForbiddenTools: []string{"ask_user_confirmation"},
		},
	}
}

// ========== I6: 历史规律利用（3 个）==========

func infScenariosI6() []InferenceScenario {
	return []InferenceScenario{
		{
			Category:    InfCatHistory,
			Description: "有历史规律-同时间段通常在工作",
			SensorData: map[string]interface{}{
				"location": map[string]interface{}{"place_type": "work", "at_place_since_minutes": 60},
			},
			TimeContext:  "10:00 周二",
			HasHistory:   true,
			HasMemory:    true,
			MustUseTools: []string{"query_sensor_data", "query_status_history", "finalize_status"},
			MustFinalize: true,
		},
		{
			Category:    InfCatHistory,
			Description: "有修正记录-用户上次在这种场景修正过",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": false, "last_active_minutes_ago": 15},
				"location": map[string]interface{}{"place_type": "work", "at_place_since_minutes": 300},
			},
			TimeContext:  "18:00 周五",
			HasHistory:   true,
			MustUseTools: []string{"query_sensor_data", "query_status_history", "finalize_status"},
			MustFinalize: true,
		},
		{
			Category:    InfCatHistory,
			Description: "新用户无历史-应降低置信度",
			SensorData: map[string]interface{}{
				"location": map[string]interface{}{"place_type": "home", "at_place_since_minutes": 30},
			},
			TimeContext:        "20:00 周三",
			IsNewUser:          true,
			MustUseTools:       []string{"query_sensor_data", "finalize_status"},
			MustFinalize:       true,
			ExpectedConfidence: "medium",
		},
	}
}

// ========== I7: 用户修正场景（2 个）==========

func infScenariosI7() []InferenceScenario {
	return []InferenceScenario{
		{
			Category:    InfCatCorrection,
			Description: "用户修正后相同场景-应使用修正后的结果",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": true, "activity_type": "productivity", "session_duration_minutes": 60},
				"location": map[string]interface{}{"place_type": "home", "at_place_since_minutes": 120},
			},
			TimeContext:  "10:00 周三",
			HasHistory:   true,
			MustUseTools: []string{"query_sensor_data", "query_status_history", "finalize_status"},
			MustFinalize: true,
			ReasoningMustContain: []string{"修正"},
		},
		{
			Category:    InfCatCorrection,
			Description: "多次被修正的模式-应问用户确认",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": false, "last_active_minutes_ago": 30},
				"location": map[string]interface{}{"place_type": "work", "at_place_since_minutes": 480},
			},
			TimeContext:   "19:00 周四",
			HasHistory:    true,
			MustUseTools:  []string{"query_sensor_data", "query_status_history"},
			MustFinalize:  true,
			ExpectAskUser: true,
		},
	}
}

// ========== I8: 边界场景（4 个）==========

func infScenariosI8() []InferenceScenario {
	return []InferenceScenario{
		{
			Category:       InfCatEdgeCase,
			Description:    "凌晨3点-绝大多数人在睡觉",
			SensorData: map[string]interface{}{
				"screen": map[string]interface{}{"is_active": false, "last_active_minutes_ago": 180},
				"device": map[string]interface{}{"is_charging": true},
			},
			TimeContext:        "03:00 周四",
			MustUseTools:       []string{"query_sensor_data", "finalize_status"},
			MustFinalize:       true,
			ForbiddenTools:     []string{"ask_user_confirmation"},
			ExpectedConfidence: "high",
			ExpectedAvailable:  boolPtr(false),
		},
		{
			Category:    InfCatEdgeCase,
			Description: "节假日-不同于工作日的推断逻辑",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": true, "activity_type": "entertainment", "session_duration_minutes": 40},
				"location": map[string]interface{}{"place_type": "home", "at_place_since_minutes": 180},
			},
			TimeContext:        "10:00 春节",
			MustUseTools:       []string{"query_sensor_data", "finalize_status"},
			MustFinalize:       true,
			ExpectedAvailable:  boolPtr(true),
		},
		{
			Category:    InfCatEdgeCase,
			Description: "所有传感器返回空-极端降级",
			TimeContext: "14:00 周二",
			MustUseTools:       []string{"query_sensor_data"},
			MustFinalize:       true,
			ExpectedConfidence: "low",
		},
		{
			Category:    InfCatEdgeCase,
			Description: "深夜但屏幕活跃-可能熬夜",
			SensorData: map[string]interface{}{
				"screen":   map[string]interface{}{"is_active": true, "activity_type": "entertainment", "session_duration_minutes": 90},
				"location": map[string]interface{}{"place_type": "home", "at_place_since_minutes": 300},
			},
			TimeContext:  "01:00 周六",
			MustUseTools: []string{"query_sensor_data", "finalize_status"},
			MustFinalize: true,
		},
	}
}
