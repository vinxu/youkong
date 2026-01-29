import Foundation

// MARK: - Holmes 原始线索

/// Holmes 收集的原始数据
struct HolmesRawData: Codable, Equatable {
    let timestamp: String           // ISO 8601
    let weekday: String             // "周六"
    let timePeriod: String          // "下午3点42分"
    let isWeekend: Bool
    let placeName: String?          // "星巴克(三里屯店)"
    let placeType: String           // leisure
    let atPlaceSinceMinutes: Int
    let stepsToday: Int?
    let stepsLastHour: Int?
    let isMoving: Bool
    let stationaryMinutes: Int?
    let screenActive: Bool
    let screenDurationMins: Int?
    let activityType: String?
    let hasCalendarEvent: Bool
    let todayRemainingEvents: Int
    let headphonesConnected: Bool?
    let focusModeOn: Bool?
    let batteryLevel: Int?

    enum CodingKeys: String, CodingKey {
        case timestamp
        case weekday
        case timePeriod = "time_period"
        case isWeekend = "is_weekend"
        case placeName = "place_name"
        case placeType = "place_type"
        case atPlaceSinceMinutes = "at_place_since_minutes"
        case stepsToday = "steps_today"
        case stepsLastHour = "steps_last_hour"
        case isMoving = "is_moving"
        case stationaryMinutes = "stationary_minutes"
        case screenActive = "screen_active"
        case screenDurationMins = "screen_duration_mins"
        case activityType = "activity_type"
        case hasCalendarEvent = "has_calendar_event"
        case todayRemainingEvents = "today_remaining_events"
        case headphonesConnected = "headphones_connected"
        case focusModeOn = "focus_mode_on"
        case batteryLevel = "battery_level"
    }
}

// MARK: - Holmes 提取特征

/// Holmes 从原始数据中提取的特征
struct HolmesFeatures: Codable, Equatable {
    let locationType: String        // "星巴克(三里屯店)"
    let movementState: String       // "静止（已23分钟）"
    let timePeriod: String          // "周末下午"
    let activity: String            // "娱乐内容（已18分钟）"
    let schedule: String            // "无日程"
    let deviceState: String         // "正常"

    enum CodingKeys: String, CodingKey {
        case locationType = "location_type"
        case movementState = "movement_state"
        case timePeriod = "time_period"
        case activity
        case schedule
        case deviceState = "device_state"
    }
}

// MARK: - Holmes 推理过程

/// Holmes 的推理过程
struct HolmesReasoning: Codable, Equatable {
    let model: String               // "qwen3-max-2026-01-23" 或 "rules"
    let thinking: String            // LLM 思考过程
    let conclusion: String          // "很可能有空约朋友"

    static let empty = HolmesReasoning(
        model: "unknown",
        thinking: "",
        conclusion: ""
    )
}

// MARK: - Holmes 结果

/// Holmes 的最终判断结果
struct HolmesResult: Codable, Equatable {
    let available: Bool
    let probability: Int            // 0-100
    let confidence: String          // high/medium/low
    let summary: String             // "在咖啡厅休闲"
    let emoji: String?              // "☕"
    let color: String               // "#22C55E"

    enum CodingKeys: String, CodingKey {
        case available
        case probability
        case confidence
        case summary
        case emoji
        case color
    }

    /// 是否有空的简单状态
    var simpleStatus: String {
        available ? "有空" : "忙碌"
    }

    /// 置信度的中文显示
    var confidenceDisplay: String {
        switch confidence {
        case "high": return "高"
        case "medium": return "中"
        case "low": return "低"
        default: return confidence
        }
    }
}

// MARK: - Holmes 好友响应（完整）

/// 包含完整 Holmes 分析的好友响应
struct HolmesFriendResponse: Codable, Identifiable, Equatable {
    let friendId: String
    let name: String
    let avatar: String?
    let rawData: HolmesRawData?
    let features: HolmesFeatures?
    let reasoning: HolmesReasoning?
    let result: HolmesResult
    let updatedAt: Int              // 毫秒时间戳

    var id: String { friendId }

    enum CodingKeys: String, CodingKey {
        case friendId = "friend_id"
        case name
        case avatar
        case rawData = "raw_data"
        case features
        case reasoning
        case result
        case updatedAt = "updated_at"
    }

    /// 是否有详细数据（非降级版）
    var hasDetailedData: Bool {
        rawData != nil && features != nil
    }
}

// MARK: - Holmes 列表响应

/// Holmes 好友列表响应
struct HolmesProbabilityResponse: Codable {
    let friends: [HolmesFriendResponse]
    let generatedAt: Int

    enum CodingKeys: String, CodingKey {
        case friends
        case generatedAt = "generated_at"
    }
}

// MARK: - Holmes 自我分析

/// 用于「我的 Agent 数据」面板展示的完整 Holmes 分析
struct HolmesSelfAnalysis: Codable, Equatable {
    let rawData: HolmesRawData
    let features: HolmesFeatures
    let reasoning: HolmesReasoning
    let result: HolmesResult
    let generatedAt: Int

    enum CodingKeys: String, CodingKey {
        case rawData = "raw_data"
        case features
        case reasoning
        case result
        case generatedAt = "generated_at"
    }
}

// MARK: - Holmes 状态上报响应（扩展）

/// 状态上报后返回的 Holmes 分析结果
struct HolmesStatusReportResponse: Codable {
    let success: Bool
    let nextReportIn: Int
    let holmes: HolmesSelfAnalysis?

    enum CodingKeys: String, CodingKey {
        case success
        case nextReportIn = "next_report_in"
        case holmes
    }
}
