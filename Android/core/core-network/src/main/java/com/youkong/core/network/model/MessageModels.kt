package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

@Serializable
data class ConversationResponse(
    val id: String,
    val partner: UserProfileResponse,
    @SerialName("lastMessage")
    val lastMessage: MessageResponse? = null,
    @SerialName("unreadCount")
    val unreadCount: Int,
    @SerialName("createdAt")
    val createdAt: String,
)

@Serializable
data class MessageResponse(
    val id: String,
    val sender: UserProfileResponse,
    val type: String,
    val content: String? = null,
    val metadata: JsonObject? = null,
    @SerialName("createdAt")
    val createdAt: String,
    @SerialName("isRead")
    val isRead: Boolean,
)

@Serializable
data class SendMessageRequest(
    val type: String,
    val content: String? = null,
    val metadata: JsonObject? = null,
)

@Serializable
data class CreateConversationRequest(
    @SerialName("userId")
    val userId: String,
)
