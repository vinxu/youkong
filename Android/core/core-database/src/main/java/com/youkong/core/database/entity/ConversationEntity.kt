package com.youkong.core.database.entity

import androidx.room.Entity
import androidx.room.PrimaryKey
import com.youkong.core.domain.model.Conversation
import com.youkong.core.domain.model.UserProfile
import kotlinx.datetime.Instant

@Entity(tableName = "conversations")
data class ConversationEntity(
    @PrimaryKey
    val id: String,
    val partnerId: String,
    val partnerNickname: String,
    val partnerAvatar: String?,
    val unreadCount: Int,
    val createdAt: Long,
    val updatedAt: Long,
)

fun ConversationEntity.toDomain(): Conversation = Conversation(
    id = id,
    partner = UserProfile(
        id = partnerId,
        nickname = partnerNickname,
        avatar = partnerAvatar,
    ),
    lastMessage = null, // 需要单独查询
    unreadCount = unreadCount,
    createdAt = Instant.fromEpochMilliseconds(createdAt),
)

fun Conversation.toEntity(): ConversationEntity = ConversationEntity(
    id = id,
    partnerId = partner.id,
    partnerNickname = partner.nickname,
    partnerAvatar = partner.avatar,
    unreadCount = unreadCount,
    createdAt = createdAt.toEpochMilliseconds(),
    updatedAt = lastMessage?.createdAt?.toEpochMilliseconds() ?: createdAt.toEpochMilliseconds(),
)
