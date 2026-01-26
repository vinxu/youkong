package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class InviterResponse(
    val id: String,
    val nickname: String,
    val avatar: String? = null,
)

@Serializable
data class InviteCircleResponse(
    val id: String,
    val name: String,
    val emoji: String,
    val color: String? = null,
    @SerialName("memberCount")
    val memberCount: Int = 0,
)

@Serializable
data class CreateInvitationRequest(
    @SerialName("circleId")
    val circleId: String? = null,
    @SerialName("maxUses")
    val maxUses: Int? = null,
    @SerialName("expiresDays")
    val expiresDays: Int? = null,
)

@Serializable
data class InvitationResponse(
    val id: String,
    val code: String,
    @SerialName("inviteUrl")
    val inviteUrl: String,
    val inviter: InviterResponse? = null,
    val circle: InviteCircleResponse? = null,
    @SerialName("maxUses")
    val maxUses: Int,
    @SerialName("useCount")
    val useCount: Int,
    @SerialName("expiresAt")
    val expiresAt: String? = null,
    val status: String,
    @SerialName("isValid")
    val isValid: Boolean,
    @SerialName("createdAt")
    val createdAt: String,
)

@Serializable
data class PublicInvitationResponse(
    val inviter: InviterResponse,
    val circle: InviteCircleResponse? = null,
    @SerialName("isValid")
    val isValid: Boolean,
)

@Serializable
data class AcceptInvitationResponse(
    @SerialName("joinedCircle")
    val joinedCircle: InviteCircleResponse? = null,
)
