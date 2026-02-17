package com.youkong.feature.home.viewmodel

import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.datastore.UserPreferences
import com.youkong.core.domain.model.Message
import com.youkong.core.domain.model.MessageType
import com.youkong.core.domain.model.UserProfile
import com.youkong.core.network.api.HomeApi
import com.youkong.core.network.model.SendRoomMessageRequest
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

data class GroupChatUiState(
    val messages: List<Message> = emptyList(),
    val currentUserId: String? = null,
    val isLoading: Boolean = true,
    val isSending: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class GroupChatViewModel @Inject constructor(
    private val homeApi: HomeApi,
    private val userPreferences: UserPreferences,
) : ViewModel() {

    private val _uiState = MutableStateFlow(GroupChatUiState())
    val uiState: StateFlow<GroupChatUiState> = _uiState.asStateFlow()

    private val messageIds = mutableSetOf<String>()
    private val messageMutex = Mutex()

    init {
        loadCurrentUser()
    }

    private fun loadCurrentUser() {
        viewModelScope.launch {
            val userId = userPreferences.userId.first()
            _uiState.update { it.copy(currentUserId = userId) }
        }
    }

    fun loadMessages(roomId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                val response = homeApi.getCardRoomMessages(roomId)
                val messagesData = response.getOrNull() ?: emptyList()
                // API 返回倒序，反转为正序
                val reversedMessages = messagesData.reversed()
                reversedMessages.forEach { msg ->
                    addMessageIfNew(
                        Message(
                            id = msg.id,
                            sender = UserProfile(
                                id = msg.sender.id,
                                nickname = msg.sender.nickname,
                                avatar = msg.sender.avatar,
                            ),
                            type = MessageType.TEXT,
                            content = msg.content,
                            metadata = null,
                            createdAt = try {
                                Instant.parse(msg.createdAt)
                            } catch (e: Exception) {
                                kotlinx.datetime.Clock.System.now()
                            },
                            isRead = msg.isRead,
                        )
                    )
                }
                _uiState.update { it.copy(isLoading = false) }
            } catch (e: Exception) {
                Log.e("GroupChat", "加载消息失败: ${e.message}")
                _uiState.update { it.copy(isLoading = false, error = e.message) }
            }
        }
    }

    fun sendMessage(roomId: String, content: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isSending = true) }
            try {
                val response = homeApi.sendCardRoomMessage(
                    roomId = roomId,
                    request = SendRoomMessageRequest(content = content),
                )
                val msgData = response.getOrThrow()
                addMessageIfNew(
                    Message(
                        id = msgData.id,
                        sender = UserProfile(
                            id = msgData.sender.id,
                            nickname = msgData.sender.nickname,
                            avatar = msgData.sender.avatar,
                        ),
                        type = MessageType.TEXT,
                        content = msgData.content,
                        metadata = null,
                        createdAt = try {
                            Instant.parse(msgData.createdAt)
                        } catch (e: Exception) {
                            kotlinx.datetime.Clock.System.now()
                        },
                        isRead = msgData.isRead,
                    )
                )
                _uiState.update { it.copy(isSending = false) }
            } catch (e: Exception) {
                Log.e("GroupChat", "发送消息失败: ${e.message}")
                _uiState.update { it.copy(isSending = false, error = e.message) }
            }
        }
    }

    private suspend fun addMessageIfNew(message: Message) {
        messageMutex.withLock {
            if (messageIds.contains(message.id)) return
            messageIds.add(message.id)
            _uiState.update { it.copy(messages = it.messages + message) }
        }
    }
}
