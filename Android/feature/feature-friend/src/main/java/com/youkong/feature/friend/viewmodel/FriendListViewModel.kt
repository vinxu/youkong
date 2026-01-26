package com.youkong.feature.friend.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.model.Friend
import com.youkong.core.domain.repository.FriendRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

enum class FriendTab {
    ALL, INVITED_BY_ME, INVITED_ME
}

data class FriendListUiState(
    val allFriends: List<Friend> = emptyList(),
    val invitedByMe: List<Friend> = emptyList(),
    val invitedMe: List<Friend> = emptyList(),
    val currentTab: FriendTab = FriendTab.ALL,
    val isLoading: Boolean = true,
    val isRefreshing: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class FriendListViewModel @Inject constructor(
    private val friendRepository: FriendRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(FriendListUiState())
    val uiState: StateFlow<FriendListUiState> = _uiState.asStateFlow()

    init {
        loadFriends()
    }

    fun setTab(tab: FriendTab) {
        _uiState.update { it.copy(currentTab = tab) }

        when (tab) {
            FriendTab.ALL -> if (_uiState.value.allFriends.isEmpty()) loadAllFriends()
            FriendTab.INVITED_BY_ME -> if (_uiState.value.invitedByMe.isEmpty()) loadInvitedByMe()
            FriendTab.INVITED_ME -> if (_uiState.value.invitedMe.isEmpty()) loadInvitedMe()
        }
    }

    private fun loadFriends() {
        loadAllFriends()
    }

    private fun loadAllFriends() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            friendRepository.getFriends()
                .onSuccess { friends ->
                    _uiState.update {
                        it.copy(isLoading = false, allFriends = friends)
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(isLoading = false, error = e.message ?: "加载失败")
                    }
                }
        }
    }

    private fun loadInvitedByMe() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            friendRepository.getInvitedByMe()
                .onSuccess { friends ->
                    _uiState.update {
                        it.copy(isLoading = false, invitedByMe = friends)
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(isLoading = false, error = e.message ?: "加载失败")
                    }
                }
        }
    }

    private fun loadInvitedMe() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            friendRepository.getInvitedMe()
                .onSuccess { friends ->
                    _uiState.update {
                        it.copy(isLoading = false, invitedMe = friends)
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

            when (_uiState.value.currentTab) {
                FriendTab.ALL -> {
                    friendRepository.getFriends()
                        .onSuccess { friends ->
                            _uiState.update {
                                it.copy(isRefreshing = false, allFriends = friends)
                            }
                        }
                        .onFailure { e ->
                            _uiState.update {
                                it.copy(isRefreshing = false, error = e.message ?: "刷新失败")
                            }
                        }
                }

                FriendTab.INVITED_BY_ME -> {
                    friendRepository.getInvitedByMe()
                        .onSuccess { friends ->
                            _uiState.update {
                                it.copy(isRefreshing = false, invitedByMe = friends)
                            }
                        }
                        .onFailure { e ->
                            _uiState.update {
                                it.copy(isRefreshing = false, error = e.message ?: "刷新失败")
                            }
                        }
                }

                FriendTab.INVITED_ME -> {
                    friendRepository.getInvitedMe()
                        .onSuccess { friends ->
                            _uiState.update {
                                it.copy(isRefreshing = false, invitedMe = friends)
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
    }

    fun deleteFriend(userId: String) {
        viewModelScope.launch {
            friendRepository.deleteFriend(userId)
                .onSuccess {
                    _uiState.update { state ->
                        state.copy(
                            allFriends = state.allFriends.filter { it.user.id != userId },
                            invitedByMe = state.invitedByMe.filter { it.user.id != userId },
                            invitedMe = state.invitedMe.filter { it.user.id != userId },
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(error = e.message ?: "删除失败")
                    }
                }
        }
    }
}
