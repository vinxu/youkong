import Foundation

protocol CircleRepositoryProtocol {
    func getCircles() async throws -> [FriendCircle]
    func createCircle(name: String, emoji: String, color: String) async throws -> FriendCircle
    func getCircle(id: String) async throws -> FriendCircleDetail
    func updateCircle(id: String, name: String, emoji: String, color: String) async throws -> FriendCircle
    func deleteCircle(id: String) async throws
    func addMember(circleId: String, userId: String) async throws
    func removeMember(circleId: String, userId: String) async throws
}
