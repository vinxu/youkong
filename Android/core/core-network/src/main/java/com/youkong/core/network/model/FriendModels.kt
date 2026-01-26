package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class FriendResponse(
    val user: UserProfileResponse,
    val source: String,
    @SerialName("circleName")
    val circleName: String? = null,
    @SerialName("createdAt")
    val createdAt: String,
)
