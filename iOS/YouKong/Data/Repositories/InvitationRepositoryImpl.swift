import Foundation
import Factory

// MARK: - Invitation Repository Implementation

final class InvitationRepositoryImpl: InvitationRepositoryProtocol {
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

    func createInvitation(circleId: String?, maxUses: Int?, expiresDays: Int?) async throws -> Invitation {
        let request = CreateInvitationRequest(
            circleId: circleId,
            maxUses: maxUses,
            expiresDays: expiresDays
        )
        let endpoint = APIEndpoint.createInvitation(request: request)
        return try await apiClient.request(endpoint)
    }

    func getMyInvitations() async throws -> [Invitation] {
        let endpoint = APIEndpoint.getMyInvitations
        return try await apiClient.request(endpoint)
    }

    func getInvitationByCode(code: String) async throws -> InvitationPublicInfo {
        let endpoint = APIEndpoint.getInvitationByCode(code: code)
        return try await apiClient.request(endpoint)
    }

    func acceptInvitation(code: String) async throws -> AcceptInvitationResponse {
        let endpoint = APIEndpoint.acceptInvitation(code: code)
        return try await apiClient.request(endpoint)
    }

    func disableInvitation(id: String) async throws {
        let endpoint = APIEndpoint.disableInvitation(id: id)
        let _: EmptyResponse = try await apiClient.request(endpoint)
    }

    func getInvitationPosterURL(id: String) -> URL? {
        guard let token = KeychainManager.shared.getAccessToken() else { return nil }
        return URL(string: "\(baseURL)/api/v1/invitations/\(id)/poster?token=\(token)")
    }

    func getInvitationQRCodeURL(id: String) -> URL? {
        guard let token = KeychainManager.shared.getAccessToken() else { return nil }
        return URL(string: "\(baseURL)/api/v1/invitations/\(id)/qrcode?token=\(token)")
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
