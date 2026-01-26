import Foundation

final class CircleRepositoryImpl: CircleRepositoryProtocol {
    private let apiClient: APIClient

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
    }

    func getCircles() async throws -> [FriendCircle] {
        try await apiClient.request(.getCircles)
    }

    func createCircle(name: String, emoji: String, color: String) async throws -> FriendCircle {
        try await apiClient.request(.createCircle(name: name, emoji: emoji, color: color))
    }

    func getCircle(id: String) async throws -> FriendCircleDetail {
        try await apiClient.request(.getCircle(id: id))
    }

    func updateCircle(id: String, name: String, emoji: String, color: String) async throws -> FriendCircle {
        try await apiClient.request(.updateCircle(id: id, name: name, emoji: emoji, color: color))
    }

    func deleteCircle(id: String) async throws {
        try await apiClient.requestWithEmptyResponse(.deleteCircle(id: id))
    }

    func addMember(circleId: String, userId: String) async throws {
        try await apiClient.requestWithEmptyResponse(.addCircleMember(circleId: circleId, userId: userId))
    }

    func removeMember(circleId: String, userId: String) async throws {
        try await apiClient.requestWithEmptyResponse(.removeCircleMember(circleId: circleId, userId: userId))
    }
}
