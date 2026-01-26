package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class CircleResponse(
    val id: String,
    val name: String,
    val emoji: String,
    val color: String,
    @SerialName("ownerId")
    val ownerId: String,
    @SerialName("createdAt")
    val createdAt: String,
    @SerialName("updatedAt")
    val updatedAt: String? = null,
)

@Serializable
data class CircleDetailResponse(
    val id: String,
    val name: String,
    val emoji: String,
    val color: String,
    @SerialName("ownerId")
    val ownerId: String,
    @SerialName("memberCount")
    val memberCount: Int,
    val members: List<UserProfileResponse>? = null,
    @SerialName("createdAt")
    val createdAt: String,
)

@Serializable
data class CreateCircleRequest(
    val name: String,
    val emoji: String,
    val color: String,
)

@Serializable
data class UpdateCircleRequest(
    val name: String? = null,
    val emoji: String? = null,
    val color: String? = null,
)

@Serializable
data class AddMemberRequest(
    val userId: String,
)
