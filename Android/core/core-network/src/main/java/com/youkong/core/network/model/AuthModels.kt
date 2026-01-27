package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class SendSmsRequest(
    val phone: String,
)

@Serializable
data class VerifySmsRequest(
    val phone: String,
    val code: String,
)

@Serializable
data class RefreshTokenRequest(
    val token: String,
)

@Serializable
data class LoginResponse(
    val token: String,
    val user: UserResponse,
    @SerialName("isNewUser")
    val isNewUser: Boolean,
)

@Serializable
data class TokenResponse(
    val token: String,
)

@Serializable
data class UserResponse(
    val id: String,
    val phone: String,
    val nickname: String,
    val avatar: String? = null,
    @SerialName("createdAt")
    val createdAt: String,
    @SerialName("updatedAt")
    val updatedAt: String,
)

@Serializable
data class UserProfileResponse(
    val id: String,
    val nickname: String,
    val avatar: String? = null,
)

@Serializable
data class UpdateUserRequest(
    val nickname: String? = null,
    val avatar: String? = null,
)

@Serializable
data class SearchUsersRequest(
    val keyword: String,
)

@Serializable
data class MyInviteResponse(
    val code: String,
    val inviteUrl: String,
)
