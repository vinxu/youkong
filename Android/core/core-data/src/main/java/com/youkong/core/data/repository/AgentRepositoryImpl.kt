package com.youkong.core.data.repository

import com.youkong.core.domain.model.Confidence
import com.youkong.core.domain.model.FriendWithProbability
import com.youkong.core.domain.model.UserProfile
import com.youkong.core.domain.repository.AgentRepository
import com.youkong.core.network.api.AgentApi
import com.youkong.core.network.api.FriendApi
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.LocationDataRequest
import com.youkong.core.network.model.ScreenDataRequest
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AgentRepositoryImpl @Inject constructor(
    private val agentApi: AgentApi,
    private val friendApi: FriendApi,
) : AgentRepository {

    private val _friendsProbability = MutableStateFlow<List<FriendWithProbability>>(emptyList())
    val friendsProbability: Flow<List<FriendWithProbability>> = _friendsProbability.asStateFlow()

    override suspend fun reportStatus(
        isActive: Boolean,
        activityType: String,
        sessionDurationMinutes: Int,
        lastActiveMinutesAgo: Int,
        placeType: String?,
        atPlaceSinceMinutes: Int?,
    ): Result<Unit> {
        return try {
            val request = AgentStatusRequest(
                screen = ScreenDataRequest(
                    isActive = isActive,
                    activityType = activityType,
                    sessionDurationMinutes = sessionDurationMinutes,
                    lastActiveMinutesAgo = lastActiveMinutesAgo,
                ),
                location = if (placeType != null && atPlaceSinceMinutes != null) {
                    LocationDataRequest(
                        placeType = placeType,
                        atPlaceSinceMinutes = atPlaceSinceMinutes,
                    )
                } else null,
            )
            val response = agentApi.reportStatus(request)
            if (response.isSuccess) {
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getFriendsProbability(): Result<List<FriendWithProbability>> {
        return try {
            val response = friendApi.getFriendsProbability()
            val data = response.data
            if (response.isSuccess && data != null) {
                val friends = data.friends.map { it.toDomain() }
                _friendsProbability.value = friends
                Result.success(friends)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    private fun com.youkong.core.network.model.FriendProbabilityResponse.toDomain(): FriendWithProbability {
        return FriendWithProbability(
            user = UserProfile(
                id = friendId,
                nickname = name,
                avatar = avatar,
            ),
            probability = if (probability < 0) null else probability,
            confidence = Confidence.fromString(confidence),
            reason = reason,
            color = color,
            emoji = emoji,
            activity = activity,
            updatedAt = updatedAt,
        )
    }
}
