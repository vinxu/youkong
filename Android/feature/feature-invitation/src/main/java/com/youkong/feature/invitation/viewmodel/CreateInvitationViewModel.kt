package com.youkong.feature.invitation.viewmodel

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.model.CircleDetail
import com.youkong.core.domain.model.Invitation
import com.youkong.core.domain.repository.CircleRepository
import com.youkong.core.domain.repository.InvitationRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class CreateInvitationUiState(
    val circle: CircleDetail? = null,
    val expireDays: Int = 7,
    val maxUses: Int = 100,
    val isLoading: Boolean = false,
    val isCreating: Boolean = false,
    val createdInvitation: Invitation? = null,
    val error: String? = null,
)

@HiltViewModel
class CreateInvitationViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val circleRepository: CircleRepository,
    private val invitationRepository: InvitationRepository,
) : ViewModel() {

    private val circleId: String? = savedStateHandle["circleId"]

    private val _uiState = MutableStateFlow(CreateInvitationUiState())
    val uiState: StateFlow<CreateInvitationUiState> = _uiState.asStateFlow()

    init {
        if (circleId != null) {
            loadCircle()
        }
    }

    private fun loadCircle() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            circleRepository.getCircle(circleId!!)
                .onSuccess { circle ->
                    _uiState.update { it.copy(isLoading = false, circle = circle) }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(isLoading = false, error = e.message ?: "加载失败")
                    }
                }
        }
    }

    fun setExpireDays(days: Int) {
        _uiState.update { it.copy(expireDays = days) }
    }

    fun setMaxUses(uses: Int) {
        _uiState.update { it.copy(maxUses = uses) }
    }

    fun createInvitation() {
        viewModelScope.launch {
            _uiState.update { it.copy(isCreating = true, error = null) }

            invitationRepository.createInvitation(
                circleId = circleId,
                maxUses = _uiState.value.maxUses,
                expiresDays = _uiState.value.expireDays,
            )
                .onSuccess { invitation ->
                    _uiState.update {
                        it.copy(isCreating = false, createdInvitation = invitation)
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(isCreating = false, error = e.message ?: "创建失败")
                    }
                }
        }
    }
}
