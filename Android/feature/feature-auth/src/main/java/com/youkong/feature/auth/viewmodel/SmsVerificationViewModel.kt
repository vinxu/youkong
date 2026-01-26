package com.youkong.feature.auth.viewmodel

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.repository.AuthRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class SmsVerificationUiState(
    val phone: String = "",
    val code: String = "",
    val isLoading: Boolean = false,
    val error: String? = null,
    val countdown: Int = 60,
    val canResend: Boolean = false,
    val loginSuccess: Boolean = false,
    val isNewUser: Boolean = false,
)

@HiltViewModel
class SmsVerificationViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val authRepository: AuthRepository,
) : ViewModel() {

    private val phone: String = checkNotNull(savedStateHandle["phone"])

    private val _uiState = MutableStateFlow(SmsVerificationUiState(phone = phone))
    val uiState: StateFlow<SmsVerificationUiState> = _uiState.asStateFlow()

    private var countdownJob: Job? = null

    init {
        startCountdown()
    }

    fun onCodeChange(code: String) {
        if (code.length <= 6 && code.all { it.isDigit() }) {
            _uiState.update {
                it.copy(
                    code = code,
                    error = null,
                )
            }

            // 自动验证
            if (code.length == 6) {
                verifyCode()
            }
        }
    }

    fun verifyCode() {
        val code = _uiState.value.code

        if (code.length != 6) {
            _uiState.update { it.copy(error = "请输入6位验证码") }
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            authRepository.verifySms(phone, code)
                .onSuccess { result ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            loginSuccess = true,
                            isNewUser = result.isNewUser,
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = e.message ?: "验证码错误"
                        )
                    }
                }
        }
    }

    fun resendCode() {
        if (!_uiState.value.canResend) return

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            authRepository.sendSms(phone)
                .onSuccess {
                    _uiState.update { it.copy(isLoading = false) }
                    startCountdown()
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = e.message ?: "发送失败"
                        )
                    }
                }
        }
    }

    private fun startCountdown() {
        countdownJob?.cancel()
        countdownJob = viewModelScope.launch {
            _uiState.update { it.copy(countdown = 60, canResend = false) }

            repeat(60) {
                delay(1000)
                _uiState.update { state ->
                    val newCountdown = state.countdown - 1
                    state.copy(
                        countdown = newCountdown,
                        canResend = newCountdown <= 0,
                    )
                }
            }
        }
    }

    override fun onCleared() {
        super.onCleared()
        countdownJob?.cancel()
    }
}
