package com.youkong.core.domain.repository

import com.youkong.core.domain.model.MyInvite
import com.youkong.core.domain.model.User
import com.youkong.core.domain.model.UserProfile
import kotlinx.coroutines.flow.Flow

interface UserRepository {
    val currentUser: Flow<User?>

    suspend fun getCurrentUser(): Result<User>
    suspend fun updateUser(nickname: String?, avatar: String?): Result<User>
    suspend fun searchUsers(keyword: String): Result<List<UserProfile>>
    suspend fun getUser(userId: String): Result<UserProfile>

    // 邀请相关
    suspend fun getMyInvite(): Result<MyInvite>
    fun getMyPosterUrl(): String
}
