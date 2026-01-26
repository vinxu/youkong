package com.youkong.feature.auth.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.repository.AuthRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class PhoneInputUiState(
    val phone: String = "",
    val isLoading: Boolean = false,
    val error: String? = null,
    val isSmsSent: Boolean = false,
)

@HiltViewModel
class PhoneInputViewModel @Inject constructor(
    private val authRepository: AuthRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(PhoneInputUiState())
    val uiState: StateFlow<PhoneInputUiState> = _uiState.asStateFlow()

    fun onPhoneChange(phone: String) {
        _uiState.update {
            it.copy(
                phone = phone,
                error = null,
            )
        }
    }

    fun sendSms() {
        val phone = _uiState.value.phone

        // 验证手机号
        if (phone.length != 11) {
            _uiState.update { it.copy(error = "请输入11位手机号") }
            return
        }

        if (!phone.startsWith("1")) {
            _uiState.update { it.copy(error = "请输入有效的手机号") }
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            authRepository.sendSms(phone)
                .onSuccess {
                    _uiState.update { it.copy(isLoading = false, isSmsSent = true) }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = e.message ?: "发送验证码失败"
                        )
                    }
                }
        }
    }

    fun resetSmsSent() {
        _uiState.update { it.copy(isSmsSent = false) }
    }
}
