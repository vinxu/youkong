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

    var id: String { userId }

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case nickname
        case avatar
        case emoji
        case status
        case updatedAt = "updated_at"
        case relativeTime = "relative_time"
    }
}

// MARK: - Poster Models

/// 海报响应
struct PosterResponse: Codable {
    let posterUrl: String
    let message: String?

    enum CodingKeys: String, CodingKey {
        case posterUrl = "poster_url"
        case message
    }
}
