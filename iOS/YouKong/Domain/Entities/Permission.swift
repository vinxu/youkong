import Foundation

// MARK: - Permission Status

/// 权限状态
struct PermissionStatus: Codable, Equatable {
    var screenTime: Bool
    var location: Bool
    var contacts: Bool

    /// 所有权限是否都已授权
    var allGranted: Bool {
        screenTime && location && contacts
    }

    /// 已授权的权限数量
    var grantedCount: Int {
        [screenTime, location, contacts].filter { $0 }.count
    }

    /// 总权限数量
    static let totalCount = 3

    /// 初始状态（全部未授权）
    static let initial = PermissionStatus(
        screenTime: false,
        location: false,
        contacts: false
    )
}

// MARK: - Permission Type

enum PermissionType: String, CaseIterable {
    case screenTime
    case location
    case contacts

    var title: String {
        switch self {
        case .screenTime: return "屏幕使用时间"
        case .location: return "地理位置"
        case .contacts: return "通讯录"
        }
    }

    var description: String {
        switch self {
        case .screenTime: return "判断你是否有空"
        case .location: return "知道你在家、公司还是外面"
        case .contacts: return "找到你的朋友"
        }
    }

    var iconName: String {
        switch self {
        case .screenTime: return "iphone"
        case .location: return "location.fill"
        case .contacts: return "person.2.fill"
        }
    }
}
