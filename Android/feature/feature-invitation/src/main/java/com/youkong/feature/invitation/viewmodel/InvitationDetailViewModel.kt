package com.youkong.feature.invitation.viewmodel

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.model.Invitation
import com.youkong.core.domain.repository.InvitationRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class InvitationDetailUiState(
    val invitation: Invitation? = null,
    val posterUrl: String? = null,
    val qrCodeUrl: String? = null,
    val isLoading: Boolean = true,
    val isDisabling: Boolean = false,
    val disableSuccess: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class InvitationDetailViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val invitationRepository: InvitationRepository,
) : ViewModel() {

    private val invitationId: String = checkNotNull(savedStateHandle["invitationId"])

    private val _uiState = MutableStateFlow(InvitationDetailUiState())
    val uiState: StateFlow<InvitationDetailUiState> = _uiState.asStateFlow()

    init {
        loadInvitationDetail()
    }

    private fun loadInvitationDetail() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            invitationRepository.getInvitationDetail(invitationId)
                .onSuccess { invitation ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            invitation = invitation,
                            posterUrl = invitationRepository.getInvitationPosterUrl(invitationId),
                            qrCodeUrl = invitationRepository.getInvitationQrCodeUrl(invitationId),
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(isLoading = false, error = e.message ?: "加载失败")
                    }
                }
        }
    }

    fun disableInvitation() {
        viewModelScope.launch {
            _uiState.update { it.copy(isDisabling = true, error = null) }

            invitationRepository.disableInvitation(invitationId)
                .onSuccess {
                    _uiState.update {
                        it.copy(isDisabling = false, disableSuccess = true)
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(isDisabling = false, error = e.message ?: "禁用失败")
                    }
                }
        }
    }

    fun getInviteUrl(): String {
        return _uiState.value.invitation?.inviteUrl ?: ""
    }
}
