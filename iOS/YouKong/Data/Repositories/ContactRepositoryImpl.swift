import Foundation
import Combine
import Factory

// MARK: - Contact Repository Implementation

class ContactRepositoryImpl: ContactRepositoryProtocol {
    @Injected(\.apiClient) private var apiClient

    // MARK: - Get Contact Friends

    func getContactFriends() async throws -> [UserProfile] {
        // 通讯录功能已移除
        return []
    }

    // MARK: - Sync Contacts

    func syncContacts(phones: [String]) async throws -> [UserProfile] {
        let endpoint = APIEndpoint.syncContacts(phones: phones)
        let response: SyncContactsResponse = try await apiClient.request(endpoint)
        return response.friends
    }

    // MARK: - Match Contacts (隐私保护方式)

    func matchContacts(phoneHashes: [String]) async throws -> ContactMatchResponse {
        let endpoint = APIEndpoint.matchContacts(phoneHashes: phoneHashes)
        return try await apiClient.request(endpoint)
    }

    // MARK: - Add Friends (批量)

    func addFriends(userIds: [String]) async throws -> AddFriendsResponse {
        let endpoint = APIEndpoint.addFriends(userIds: userIds)
        return try await apiClient.request(endpoint)
    }

    // MARK: - Friend Requests (好友请求流程)

    func sendFriendRequest(phone: String, message: String?) async throws -> SendFriendRequestResponse {
        let endpoint = APIEndpoint.sendFriendRequest(phone: phone, message: message)
        return try await apiClient.request(endpoint)
    }

    func getReceivedRequests() async throws -> [FriendRequest] {
        let endpoint = APIEndpoint.getReceivedFriendRequests
        return try await apiClient.request(endpoint)
    }

    func getSentRequests() async throws -> [FriendRequest] {
        let endpoint = APIEndpoint.getSentFriendRequests
        return try await apiClient.request(endpoint)
    }

    func handleFriendRequest(requestId: String, accept: Bool) async throws -> HandleFriendRequestResponse {
        let endpoint = APIEndpoint.handleFriendRequest(requestId: requestId, accept: accept)
        return try await apiClient.request(endpoint)
    }

    func getPendingRequestCount() async throws -> Int {
        let endpoint = APIEndpoint.getPendingRequestCount
        let response: FriendRequestCountResponse = try await apiClient.request(endpoint)
        return response.count
    }
}
