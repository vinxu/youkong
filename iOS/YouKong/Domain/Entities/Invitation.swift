import Foundation

// MARK: - My Invite Info (简化的邀请信息)

/// 用户固定的邀请信息
struct MyInviteInfo: Codable {
    let code: String       // 邀请码（固定，用户ID前8位）
    let inviteUrl: String  // 邀请链接
}

// MARK: - Invitation Public Info (接受邀请时使用)

/// 邀请公开信息（用于落地页展示）
struct InvitationPublicInfo: Codable {
    let inviter: UserProfile
    let circle: CircleInfo?
    let isValid: Bool
}

// MARK: - Circle Info

struct CircleInfo: Codable, Equatable {
    let id: String
    let name: String
    let emoji: String
    let color: String?
    let memberCount: Int?
}

// MARK: - Accept Invitation Response

struct AcceptInvitationResponse: Codable {
    let message: String
    let friendship: FriendshipInfo?
    let circle: CircleInfo?
}

struct FriendshipInfo: Codable {
    let userId: String
    let nickname: String
}

// MARK: - Friend Info

struct FriendInfo: Codable, Identifiable, Equatable {
    let user: UserProfile
    let source: FriendshipSource
    let createdAt: Date

    var id: String { user.id }

    // API 返回 camelCase
}

// MARK: - Friend With Invitation (我邀请的/邀请我的好友)

struct FriendWithInvitation: Codable, Identifiable, Equatable {
    let user: UserProfile
    let source: FriendshipSource
    let circleName: String?
    let createdAt: Date

    var id: String { user.id }

    // API 返回 camelCase
}

// MARK: - Friendship Source

enum FriendshipSource: String, Codable {
    case invitation = "INVITATION"
    case search = "SEARCH"
    case manual = "MANUAL"
}
