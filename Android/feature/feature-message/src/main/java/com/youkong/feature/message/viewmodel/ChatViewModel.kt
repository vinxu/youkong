package com.youkong.feature.message.viewmodel

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.datastore.UserPreferences
import com.youkong.core.domain.model.Message
import com.youkong.core.domain.model.MessageType
import com.youkong.core.domain.repository.MessageRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
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
) : ViewModel() {

    private val conversationIdParam: String? = savedStateHandle["conversationId"]
    private val partnerIdParam: String? = savedStateHandle["partnerId"]

    private var conversationId: String? = null

    private val _uiState = MutableStateFlow(ChatUiState())
    val uiState: StateFlow<ChatUiState> = _uiState.asStateFlow()

    init {
        loadCurrentUser()
        initializeConversation()
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

            messageRepository.getMessages(convId)
                .onSuccess { messages ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            messages = messages,
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = e.message ?: "加载失败"
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
                _uiState.update {
                    it.copy(
                        isSending = false,
                        messages = it.messages + message,
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
