package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.LoginResponse
import com.youkong.core.network.model.RefreshTokenRequest
import com.youkong.core.network.model.SendSmsRequest
import com.youkong.core.network.model.TokenResponse
import com.youkong.core.network.model.VerifySmsRequest
import retrofit2.http.Body
import retrofit2.http.POST

interface AuthApi {

    @POST("auth/sms/send")
    suspend fun sendSms(@Body request: SendSmsRequest): ApiResponse<Unit>

    @POST("auth/sms/verify")
    suspend fun verifySms(@Body request: VerifySmsRequest): ApiResponse<LoginResponse>

    @POST("auth/refresh")
    suspend fun refreshToken(@Body request: RefreshTokenRequest): ApiResponse<TokenResponse>
}
