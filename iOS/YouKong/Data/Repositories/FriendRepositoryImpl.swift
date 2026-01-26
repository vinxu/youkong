//
//  FriendRepositoryImpl.swift
//  YouKong
//
//  好友系统 Repository 实现
//

import Foundation

final class FriendRepositoryImpl: FriendRepositoryProtocol {
    private let apiClient = APIClient.shared

    func getFriends() async throws -> [Friend] {
        try await apiClient.request(.getFriends)
    }

    func deleteFriend(userId: String) async throws {
        try await apiClient.requestWithEmptyResponse(.deleteFriend(userId: userId))
    }

    func getFriendsInvitedByMe() async throws -> [InvitedFriend] {
        try await apiClient.request(.getFriendsInvitedByMe)
    }

    func getFriendsWhoInvitedMe() async throws -> [InvitedFriend] {
        try await apiClient.request(.getFriendsWhoInvitedMe)
    }
}
