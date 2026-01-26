import Foundation

// MARK: - Agent Exposed Data

/// Agent 暴露给其他 Agent 的数据
struct AgentExposedData: Codable {
    let realtime: RealtimeStatus
    let patterns: UserPatterns
    let dataQuality: DataQuality
}

// MARK: - Realtime Status

struct RealtimeStatus: Codable {
    let screen: ScreenStatus
    let location: LocationStatus
}

// MARK: - Screen Status

struct ScreenStatus: Codable, Equatable {
    let isActive: Bool
    let activityType: ActivityType
    let sessionDurationMinutes: Int
    let lastActiveMinutesAgo: Int

    enum CodingKeys: String, CodingKey {
        case isActive = "is_active"
        case activityType = "activity_type"
        case sessionDurationMinutes = "session_duration_minutes"
        case lastActiveMinutesAgo = "last_active_minutes_ago"
    }

    static let idle = ScreenStatus(
        isActive: false,
        activityType: .idle,
        sessionDurationMinutes: 0,
        lastActiveMinutesAgo: 0
    )
}

// MARK: - Location Status

struct LocationStatus: Codable, Equatable {
    let placeType: PlaceType
    let atPlaceSinceMinutes: Int

    enum CodingKeys: String, CodingKey {
        case placeType = "place_type"
        case atPlaceSinceMinutes = "at_place_since_minutes"
    }

    static let unknown = LocationStatus(
        placeType: .unknown,
        atPlaceSinceMinutes: 0
    )
}

// MARK: - Activity Type

enum ActivityType: String, Codable {
    case entertainment
    case productivity
    case communication
    case idle

    var displayName: String {
        switch self {
        case .entertainment: return "娱乐"
        case .productivity: return "工作"
        case .communication: return "聊天"
        case .idle: return "空闲"
        }
    }
}

// MARK: - Place Type

enum PlaceType: String, Codable {
    case home
    case work
    case leisure
    case transit
    case unknown

    var displayName: String {
        switch self {
        case .home: return "在家"
        case .work: return "在公司"
        case .leisure: return "在外面"
        case .transit: return "在路上"
        case .unknown: return "未知"
        }
    }
}

// MARK: - User Patterns

struct UserPatterns: Codable {
    let currentHourFreeRate: Int
    let currentWeekdayFreeRate: Int
    let atHomeFreeRate: Int
    let atWorkAfterHoursFreeRate: Int
    let avgResponseTimeMinutes: Int
    let responseRate: Int

    enum CodingKeys: String, CodingKey {
        case currentHourFreeRate = "current_hour_free_rate"
        case currentWeekdayFreeRate = "current_weekday_free_rate"
        case atHomeFreeRate = "at_home_free_rate"
        case atWorkAfterHoursFreeRate = "at_work_after_hours_free_rate"
        case avgResponseTimeMinutes = "avg_response_time_minutes"
        case responseRate = "response_rate"
    }
}

// MARK: - Data Quality

struct DataQuality: Codable {
    let screenDataAgeSeconds: Int
    let locationDataAgeSeconds: Int
    let patternsSampleSize: Int

    enum CodingKeys: String, CodingKey {
        case screenDataAgeSeconds = "screen_data_age_seconds"
        case locationDataAgeSeconds = "location_data_age_seconds"
        case patternsSampleSize = "patterns_sample_size"
    }
}
