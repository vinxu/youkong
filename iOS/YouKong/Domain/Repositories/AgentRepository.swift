import Foundation

// MARK: - Agent Repository Protocol

protocol AgentRepositoryProtocol {
    /// 上报自己的状态，返回 LLM 预测结果
    func reportStatus(request: StatusReportRequest) async throws -> StatusReportResponse

    /// 获取朋友有空概率列表
    func getFriendsFreeProbability() async throws -> [FriendRecommendation]

    /// 请求朋友 Agent 数据
    func queryAgentData(agentId: String) async throws -> AgentExposedData
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

    enum CodingKeys: String, CodingKey {
        case placeType = "place_type"
        case atPlaceSinceMinutes = "at_place_since_minutes"
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

    enum CodingKeys: String, CodingKey {
        case availability
        case lifeStatus = "life_status"
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
