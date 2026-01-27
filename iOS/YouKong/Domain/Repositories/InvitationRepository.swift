import Foundation

// MARK: - Invite Repository Protocol (简化版)

protocol InviteRepositoryProtocol {
    /// 获取我的邀请信息
    func getMyInvite() async throws -> MyInviteInfo

    /// 获取我的邀请海报图片数据
    func fetchMyPoster() async throws -> Data

    /// 获取邀请信息（公开，落地页用）
    func getInvitationByCode(code: String) async throws -> InvitationPublicInfo

    /// 接受邀请
    func acceptInvitation(code: String) async throws
}

// MARK: - Friendship Repository Protocol

protocol FriendshipRepositoryProtocol {
    /// 获取好友列表
    func getFriends() async throws -> [FriendInfo]

    /// 删除好友
    func deleteFriend(userId: String) async throws

    /// 我邀请的好友
    func getFriendsInvitedByMe() async throws -> [FriendWithInvitation]

    /// 邀请我的好友
    func getFriendsInvitedMe() async throws -> [FriendWithInvitation]
}
