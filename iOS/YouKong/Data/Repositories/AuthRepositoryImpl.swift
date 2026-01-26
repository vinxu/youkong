import Foundation

final class AuthRepositoryImpl: AuthRepositoryProtocol {
    private let apiClient: APIClient

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
    }

    func sendSMSCode(phone: String) async throws {
        try await apiClient.requestWithEmptyResponse(.sendSMSCode(phone: phone))
    }

    func verifySMSCode(phone: String, code: String) async throws -> LoginResult {
        try await apiClient.request(.verifySMSCode(phone: phone, code: code))
    }

    func refreshToken(refreshToken: String) async throws -> String {
        struct RefreshResponse: Codable {
            let token: String
        }
        let response: RefreshResponse = try await apiClient.request(.refreshToken(refreshToken: refreshToken))
        return response.token
    }
}
