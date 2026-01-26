import Foundation

// MARK: - Invitation

struct Invitation: Codable, Identifiable, Equatable {
    let id: String
    let code: String
    let inviteUrl: String
    let qrcodeUrl: String?
    let inviter: UserProfile?
    let circle: CircleInfo?
    let maxUses: Int
    let useCount: Int
    let expiresAt: Date?
    let status: InvitationStatus
    let isValid: Bool
    let createdAt: Date

    // API 返回 camelCase，不需要 CodingKeys
}

// MARK: - Invitation Status

enum InvitationStatus: String, Codable {
    case active = "ACTIVE"
    case disabled = "DISABLED"
    case expired = "EXPIRED"
}

// MARK: - Circle Info

struct CircleInfo: Codable, Equatable {
    let id: String
    let name: String
    let emoji: String
    let color: String?
    let memberCount: Int?

    // API 返回 camelCase
}

// MARK: - Invitation Public Info (落地页用)

struct InvitationPublicInfo: Codable {
    let inviter: UserProfile
    let circle: CircleInfo?
    let isValid: Bool

    // API 返回 camelCase
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

    // API 返回 camelCase
}

// MARK: - Create Invitation Request

struct CreateInvitationRequest: Encodable {
    let circleId: String?
    let maxUses: Int?
    let expiresDays: Int?

    enum CodingKeys: String, CodingKey {
        case circleId = "circle_id"
        case maxUses = "max_uses"
        case expiresDays = "expires_days"
    }
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
