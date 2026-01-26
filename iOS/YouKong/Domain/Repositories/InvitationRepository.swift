//
//  InvitationRepository.swift
//  YouKong
//
//  邀请系统 Repository 协议
//

import Foundation

protocol InvitationRepositoryProtocol {
    /// 创建邀请
    func createInvitation(circleId: String?, maxUses: Int?, expiresDays: Int?) async throws -> InvitationDetail

    /// 获取我的邀请列表
    func getMyInvitations() async throws -> [Invitation]

    /// 获取邀请详情（公开，无需认证）
    func getPublicInvitationInfo(code: String) async throws -> PublicInvitationInfo

    /// 获取邀请完整信息
    func getInvitationDetail(id: String) async throws -> InvitationDetail

    /// 禁用邀请
    func disableInvitation(id: String) async throws

    /// 接受邀请
    func acceptInvitation(code: String) async throws -> AcceptInvitationResponse

    /// 获取邀请海报 URL
    func getInvitationPosterURL(id: String) -> URL?

    /// 获取邀请二维码 URL
    func getInvitationQRCodeURL(id: String) -> URL?
}
