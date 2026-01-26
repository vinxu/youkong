package com.youkong.core.database.entity

import androidx.room.Entity
import androidx.room.ForeignKey
import androidx.room.Index
import androidx.room.PrimaryKey
import com.youkong.core.domain.model.Message
import com.youkong.core.domain.model.MessageType
import com.youkong.core.domain.model.UserProfile
import kotlinx.datetime.Instant
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject

@Entity(
    tableName = "messages",
    indices = [Index("conversationId", "createdAt")]
)
data class MessageEntity(
    @PrimaryKey
    val id: String,
    val conversationId: String,
    val senderId: String,
    val senderNickname: String,
    val senderAvatar: String?,
    val type: String,
    val content: String?,
    val metadata: String?, // JSON string
    val createdAt: Long,
    val isRead: Boolean,
)

fun MessageEntity.toDomain(): Message = Message(
    id = id,
    sender = UserProfile(
        id = senderId,
        nickname = senderNickname,
        avatar = senderAvatar,
    ),
    type = MessageType.valueOf(type),
    content = content,
    metadata = metadata?.let { Json.decodeFromString<JsonObject>(it) },
    createdAt = Instant.fromEpochMilliseconds(createdAt),
    isRead = isRead,
)

fun Message.toEntity(conversationId: String): MessageEntity = MessageEntity(
    id = id,
    conversationId = conversationId,
    senderId = sender.id,
    senderNickname = sender.nickname,
    senderAvatar = sender.avatar,
    type = type.name,
    content = content,
    metadata = metadata?.toString(),
    createdAt = createdAt.toEpochMilliseconds(),
    isRead = isRead,
)
