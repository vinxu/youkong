import Foundation

// MARK: - Permission Status

/// 权限状态
struct PermissionStatus: Codable, Equatable {
    var screenTime: Bool
    var location: Bool
    var contacts: Bool
    var calendar: Bool      // 日历权限
    var motion: Bool        // 运动与健身权限

    /// 核心权限是否都已授权（位置 + 通讯录）
    var coreGranted: Bool {
        location && contacts
    }

    /// 所有权限是否都已授权
    var allGranted: Bool {
        screenTime && location && contacts && calendar && motion
    }

    /// 已授权的权限数量
    var grantedCount: Int {
        [screenTime, location, contacts, calendar, motion].filter { $0 }.count
    }

    /// 总权限数量
    static let totalCount = 5

    /// 初始状态（全部未授权）
    static let initial = PermissionStatus(
        screenTime: false,
        location: false,
        contacts: false,
        calendar: false,
        motion: false
    )
}

// MARK: - Permission Type

enum PermissionType: String, CaseIterable {
    case screenTime
    case location
    case contacts
    case calendar
    case motion

    /// 需要用户授权的权限列表（排除 screenTime，因为它已废弃）
    static var requiredPermissions: [PermissionType] {
        [.location, .contacts, .calendar, .motion]
    }

    var title: String {
        switch self {
        case .screenTime: return "屏幕使用时间"
        case .location: return "地理位置"
        case .contacts: return "通讯录"
        case .calendar: return "日历"
        case .motion: return "运动与健身"
        }
    }

    var description: String {
        switch self {
        case .screenTime: return "判断你是否有空"
        case .location: return "知道你在家、公司还是外面"
        case .contacts: return "找到你的朋友"
        case .calendar: return "了解你的日程安排"
        case .motion: return "判断你是否在移动"
        }
    }

    var iconName: String {
        switch self {
        case .screenTime: return "iphone"
        case .location: return "location.fill"
        case .contacts: return "person.2.fill"
        case .calendar: return "calendar"
        case .motion: return "figure.walk"
        }
    }

    /// 是否为核心权限
    var isCore: Bool {
        switch self {
        case .location, .contacts:
            return true
        default:
            return false
        }
    }
}
