import Foundation

// MARK: - Permission Status

/// 权限状态
struct PermissionStatus: Codable, Equatable {
    var location: Bool
    var calendar: Bool      // 日历权限
    var motion: Bool        // 运动与健身权限

    /// 核心权限是否都已授权（位置）
    var coreGranted: Bool {
        location
    }

    /// 所有权限是否都已授权
    var allGranted: Bool {
        location && calendar && motion
    }

    /// 已授权的权限数量
    var grantedCount: Int {
        [location, calendar, motion].filter { $0 }.count
    }

    /// 总权限数量
    static let totalCount = 3

    /// 初始状态（全部未授权）
    static let initial = PermissionStatus(
        location: false,
        calendar: false,
        motion: false
    )
}

// MARK: - Permission Type

enum PermissionType: String, CaseIterable {
    case location
    case calendar
    case motion

    /// 需要用户授权的权限列表
    static var requiredPermissions: [PermissionType] {
        [.location, .calendar, .motion]
    }

    var title: String {
        switch self {
        case .location: return "地理位置"
        case .calendar: return "日历"
        case .motion: return "运动与健身"
        }
    }

    var description: String {
        switch self {
        case .location: return "知道你在家、公司还是外面"
        case .calendar: return "了解你的日程安排"
        case .motion: return "判断你是否在移动"
        }
    }

    var iconName: String {
        switch self {
        case .location: return "location.fill"
        case .calendar: return "calendar"
        case .motion: return "figure.walk"
        }
    }

    /// 是否为核心权限
    var isCore: Bool {
        switch self {
        case .location:
            return true
        default:
            return false
        }
    }
}
