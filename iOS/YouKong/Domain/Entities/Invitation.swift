//
//  Invitation.swift
//  YouKong
//
//  邀请系统实体定义
//

import Foundation

// MARK: - 邀请状态

enum InvitationStatus: String, Codable {
    case active = "ACTIVE"
    case disabled = "DISABLED"
    case expired = "EXPIRED"
}

// MARK: - 邀请关联的圈子简化信息

struct InvitationCircle: Codable, Equatable {
    let id: String
    let name: String
    let emoji: String
    let color: String?
    let memberCount: Int?
}

// MARK: - 邀请基础信息（用于列表展示）

struct Invitation: Codable, Identifiable, Equatable {
    let id: String
    let code: String
    let inviteUrl: String
    let circle: InvitationCircle?
    let maxUses: Int
    let useCount: Int
    let expiresAt: Date?
    let status: InvitationStatus
    let isValid: Bool
    let createdAt: Date
}

// MARK: - 邀请详情（包含邀请者信息）

struct InvitationDetail: Codable, Identifiable, Equatable {
    let id: String
    let code: String
    let inviteUrl: String
    let qrcodeUrl: String?
    let inviter: UserProfile?
    let circle: InvitationCircle?
    let maxUses: Int
    let useCount: Int
    let expiresAt: Date?
    let status: InvitationStatus
    let isValid: Bool
    let createdAt: Date
}

// MARK: - 公开邀请信息（无需认证）

struct PublicInvitationInfo: Codable, Equatable {
    let inviter: UserProfile
    let circle: InvitationCircle?
    let isValid: Bool
}

// MARK: - 创建邀请请求

struct CreateInvitationRequest: Codable {
    let circleId: String?
    let maxUses: Int?
    let expiresDays: Int?

    init(circleId: String? = nil, maxUses: Int? = nil, expiresDays: Int? = nil) {
        self.circleId = circleId
        self.maxUses = maxUses
        self.expiresDays = expiresDays
    }
}

// MARK: - 接受邀请响应

struct AcceptInvitationResponse: Codable {
    let joinedCircle: InvitationCircle?
}
