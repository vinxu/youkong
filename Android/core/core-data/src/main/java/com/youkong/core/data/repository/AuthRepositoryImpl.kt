package com.youkong.core.data.repository

import com.youkong.core.data.mapper.toDomain
import com.youkong.core.datastore.TokenManager
import com.youkong.core.datastore.UserPreferences
import com.youkong.core.domain.model.LoginResult
import com.youkong.core.domain.repository.AuthRepository
import com.youkong.core.network.api.AuthApi
import com.youkong.core.network.model.SendSmsRequest
import com.youkong.core.network.model.VerifySmsRequest
import kotlinx.coroutines.flow.Flow
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AuthRepositoryImpl @Inject constructor(
    private val authApi: AuthApi,
    private val tokenManager: TokenManager,
    private val userPreferences: UserPreferences,
) : AuthRepository {

    override val isLoggedIn: Flow<Boolean> = tokenManager.isLoggedIn

    override suspend fun sendSms(phone: String): Result<Unit> {
        return try {
            val response = authApi.sendSms(SendSmsRequest(phone))
            if (response.isSuccess) {
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun verifySms(phone: String, code: String): Result<LoginResult> {
        return try {
            val response = authApi.verifySms(VerifySmsRequest(phone, code))
            val loginData = response.data
            if (response.isSuccess && loginData != null) {
                val user = loginData.user.toDomain()

                // 保存 token
                tokenManager.saveAccessToken(loginData.token)

                // 保存用户信息
                userPreferences.saveUser(
                    id = user.id,
                    phone = user.phone,
                    nickname = user.nickname,
                    avatar = user.avatar,
                    isNewUser = loginData.isNewUser,
                )

                Result.success(
                    LoginResult(
                        token = loginData.token,
                        user = user,
                        isNewUser = loginData.isNewUser,
                    )
                )
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun logout() {
        tokenManager.clearTokens()
        userPreferences.clear()
    }
}
