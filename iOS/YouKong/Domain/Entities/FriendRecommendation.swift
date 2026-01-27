import Foundation

// MARK: - Friend Recommendation

/// 有空推荐结果（主页面展示）
struct FriendRecommendation: Codable, Identifiable, Equatable, Hashable {
    let friendId: String
    let name: String
    let avatar: String?
    let probability: Int    // 0-100, -1 表示无数据
    let confidence: Confidence
    let reason: String      // 有空程度状态，如"可能有空"
    let color: String       // 颜色代码，如 "#22C55E"
    let emoji: String?      // 活动 emoji，如 "🎮"
    let activity: String?   // 活动描述，如 "在玩游戏"
    let updatedAt: Int      // 毫秒时间戳

    var id: String { friendId }

    enum CodingKeys: String, CodingKey {
        case friendId = "friend_id"
        case name
        case avatar
        case probability
        case confidence
        case reason
        case color
        case emoji
        case activity
        case updatedAt = "updated_at"
    }

    /// 是否有有效数据
    var hasData: Bool {
        probability >= 0
    }

    /// 格式化的概率文本
    var probabilityText: String {
        if probability < 0 {
            return "无数据"
        }
        return "\(probability)%"
    }

    /// 显示用的 emoji（如果没有则用默认）
    var displayEmoji: String {
        emoji ?? "📱"
    }

    /// 显示用的活动描述
    var displayActivity: String {
        activity ?? reason
    }
}

// MARK: - Confidence Level

enum Confidence: String, Codable {
    case high
    case medium
    case low

    var displayName: String {
        switch self {
        case .high: return "高"
        case .medium: return "中"
        case .low: return "低"
        }
    }
}

// MARK: - Free Probability Response

/// 有空概率列表响应
struct FreeProbabilityResponse: Codable {
    let friends: [FriendRecommendation]
    let generatedAt: Int    // 毫秒时间戳

    enum CodingKeys: String, CodingKey {
        case friends
        case generatedAt = "generated_at"
    }
}
