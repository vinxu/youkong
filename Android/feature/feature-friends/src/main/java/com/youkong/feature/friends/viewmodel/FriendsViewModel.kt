package com.youkong.feature.friends.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.model.FriendWithProbability
import com.youkong.core.domain.repository.AgentRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class FriendsUiState(
    val isLoading: Boolean = false,
    val isRefreshing: Boolean = false,
    val friends: List<FriendWithProbability> = emptyList(),
    val error: String? = null,
)

@HiltViewModel
class FriendsViewModel @Inject constructor(
    private val agentRepository: AgentRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(FriendsUiState())
    val uiState: StateFlow<FriendsUiState> = _uiState.asStateFlow()

    init {
        loadFriends()
    }

    fun loadFriends() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            agentRepository.getFriendsProbability()
                .onSuccess { friends ->
                    // 按概率降序排列，无数据的放最后
                    val sortedFriends = friends.sortedByDescending { it.probability ?: -1 }
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            friends = sortedFriends,
                        )
                    }
                }
                .onFailure { error ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = error.message ?: "加载失败",
                        )
                    }
                }
        }
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isRefreshing = true, error = null) }

            agentRepository.getFriendsProbability()
                .onSuccess { friends ->
                    val sortedFriends = friends.sortedByDescending { it.probability ?: -1 }
                    _uiState.update {
                        it.copy(
                            isRefreshing = false,
                            friends = sortedFriends,
                        )
                    }
                }
                .onFailure { error ->
                    _uiState.update {
                        it.copy(
                            isRefreshing = false,
                            error = error.message ?: "刷新失败",
                        )
                    }
                }
        }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }
}
