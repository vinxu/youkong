package com.youkong.core.domain.model

import kotlinx.datetime.Instant

data class User(
    val id: String,
    val phone: String,
    val nickname: String,
    val avatar: String? = null,
    val createdAt: Instant,
    val updatedAt: Instant,
)

data class UserProfile(
    val id: String,
    val nickname: String,
    val avatar: String? = null,
)

data class MyInvite(
    val code: String,
    val inviteUrl: String,
)
