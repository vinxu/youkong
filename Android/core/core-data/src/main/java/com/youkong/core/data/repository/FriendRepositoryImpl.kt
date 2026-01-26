package com.youkong.core.data.repository

import com.youkong.core.data.mapper.toDomain
import com.youkong.core.domain.model.Friend
import com.youkong.core.domain.repository.FriendRepository
import com.youkong.core.network.api.FriendApi
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class FriendRepositoryImpl @Inject constructor(
    private val friendApi: FriendApi,
) : FriendRepository {

    private val _friends = MutableStateFlow<List<Friend>>(emptyList())
    override val friends: Flow<List<Friend>> = _friends.asStateFlow()

    override suspend fun getFriends(): Result<List<Friend>> {
        return try {
            val response = friendApi.getFriends()
            val data = response.data
            if (response.isSuccess && data != null) {
                val friends = data.map { it.toDomain() }
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
                Result.success(data.map { it.toDomain() })
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
                Result.success(data.map { it.toDomain() })
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
