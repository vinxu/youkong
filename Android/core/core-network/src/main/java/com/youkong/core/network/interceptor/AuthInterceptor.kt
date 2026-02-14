package com.youkong.core.network.interceptor

import com.youkong.core.datastore.TokenManager
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import okhttp3.Interceptor
import okhttp3.Response
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AuthInterceptor @Inject constructor(
    private val tokenManager: TokenManager,
) : Interceptor {

    override fun intercept(chain: Interceptor.Chain): Response {
        val originalRequest = chain.request()

        // 不需要认证的接口
        val noAuthPaths = listOf(
            "auth/sms/send",
            "auth/sms/verify",
            "auth/refresh",
            "app/version",
        )
        val path = originalRequest.url.encodedPath
        if (noAuthPaths.any { path.contains(it) }) {
            return chain.proceed(originalRequest)
        }

        // 获取 token
        val token = runBlocking { tokenManager.accessToken.first() }

        if (token.isNullOrBlank()) {
            return chain.proceed(originalRequest)
        }

        // 添加 Authorization header
        val authorizedRequest = originalRequest.newBuilder()
            .header("Authorization", "Bearer $token")
            .build()

        return chain.proceed(authorizedRequest)
    }
}
