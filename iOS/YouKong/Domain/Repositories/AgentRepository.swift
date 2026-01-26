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
    let battery: BatteryRequestData?
    let mode: ModeRequestData?
    let connection: ConnectionRequestData?
    let display: DisplayRequestData?
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

// MARK: - Status Report Response

struct StatusReportResponse: Codable {
    let success: Bool
    let nextReportIn: Int
    let analysis: AnalysisData?

    enum CodingKeys: String, CodingKey {
        case success
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
