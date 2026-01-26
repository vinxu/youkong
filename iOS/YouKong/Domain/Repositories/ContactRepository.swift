import Foundation
import CryptoKit

// MARK: - Contact Repository Protocol

protocol ContactRepositoryProtocol {
    /// 获取通讯录中的朋友（已注册用户）
    func getContactFriends() async throws -> [UserProfile]

    /// 同步通讯录
    func syncContacts(phones: [String]) async throws -> [UserProfile]

    /// 通讯录匹配（隐私保护方式，使用 SHA256 Hash）
    func matchContacts(phoneHashes: [String]) async throws -> ContactMatchResponse

    /// 批量添加好友
    func addFriends(userIds: [String]) async throws -> AddFriendsResponse

    // MARK: - Friend Request (好友请求流程)

    /// 发送好友请求
    func sendFriendRequest(phone: String, message: String?) async throws -> SendFriendRequestResponse

    /// 获取收到的好友请求
    func getReceivedRequests() async throws -> [FriendRequest]

    /// 获取发出的好友请求
    func getSentRequests() async throws -> [FriendRequest]

    /// 处理好友请求（同意/拒绝）
    func handleFriendRequest(requestId: String, accept: Bool) async throws -> HandleFriendRequestResponse

    /// 获取待处理请求数量（用于红点）
    func getPendingRequestCount() async throws -> Int
}

// MARK: - Sync Contacts Request

struct SyncContactsRequest: Encodable {
    let phones: [String]
}

// MARK: - Sync Contacts Response

struct SyncContactsResponse: Decodable {
    let friends: [UserProfile]
}

// MARK: - Contact Match Request

struct ContactMatchRequest: Encodable {
    let phoneHashes: [String]
}

// MARK: - Contact Match Response

struct ContactMatchResponse: Decodable {
    let matches: [ContactMatch]
    let totalFound: Int
}

struct ContactMatch: Codable, Identifiable {
    let phoneHash: String
    let user: UserProfile
    let isFriend: Bool

    var id: String { user.id }
}

// MARK: - Add Friends Request

struct AddFriendsRequest: Encodable {
    let userIds: [String]
}

// MARK: - Add Friends Response

struct AddFriendsResponse: Decodable {
    let added: Int
    let skipped: Int
    let userIds: [String]
}

// MARK: - Friend Request

struct FriendRequest: Codable, Identifiable, Equatable {
    let id: String
    let user: UserProfile  // 对方用户信息
    let message: String?
    let status: FriendRequestStatus
    let createdAt: Date
}

enum FriendRequestStatus: String, Codable {
    case pending = "PENDING"
    case accepted = "ACCEPTED"
    case rejected = "REJECTED"
    case cancelled = "CANCELLED"
    case alreadyFriends = "ALREADY_FRIENDS"
    case alreadyRequested = "ALREADY_REQUESTED"
}

// MARK: - Send Friend Request

struct SendFriendRequestRequest: Encodable {
    let phone: String
    let message: String?
}

struct SendFriendRequestResponse: Decodable {
    let requestId: String?
    let user: UserProfile?
    let status: FriendRequestStatus
    let message: String?
}

// MARK: - Handle Friend Request

struct HandleFriendRequestRequest: Encodable {
    let accept: Bool
}

struct HandleFriendRequestResponse: Decodable {
    let success: Bool
    let message: String?
}

// MARK: - Friend Request Count

struct FriendRequestCountResponse: Decodable {
    let count: Int
}

// MARK: - Phone Hash Helper

extension String {
    /// 计算手机号的 SHA256 Hash
    var sha256Hash: String {
        let data = Data(self.utf8)
        let hash = SHA256.hash(data: data)
        return hash.compactMap { String(format: "%02x", $0) }.joined()
    }
}
