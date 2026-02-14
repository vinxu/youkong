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

	// 个性化人设描述（由每日 LLM 总结生成）
	// 例："互联网产品经理，工作日9点到公司，中午外卖，回家追剧到23点。周末睡到10点，偏宅。"
	PersonaText string `json:"persona_text" db:"persona_text"`

	// 时间模式频率表（JSON）
	// 格式: {"weekday":{"9":{"工作":5},"20":{"追剧":3}},"weekend":{"10":{"赖床":2}},"stats":{...}}
	TimePatternStats string `json:"time_pattern_stats" db:"time_pattern_stats"`

	// persona 最近生成时间
	PersonaGeneratedAt *time.Time `json:"persona_generated_at" db:"persona_generated_at"`

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

// CalendarData 日历数据
type CalendarData struct {
	HasCurrentEvent     bool   `json:"has_current_event"`               // 当前是否有日程
	CurrentEventTitle   string `json:"current_event_title,omitempty"`   // 当前日程标题（脱敏后）
	EventEndMinutes     int    `json:"event_end_minutes,omitempty"`     // 当前日程还有多久结束（分钟）
	NextEventInMinutes  int    `json:"next_event_in_minutes,omitempty"` // 下个日程还有多久开始（-1表示无）
	TodayRemainingCount int    `json:"today_remaining_count"`           // 今天剩余日程数量
}

// MovementData 移动状态数据
type MovementData struct {
	IsMoving           bool    `json:"is_moving"`                       // 是否在移动
	MovementType       string  `json:"movement_type,omitempty"`         // stationary/walking/running/driving/cycling
	StepsToday         int     `json:"steps_today,omitempty"`           // 今日步数
	StepsLastHour      int     `json:"steps_last_hour,omitempty"`       // 最近1小时步数
	StationaryMinutes  int     `json:"stationary_minutes,omitempty"`    // 静止了多久（分钟）
	MovingDirection    string  `json:"moving_direction,omitempty"`      // 移动方向：to_work/to_home/unknown
	CurrentSpeedKmH    float64 `json:"current_speed_kmh,omitempty"`     // 当前速度（km/h）
}

