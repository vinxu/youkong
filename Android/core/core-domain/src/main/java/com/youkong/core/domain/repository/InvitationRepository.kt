package com.youkong.core.domain.repository

import com.youkong.core.domain.model.InviteCircle
import com.youkong.core.domain.model.Invitation
import com.youkong.core.domain.model.PublicInvitation

interface InvitationRepository {
    suspend fun createInvitation(
        circleId: String? = null,
        maxUses: Int? = null,
        expiresDays: Int? = null
    ): Result<Invitation>

    suspend fun getMyInvitations(): Result<List<Invitation>>

    suspend fun getPublicInvitation(code: String): Result<PublicInvitation>

    suspend fun getInvitationDetail(id: String): Result<Invitation>

    suspend fun disableInvitation(id: String): Result<Unit>

    suspend fun acceptInvitation(code: String): Result<InviteCircle?>

    fun getInvitationPosterUrl(id: String): String

    fun getInvitationQrCodeUrl(id: String): String
}
