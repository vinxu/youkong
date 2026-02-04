import Foundation

// MARK: - Agent Repository Protocol

protocol AgentRepositoryProtocol {
    /// 上报自己的状态，返回 LLM 预测结果
    func reportStatus(request: StatusReportRequest) async throws -> StatusReportResponse

    /// 上报状态（便捷方法，自动收集设备数据）
    func reportStatus() async throws

    /// 获取朋友有空概率列表
    func getFriendsFreeProbability() async throws -> [FriendRecommendation]

    /// 获取自己的分析结果
    func getMyAnalysis() async throws -> AnalysisData?

    /// 请求朋友 Agent 数据
    func queryAgentData(agentId: String) async throws -> AgentExposedData

    /// 获取宫格数据（所有好友的实时状态）
    func getGridData() async throws -> GridResponse
}

// MARK: - Status Report Request (按 API 规范分组)

struct StatusReportRequest: Encodable {
    let screen: ScreenRequestData?
    let location: LocationRequestData?
    let extendedLocation: ExtendedLocationRequestData?  // 扩展位置数据
    let battery: BatteryRequestData?
    let mode: ModeRequestData?
    let connection: ConnectionRequestData?
    let display: DisplayRequestData?
    let calendar: CalendarRequestData?      // 日历数据
    let movement: MovementRequestData?      // 运动数据

    enum CodingKeys: String, CodingKey {
        case screen
        case location
        case extendedLocation = "extended_location"
        case battery
        case mode
        case connection
        case display
        case calendar
        case movement
    }
}

struct ScreenRequestData: Encodable {
    let isActive: Bool
    let activityType: String
    let sessionDurationMinutes: Int
    let lastActiveMinutesAgo: Int
    let lastActiveCategory: String?  // 最近活跃的应用分类（如"游戏"、"社交"）

    enum CodingKeys: String, CodingKey {
        case isActive = "is_active"
        case activityType = "activity_type"
        case sessionDurationMinutes = "session_duration_minutes"
        case lastActiveMinutesAgo = "last_active_minutes_ago"
        case lastActiveCategory = "last_active_category"
    }
}

struct LocationRequestData: Encodable {
    let placeType: String
    let atPlaceSinceMinutes: Int
    let city: String?  // 城市名称（如"上海"、"北京"）

    enum CodingKeys: String, CodingKey {
        case placeType = "place_type"
        case atPlaceSinceMinutes = "at_place_since_minutes"
        case city
    }
}

// MARK: - 扩展位置数据（包含地点名称和坐标）

struct ExtendedLocationRequestData: Encodable {
    let placeType: String
    let placeName: String?          // 地点名称（反向地理编码）
    let atPlaceSinceMinutes: Int
    let latitude: Double?
    let longitude: Double?

    enum CodingKeys: String, CodingKey {
        case placeType = "place_type"
        case placeName = "place_name"
        case atPlaceSinceMinutes = "at_place_since_minutes"
        case latitude
        case longitude
    }
}

struct BatteryRequestData: Encodable {
    let batteryLevel: Int
    let batteryState: String
    let isCharging: Bool

    enum CodingKeys: String, CodingKey {
        case batteryLevel = "battery_level"
        case batteryState = "battery_state"
        case isCharging = "is_charging"
    }
}

struct ModeRequestData: Encodable {
    let isLowPowerMode: Bool
    let isFocusModeOn: Bool

    enum CodingKeys: String, CodingKey {
        case isLowPowerMode = "is_low_power_mode"
        case isFocusModeOn = "is_focus_mode_on"
    }
}

struct ConnectionRequestData: Encodable {
    let isHeadphonesConnected: Bool
    let networkType: String

    enum CodingKeys: String, CodingKey {
        case isHeadphonesConnected = "is_headphones_connected"
        case networkType = "network_type"
    }
}

struct DisplayRequestData: Encodable {
    let screenBrightness: Float

    enum CodingKeys: String, CodingKey {
        case screenBrightness = "screen_brightness"
    }
}

// MARK: - 日历请求数据

struct CalendarRequestData: Encodable {
    let hasCurrentEvent: Bool
    let currentEventTitle: String?
    let eventEndMinutes: Int?
    let nextEventInMinutes: Int?
    let todayRemainingCount: Int

    enum CodingKeys: String, CodingKey {
        case hasCurrentEvent = "has_current_event"
        case currentEventTitle = "current_event_title"
        case eventEndMinutes = "event_end_minutes"
        case nextEventInMinutes = "next_event_in_minutes"
        case todayRemainingCount = "today_remaining_count"
    }
}

// MARK: - 运动请求数据

struct MovementRequestData: Encodable {
    let isMoving: Bool
    let movementType: String
    let stepsToday: Int
    let stepsLastHour: Int
    let stationaryMinutes: Int

    enum CodingKeys: String, CodingKey {
        case isMoving = "is_moving"
        case movementType = "movement_type"
        case stepsToday = "steps_today"
        case stepsLastHour = "steps_last_hour"
        case stationaryMinutes = "stationary_minutes"
    }
}

// MARK: - Status Report Response

struct StatusReportResponse: Codable {
    let success: Bool
    let message: String?
    let nextReportIn: Int?  // 已废弃，手动分析模式不再需要
    let analysis: AnalysisData?

    enum CodingKeys: String, CodingKey {
        case success
        case message
        case nextReportIn = "next_report_in"
        case analysis
    }
}

struct AnalysisData: Codable {
    let availability: AvailabilityAnalysis
    let lifeStatus: LifeStatusData
    let updatedAt: String?  // ISO 8601 格式时间

    enum CodingKeys: String, CodingKey {
        case availability
        case lifeStatus = "life_status"
        case updatedAt = "updated_at"
    }
}

