package com.youkong.feature.message.viewmodel

import android.util.Log
import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.datastore.UserPreferences
import com.youkong.core.domain.manager.AppNotificationManager
import com.youkong.core.domain.manager.UnreadMessageManager
import com.youkong.core.domain.model.Message
import com.youkong.core.domain.model.MessageType
import com.youkong.core.domain.repository.MessageRepository
import com.youkong.core.websocket.WebSocketManager
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.datetime.Instant
import javax.inject.Inject

data class ChatUiState(
    val messages: List<Message> = emptyList(),
    val partnerName: String? = null,
    val currentUserId: String? = null,
    val isLoading: Boolean = true,
    val isSending: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class ChatViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val messageRepository: MessageRepository,
    private val userPreferences: UserPreferences,
    private val webSocketManager: WebSocketManager,
) : ViewModel() {

    private val conversationIdParam: String? = savedStateHandle["conversationId"]
    private val partnerIdParam: String? = savedStateHandle["partnerId"]

    private var conversationId: String? = null

    private val _uiState = MutableStateFlow(ChatUiState())
    val uiState: StateFlow<ChatUiState> = _uiState.asStateFlow()

    private val messageIds = mutableSetOf<String>()
    private val messageMutex = Mutex() // 确保消息添加的原子性

    init {
        loadCurrentUser()
        initializeConversation()
        observeWebSocketMessages()
    }

    override fun onCleared() {
        super.onCleared()
        // 离开聊天页面时清除当前会话 ID
        AppNotificationManager.currentConversationId = null
        Log.d("ChatViewModel", "💬 [ChatViewModel] 清除当前会话ID")
    }

    private fun observeWebSocketMessages() {
        viewModelScope.launch {
            webSocketManager.newMessages.collect { newMessageData ->
                // 只处理当前会话的消息
                if (newMessageData.conversationId == conversationId) {
                    val message = convertToMessage(newMessageData.message)
                    addMessageIfNew(message)
                }
            }
        }
    }

    /**
     * 添加消息（如果不存在）
     * 使用 Mutex 确保原子性，避免并发添加导致的重复
     * 类似 iOS 的 addMessageIfNew 实现
     */
    private suspend fun addMessageIfNew(message: Message) {
        messageMutex.withLock {
            // 去重检查
            if (messageIds.contains(message.id)) {
                return
            }

            // 立即加入去重集合
            messageIds.add(message.id)

            // 更新消息列表
            _uiState.update {
                it.copy(messages = it.messages + message)
            }
        }
    }

    private fun convertToMessage(messageData: com.youkong.core.websocket.MessageData): Message {
        return Message(
            id = messageData.id,
            sender = com.youkong.core.domain.model.UserProfile(
                id = messageData.sender.id,
                nickname = messageData.sender.nickname,
                avatar = messageData.sender.avatar
            ),
            type = MessageType.valueOf(messageData.type),
            content = messageData.content,
            metadata = null,
            createdAt = Instant.parse(messageData.createdAt),
            isRead = messageData.isRead
        )
    }

    private fun initializeConversation() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            // 如果有 partnerId，先创建或获取会话
            if (partnerIdParam != null) {
                messageRepository.getOrCreateConversation(partnerIdParam)
                    .onSuccess { conversation ->
                        conversationId = conversation.id
                        _uiState.update { it.copy(partnerName = conversation.partner.nickname) }
                        loadMessages()
                    }
                    .onFailure { e ->
                        _uiState.update {
                            it.copy(
                                isLoading = false,
                                error = "创建会话失败: ${e.message}"
                            )
                        }
                    }
            } else if (conversationIdParam != null) {
                // 直接使用 conversationId
                conversationId = conversationIdParam
                loadMessages()
            } else {
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        error = "缺少会话参数"
                    )
                }
            }
        }
    }

    private fun loadCurrentUser() {
        viewModelScope.launch {
            val userId = userPreferences.userId.first()
            _uiState.update { it.copy(currentUserId = userId) }
        }
    }

    private fun loadMessages() {
        val convId = conversationId
        if (convId == null) {
            _uiState.update {
                it.copy(
                    isLoading = false,
                    error = "会话ID无效"
                )
            }
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            // ✅ 会话 ID 获取成功后，设置当前会话 ID 并清除未读计数
            Log.d("ChatViewModel", "💬 [ChatViewModel] 设置当前会话ID: $convId")
            AppNotificationManager.currentConversationId = convId

            Log.d("ChatViewModel", "💬 [ChatViewModel] 清除会话 $convId 的未读计数")
            UnreadMessageManager.clearUnread(convId)

            messageRepository.getMessages(convId)
                .onSuccess { messages ->
                    // 使用统一的添加方法逐个添加消息
                    messages.forEach { message ->
                        addMessageIfNew(message)
                    }
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                        )
                    }
                }
                .onFailure { e ->
                    // 如果会话不存在（可能被删除），尝试重新创建
                    if (e.message?.contains("会话不存在") == true || e.message?.contains("403") == true) {
                        if (partnerIdParam != null) {
                            // 如果有 partnerId，重新创建会话
                            recreateConversation()
                        } else {
                            _uiState.update {
                                it.copy(
                                    isLoading = false,
                                    error = "会话已失效，请从好友列表重新发起聊天"
                                )
                            }
                        }
                    } else {
                        _uiState.update {
                            it.copy(
                                isLoading = false,
                                error = e.message ?: "加载失败"
                            )
                        }
                    }
                }
        }
    }

    private fun recreateConversation() {
        val partnerId = partnerIdParam ?: return

        viewModelScope.launch {
            messageRepository.getOrCreateConversation(partnerId)
                .onSuccess { conversation ->
                    conversationId = conversation.id
                    _uiState.update { it.copy(partnerName = conversation.partner.nickname) }
                    loadMessages()
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = "重新创建会话失败: ${e.message}"
                        )
                    }
                }
        }
    }

    fun sendMessage(content: String) {
        val convId = conversationId
        if (convId == null) {
            _uiState.update { it.copy(error = "会话ID无效") }
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isSending = true) }

            messageRepository.sendMessage(
                conversationId = convId,
                type = MessageType.TEXT,
                content = content,
                metadata = null,
            ).onSuccess { message ->
                // 使用统一的添加方法，确保原子性
                addMessageIfNew(message)
                _uiState.update {
                    it.copy(
                        isSending = false,
                    )
                }
            }.onFailure { e ->
                _uiState.update {
                    it.copy(
                        isSending = false,
                        error = e.message ?: "发送失败"
                    )
                }
            }
        }
    }
}
