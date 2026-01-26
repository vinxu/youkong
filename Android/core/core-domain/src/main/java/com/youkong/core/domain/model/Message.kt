package com.youkong.core.domain.model

import kotlinx.datetime.Instant
import kotlinx.serialization.json.JsonObject

data class Conversation(
    val id: String,
    val partner: UserProfile,
    val lastMessage: Message? = null,
    val unreadCount: Int,
    val createdAt: Instant,
)

data class Message(
    val id: String,
    val sender: UserProfile,
    val type: MessageType,
    val content: String? = null,
    val metadata: JsonObject? = null,
    val createdAt: Instant,
    val isRead: Boolean,
)

enum class MessageType {
    TEXT,
    AVAILABILITY_CARD,
    CONFIRM_REQUEST,
    CONFIRM_RESPONSE,
}
