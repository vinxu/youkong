package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class CardRoomMemberBrief(
    @SerialName("user_id") val userId: String = "",
    val nickname: String = "",
    val avatar: String = "",
    val gender: String = "",
    @SerialName("rive_character") val riveCharacter: String = "",
)

@Serializable
data class JoinCardRequest(
    @SerialName("owner_id") val ownerId: String,
    val emoji: String = "",
    @SerialName("status_text") val statusText: String = "",
)

@Serializable
data class JoinCardResponse(
    @SerialName("room_id") val roomId: String = "",
    val members: List<CardRoomMemberBrief> = emptyList(),
)

@Serializable
data class SendRoomMessageRequest(
    val content: String,
)

@Serializable
data class CardRoomMessageResponse(
    val id: String = "",
    val sender: CardRoomMessageSender = CardRoomMessageSender(),
    val content: String = "",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("is_read") val isRead: Boolean = false,
)

@Serializable
data class CardRoomMessageSender(
    val id: String = "",
    val nickname: String = "",
    val avatar: String = "",
)
