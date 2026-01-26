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

// 发送好友请求
@Serializable
data class SendFriendRequestRequest(
    val phone: String,
    val message: String? = null,
)

@Serializable
data class SendFriendRequestResponse(
    @SerialName("requestId")
    val requestId: String,
    val user: UserProfileResponse,
    val status: String, // PENDING, ALREADY_FRIENDS, ALREADY_REQUESTED, ACCEPTED
)

// 好友请求
@Serializable
data class FriendRequestResponse(
    val id: String,
    val user: UserProfileResponse,
    val message: String? = null,
    val status: String, // PENDING, ACCEPTED, REJECTED, CANCELLED
    @SerialName("createdAt")
    val createdAt: String,
)

// 处理好友请求
@Serializable
data class HandleFriendRequestRequest(
    val accept: Boolean,
)

// 待处理请求数量
@Serializable
data class FriendRequestCountResponse(
    val count: Int,
)
