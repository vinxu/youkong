import Foundation

// MARK: - Contact Repository Protocol

protocol ContactRepositoryProtocol {
    /// 获取通讯录中的朋友（已注册用户）
    func getContactFriends() async throws -> [UserProfile]

    /// 同步通讯录
    func syncContacts(phones: [String]) async throws -> [UserProfile]
}

// MARK: - Sync Contacts Request

struct SyncContactsRequest: Encodable {
    let phones: [String]
}

// MARK: - Sync Contacts Response

struct SyncContactsResponse: Decodable {
    let friends: [UserProfile]
}
