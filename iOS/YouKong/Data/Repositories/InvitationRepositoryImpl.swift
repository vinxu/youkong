import Foundation
import Factory

// MARK: - Invite Repository Implementation (简化版)

final class InviteRepositoryImpl: InviteRepositoryProtocol {
    @Injected(\.apiClient) private var apiClient

    // API Base URL
    private var baseURL: String {
        #if DEBUG
        let custom = UserDefaults.standard.string(forKey: "debug_baseURL") ?? ""
        return custom.isEmpty ? "http://49.232.13.41:8080" : custom
        #else
        return "http://49.232.13.41:8080"
        #endif
    }

    func getMyInvite() async throws -> MyInviteInfo {
        let endpoint = APIEndpoint.getMyInvite
        return try await apiClient.request(endpoint)
    }

    func fetchMyPoster() async throws -> Data {
        guard let token = KeychainManager.shared.getAccessToken() else {
            throw NSError(domain: "InviteRepository", code: 401, userInfo: [NSLocalizedDescriptionKey: "未登录"])
        }

        let url = URL(string: "\(baseURL)/api/v1/users/me/poster")!
        var request = URLRequest(url: url)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let (data, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
            throw NSError(domain: "InviteRepository", code: 500, userInfo: [NSLocalizedDescriptionKey: "获取海报失败"])
        }

        return data
    }

    func getInvitationByCode(code: String) async throws -> InvitationPublicInfo {
        let endpoint = APIEndpoint.getInvitationByCode(code: code)
        return try await apiClient.request(endpoint)
    }

    func acceptInvitation(code: String) async throws {
        let endpoint = APIEndpoint.acceptInvitation(code: code)
        let _: EmptyResponse = try await apiClient.request(endpoint)
    }
}

// MARK: - Friendship Repository Implementation

final class FriendshipRepositoryImpl: FriendshipRepositoryProtocol {
    @Injected(\.apiClient) private var apiClient

    func getFriends() async throws -> [FriendInfo] {
        let endpoint = APIEndpoint.getFriends
        return try await apiClient.request(endpoint)
    }

    func deleteFriend(userId: String) async throws {
        let endpoint = APIEndpoint.deleteFriend(userId: userId)
        let _: EmptyResponse = try await apiClient.request(endpoint)
    }

    func getFriendsInvitedByMe() async throws -> [FriendWithInvitation] {
        let endpoint = APIEndpoint.getFriendsInvitedByMe
        return try await apiClient.request(endpoint)
    }

    func getFriendsInvitedMe() async throws -> [FriendWithInvitation] {
        let endpoint = APIEndpoint.getFriendsInvitedMe
        return try await apiClient.request(endpoint)
    }
}
