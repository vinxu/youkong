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
    }
}
