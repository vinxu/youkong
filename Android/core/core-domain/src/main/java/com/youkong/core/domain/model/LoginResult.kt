package com.youkong.core.domain.model

data class LoginResult(
    val token: String,
    val user: User,
    val isNewUser: Boolean,
)
