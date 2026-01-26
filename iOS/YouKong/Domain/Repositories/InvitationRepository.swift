import Foundation

// MARK: - Invitation Repository Protocol

protocol InvitationRepositoryProtocol {
    /// 创建邀请链接
    func createInvitation(circleId: String?, maxUses: Int?, expiresDays: Int?) async throws -> Invitation

    /// 获取我创建的邀请列表
    func getMyInvitations() async throws -> [Invitation]

    /// 获取邀请信息（公开，落地页用）
    func getInvitationByCode(code: String) async throws -> InvitationPublicInfo

    /// 接受邀请（加好友+加圈子）
    func acceptInvitation(code: String) async throws -> AcceptInvitationResponse

    /// 禁用邀请链接
    func disableInvitation(id: String) async throws

    /// 获取邀请海报图片 URL
    func getInvitationPosterURL(id: String) -> URL?

    /// 获取邀请二维码 URL
    func getInvitationQRCodeURL(id: String) -> URL?
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