// AltitudeData 海拔/气压计数据
type AltitudeData struct {
	Altitude        float64 `json:"altitude"`                   // 当前海拔（米）
	RelativeAltitude float64 `json:"relative_altitude,omitempty"` // 相对海拔变化（米）
	Floor           int     `json:"floor,omitempty"`            // 推测楼层（如果可计算）
	IsClimbing      bool    `json:"is_climbing,omitempty"`      // 是否在爬楼/上升
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

// ExtendedLocationData 扩展的位置数据（包含精确坐标，仅用于后端计算）
type ExtendedLocationData struct {
	PlaceType           PlaceType `json:"place_type"`                      // 位置类型
	PlaceName           string    `json:"place_name,omitempty"`            // 地点名称（如"星巴克(三里屯店)"）
	AtPlaceSinceMinutes int       `json:"at_place_since_minutes"`          // 在此位置待了多久(分钟)
	Latitude            float64   `json:"latitude,omitempty"`              // 纬度（用于计算，不传给 LLM）
	Longitude           float64   `json:"longitude,omitempty"`             // 经度（用于计算，不传给 LLM）
}

// ExtendedStatusReportRequest 扩展的状态上报请求（包含所有数据维度）
type ExtendedStatusReportRequest struct {
	Screen           *ScreenData           `json:"screen"`
	Location         *LocationData         `json:"location"`
	ExtendedLocation *ExtendedLocationData `json:"extended_location,omitempty"` // 扩展位置数据
	Battery          *BatteryData          `json:"battery"`
	Mode             *ModeData             `json:"mode"`
	Connection       *ConnectionData       `json:"connection"`
	Display          *DisplayData          `json:"display"`
	Calendar         *CalendarData         `json:"calendar,omitempty"`  // 日历数据
	Movement         *MovementData         `json:"movement,omitempty"`  // 移动状态
	Altitude         *AltitudeData         `json:"altitude,omitempty"`  // 海拔数据
}

// ========== 分析结果数据结构 ==========

// LifeStatus 生活状态
type LifeStatus struct {
	Emoji       string `json:"emoji"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	GifURL      string `json:"gif_url,omitempty"`      // GIF 动图 URL（仅有空时）
	GiphyQuery  string `json:"giphy_query,omitempty"`   // Giphy 搜索词
	UseGif      bool   `json:"use_gif"`                 // 是否使用 GIF 显示模式
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
	Availability AvailabilityAnalysis `json:"availability"`          // 保留用于兼容
	LifeStatus   LifeStatus           `json:"life_status"`           // 主要状态（emoji + label + description）
	Mood         string               `json:"mood,omitempty"`        // 心情（如"开心"、"平静"、"焦虑"）
	Activity     string               `json:"activity,omitempty"`    // 活动（如"工作"、"运动"、"休息"）
	Context      string               `json:"context,omitempty"`     // 上下文描述
	MemoryUpdate *MemoryUpdate        `json:"memory_update,omitempty"`
	UpdatedAt    time.Time            `json:"updated_at,omitempty"` // 缓存更新时间（用于时效检查）
	IsAIGuess    bool                 `json:"is_ai_guess,omitempty"` // 是否为 AI 推测的状态（false=用户设置，true=AI推测）
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
	FriendID    string `json:"friend_id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar,omitempty"`
	Probability int    `json:"probability"`
	Confidence  string `json:"confidence"`
	Reason      string `json:"reason"`
	Color       string `json:"color"`
	Emoji       string `json:"emoji,omitempty"`    // 生活状态 emoji
	Activity    string `json:"activity,omitempty"` // 正在做什么
	UpdatedAt   int64  `json:"updated_at"`
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
	"gaming":     {Emoji: "🎮", Label: "在玩游戏"},
	"watching":   {Emoji: "📺", Label: "在追剧"},
	"working":    {Emoji: "💼", Label: "在工作"},
	"slacking":   {Emoji: "☕", Label: "在摸鱼"},
	"eating":     {Emoji: "🍜", Label: "在吃饭"},
	"resting":    {Emoji: "🛋️", Label: "在家躺着"},
	"walking":    {Emoji: "🚶", Label: "在外面逛"},
	"sleeping":   {Emoji: "😴", Label: "可能在睡觉"},
	"scrolling":  {Emoji: "📱", Label: "在刷手机"},
	"chatting":   {Emoji: "💬", Label: "在聊天"},
	"exercising": {Emoji: "🏃", Label: "在运动"},
	"partying":   {Emoji: "🍻", Label: "可能在聚会"},
	"listening":  {Emoji: "🎧", Label: "在听音乐"},
	"busy":       {Emoji: "🔕", Label: "不想被打扰"},
	"lowbattery": {Emoji: "🪫", Label: "电量告急"},
	"unknown":    {Emoji: "🤔", Label: "状态未知"},
	"cafe":       {Emoji: "☕", Label: "在咖啡厅"},
	"commuting":  {Emoji: "🚇", Label: "在通勤"},
	"meeting":    {Emoji: "📊", Label: "在开会"},
	"shopping":   {Emoji: "🛍️", Label: "在逛街"},
}

// ========== 福尔摩斯推理框架数据结构 ==========

// HolmesClue 福尔摩斯线索 - Layer 1 原始数据
type HolmesClue struct {
	// 时间线索
	Timestamp   time.Time `json:"timestamp"`
	Weekday     string    `json:"weekday"`      // 星期几（中文）
	TimePeriod  string    `json:"time_period"`  // 时间段描述（如"下午3点42分"）
	IsWeekend   bool      `json:"is_weekend"`

	// 位置线索
	PlaceName          string  `json:"place_name,omitempty"`            // 地点名称
	PlaceType          string  `json:"place_type"`                      // 位置类型
	AtPlaceSinceMinutes int    `json:"at_place_since_minutes"`          // 在此地多久
	Altitude           float64 `json:"altitude,omitempty"`              // 海拔（米）
	Floor              int     `json:"floor,omitempty"`                 // 推测楼层

	// 移动线索
	IsMoving          bool   `json:"is_moving"`
	MovementType      string `json:"movement_type,omitempty"`      // stationary/walking/running/driving
	StepsToday        int    `json:"steps_today,omitempty"`
	StepsLastHour     int    `json:"steps_last_hour,omitempty"`
	StationaryMinutes int    `json:"stationary_minutes,omitempty"`

	// 屏幕线索
	ScreenActive        bool   `json:"screen_active"`
	ScreenDurationMins  int    `json:"screen_duration_mins,omitempty"`
	ActivityType        string `json:"activity_type,omitempty"`          // entertainment/productivity/communication/idle
	LastActiveMinutesAgo int   `json:"last_active_minutes_ago,omitempty"`

	// 日历线索
	HasCalendarEvent     bool   `json:"has_calendar_event"`
	CalendarEventTitle   string `json:"calendar_event_title,omitempty"`   // 脱敏后的标题
	EventEndMinutes      int    `json:"event_end_minutes,omitempty"`
	TodayRemainingEvents int    `json:"today_remaining_events"`

	// 设备线索
	HeadphonesConnected bool `json:"headphones_connected,omitempty"`
	FocusModeOn         bool `json:"focus_mode_on,omitempty"`
	LowBatteryMode      bool `json:"low_battery_mode,omitempty"`
	BatteryLevel        int  `json:"battery_level,omitempty"`
}

// HolmesFeatures 福尔摩斯特征层 - Layer 2 提取的特征
type HolmesFeatures struct {
	LocationType  string `json:"location_type"`   // 咖啡厅/办公室/家/商场/...
	MovementState string `json:"movement_state"`  // 静止/步行/乘车/跑步
	TimePeriod    string `json:"time_period"`     // 工作日上午/周末下午/...
	Activity      string `json:"activity"`        // 娱乐内容/工作内容/通讯/闲置
	Schedule      string `json:"schedule"`        // 有会议/无日程/会议即将结束
	DeviceState   string `json:"device_state"`    // 专注模式/低电量/戴耳机/...
}

// HolmesReasoning 福尔摩斯推理过程 - Layer 3 推理结果
type HolmesReasoning struct {
	Model     string `json:"model"`     // 使用的模型
	Thinking  string `json:"thinking"`  // 思考过程（<think>标签内容）
	Conclusion string `json:"conclusion"` // 推理结论
}

// HolmesResult 福尔摩斯完整分析结果
type HolmesResult struct {
	// 原始数据（Layer 1）
	RawData *HolmesClue `json:"raw_data"`

	// 特征提取（Layer 2）
	Features *HolmesFeatures `json:"features"`

	// 推理过程（Layer 3）
	Reasoning *HolmesReasoning `json:"reasoning"`

	// 最终结果
	Result struct {
		Available   bool   `json:"available"`
		Probability int    `json:"probability"`   // 0-100
		Confidence  string `json:"confidence"`    // high/medium/low
		Summary     string `json:"summary"`       // 简短总结（15字以内）
		Emoji       string `json:"emoji"`         // 状态 emoji
	} `json:"result"`

	// 生成时间
	GeneratedAt time.Time `json:"generated_at"`
}

// HolmesAPIResponse 福尔摩斯 API 响应（前端展示用）
type HolmesAPIResponse struct {
	FriendID string `json:"friend_id"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar,omitempty"`

	// 原始线索（供前端展示推理过程）
	RawData *HolmesClue `json:"raw_data,omitempty"`

	// 特征层
	Features *HolmesFeatures `json:"features,omitempty"`

	// 推理过程
	Reasoning *HolmesReasoning `json:"reasoning,omitempty"`

	// 最终结果
	Result struct {
		Available   bool   `json:"available"`
		Probability int    `json:"probability"`
		Confidence  string `json:"confidence"`
		Summary     string `json:"summary"`
		Emoji       string `json:"emoji,omitempty"`
		Color       string `json:"color"`
	} `json:"result"`

	UpdatedAt int64 `json:"updated_at"`
}

// HolmesFriendListResponse 福尔摩斯好友列表响应
type HolmesFriendListResponse struct {
	Friends     []HolmesAPIResponse `json:"friends"`
	GeneratedAt int64               `json:"generated_at"`
}

// ========== Holmes 2.0 语义上下文建模 ==========

// SpaceSemantic 空间语义 - 描述"这是什么样的地方"
type SpaceSemantic struct {
	Nature string `json:"nature"` // "私密空间"/"公共空间"/"移动中"/"户外"
	Vibe   string `json:"vibe"`   // "安静"/"喧嚣"/"专业"/"休闲"
	Social string `json:"social"` // "独处"/"可能有他人"/"社交场合"
}

// TimeSemantic 时间语义 - 描述"这是什么样的时刻"
type TimeSemantic struct {
	Phase      string `json:"phase"`      // "苏醒期"/"高效期"/"放松期"/"入睡期"
	Rhythm     string `json:"rhythm"`     // "工作节奏"/"休闲节奏"/"过渡期"
	Continuity string `json:"continuity"` // "刚开始"/"进行中"/"即将结束"
}

// ActivitySemantic 活动语义 - 描述"人在做什么类型的事"
type ActivitySemantic struct {
	BodyState  string `json:"body_state"`  // "静态"/"轻度活动"/"运动中"/"移动中"
	MindState  string `json:"mind_state"`  // "专注"/"消遣"/"社交"/"休息"
	Engagement string `json:"engagement"`  // "深度投入"/"浅层互动"/"被动接收"/"闲置"
}

// EnergyLevel 能量状态 - 描述"此刻的能量水平"
type EnergyLevel struct {
	Physical string `json:"physical"` // "充沛"/"正常"/"疲惫"
	Mental   string `json:"mental"`   // "清醒"/"平静"/"低迷"
	Social   string `json:"social"`   // "开放"/"中性"/"封闭"
}

// SemanticContext 语义上下文 - Layer 2 从原始数据提取的语义
type SemanticContext struct {
	Space    *SpaceSemantic    `json:"space"`
	Time     *TimeSemantic     `json:"time"`
	Activity *ActivitySemantic `json:"activity"`
	Energy   *EnergyLevel      `json:"energy"`
}

// Anomaly 异常标记
type Anomaly struct {
	Type   string `json:"type"`   // "unusual_location"/"unusual_time"/"behavior_change"
	Detail string `json:"detail"` // "通常这时候在家，今天在外面"
}

// MoodVector 心情向量 - 连续维度表示
type MoodVector struct {
	Valence  float64 `json:"valence"`  // 效价：-1(消极) 到 1(积极)
	Arousal  float64 `json:"arousal"`  // 唤醒度：0(平静) 到 1(激动)
	Openness float64 `json:"openness"` // 社交开放度：0(封闭) 到 1(开放)
}

// HolmesCreativeResult Holmes 2.0 创意叙事结果
type HolmesCreativeResult struct {
	Narrative   string      `json:"narrative"`    // 叙事推理过程
	Scene       string      `json:"scene"`        // 场景描述（展示用）
	Emoji       string      `json:"emoji"`        // 表情
	Mood        *MoodVector `json:"mood"`         // 心情向量
	Confidence  string      `json:"confidence"`   // 置信度
	Basis       []string    `json:"basis"`        // 推断依据
	GeneratedAt int64       `json:"generated_at"` // 生成时间戳
}

// InferenceContext 推断上下文 - 传给 LLM 的完整上下文
type InferenceContext struct {
	Current      *SemanticContext `json:"current"`       // 当前上下文
	Memory       *CoreMemory      `json:"memory"`        // 历史记忆
	RecentStates []*RecentState   `json:"recent_states"` // 最近状态序列
	Anomalies    []Anomaly        `json:"anomalies"`     // 异常标记
}

// RecentState 最近的状态快照
type RecentState struct {
	Timestamp int64  `json:"timestamp"` // 毫秒时间戳
	Summary   string `json:"summary"`   // 之前的状态摘要
	Duration  int    `json:"duration"`  // 持续多久（分钟）
}

// Holmes2Result Holmes 2.0 完整分析结果
type Holmes2Result struct {
	// 原始数据（Layer 1）
	RawData *HolmesClue `json:"raw_data"`

	// 语义上下文（Layer 2）
	Context *SemanticContext `json:"context"`

	// 异常检测
	Anomalies []Anomaly `json:"anomalies,omitempty"`

	// 创意叙事结果（Layer 4）
	Creative *HolmesCreativeResult `json:"creative"`

	// 兼容旧版结果（供宫格显示）
	Result struct {
		Available   bool   `json:"available"`
		Probability int    `json:"probability"`
		Confidence  string `json:"confidence"`
		Summary     string `json:"summary"`
		Emoji       string `json:"emoji"`
	} `json:"result"`

	// 生成时间
	GeneratedAt int64 `json:"generated_at"`
}

// ========== 训练 AI - 状态选择功能 ==========

// StatusOption 状态选项
type StatusOption struct {
	Emoji  string `json:"emoji"`            // 1-2 个 emoji（场所+行为），如 "🏠🍽️"
	Status string `json:"status"`           // 完整描述，如 "在家里吃饭"
	Place  string `json:"place,omitempty"`  // 可选：场所名称（用于分离展示），如 "家里"
	Action string `json:"action,omitempty"` // 可选：行为名称（用于分离展示），如 "吃饭"
}

// StatusOptionsResult 状态选项生成结果
type StatusOptionsResult struct {
	Options []StatusOption `json:"options"` // 4 个状态选项
}

// SelectStatusRequest 选择状态请求
type SelectStatusRequest struct {
	Emoji      string                       `json:"emoji" binding:"required"`
	Status     string                       `json:"status" binding:"required"`
	DeviceData *ExtendedStatusReportRequest `json:"device_data,omitempty"` // 选择时的设备数据
}

// UserStatusMemory 用户状态记忆（数据库模型）
type UserStatusMemory struct {
	ID        int64     `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Emoji     string    `json:"emoji" db:"emoji"`
	Status    string    `json:"status" db:"status"`
	Context   string    `json:"context" db:"context"` // JSON 格式的设备上下文
	Source    string    `json:"source" db:"source"`   // 来源: user_confirmed | ai_auto | ai_oneclick | unknown
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// DailyActivitySummary 日维度传感器摘要
type DailyActivitySummary struct {
	ID                int64   `json:"id" db:"id"`
	UserID            string  `json:"user_id" db:"user_id"`
	SummaryDate       string  `json:"summary_date" db:"summary_date"`
	HomeHours         float64 `json:"home_hours" db:"home_hours"`
	WorkHours         float64 `json:"work_hours" db:"work_hours"`
	TransitHours      float64 `json:"transit_hours" db:"transit_hours"`
	ScreenActiveHours float64 `json:"screen_active_hours" db:"screen_active_hours"`
	TotalSteps        int     `json:"total_steps" db:"total_steps"`
	MostActivePeriod  string  `json:"most_active_period" db:"most_active_period"`
	SampleCount       int     `json:"sample_count" db:"sample_count"`
	TextSummary       string  `json:"text_summary" db:"text_summary"`
	CreatedAt         string  `json:"created_at" db:"created_at"`
}
