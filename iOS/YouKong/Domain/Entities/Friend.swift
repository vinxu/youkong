//
//  Friend.swift
//  YouKong
//
//  好友系统实体定义
//

import Foundation

// MARK: - 好友来源

enum FriendSource: String, Codable {
    case invitation = "INVITATION"
    case search = "SEARCH"
    case manual = "MANUAL"
}

// MARK: - 好友信息

struct Friend: Codable, Identifiable, Equatable {
    let user: UserProfile
    let source: FriendSource
    let createdAt: Date

    var id: String { user.id }
}

// MARK: - 邀请关系好友（包含圈子信息）

struct InvitedFriend: Codable, Identifiable, Equatable {
    let user: UserProfile
    let source: FriendSource
    let circleName: String?
    let createdAt: Date

    var id: String { user.id }
}
