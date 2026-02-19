import Foundation

// MARK: - Grid Models

/// 宫格响应
struct GridResponse: Codable {
    let gridSize: Int
    let friends: [FriendGridItem]

    enum CodingKeys: String, CodingKey {
        case gridSize = "grid_size"
        case friends
    }
}

/// 宫格中的好友项
struct FriendGridItem: Codable, Identifiable {
    let userId: String
    let nickname: String
    let avatar: String?
    let emoji: String
    let status: String
    let updatedAt: String
    let relativeTime: String
    let city: String?
    let isAvailable: Bool?
    let isVisiting: Bool?
    let gifUrl: String?
    let giphyQuery: String?
    let useGif: Bool?
    let needsSchedule: Bool?
    // 像素场景
    let scenePose: String?
    let sceneArms: String?
    let sceneExpression: String?
    let sceneProp: String?
    let sceneSurface: String?
    // AI 互动选项
    let interactions: [InteractionOptionItem]?
    // 今日互动计数
    let interactionCount: Int?
    // 是否已对该好友当前状态 +1
    let hasPlusOned: Bool?
    // 是否是自己
    let isSelf: Bool?

    var id: String { userId }

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case nickname
        case avatar
        case emoji
        case status
        case updatedAt = "updated_at"
        case relativeTime = "relative_time"
        case city
        case isAvailable = "is_available"
        case isVisiting = "is_visiting"
        case gifUrl = "gif_url"
        case giphyQuery = "giphy_query"
        case useGif = "use_gif"
        case needsSchedule = "needs_schedule"
        case scenePose = "scene_pose"
        case sceneArms = "scene_arms"
        case sceneExpression = "scene_expression"
        case sceneProp = "scene_prop"
        case sceneSurface = "scene_surface"
        case interactions
        case interactionCount = "interaction_count"
        case hasPlusOned = "has_plus_oned"
        case isSelf = "is_self"
    }

    /// 转换为 PixelSceneConfig
    var pixelSceneConfig: PixelSceneConfig? {
        guard let pose = scenePose, !pose.isEmpty else { return nil }
        return PixelSceneConfig(
            pose: pose,
            arms: sceneArms ?? "down",
            expression: sceneExpression ?? "normal",
            prop: sceneProp ?? "none",
            surface: sceneSurface ?? "none"
        )
    }
}

/// AI 互动选项
struct InteractionOptionItem: Codable, Identifiable {
    let emoji: String
    let label: String
    let pushText: String

    var id: String { emoji + label }

    enum CodingKeys: String, CodingKey {
        case emoji
        case label
        case pushText = "push_text"
    }
}

/// 发送互动请求
struct SendInteractionRequest: Codable {
    let receiverId: String
    let actionEmoji: String
    let actionLabel: String
    let actionPushText: String

    enum CodingKeys: String, CodingKey {
        case receiverId = "receiver_id"
        case actionEmoji = "action_emoji"
        case actionLabel = "action_label"
        case actionPushText = "action_push_text"
    }
}
