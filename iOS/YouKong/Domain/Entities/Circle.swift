import Foundation
import SwiftUI

struct FriendCircle: Codable, Identifiable, Equatable, Hashable {
    let id: String
    let name: String
    let emoji: String
    let color: String
    let ownerId: String
    let createdAt: Date

    var colorValue: Color {
        Color(hex: color)
    }
}

struct FriendCircleDetail: Codable, Identifiable, Equatable {
    let id: String
    let name: String
    let emoji: String
    let color: String
    let ownerId: String
    let memberCount: Int
    let members: [UserProfile]?
    let createdAt: Date

    var colorValue: Color {
        Color(hex: color)
    }
}

struct CreateCircleRequest: Codable {
    let name: String
    let emoji: String
    let color: String
}

struct UpdateCircleRequest: Codable {
    let name: String
    let emoji: String
    let color: String
}
