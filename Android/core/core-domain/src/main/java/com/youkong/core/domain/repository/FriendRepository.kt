package com.youkong.core.domain.repository

import com.youkong.core.domain.model.Friend
import kotlinx.coroutines.flow.Flow

interface FriendRepository {
    val friends: Flow<List<Friend>>

    suspend fun getFriends(): Result<List<Friend>>

    suspend fun deleteFriend(userId: String): Result<Unit>

    suspend fun getInvitedByMe(): Result<List<Friend>>

    suspend fun getInvitedMe(): Result<List<Friend>>
}
