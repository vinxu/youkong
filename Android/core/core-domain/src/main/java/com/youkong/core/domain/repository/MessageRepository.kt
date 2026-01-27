package com.youkong.core.domain.repository

import com.youkong.core.domain.model.Conversation
import com.youkong.core.domain.model.Message
import com.youkong.core.domain.model.MessageType
import kotlinx.coroutines.flow.Flow
import kotlinx.serialization.json.JsonObject

interface MessageRepository {
    val conversations: Flow<List<Conversation>>

    suspend fun getConversations(): Result<List<Conversation>>
    suspend fun getMessages(conversationId: String): Result<List<Message>>
    suspend fun sendMessage(
        conversationId: String,
        type: MessageType,
        content: String?,
        metadata: JsonObject?,
    ): Result<Message>

    // 实时消息流
    fun observeMessages(conversationId: String): Flow<Message>

    // 获取或创建与指定用户的会话
    suspend fun getOrCreateConversation(userId: String): Result<Conversation>

    // Agent 自动回复
    suspend fun agentReply(conversationId: String): Result<Message>
}
