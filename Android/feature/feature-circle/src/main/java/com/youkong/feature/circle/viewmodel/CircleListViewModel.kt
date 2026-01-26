package com.youkong.feature.circle.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.model.Circle
import com.youkong.core.domain.repository.CircleRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class CircleListUiState(
    val circles: List<Circle> = emptyList(),
    val isLoading: Boolean = true,
    val error: String? = null,
)

@HiltViewModel
class CircleListViewModel @Inject constructor(
    private val circleRepository: CircleRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(CircleListUiState())
    val uiState: StateFlow<CircleListUiState> = _uiState.asStateFlow()

    init {
        loadCircles()
        observeCircles()
    }

    private fun observeCircles() {
        viewModelScope.launch {
            circleRepository.myCircles.collect { circles ->
                _uiState.update { it.copy(circles = circles) }
            }
        }
    }

    fun loadCircles() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            circleRepository.getMyCircles()
                .onSuccess {
                    _uiState.update { it.copy(isLoading = false) }
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

    fun deleteCircle(circleId: String) {
        viewModelScope.launch {
            circleRepository.deleteCircle(circleId)
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message ?: "删除失败") }
                }
        }
    }
}
