package com.youkong.core.domain.repository

import com.youkong.core.domain.model.LoginResult
import kotlinx.coroutines.flow.Flow

interface AuthRepository {
    val isLoggedIn: Flow<Boolean>

    suspend fun sendSms(phone: String): Result<Unit>
    suspend fun verifySms(phone: String, code: String): Result<LoginResult>
    suspend fun logout()
}
