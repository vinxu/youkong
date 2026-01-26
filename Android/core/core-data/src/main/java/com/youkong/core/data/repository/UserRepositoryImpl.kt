package com.youkong.core.data.repository

import com.youkong.core.data.mapper.toDomain
import com.youkong.core.database.dao.UserDao
import com.youkong.core.database.entity.toDomain
import com.youkong.core.database.entity.toEntity
import com.youkong.core.datastore.UserPreferences
import com.youkong.core.domain.model.User
import com.youkong.core.domain.model.UserProfile
import com.youkong.core.domain.repository.UserRepository
import com.youkong.core.network.api.UserApi
import com.youkong.core.network.model.UpdateUserRequest
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class UserRepositoryImpl @Inject constructor(
    private val userApi: UserApi,
    private val userDao: UserDao,
    private val userPreferences: UserPreferences,
) : UserRepository {

    override val currentUser: Flow<User?> = userPreferences.userId.map { userId ->
        userId?.let { userDao.getUser(it)?.toDomain() }
    }

    override suspend fun getCurrentUser(): Result<User> {
        return try {
            val response = userApi.getCurrentUser()
            val data = response.data
            if (response.isSuccess && data != null) {
                val user = data.toDomain()
                userDao.insert(user.toEntity())
                userPreferences.saveUser(
                    id = user.id,
                    phone = user.phone,
                    nickname = user.nickname,
                    avatar = user.avatar,
                    isNewUser = false,
                )
                Result.success(user)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            // 尝试从本地获取
            val userId = userPreferences.userId.first()
            if (userId != null) {
                val cachedUser = userDao.getUser(userId)?.toDomain()
                if (cachedUser != null) {
                    return Result.success(cachedUser)
                }
            }
            Result.failure(e)
        }
    }

    override suspend fun updateUser(nickname: String?, avatar: String?): Result<User> {
        return try {
            val response = userApi.updateCurrentUser(UpdateUserRequest(nickname, avatar))
            val data = response.data
            if (response.isSuccess && data != null) {
                val user = data.toDomain()
                userDao.insert(user.toEntity())
                nickname?.let { userPreferences.updateNickname(it) }
                avatar?.let { userPreferences.updateAvatar(it) }
                Result.success(user)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun searchUsers(keyword: String): Result<List<UserProfile>> {
        return try {
            val response = userApi.searchUsers(keyword)
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

    override suspend fun getUser(userId: String): Result<UserProfile> {
        return try {
            val response = userApi.getUser(userId)
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
}
