package com.youkong.feature.home.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.model.Availability
import com.youkong.core.domain.repository.AvailabilityRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class HomeUiState(
    val friendsAvailabilities: List<Availability> = emptyList(),
    val myAvailabilities: List<Availability> = emptyList(),
    val isLoading: Boolean = true,
    val isRefreshing: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class HomeViewModel @Inject constructor(
    private val availabilityRepository: AvailabilityRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(HomeUiState())
    val uiState: StateFlow<HomeUiState> = _uiState.asStateFlow()

    init {
        loadData()
        observeAvailabilities()
    }

    private fun observeAvailabilities() {
        viewModelScope.launch {
            availabilityRepository.friendsAvailabilities.collect { availabilities ->
                _uiState.update { it.copy(friendsAvailabilities = availabilities) }
            }
        }

        viewModelScope.launch {
            availabilityRepository.myAvailabilities.collect { availabilities ->
                _uiState.update { it.copy(myAvailabilities = availabilities) }
            }
        }
    }

    fun loadData() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            val friendsResult = availabilityRepository.getFriendsAvailabilities()
            val myResult = availabilityRepository.getMyAvailabilities()

            _uiState.update {
                it.copy(
                    isLoading = false,
                    error = if (friendsResult.isFailure && myResult.isFailure) {
                        "加载失败，请稀释重试"
                    } else null,
                )
            }
        }
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isRefreshing = true, error = null) }

            availabilityRepository.getFriendsAvailabilities()
            availabilityRepository.getMyAvailabilities()

            _uiState.update { it.copy(isRefreshing = false) }
        }
    }

    fun cancelAvailability(availabilityId: String) {
        viewModelScope.launch {
            availabilityRepository.cancelAvailability(availabilityId)
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message ?: "取消失败") }
                }
        }
    }
}
