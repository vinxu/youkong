package com.youkong.feature.auth.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.datastore.UserPreferences
import com.youkong.core.domain.repository.UserRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class ProfileSetupUiState(
    val nickname: String = "",
    val isLoading: Boolean = false,
    val error: String? = null,
    val isComplete: Boolean = false,
)

@HiltViewModel
class ProfileSetupViewModel @Inject constructor(
    private val userRepository: UserRepository,
    private val userPreferences: UserPreferences,
) : ViewModel() {

    private val _uiState = MutableStateFlow(ProfileSetupUiState())
    val uiState: StateFlow<ProfileSetupUiState> = _uiState.asStateFlow()

    fun onNicknameChange(nickname: String) {
        if (nickname.length <= 20) {
            _uiState.update {
                it.copy(
                    nickname = nickname,
                    error = null,
                )
            }
        }
    }

    fun saveProfile() {
        val nickname = _uiState.value.nickname.trim()

        if (nickname.isEmpty()) {
            _uiState.update { it.copy(error = "请输入昵称") }
            return
        }

        if (nickname.length < 2) {
            _uiState.update { it.copy(error = "昵称至少需要2个字符") }
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            userRepository.updateUser(nickname = nickname, avatar = null)
                .onSuccess {
                    userPreferences.setOnboardingCompleted()
                    _uiState.update { it.copy(isLoading = false, isComplete = true) }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = e.message ?: "保存失败"
                        )
                    }
                }
        }
    }
}
