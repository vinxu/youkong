package com.youkong.core.data.mapper

import com.youkong.core.domain.model.User
import com.youkong.core.domain.model.UserProfile
import com.youkong.core.network.model.UserProfileResponse
import com.youkong.core.network.model.UserResponse
import kotlinx.datetime.Instant

fun UserResponse.toModel(): User = User(
    id = id,
    phone = phone,
    nickname = nickname,
    avatar = avatar,
    createdAt = Instant.parse(createdAt),
    updatedAt = Instant.parse(updatedAt),
)

fun UserProfileResponse.toModel(): UserProfile = UserProfile(
    id = id,
    nickname = nickname,
    avatar = avatar,
)
