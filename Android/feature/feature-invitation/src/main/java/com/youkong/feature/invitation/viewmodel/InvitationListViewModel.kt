package com.youkong.feature.invitation.viewmodel

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

data class InvitationListUiState(
    val invitations: List<Invitation> = emptyList(),
    val isLoading: Boolean = true,
    val isRefreshing: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class InvitationListViewModel @Inject constructor(
    private val invitationRepository: InvitationRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(InvitationListUiState())
    val uiState: StateFlow<InvitationListUiState> = _uiState.asStateFlow()

    init {
        loadInvitations()
    }

    fun loadInvitations() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            invitationRepository.getMyInvitations()
                .onSuccess { invitations ->
                    _uiState.update {
                        it.copy(isLoading = false, invitations = invitations)
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(isLoading = false, error = e.message ?: "加载失败")
                    }
                }
        }
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isRefreshing = true, error = null) }

            invitationRepository.getMyInvitations()
                .onSuccess { invitations ->
                    _uiState.update {
                        it.copy(isRefreshing = false, invitations = invitations)
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(isRefreshing = false, error = e.message ?: "刷新失败")
                    }
                }
        }
    }
}
