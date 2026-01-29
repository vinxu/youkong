package com.youkong.core.websocket

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

/**
 * WebSocket 消息类型
 */
object WebSocketMessageType {
    const val PING = "PING"
    const val PONG = "PONG"
    const val NEW_MESSAGE = "NEW_MESSAGE"
    const val MESSAGE_READ = "MESSAGE_READ"
    const val TYPING = "TYPING"
}

/**
 * WebSocket 消息包装
 */
@Serializable
data class WebSocketMessage(
    val type: String,
    val data: JsonObject? = null,
)

/**
 * NEW_MESSAGE 消息数据
 */
@Serializable
data class NewMessageData(
    @SerialName("conversation_id")
    val conversationId: String,
    val message: MessageData,
)

/**
 * 消息数据
 */
@Serializable
data class MessageData(
    val id: String,
    val sender: SenderData,
    val type: String,
    val content: String? = null,
    val metadata: JsonObject? = null,
    @SerialName("createdAt")
    val createdAt: String,
    @SerialName("isRead")
    val isRead: Boolean = false,
)

/**
 * 发送者数据
 */
@Serializable
data class SenderData(
    val id: String,
    val nickname: String,
    val avatar: String? = null,
)
