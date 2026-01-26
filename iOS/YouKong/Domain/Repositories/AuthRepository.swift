import Foundation

protocol AuthRepositoryProtocol {
    func sendSMSCode(phone: String) async throws
    func verifySMSCode(phone: String, code: String) async throws -> LoginResult
    func refreshToken(refreshToken: String) async throws -> String
}
