package com.youkong.core.data.repository

import com.youkong.core.data.mapper.toModel
import com.youkong.core.domain.model.Friend
import com.youkong.core.domain.model.FriendRequest
import com.youkong.core.domain.model.FriendRequestStatus
import com.youkong.core.domain.model.SendFriendRequestResult
import com.youkong.core.domain.model.SendRequestStatus
import com.youkong.core.domain.model.UserProfile
import com.youkong.core.domain.repository.FriendRepository
import com.youkong.core.network.api.FriendApi
import com.youkong.core.network.model.FriendRequestResponse
import com.youkong.core.network.model.HandleFriendRequestRequest
import com.youkong.core.network.model.SendFriendRequestRequest
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.datetime.Instant
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class FriendRepositoryImpl @Inject constructor(
    private val friendApi: FriendApi,
) : FriendRepository {

    private val _friends = MutableStateFlow<List<Friend>>(emptyList())
    override val friends: Flow<List<Friend>> = _friends.asStateFlow()

    private val _pendingRequestCount = MutableStateFlow(0)
    override val pendingRequestCount: Flow<Int> = _pendingRequestCount.asStateFlow()

    override suspend fun getFriends(): Result<List<Friend>> {
        return try {
            val response = friendApi.getFriends()
            val data = response.data
            if (response.isSuccess && data != null) {
                val friends = data.map { it.toModel() }
                _friends.value = friends
                Result.success(friends)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun deleteFriend(userId: String): Result<Unit> {
        return try {
            val response = friendApi.deleteFriend(userId)
            if (response.isSuccess) {
                _friends.value = _friends.value.filter { it.user.id != userId }
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getInvitedByMe(): Result<List<Friend>> {
        return try {
            val response = friendApi.getInvitedByMe()
            val data = response.data
            if (response.isSuccess && data != null) {
                Result.success(data.map { it.toModel() })
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getInvitedMe(): Result<List<Friend>> {
        return try {
            val response = friendApi.getInvitedMe()
            val data = response.data
            if (response.isSuccess && data != null) {
                Result.success(data.map { it.toModel() })
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun sendFriendRequest(phone: String, message: String?): Result<SendFriendRequestResult> {
        return try {
            val response = friendApi.sendFriendRequest(SendFriendRequestRequest(phone, message))
            val data = response.data
            if (response.isSuccess && data != null) {
                val result = SendFriendRequestResult(
                    requestId = data.requestId,
                    user = UserProfile(
                        id = data.user.id,
                        nickname = data.user.nickname,
                        avatar = data.user.avatar,
                    ),
                    status = SendRequestStatus.fromString(data.status),
                )
                // 如果直接成为好友（双方已互发请求），刷新好友列表
                if (result.status == SendRequestStatus.ACCEPTED) {
                    getFriends()
                }
                Result.success(result)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getReceivedRequests(): Result<List<FriendRequest>> {
        return try {
            val response = friendApi.getReceivedRequests()
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

    override suspend fun getSentRequests(): Result<List<FriendRequest>> {
        return try {
            val response = friendApi.getSentRequests()
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

    override suspend fun handleFriendRequest(requestId: String, accept: Boolean): Result<Unit> {
        return try {
            val response = friendApi.handleFriendRequest(requestId, HandleFriendRequestRequest(accept))
            if (response.isSuccess) {
                // 如果同意，刷新好友列表和待处理数量
                if (accept) {
                    getFriends()
                }
                getPendingRequestCount()
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getPendingRequestCount(): Result<Int> {
        return try {
            val response = friendApi.getPendingRequestCount()
            val data = response.data
            if (response.isSuccess && data != null) {
                _pendingRequestCount.value = data.count
                Result.success(data.count)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    private fun FriendRequestResponse.toDomain(): FriendRequest {
        return FriendRequest(
            id = id,
            user = UserProfile(
                id = user.id,
                nickname = user.nickname,
                avatar = user.avatar,
            ),
            message = message,
            status = FriendRequestStatus.fromString(status),
            createdAt = Instant.parse(createdAt),
        )
    }
}
