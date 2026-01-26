import Foundation

final class AvailabilityRepositoryImpl: AvailabilityRepositoryProtocol {
    private let apiClient: APIClient

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
    }

    func getFriendsAvailabilities() async throws -> [Availability] {
        try await apiClient.request(.getFriendsAvailabilities)
    }

    func getMyAvailabilities() async throws -> [Availability] {
        try await apiClient.request(.getMyAvailabilities)
    }

    func createAvailability(request: CreateAvailabilityRequest) async throws -> Availability {
        try await apiClient.request(.createAvailability(request: request))
    }

    func cancelAvailability(id: String) async throws {
        try await apiClient.requestWithEmptyResponse(.cancelAvailability(id: id))
    }
}
