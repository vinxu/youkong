package model

import "time"

// ActivityType 屏幕使用类型
type ActivityType string

const (
	ActivityEntertainment ActivityType = "entertainment" // 娱乐类APP
	ActivityProductivity  ActivityType = "productivity"  // 工作类APP
	ActivityCommunication ActivityType = "communication" // 通讯类APP
	ActivityIdle          ActivityType = "idle"          // 闲置
)

// PlaceType 位置类型
type PlaceType string

const (
	PlaceHome    PlaceType = "home"    // 家
	PlaceWork    PlaceType = "work"    // 公司
	PlaceLeisure PlaceType = "leisure" // 休闲场所
	PlaceTransit PlaceType = "transit" // 路上
	PlaceUnknown PlaceType = "unknown" // 未知
)

// ScreenData 屏幕使用数据
type ScreenData struct {
	IsActive               bool         `json:"is_active"`                 // 当前是否在用手机
	ActivityType           ActivityType `json:"activity_type"`             // 使用类型
	SessionDurationMinutes int          `json:"session_duration_minutes"`  // 本次使用时长(分钟)
	LastActiveMinutesAgo   int          `json:"last_active_minutes_ago"`   // 上次活跃是多久前(分钟)
}

// LocationData 位置数据
type LocationData struct {
	PlaceType           PlaceType `json:"place_type"`                    // 位置类型
	AtPlaceSinceMinutes int       `json:"at_place_since_minutes"`        // 在此位置待了多久(分钟)
	City                string    `json:"city,omitempty"`                // 城市名称（客户端上报，如"上海"、"北京"）
	Latitude            float64   `json:"latitude,omitempty"`            // 纬度
	Longitude           float64   `json:"longitude,omitempty"`           // 经度
}

// UserPatterns 用户历史规律（Agent学习得到）
type UserPatterns struct {
	CurrentHourFreeRate       int `json:"current_hour_free_rate"`        // 0-100，当前小时历史有空率
	CurrentWeekdayFreeRate    int `json:"current_weekday_free_rate"`     // 0-100，今天(周几)历史有空率
	AtHomeFreeRate            int `json:"at_home_free_rate"`             // 0-100，在家时有空率
	AtWorkAfterHoursFreeRate  int `json:"at_work_after_hours_free_rate"` // 0-100，下班后在公司有空率
	AvgResponseTimeMinutes    int `json:"avg_response_time_minutes"`     // 平均回复时间(分钟)
	ResponseRate              int `json:"response_rate"`                 // 0-100，回复率
}

// DataQuality 数据质量信息
type DataQuality struct {
	ScreenDataAgeSeconds   int `json:"screen_data_age_seconds"`   // 屏幕数据多久前更新
	LocationDataAgeSeconds int `json:"location_data_age_seconds"` // 位置数据多久前更新
	PatternsSampleSize     int `json:"patterns_sample_size"`      // 历史规律基于多少样本
}

// AgentExposedData Agent对外暴露的数据（给其他Agent）
type AgentExposedData struct {
	Realtime struct {
		Screen   ScreenData   `json:"screen"`
		Location LocationData `json:"location"`
	} `json:"realtime"`
	Patterns    UserPatterns `json:"patterns"`
	DataQuality DataQuality  `json:"data_quality"`
}

// UserRealtimeStatus 用户实时状态（存储在Redis）
type UserRealtimeStatus struct {
	UserID    string       `json:"user_id"`
	Screen    ScreenData   `json:"screen"`
	Location  LocationData `json:"location"`
	City      string       `json:"city,omitempty"` // 城市名称
	UpdatedAt time.Time    `json:"updated_at"`
}

// StatusReportRequest 状态上报请求
type StatusReportRequest struct {
	Screen   *ScreenData   `json:"screen"`
	Location *LocationData `json:"location"`
}

