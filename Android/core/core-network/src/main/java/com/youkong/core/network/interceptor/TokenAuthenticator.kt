package com.youkong.core.network.interceptor

import com.youkong.core.datastore.TokenManager
import com.youkong.core.network.api.AuthApi
import com.youkong.core.network.model.RefreshTokenRequest
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import okhttp3.Authenticator
import okhttp3.Request
import okhttp3.Response
import okhttp3.Route
import javax.inject.Inject
import javax.inject.Provider
import javax.inject.Singleton

@Singleton
class TokenAuthenticator @Inject constructor(
    private val tokenManager: TokenManager,
    private val authApiProvider: Provider<AuthApi>,
) : Authenticator {

    override fun authenticate(route: Route?, response: Response): Request? {
        // 避免无限循环
        if (response.request.header("Authorization-Retry") != null) {
            return null
        }

        // 获取当前 token
        val currentToken = runBlocking { tokenManager.accessToken.first() }
        if (currentToken.isNullOrBlank()) {
            return null
        }

        synchronized(this) {
            // 再次检查 token 是否已被其他线程刷新
            val latestToken = runBlocking { tokenManager.accessToken.first() }
            if (latestToken != currentToken) {
                // Token 已被刷新，使用新 token 重试
                return response.request.newBuilder()
                    .header("Authorization", "Bearer $latestToken")
                    .header("Authorization-Retry", "true")
                    .build()
            }

            // 尝试刷新 token
            return try {
                val refreshResponse = runBlocking {
                    authApiProvider.get().refreshToken(RefreshTokenRequest(currentToken))
                }

                if (refreshResponse.isSuccess && refreshResponse.data != null) {
                    val newToken = refreshResponse.data.token
                    runBlocking { tokenManager.saveAccessToken(newToken) }

                    response.request.newBuilder()
                        .header("Authorization", "Bearer $newToken")
                        .header("Authorization-Retry", "true")
                        .build()
                } else {
                    // 刷新失败，清除 token
                    runBlocking { tokenManager.clearTokens() }
                    null
                }
            } catch (e: Exception) {
                runBlocking { tokenManager.clearTokens() }
                null
            }
        }
    }
}
