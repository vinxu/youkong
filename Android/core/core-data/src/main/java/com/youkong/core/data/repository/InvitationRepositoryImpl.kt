package com.youkong.core.data.repository

import com.youkong.core.data.mapper.toDomain
import com.youkong.core.domain.model.InviteCircle
import com.youkong.core.domain.model.Invitation
import com.youkong.core.domain.model.PublicInvitation
import com.youkong.core.domain.repository.InvitationRepository
import com.youkong.core.network.BuildConfig
import com.youkong.core.network.api.InvitationApi
import com.youkong.core.network.model.CreateInvitationRequest
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class InvitationRepositoryImpl @Inject constructor(
    private val invitationApi: InvitationApi,
) : InvitationRepository {

    override suspend fun createInvitation(
        circleId: String?,
        maxUses: Int?,
        expiresDays: Int?
    ): Result<Invitation> {
        return try {
            val response = invitationApi.createInvitation(
                CreateInvitationRequest(circleId, maxUses, expiresDays)
            )
            val data = response.data
            if (response.isSuccess && data != null) {
                Result.success(data.toDomain())
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getMyInvitations(): Result<List<Invitation>> {
        return try {
            val response = invitationApi.getMyInvitations()
            val data = response.data
            if (response.isSuccess && data != null) {
                Result.success(data.map { it.toDomain() })
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getPublicInvitation(code: String): Result<PublicInvitation> {
        return try {
            val response = invitationApi.getPublicInvitation(code)
            val data = response.data
            if (response.isSuccess && data != null) {
                Result.success(data.toDomain())
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getInvitationDetail(id: String): Result<Invitation> {
        return try {
            val response = invitationApi.getInvitationDetail(id)
            val data = response.data
            if (response.isSuccess && data != null) {
                Result.success(data.toDomain())
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun disableInvitation(id: String): Result<Unit> {
        return try {
            val response = invitationApi.disableInvitation(id)
            if (response.isSuccess) {
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun acceptInvitation(code: String): Result<InviteCircle?> {
        return try {
            val response = invitationApi.acceptInvitation(code)
            val data = response.data
            if (response.isSuccess) {
                Result.success(data?.joinedCircle?.toDomain())
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override fun getInvitationPosterUrl(id: String): String {
        return "${BuildConfig.BASE_URL}invitations/$id/poster"
    }

    override fun getInvitationQrCodeUrl(id: String): String {
        return "${BuildConfig.BASE_URL}invitations/$id/qrcode"
    }
}