// FriendRecommendation 好友有空推荐结果
type FriendRecommendation struct {
	FriendID    string `json:"friend_id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar,omitempty"`
	Probability int    `json:"probability"`        // 0-100
	Confidence  string `json:"confidence"`         // high/medium/low
	Reason      string `json:"reason"`             // 口语化理由，≤15字
	Color       string `json:"color"`              // 颜色代码
	UpdatedAt   int64  `json:"updated_at"`         // 数据更新时间戳
}

// FreeProbabilityResponse 有空概率列表响应
type FreeProbabilityResponse struct {
	Friends     []FriendRecommendation `json:"friends"`
	GeneratedAt int64                  `json:"generated_at"`
}

// ProbabilityColor 概率颜色映射
type ProbabilityColor struct {
	Dot        string
	Background string
}

// 颜色常量
var (
	ColorVeryLikely = ProbabilityColor{Dot: "#22C55E", Background: "#F0FDF4"} // 80-100%
	ColorLikely     = ProbabilityColor{Dot: "#86EFAC", Background: "#F0FDF4"} // 60-79%
	ColorMaybe      = ProbabilityColor{Dot: "#FACC15", Background: "#FEFCE8"} // 40-59%
	ColorUnlikely   = ProbabilityColor{Dot: "#FB923C", Background: "#FFF7ED"} // 20-39%
	ColorBusy       = ProbabilityColor{Dot: "#EF4444", Background: "#FEF2F2"} // 0-19%
	ColorUnknown    = ProbabilityColor{Dot: "#9CA3AF", Background: "#F9FAFB"} // 无数据
)

// GetProbabilityColor 根据概率返回颜色
func GetProbabilityColor(probability int) string {
	if probability < 0 {
		return ColorUnknown.Dot
	}
	if probability >= 80 {
		return ColorVeryLikely.Dot
	}
	if probability >= 60 {
		return ColorLikely.Dot
	}
	if probability >= 40 {
		return ColorMaybe.Dot
	}
	if probability >= 20 {
		return ColorUnlikely.Dot
	}
	return ColorBusy.Dot
}

// KnownPlaces 用户已知地点
type KnownPlaces struct {
	Home *GeoPoint `json:"home,omitempty"`
	Work *GeoPoint `json:"work,omitempty"`
}

// GeoPoint 地理坐标
type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// CurrentStatusInference 当下状态推理结果
type CurrentStatusInference struct {
	Emoji        string `json:"emoji"`                      // 1-3个emoji，如"🎮"或"🎮🎧"
	Place        string `json:"place,omitempty"`            // 可选，如"家里"、"公司"、"咖啡厅"
	Activity     string `json:"activity"`                   // 活动描述，如"打游戏"、"休息中"
	IsAvailable  bool   `json:"is_available"`               // 是否有空
	DurationHint string `json:"duration_hint,omitempty"`    // 预计持续时长，如"约2小时"、"半天"
	Confidence   string `json:"confidence"`                 // 置信度：high/medium/low
	InferredAt   int64  `json:"inferred_at"`                // 推理时间戳（毫秒）
	Reasoning    string `json:"reasoning,omitempty"`        // 推理依据（内部使用，可选返回）
	GifURL       string `json:"gif_url,omitempty"`          // GIF URL（仅有空时填充）
	GifSmallURL  string `json:"gif_small_url,omitempty"`    // 缩略版 GIF URL
	GiphyQuery   string `json:"giphy_query,omitempty"`      // Giphy 搜索词（英文，供客户端直接搜索）
	UseGif       bool   `json:"use_gif"`                    // 是否使用 GIF 显示模式
}

// StatusFeedbackRequest 状态反馈请求（用户修正状态）
type StatusFeedbackRequest struct {
	OriginalEmoji       string `json:"original_emoji,omitempty"`
	OriginalActivity    string `json:"original_activity,omitempty"`
	CorrectedEmoji      string `json:"corrected_emoji" binding:"required"`
	CorrectedActivity   string `json:"corrected_activity" binding:"required"`
	CorrectedPlace      string `json:"corrected_place,omitempty"`
	CorrectedIsAvailable *bool `json:"corrected_is_available,omitempty"`
	GifURL              string `json:"gif_url,omitempty"`         // 客户端获取的 GIF URL
	GiphyQuery          string `json:"giphy_query,omitempty"`     // Giphy 搜索词（用于后续重新搜索）
	UseGif              bool   `json:"use_gif"`                   // 是否使用 GIF 显示模式（默认 false 显示 emoji）
}

