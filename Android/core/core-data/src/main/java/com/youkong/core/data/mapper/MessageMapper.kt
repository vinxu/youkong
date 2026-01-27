package com.youkong.core.data.mapper

import com.youkong.core.domain.model.Conversation
import com.youkong.core.domain.model.Message
import com.youkong.core.domain.model.MessageType
import com.youkong.core.network.model.ConversationResponse
import com.youkong.core.network.model.MessageResponse
import kotlinx.datetime.Instant

fun ConversationResponse.toModel(): Conversation = Conversation(
    id = id,
    partner = partner.toModel(),
    lastMessage = lastMessage?.toModel(),
    unreadCount = unreadCount,
    createdAt = Instant.parse(createdAt),
)

fun MessageResponse.toModel(): Message = Message(
    id = id,
    sender = sender.toModel(),
    type = MessageType.valueOf(type),
    content = content,
    metadata = metadata,
    createdAt = Instant.parse(createdAt),
    isRead = isRead,
)
