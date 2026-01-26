import Foundation

protocol UserRepositoryProtocol {
    func getCurrentUser() async throws -> User
    func updateCurrentUser(nickname: String?, avatar: String?) async throws -> User
    func searchUsers(keyword: String) async throws -> [UserProfile]
    func getUser(id: String) async throws -> UserProfile
}