// StatusMemoryEntry 状态记忆条目（存储用户修正历史，用于改进推理）
type StatusMemoryEntry struct {
	ID              int64  `json:"id" db:"id"`
	UserID          string `json:"user_id" db:"user_id"`
	OriginalEmoji   string `json:"original_emoji" db:"original_emoji"`
	OriginalActivity string `json:"original_activity" db:"original_activity"`
	CorrectedEmoji  string `json:"corrected_emoji" db:"corrected_emoji"`
	CorrectedActivity string `json:"corrected_activity" db:"corrected_activity"`
	CorrectedPlace  string `json:"corrected_place" db:"corrected_place"`
	CorrectedIsAvailable bool `json:"corrected_is_available" db:"corrected_is_available"`
	DeviceContext   string `json:"device_context" db:"device_context"` // JSON: 当时的设备状态
	CreatedAt       string `json:"created_at" db:"created_at"`
}

// ========== V3 推断系统模型 ==========

// InferencePhase V3 推断响应阶段
type InferencePhase string

const (
	InferencePhaseCompleted     InferencePhase = "completed"      // 推断完成
	InferencePhaseAwaitingChoice InferencePhase = "awaiting_choice" // 等待用户选择
)

// InferenceOption 推断选项（不确定时让用户选）
type InferenceOption struct {
	Index    int    `json:"index"`
	Emoji    string `json:"emoji"`
	Activity string `json:"activity"`
	Reason   string `json:"reason,omitempty"` // 为何推测此选项
}

// InferenceSession 推断交互 session（Redis, 5min TTL）
type InferenceSession struct {
	SessionID   string              `json:"session_id"`
	UserID      string              `json:"user_id"`
	Options     []InferenceOption   `json:"options"`
	Question    string              `json:"question"`
	DefaultIdx  int                 `json:"default_index"`
	SensorData  *ExtendedStatusReportRequest `json:"sensor_data,omitempty"` // 推断时的传感器快照
	CreatedAt   int64               `json:"created_at"`
}

// UserInferencePreference 用户推断偏好（从修正历史提炼）
type UserInferencePreference struct {
	UserID      string   `json:"user_id"`
	Preferences []string `json:"preferences"` // 偏好规则列表，如 "晚上在家时更偏好说'刷剧'而不是'看手机'"
	UpdatedAt   int64    `json:"updated_at"`
}

// InferenceResponse V3 统一推断响应
type InferenceResponse struct {
	Phase     InferencePhase        `json:"phase"`
	Result    *CurrentStatusInference `json:"result,omitempty"`     // phase=completed 时有值
	SessionID string                `json:"session_id,omitempty"` // phase=awaiting_choice 时有值
	Question  string                `json:"question,omitempty"`   // phase=awaiting_choice 时有值
	Options   []InferenceOption     `json:"options,omitempty"`    // phase=awaiting_choice 时有值
	DefaultIdx int                  `json:"default_index,omitempty"`
}

// V3InferenceContext 预聚合的推断上下文（Go 代码收集，注入 system prompt）
type V3InferenceContext struct {
	DeviceSignals    map[string]interface{} `json:"device_signals,omitempty"`
	TodaySchedule    map[string]interface{} `json:"today_schedule,omitempty"`
	RecentStatuses   []map[string]string    `json:"recent_statuses,omitempty"`
	Corrections      []map[string]string    `json:"corrections,omitempty"`
	Conversation     string                 `json:"conversation,omitempty"`
	Profile          map[string]interface{} `json:"profile,omitempty"`
	CoreMemory       map[string]interface{} `json:"core_memory,omitempty"`
	PrevInference    *CurrentStatusInference `json:"prev_inference,omitempty"`
	Preferences      []string               `json:"preferences,omitempty"`
}
