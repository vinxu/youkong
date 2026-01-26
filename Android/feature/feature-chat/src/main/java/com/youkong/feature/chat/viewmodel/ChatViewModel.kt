package com.youkong.feature.chat.viewmodel

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.datastore.UserPreferences
import com.youkong.core.domain.model.Message
import com.youkong.core.domain.model.MessageType
import com.youkong.core.domain.model.UserProfile
import com.youkong.core.domain.repository.MessageRepository
import com.youkong.core.domain.repository.UserRepository
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
    val partner: UserProfile? = null,
    val currentUserId: String? = null,
    val conversationId: String? = null,
    val isLoading: Boolean = true,
    val isSending: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class ChatViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val messageRepository: MessageRepository,
    private val userRepository: UserRepository,
    private val userPreferences: UserPreferences,
) : ViewModel() {

    private val userId: String = checkNotNull(savedStateHandle["userId"])

    private val _uiState = MutableStateFlow(ChatUiState())
    val uiState: StateFlow<ChatUiState> = _uiState.asStateFlow()

    init {
        loadCurrentUser()
        loadPartnerInfo()
        loadOrCreateConversation()
    }

    private fun loadCurrentUser() {
        viewModelScope.launch {
            val currentUserId = userPreferences.userId.first()
            _uiState.update { it.copy(currentUserId = currentUserId) }
        }
    }

    private fun loadPartnerInfo() {
        viewModelScope.launch {
            userRepository.getUser(userId)
                .onSuccess { user ->
                    _uiState.update {
                        it.copy(partner = user)
                    }
                }
        }
    }

    private fun loadOrCreateConversation() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            // 获取或创建会话
            messageRepository.getOrCreateConversation(userId)
                .onSuccess { conversation ->
                    _uiState.update {
                        it.copy(conversationId = conversation.id)
                    }
                    loadMessages(conversation.id)
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = e.message ?: "加载会话失败"
                        )
                    }
                }
        }
    }

    private fun loadMessages(conversationId: String) {
        viewModelScope.launch {
            messageRepository.getMessages(conversationId)
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
                            error = e.message ?: "加载消息失败"
                        )
                    }
                }
        }
    }

    fun sendMessage(content: String) {
        val conversationId = _uiState.value.conversationId ?: return

        viewModelScope.launch {
            _uiState.update { it.copy(isSending = true) }

            messageRepository.sendMessage(
                conversationId = conversationId,
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

    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }
}
