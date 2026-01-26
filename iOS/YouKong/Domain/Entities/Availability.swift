import Foundation

enum LocationType: String, Codable, CaseIterable {
    case preset = "PRESET"
    case flexible = "FLEXIBLE"
    case custom = "CUSTOM"

    var displayName: String {
        switch self {
        case .preset: return "预设地点"
        case .flexible: return "灵活"
        case .custom: return "自定义"
        }
    }
}

enum AvailabilityStatus: String, Codable {
    case active = "ACTIVE"
    case expired = "EXPIRED"
    case cancelled = "CANCELLED"
    case fulfilled = "FULFILLED"

    var displayName: String {
        switch self {
        case .active: return "进行中"
        case .expired: return "已过期"
        case .cancelled: return "已取消"
        case .fulfilled: return "已完成"
        }
    }
}

struct Location: Codable, Equatable {
    let type: LocationType
    let name: String?
    let latitude: Double?
    let longitude: Double?
}

struct Availability: Codable, Identifiable, Equatable {
    let id: String
    let user: UserProfile
    let startTime: Date
    let endTime: Date
    let location: Location
    let status: AvailabilityStatus
    let circleIds: [String]?
    let createdAt: Date

    var timeRangeText: String {
        let startText = startTime.relativeDescription + " " + startTime.timeString
        let endText = startTime.isToday == endTime.isToday ? endTime.timeString : endTime.relativeDescription + " " + endTime.timeString
        return "\(startText) - \(endText)"
    }

    var locationText: String {
        switch location.type {
        case .preset, .custom:
            return location.name ?? "未知地点"
        case .flexible:
            return "位置灵活"
        }
    }
}

struct CreateAvailabilityRequest: Codable {
    let startTime: Date
    let endTime: Date
    let locationType: LocationType
    let locationName: String?
    let latitude: Double?
    let longitude: Double?
    let circleIds: [String]
}
