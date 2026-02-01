import Foundation

protocol UserProfileRepositoryProtocol {
    func getProfile() async throws -> UserProfileData?
    func upsertProfile(_ request: UserProfileDataRequest) async throws -> UserProfileData
    func getProfileStatus() async throws -> UserProfileStatusResponse
    func deleteProfile() async throws
}
