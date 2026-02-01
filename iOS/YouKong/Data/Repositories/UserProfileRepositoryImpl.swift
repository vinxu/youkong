import Foundation

final class UserProfileRepositoryImpl: UserProfileRepositoryProtocol {
    private let apiClient: APIClient

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
    }

    func getProfile() async throws -> UserProfileData? {
        let response: APIResponse<ProfileGetResponse> = try await apiClient.request(.getProfile)
        return response.data?.exists == true ? response.data?.profile : nil
    }

    func upsertProfile(_ request: UserProfileDataRequest) async throws -> UserProfileData {
        let response: APIResponse<UserProfileData> = try await apiClient.request(
            .upsertProfile(request)
        )
        guard let profile = response.data else {
            throw APIError.invalidResponse
        }
        return profile
    }

    func getProfileStatus() async throws -> UserProfileStatusResponse {
        let response: APIResponse<UserProfileStatusResponse> = try await apiClient.request(.getProfileStatus)
        guard let status = response.data else {
            throw APIError.invalidResponse
        }
        return status
    }

    func deleteProfile() async throws {
        let _: APIResponse<EmptyResponse> = try await apiClient.request(.deleteProfile)
    }
}

// MARK: - API Endpoint Extension

extension APIEndpoint {
    static var getProfile: APIEndpoint {
        .init(path: "/profile", method: .get)
    }

    static func upsertProfile(_ request: UserProfileDataRequest) -> APIEndpoint {
        .init(path: "/profile", method: .put, body: request)
    }

    static var getProfileStatus: APIEndpoint {
        .init(path: "/profile/status", method: .get)
    }

    static var deleteProfile: APIEndpoint {
        .init(path: "/profile", method: .delete)
    }
}

// MARK: - Response Models

struct ProfileGetResponse: Codable {
    let exists: Bool
    let profile: UserProfileData?
}
