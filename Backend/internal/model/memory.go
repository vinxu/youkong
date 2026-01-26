package model

import "time"

// CoreMemory 核心记忆 - 每个用户一份，存储 LLM 提炼的关键洞察
type CoreMemory struct {
	ID     int64  `json:"id" db:"id"`
	UserID string `json:"user_id" db:"user_id"`

	// 行为模式洞察（LLM 总结）
	// 例："工作日晚上8-10点通常在家刷手机，周末下午喜欢外出"
	BehaviorInsights string `json:"behavior_insights" db:"behavior_insights"`

	// 关键时间规律
	// 例："午休12:30-13:30活跃度高"，"周末起床较晚，通常10点后才活跃"
	TimePatterns string `json:"time_patterns" db:"time_patterns"`

	// 地点偏好
	// 例："工作日主要在公司和家之间"，"周末经常去三里屯附近"
	LocationPreferences string `json:"location_preferences" db:"location_preferences"`

	// 社交倾向
	// 例："响应速度快，平均5分钟内回复"，"晚上比较愿意社交"
	SocialTendency string `json:"social_tendency" db:"social_tendency"`

	// 置信度（数据量越大越高，0-100）
	ConfidenceScore int `json:"confidence_score" db:"confidence_score"`

	// 样本数量
	SampleCount int `json:"sample_count" db:"sample_count"`

	// 更新时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// StatusHistory 状态历史记录 - 保存每次上报的原始数据
type StatusHistory struct {
	ID     int64  `json:"id" db:"id"`
	UserID string `json:"user_id" db:"user_id"`

	// 原始上报数据（JSON 存储完整 StatusReport）
	RawData string `json:"raw_data" db:"raw_data"`

	// 上报时的上下文
	DayOfWeek int  `json:"day_of_week" db:"day_of_week"` // 0-6 (Sunday=0)
	HourOfDay int  `json:"hour_of_day" db:"hour_of_day"` // 0-23
	IsWeekend bool `json:"is_weekend" db:"is_weekend"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ========== 扩展的状态上报数据结构 ==========

// BatteryData 电池数据
type BatteryData struct {
	BatteryLevel int    `json:"battery_level"` // 0-100
	BatteryState string `json:"battery_state"` // charging/unplugged/full
	IsCharging   bool   `json:"is_charging"`
}

// ModeData 模式数据
type ModeData struct {
	IsLowPowerMode bool `json:"is_low_power_mode"`
	IsFocusModeOn  bool `json:"is_focus_mode_on"`
}

// ConnectionData 连接数据
type ConnectionData struct {
	IsHeadphonesConnected bool   `json:"is_headphones_connected"`
	NetworkType           string `json:"network_type"` // wifi/cellular/none
}

// DisplayData 显示数据
type DisplayData struct {
	ScreenBrightness float64 `json:"screen_brightness"` // 0.0-1.0
}

// ExtendedStatusReportRequest 扩展的状态上报请求（包含所有数据维度）
type ExtendedStatusReportRequest struct {
	Screen     *ScreenData     `json:"screen"`
	Location   *LocationData   `json:"location"`
	Battery    *BatteryData    `json:"battery"`
	Mode       *ModeData       `json:"mode"`
	Connection *ConnectionData `json:"connection"`
	Display    *DisplayData    `json:"display"`
}

// ========== 分析结果数据结构 ==========

// LifeStatus 生活状态
type LifeStatus struct {
	Emoji       string `json:"emoji"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// AvailabilityAnalysis 有空分析结果
type AvailabilityAnalysis struct {
	Status      string `json:"status"`      // 有空/忙碌/可能有空
	Probability int    `json:"probability"` // 0-100
	Reason      string `json:"reason"`
	Confidence  string `json:"confidence"` // high/medium/low
}

// AnalysisResult LLM 分析结果
type AnalysisResult struct {
	Availability AvailabilityAnalysis `json:"availability"`
	LifeStatus   LifeStatus           `json:"life_status"`
	MemoryUpdate *MemoryUpdate        `json:"memory_update,omitempty"`
}

// MemoryUpdate 记忆更新内容
type MemoryUpdate struct {
	BehaviorInsights    string `json:"behavior_insights,omitempty"`
	TimePatterns        string `json:"time_patterns,omitempty"`
	LocationPreferences string `json:"location_preferences,omitempty"`
	SocialTendency      string `json:"social_tendency,omitempty"`
	ShouldUpdate        bool   `json:"should_update"`
}

// ========== 增强的响应数据结构 ==========

// StatusReportResponse 状态上报响应
type StatusReportResponse struct {
	Success      bool            `json:"success"`
	NextReportIn int             `json:"next_report_in"` // 下次上报间隔（秒）
	Analysis     *AnalysisResult `json:"analysis,omitempty"`
}

// EnhancedFriendRecommendation 增强的好友推荐（带生活状态）
type EnhancedFriendRecommendation struct {
	FriendID    string      `json:"friend_id"`
	Name        string      `json:"name"`
	Avatar      string      `json:"avatar,omitempty"`
	Probability int         `json:"probability"`
	Confidence  string      `json:"confidence"`
	Reason      string      `json:"reason"`
	Color       string      `json:"color"`
	LifeStatus  *LifeStatus `json:"life_status,omitempty"`
	UpdatedAt   int64       `json:"updated_at"`
}

// EnhancedFreeProbabilityResponse 增强的有空概率响应
type EnhancedFreeProbabilityResponse struct {
	Friends     []EnhancedFriendRecommendation `json:"friends"`
	GeneratedAt int64                          `json:"generated_at"`
}

// CoreMemoryResponse 核心记忆查询响应
type CoreMemoryResponse struct {
	BehaviorInsights    string    `json:"behavior_insights"`
	TimePatterns        string    `json:"time_patterns"`
	LocationPreferences string    `json:"location_preferences"`
	SocialTendency      string    `json:"social_tendency"`
	ConfidenceScore     int       `json:"confidence_score"`
	SampleCount         int       `json:"sample_count"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ========== 生活状态 Emoji 映射 ==========

// LifeStatusEmoji 生活状态 Emoji 常量
var LifeStatusEmojis = map[string]LifeStatus{
	"gaming":    {Emoji: "🎮", Label: "在玩游戏"},
	"watching":  {Emoji: "📺", Label: "在追剧"},
	"working":   {Emoji: "💼", Label: "在工作"},
	"slacking":  {Emoji: "☕", Label: "在摸鱼"},
	"eating":    {Emoji: "🍜", Label: "在吃饭"},
	"resting":   {Emoji: "🛋️", Label: "在家躺着"},
	"walking":   {Emoji: "🚶", Label: "在外面逛"},
	"sleeping":  {Emoji: "😴", Label: "可能在睡觉"},
	"scrolling": {Emoji: "📱", Label: "在刷手机"},
	"chatting":  {Emoji: "💬", Label: "在聊天"},
	"exercising": {Emoji: "🏃", Label: "在运动"},
	"partying":  {Emoji: "🍻", Label: "可能在聚会"},
	"listening": {Emoji: "🎧", Label: "在听音乐"},
	"busy":      {Emoji: "🔕", Label: "不想被打扰"},
	"lowbattery": {Emoji: "🪫", Label: "电量告急"},
	"unknown":   {Emoji: "🤔", Label: "状态未知"},
}
