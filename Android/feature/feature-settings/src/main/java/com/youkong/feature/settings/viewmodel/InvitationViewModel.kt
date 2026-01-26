package com.youkong.feature.settings.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.model.Circle
import com.youkong.core.domain.model.FriendRequest
import com.youkong.core.domain.model.Invitation
import com.youkong.core.domain.model.SendFriendRequestResult
import com.youkong.core.domain.model.SendRequestStatus
import com.youkong.core.domain.repository.CircleRepository
import com.youkong.core.domain.repository.FriendRepository
import com.youkong.core.domain.repository.InvitationRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class InvitationUiState(
    val isLoading: Boolean = false,
    val invitations: List<Invitation> = emptyList(),
    val circles: List<Circle> = emptyList(),
    val error: String? = null,
    val createSuccess: Invitation? = null,
    val isCreating: Boolean = false,
    // 好友请求
    val isSendingRequest: Boolean = false,
    val sendRequestResult: SendFriendRequestResult? = null,
    val receivedRequests: List<FriendRequest> = emptyList(),
    val sentRequests: List<FriendRequest> = emptyList(),
    val pendingRequestCount: Int = 0,
    val isLoadingRequests: Boolean = false,
)

@HiltViewModel
class InvitationViewModel @Inject constructor(
    private val invitationRepository: InvitationRepository,
    private val circleRepository: CircleRepository,
    private val friendRepository: FriendRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(InvitationUiState())
    val uiState: StateFlow<InvitationUiState> = _uiState.asStateFlow()

    init {
        loadData()
    }

    private fun loadData() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            // 加载邀请列表
            invitationRepository.getMyInvitations()
                .onSuccess { invitations ->
                    _uiState.update { it.copy(invitations = invitations) }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message ?: "加载失败") }
                }

            // 加载圈子列表（用于创建邀请时选择）
            circleRepository.getMyCircles()
                .onSuccess { circles ->
                    _uiState.update { it.copy(circles = circles) }
                }

            // 加载待处理请求数量
            friendRepository.getPendingRequestCount()
                .onSuccess { count ->
                    _uiState.update { it.copy(pendingRequestCount = count) }
                }

            _uiState.update { it.copy(isLoading = false) }
        }
    }

    fun refresh() {
        loadData()
    }

    fun createInvitation(circleId: String? = null) {
        viewModelScope.launch {
            _uiState.update { it.copy(isCreating = true, error = null, createSuccess = null) }

            invitationRepository.createInvitation(
                circleId = circleId,
                maxUses = 100,
                expiresDays = 7,
            )
                .onSuccess { invitation ->
                    _uiState.update {
                        it.copy(
                            isCreating = false,
                            createSuccess = invitation,
                            invitations = listOf(invitation) + it.invitations,
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(
                            isCreating = false,
                            error = e.message ?: "创建失败",
                        )
                    }
                }
        }
    }

    fun disableInvitation(id: String) {
        viewModelScope.launch {
            invitationRepository.disableInvitation(id)
                .onSuccess {
                    _uiState.update { state ->
                        state.copy(
                            invitations = state.invitations.filter { it.id != id }
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message ?: "禁用失败") }
                }
        }
    }

    fun clearCreateSuccess() {
        _uiState.update { it.copy(createSuccess = null) }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }

    fun getInvitationQrCodeUrl(id: String): String {
        return invitationRepository.getInvitationQrCodeUrl(id)
    }

    // 好友请求相关方法

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

    fun loadFriendRequests() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoadingRequests = true) }

            // 加载收到的请求
            friendRepository.getReceivedRequests()
                .onSuccess { requests ->
                    _uiState.update { it.copy(receivedRequests = requests) }
                }

            // 加载发出的请求
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
                    // 从列表中移除已处理的请求
                    _uiState.update { state ->
                        state.copy(
                            receivedRequests = state.receivedRequests.filter { it.id != requestId },
                            pendingRequestCount = maxOf(0, state.pendingRequestCount - 1),
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message ?: "操作失败") }
                }
        }
    }

    fun getStatusMessage(status: SendRequestStatus): String {
        return when (status) {
            SendRequestStatus.PENDING -> "好友请求已发送，等待对方确认"
            SendRequestStatus.ALREADY_FRIENDS -> "你们已经是好友了"
            SendRequestStatus.ALREADY_REQUESTED -> "你已经发送过好友请求了"
            SendRequestStatus.ACCEPTED -> "对方也向你发送了请求，你们已成为好友"
        }
    }
}
