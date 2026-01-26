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
	PlaceType            PlaceType `json:"place_type"`              // 位置类型
	AtPlaceSinceMinutes  int       `json:"at_place_since_minutes"`  // 在此位置待了多久(分钟)
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
