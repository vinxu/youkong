package com.youkong.feature.friends.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.model.Friend
import com.youkong.core.domain.model.FriendRequest
import com.youkong.core.domain.model.SendFriendRequestResult
import com.youkong.core.domain.model.SendRequestStatus
import com.youkong.core.domain.repository.FriendRepository
import com.youkong.core.domain.repository.UserRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class AddFriendUiState(
    val isLoading: Boolean = false,
    val error: String? = null,
    // 好友请求
    val isSendingRequest: Boolean = false,
    val sendRequestResult: SendFriendRequestResult? = null,
    // 好友管理
    val friends: List<Friend> = emptyList(),
    val invitedByMe: List<Friend> = emptyList(),
    val invitedMe: List<Friend> = emptyList(),
    // 好友请求列表
    val receivedRequests: List<FriendRequest> = emptyList(),
    val sentRequests: List<FriendRequest> = emptyList(),
    val pendingRequestCount: Int = 0,
    val isLoadingRequests: Boolean = false,
)

@HiltViewModel
class AddFriendViewModel @Inject constructor(
    private val friendRepository: FriendRepository,
    private val userRepository: UserRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AddFriendUiState())
    val uiState: StateFlow<AddFriendUiState> = _uiState.asStateFlow()

    init {
        loadData()
    }

    private fun loadData() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            // 并行加载数据
            launch {
                friendRepository.getFriends()
                    .onSuccess { friends ->
                        _uiState.update { it.copy(friends = friends) }
                    }
            }

            launch {
                friendRepository.getInvitedByMe()
                    .onSuccess { friends ->
                        _uiState.update { it.copy(invitedByMe = friends) }
                    }
            }

            launch {
                friendRepository.getInvitedMe()
                    .onSuccess { friends ->
                        _uiState.update { it.copy(invitedMe = friends) }
                    }
            }

            launch {
                friendRepository.getPendingRequestCount()
                    .onSuccess { count ->
                        _uiState.update { it.copy(pendingRequestCount = count) }
                    }
            }

            _uiState.update { it.copy(isLoading = false) }
        }
    }

    fun refresh() {
        loadData()
    }

    // 手机号添加好友

    fun sendFriendRequest(phone: String, message: String? = null) {
        viewModelScope.launch {
            _uiState.update { it.copy(isSendingRequest = true, error = null, sendRequestResult = null) }

            friendRepository.sendFriendRequest(phone, message)
                .onSuccess { result ->
                    _uiState.update {
                        it.copy(
                            isSendingRequest = false,
                            sendRequestResult = result,
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(
                            isSendingRequest = false,
                            error = e.message ?: "发送失败",
                        )
                    }
                }
        }
    }

    fun clearSendRequestResult() {
        _uiState.update { it.copy(sendRequestResult = null) }
    }

    fun getStatusMessage(status: SendRequestStatus): String {
        return when (status) {
            SendRequestStatus.PENDING -> "好友请求已发送，等待对方确认"
            SendRequestStatus.ALREADY_FRIENDS -> "你们已经是好友了"
            SendRequestStatus.ALREADY_REQUESTED -> "你已经发送过好友请求了"
            SendRequestStatus.ACCEPTED -> "对方也向你发送了请求，你们已成为好友"
        }
    }

    // 邀请海报

    fun getMyPosterUrl(): String {
        return userRepository.getMyPosterUrl()
    }

    // 好友管理

    fun deleteFriend(userId: String) {
        viewModelScope.launch {
            friendRepository.deleteFriend(userId)
                .onSuccess {
                    _uiState.update { state ->
                        state.copy(
                            friends = state.friends.filter { it.user.id != userId },
                            invitedByMe = state.invitedByMe.filter { it.user.id != userId },
                            invitedMe = state.invitedMe.filter { it.user.id != userId },
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message ?: "删除失败") }
                }
        }
    }

    // 好友请求

    fun loadFriendRequests() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoadingRequests = true) }

            friendRepository.getReceivedRequests()
                .onSuccess { requests ->
                    _uiState.update { it.copy(receivedRequests = requests) }
                }

            friendRepository.getSentRequests()
                .onSuccess { requests ->
                    _uiState.update { it.copy(sentRequests = requests) }
                }

            _uiState.update { it.copy(isLoadingRequests = false) }
        }
    }

    fun handleFriendRequest(requestId: String, accept: Boolean) {
        viewModelScope.launch {
            friendRepository.handleFriendRequest(requestId, accept)
                .onSuccess {
                    _uiState.update { state ->
                        state.copy(
                            receivedRequests = state.receivedRequests.filter { it.id != requestId },
                            pendingRequestCount = maxOf(0, state.pendingRequestCount - 1),
                        )
                    }
                    // 如果接受了请求，刷新好友列表
                    if (accept) {
                        friendRepository.getFriends()
                            .onSuccess { friends ->
                                _uiState.update { it.copy(friends = friends) }
                            }
                    }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message ?: "操作失败") }
                }
        }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }
}