struct AvailabilityAnalysis: Codable {
    let status: String           // 有空/忙碌/可能有空
    let probability: Int         // 0-100
    let reason: String           // 理由
    let confidence: String       // high/medium/low
}

struct LifeStatusData: Codable {
    let emoji: String
    let label: String
    let description: String?
}

// MARK: - 状态选项（训练 AI 功能）

/// 状态选项
struct StatusOption: Codable, Identifiable, Equatable {
    let emoji: String
    let status: String

    var id: String { "\(emoji)_\(status)" }
}

/// 状态选项结果
struct StatusOptionsResult: Codable {
    let options: [StatusOption]
}

/// 选择状态请求
struct SelectStatusRequest: Encodable {
    let emoji: String
    let status: String
    let deviceData: StatusReportRequest?

    enum CodingKeys: String, CodingKey {
        case emoji
        case status
        case deviceData = "device_data"
    }
}

/// 选择状态响应
struct SelectStatusResponse: Codable {
    let success: Bool
    let message: String?
}

// MARK: - 语音状态时刻表（Voice Schedule）

/// 时刻表条目
struct ScheduleItem: Codable, Identifiable, Equatable {
    let startTime: String
    let endTime: String
    let emoji: String
    let status: String
    var executed: Bool?

    var id: String { "\(startTime)_\(endTime)_\(emoji)" }

    init(startTime: String, endTime: String, emoji: String, status: String, executed: Bool? = nil) {
        self.startTime = startTime
        self.endTime = endTime
        self.emoji = emoji
        self.status = status
        self.executed = executed
    }

    enum CodingKeys: String, CodingKey {
        case startTime = "start_time"
        case endTime = "end_time"
        case emoji
        case status
        case executed
    }
}

/// 澄清问题
struct ClarifyQuestion: Codable, Identifiable {
    let id: String
    let question: String
    let options: [String]
    let allowVoice: Bool

    enum CodingKeys: String, CodingKey {
        case id, question, options
        case allowVoice = "allow_voice"
    }
}

/// 语音时刻表 SSE 事件类型
enum VoiceScheduleEventType: String, Codable {
    case sessionStart = "session_start"
    case recognizing
    case transcript
    case progress           // 过程反馈
    case thinking
    case clarify
    case schedule
    case currentStatus = "current_status"
    case visibilityPrompt = "visibility_prompt"  // 可见性选择
    case circleList = "circle_list"              // 圈子列表
    case confirmed
    case error
}

/// 语音时刻表 SSE 事件
struct VoiceScheduleEvent: Codable {
    let type: VoiceScheduleEventType?
    let sessionId: String?
    let status: String?
    let text: String?
    let questions: [ClarifyQuestion]?
    let items: [ScheduleItem]?
    let emoji: String?
    let statusText: String?
    let reason: String?
    let message: String?
    let partial: Bool?

    // Progress 事件专用字段
    let action: String?        // 进度动作类型
    let detail: String?        // 详细信息

    // 推理相关字段
    let reasoning: [String]?   // 推理依据

    // 可见性相关字段
    let visibility: String?    // 默认可见性
    let circles: [CircleInfoCompact]?  // 圈子列表

    enum CodingKeys: String, CodingKey {
        case type
        case sessionId = "session_id"
        case status
        case text
        case questions
        case items
        case emoji
        case statusText = "status_text"
        case reason
        case message
        case partial
        case action
        case detail
        case reasoning
        case visibility
        case circles
    }
}

/// 圈子信息（紧凑版，用于可见性选择）
struct CircleInfoCompact: Codable, Identifiable {
    let id: String
    let name: String
    let emoji: String
    let memberCount: Int

    enum CodingKeys: String, CodingKey {
        case id, name, emoji
        case memberCount = "member_count"
    }
}

/// 可见性选项
enum ScheduleVisibility: String, Codable, CaseIterable {
    case allFriends = "all_friends"
    case circles = "circles"
    case privateOnly = "private"

    var label: String {
        switch self {
        case .allFriends: return "所有好友"
        case .circles: return "指定圈子"
        case .privateOnly: return "仅自己"
        }
    }

    var icon: String {
        switch self {
        case .allFriends: return "person.2.fill"
        case .circles: return "circle.grid.2x2.fill"
        case .privateOnly: return "lock.fill"
        }
    }

    var description: String {
        switch self {
        case .allFriends: return "你的所有好友都能看到这些状态"
        case .circles: return "只有选中圈子里的好友能看到"
        case .privateOnly: return "只有你自己能看到"
        }
    }
}

/// 语音时刻表交互动作
enum VoiceScheduleAction: String, Codable {
    case answer      // 回答澄清问题
    case voice       // 语音输入
    case supplement  // 补充说明
    case confirm     // 确认执行
    case cancel      // 取消
}

/// 语音时刻表后续交互请求
struct VoiceScheduleInteractionRequest: Encodable {
    let sessionId: String
    let action: String
    let data: VoiceScheduleInteractionData?

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case action
        case data
    }
}

/// 交互数据
struct VoiceScheduleInteractionData: Encodable {
    let answers: [String: String]?   // 问题 ID -> 回答
    let audioData: String?           // Base64 编码的音频数据
    let text: String?                // 文本输入
    let visibility: String?          // 可见性设置
    let circleIds: [String]?         // 指定圈子 ID

    enum CodingKeys: String, CodingKey {
        case answers
        case audioData = "audio_data"
        case text
        case visibility
        case circleIds = "circle_ids"
    }

    init(
        answers: [String: String]? = nil,
        audioData: String? = nil,
        text: String? = nil,
        visibility: String? = nil,
        circleIds: [String]? = nil
    ) {
        self.answers = answers
        self.audioData = audioData
        self.text = text
        self.visibility = visibility
        self.circleIds = circleIds
    }
}
