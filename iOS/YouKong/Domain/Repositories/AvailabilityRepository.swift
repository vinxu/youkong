import Foundation

protocol AvailabilityRepositoryProtocol {
    func getFriendsAvailabilities() async throws -> [Availability]
    func getMyAvailabilities() async throws -> [Availability]
    func createAvailability(request: CreateAvailabilityRequest) async throws -> Availability
    func cancelAvailability(id: String) async throws
}
