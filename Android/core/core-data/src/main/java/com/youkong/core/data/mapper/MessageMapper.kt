package com.youkong.core.data.mapper

import com.youkong.core.domain.model.Conversation
import com.youkong.core.domain.model.Message
import com.youkong.core.domain.model.MessageType
import com.youkong.core.network.model.ConversationResponse
import com.youkong.core.network.model.MessageResponse
import kotlinx.datetime.Instant

fun ConversationResponse.toDomain(): Conversation = Conversation(
    id = id,
    partner = partner.toDomain(),
    lastMessage = lastMessage?.toDomain(),
    unreadCount = unreadCount,
    createdAt = Instant.parse(createdAt),
)

fun MessageResponse.toDomain(): Message = Message(
    id = id,
    sender = sender.toDomain(),
    type = MessageType.valueOf(type),
    content = content,
    metadata = metadata,
    createdAt = Instant.parse(createdAt),
    isRead = isRead,
)
