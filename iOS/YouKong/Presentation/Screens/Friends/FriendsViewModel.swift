//
//  FriendsViewModel.swift
//  YouKong
//
//  好友列表 ViewModel
//

import Foundation
import Combine
import Factory

enum FriendTab: String, CaseIterable {
    case all = "全部"
    case invitedByMe = "我邀请的"
    case invitedMe = "邀请我的"
}

@MainActor
final class FriendsViewModel: ObservableObject {
    @Published var selectedTab: FriendTab = .all
    @Published var friends: [Friend] = []
    @Published var invitedByMe: [InvitedFriend] = []
    @Published var invitedMe: [InvitedFriend] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Injected(\.friendRepository) private var repository

    func loadFriends() async {
        isLoading = true
        defer { isLoading = false }

        do {
            async let allFriends = repository.getFriends()
            async let invitedByMeFriends = repository.getFriendsInvitedByMe()
            async let invitedMeFriends = repository.getFriendsWhoInvitedMe()

            friends = try await allFriends
            invitedByMe = try await invitedByMeFriends
            invitedMe = try await invitedMeFriends
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func refresh() async {
        await loadFriends()
    }

    func deleteFriend(userId: String) async {
        do {
            try await repository.deleteFriend(userId: userId)
            friends.removeAll { $0.user.id == userId }
            invitedByMe.removeAll { $0.user.id == userId }
            invitedMe.removeAll { $0.user.id == userId }
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
