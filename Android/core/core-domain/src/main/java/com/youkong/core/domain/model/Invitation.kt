package com.youkong.core.domain.model

import kotlinx.datetime.Instant

data class Invitation(
    val id: String,
    val code: String,
    val inviteUrl: String,
    val inviter: UserProfile? = null,
    val circle: InviteCircle? = null,
    val maxUses: Int,
    val useCount: Int,
    val expiresAt: Instant? = null,
    val status: InvitationStatus,
    val isValid: Boolean,
    val createdAt: Instant,
)

data class InviteCircle(
    val id: String,
    val name: String,
    val emoji: String,
    val color: String? = null,
    val memberCount: Int = 0,
)

data class PublicInvitation(
    val inviter: UserProfile,
    val circle: InviteCircle? = null,
    val isValid: Boolean,
)

enum class InvitationStatus {
    ACTIVE, DISABLED, EXPIRED;

    companion object {
        fun fromString(value: String): InvitationStatus {
            return when (value.uppercase()) {
                "ACTIVE" -> ACTIVE
                "DISABLED" -> DISABLED
                "EXPIRED" -> EXPIRED
                else -> ACTIVE
            }
        }
    }
}
