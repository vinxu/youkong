package com.youkong.core.data.repository

import com.youkong.core.data.mapper.toDomain
import com.youkong.core.database.dao.ConversationDao
import com.youkong.core.database.dao.MessageDao
import com.youkong.core.database.entity.toDomain
import com.youkong.core.database.entity.toEntity
import com.youkong.core.domain.model.Conversation
import com.youkong.core.domain.model.Message
import com.youkong.core.domain.model.MessageType
import com.youkong.core.domain.repository.MessageRepository
import com.youkong.core.network.api.MessageApi
import com.youkong.core.network.model.SendMessageRequest
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.emptyFlow
import kotlinx.coroutines.flow.map
import kotlinx.serialization.json.JsonObject
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class MessageRepositoryImpl @Inject constructor(
    private val messageApi: MessageApi,
    private val conversationDao: ConversationDao,
    private val messageDao: MessageDao,
) : MessageRepository {

    override val conversations: Flow<List<Conversation>> =
        conversationDao.observeAll().map { entities ->
            entities.map { entity ->
                val lastMessage = messageDao.getLastMessage(entity.id)?.toDomain()
                entity.toDomain().copy(lastMessage = lastMessage)
            }
        }

    override suspend fun getConversations(): Result<List<Conversation>> {
        return try {
            val response = messageApi.getConversations()
            val data = response.data
            if (response.isSuccess && data != null) {
                val conversations = data.map { it.toDomain() }
                conversationDao.deleteAll()
                conversationDao.insertAll(conversations.map { it.toEntity() })
                // 保存最后一条消息
                conversations.forEach { conversation ->
                    conversation.lastMessage?.let { message ->
                        messageDao.insert(message.toEntity(conversation.id))
                    }
                }
                Result.success(conversations)
            } else {
                val cached = conversationDao.getAll().map { entity ->
                    val lastMessage = messageDao.getLastMessage(entity.id)?.toDomain()
                    entity.toDomain().copy(lastMessage = lastMessage)
                }
                if (cached.isNotEmpty()) {
                    Result.success(cached)
                } else {
                    Result.failure(Exception(response.message))
                }
            }
        } catch (e: Exception) {
            val cached = conversationDao.getAll().map { entity ->
                val lastMessage = messageDao.getLastMessage(entity.id)?.toDomain()
                entity.toDomain().copy(lastMessage = lastMessage)
            }
            if (cached.isNotEmpty()) {
                Result.success(cached)
            } else {
                Result.failure(e)
            }
        }
    }

    override suspend fun getMessages(conversationId: String): Result<List<Message>> {
        return try {
            val response = messageApi.getMessages(conversationId)
            val data = response.data
            if (response.isSuccess && data != null) {
                val messages = data.map { it.toDomain() }
                messageDao.deleteByConversation(conversationId)
                messageDao.insertAll(messages.map { it.toEntity(conversationId) })
                Result.success(messages)
            } else {
                val cached = messageDao.getMessages(conversationId).map { it.toDomain() }
                if (cached.isNotEmpty()) {
                    Result.success(cached)
                } else {
                    Result.failure(Exception(response.message))
                }
            }
        } catch (e: Exception) {
            val cached = messageDao.getMessages(conversationId).map { it.toDomain() }
            if (cached.isNotEmpty()) {
                Result.success(cached)
            } else {
                Result.failure(e)
            }
        }
    }

    override suspend fun sendMessage(
        conversationId: String,
        type: MessageType,
        content: String?,
        metadata: JsonObject?,
    ): Result<Message> {
        return try {
            val response = messageApi.sendMessage(
                conversationId,
                SendMessageRequest(
                    type = type.name,
                    content = content,
                    metadata = metadata,
                )
            )
            val msgData = response.data
            if (response.isSuccess && msgData != null) {
                val message = msgData.toDomain()
                messageDao.insert(message.toEntity(conversationId))
                Result.success(message)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override fun observeMessages(conversationId: String): Flow<Message> {
        // WebSocket 实时消息会通过 WebSocketManager 处理
        return emptyFlow()
    }
}
