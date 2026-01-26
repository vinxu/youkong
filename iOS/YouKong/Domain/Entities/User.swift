import Foundation

struct User: Codable, Identifiable, Equatable {
    let id: String
    let phone: String
    let nickname: String
    let avatar: String?
    let createdAt: Date
    let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case id, phone, nickname, avatar, createdAt, updatedAt
    }
}

struct UserProfile: Codable, Identifiable, Equatable, Hashable {
    let id: String
    let nickname: String
    let avatar: String?

    func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }
}

struct LoginResult: Codable, Equatable {
    let token: String
    let user: User
    let isNewUser: Bool
}
