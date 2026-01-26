package com.youkong.core.domain.repository

import com.youkong.core.domain.model.Friend
import com.youkong.core.domain.model.FriendRequest
import com.youkong.core.domain.model.SendFriendRequestResult
import kotlinx.coroutines.flow.Flow

interface FriendRepository {
    val friends: Flow<List<Friend>>
    val pendingRequestCount: Flow<Int>

    suspend fun getFriends(): Result<List<Friend>>

    suspend fun deleteFriend(userId: String): Result<Unit>

    suspend fun getInvitedByMe(): Result<List<Friend>>

    suspend fun getInvitedMe(): Result<List<Friend>>

    // 好友请求相关
    suspend fun sendFriendRequest(phone: String, message: String? = null): Result<SendFriendRequestResult>

    suspend fun getReceivedRequests(): Result<List<FriendRequest>>

    suspend fun getSentRequests(): Result<List<FriendRequest>>

    suspend fun handleFriendRequest(requestId: String, accept: Boolean): Result<Unit>

    suspend fun getPendingRequestCount(): Result<Int>
}
