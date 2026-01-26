import Foundation

final class UserRepositoryImpl: UserRepositoryProtocol {
    private let apiClient: APIClient

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
    }

    func getCurrentUser() async throws -> User {
        try await apiClient.request(.getMe)
    }

    func updateCurrentUser(nickname: String?, avatar: String?) async throws -> User {
        try await apiClient.request(.updateMe(nickname: nickname, avatar: avatar))
    }

    func searchUsers(keyword: String) async throws -> [UserProfile] {
        try await apiClient.request(.searchUsers(keyword: keyword))
    }

    func getUser(id: String) async throws -> UserProfile {
        try await apiClient.request(.getUser(id: id))
    }
}
