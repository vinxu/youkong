//
//  FriendRepository.swift
//  YouKong
//
//  好友系统 Repository 协议
//

import Foundation

protocol FriendRepositoryProtocol {
    /// 获取好友列表
    func getFriends() async throws -> [Friend]

    /// 删除好友
    func deleteFriend(userId: String) async throws

    /// 获取我邀请的好友
    func getFriendsInvitedByMe() async throws -> [InvitedFriend]

    /// 获取邀请我的好友
    func getFriendsWhoInvitedMe() async throws -> [InvitedFriend]
}
