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
    let lastActiveCategory: String?  // 最近活跃的应用分类（如"游戏"、"社交"）

    enum CodingKeys: String, CodingKey {
        case isActive = "is_active"
        case activityType = "activity_type"
        case sessionDurationMinutes = "session_duration_minutes"
        case lastActiveMinutesAgo = "last_active_minutes_ago"
        case lastActiveCategory = "last_active_category"
    }

    static let idle = ScreenStatus(
        isActive: false,
        activityType: .idle,
        sessionDurationMinutes: 0,
        lastActiveMinutesAgo: 0,
        lastActiveCategory: nil
    )
}

// MARK: - Location Status

struct LocationStatus: Codable, Equatable {
    let placeType: PlaceType
    let atPlaceSinceMinutes: Int
    let placeName: String?          // 反向地理编码获取的地点名称
    let latitude: Double?           // 纬度
    let longitude: Double?          // 经度
    let city: String?               // 城市名称

    enum CodingKeys: String, CodingKey {
        case placeType = "place_type"
        case atPlaceSinceMinutes = "at_place_since_minutes"
        case placeName = "place_name"
        case latitude
        case longitude
        case city
    }

    static let unknown = LocationStatus(
        placeType: .unknown,
        atPlaceSinceMinutes: 0,
        placeName: nil,
        latitude: nil,
        longitude: nil,
        city: nil
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
