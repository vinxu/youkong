package com.youkong.feature.circle.viewmodel

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.model.CircleDetail
import com.youkong.core.domain.repository.CircleRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class CircleDetailUiState(
    val circle: CircleDetail? = null,
    val isLoading: Boolean = true,
    val error: String? = null,
)

@HiltViewModel
class CircleDetailViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val circleRepository: CircleRepository,
) : ViewModel() {

    private val circleId: String = checkNotNull(savedStateHandle["circleId"])

    private val _uiState = MutableStateFlow(CircleDetailUiState())
    val uiState: StateFlow<CircleDetailUiState> = _uiState.asStateFlow()

    init {
        loadCircle()
    }

    private fun loadCircle() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            circleRepository.getCircle(circleId)
                .onSuccess { circle ->
                    _uiState.update { it.copy(isLoading = false, circle = circle) }
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
}
