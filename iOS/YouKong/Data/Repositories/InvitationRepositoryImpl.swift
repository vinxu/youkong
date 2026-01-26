//
//  InvitationRepositoryImpl.swift
//  YouKong
//
//  邀请系统 Repository 实现
//

import Foundation

final class InvitationRepositoryImpl: InvitationRepositoryProtocol {
    private let apiClient = APIClient.shared
    private let baseURL = "http://49.232.13.41:8080"

    func createInvitation(circleId: String?, maxUses: Int?, expiresDays: Int?) async throws -> InvitationDetail {
        let request = CreateInvitationRequest(circleId: circleId, maxUses: maxUses, expiresDays: expiresDays)
        return try await apiClient.request(.createInvitation(request: request))
    }

    func getMyInvitations() async throws -> [Invitation] {
        try await apiClient.request(.getMyInvitations)
    }

    func getPublicInvitationInfo(code: String) async throws -> PublicInvitationInfo {
        try await apiClient.request(.getPublicInvitationInfo(code: code))
    }

    func getInvitationDetail(id: String) async throws -> InvitationDetail {
        try await apiClient.request(.getInvitationDetail(id: id))
    }

    func disableInvitation(id: String) async throws {
        try await apiClient.requestWithEmptyResponse(.disableInvitation(id: id))
    }

    func acceptInvitation(code: String) async throws -> AcceptInvitationResponse {
        try await apiClient.request(.acceptInvitation(code: code))
    }

    func getInvitationPosterURL(id: String) -> URL? {
        URL(string: "\(baseURL)/api/v1/invitations/\(id)/poster")
    }

    func getInvitationQRCodeURL(id: String) -> URL? {
        URL(string: "\(baseURL)/api/v1/invitations/\(id)/qrcode")
    }
}
