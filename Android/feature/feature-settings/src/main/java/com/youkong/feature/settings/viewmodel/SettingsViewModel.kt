package com.youkong.feature.settings.viewmodel

import android.app.Activity
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.datastore.TokenManager
import com.youkong.core.datastore.UserPreferences
import com.youkong.core.network.api.UserApi
import com.youkong.core.network.model.UpdateUserRequest
import com.youkong.core.network.model.UserSettingsRequest
import com.youkong.core.permission.PermissionManager
import com.youkong.core.permission.PermissionState
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class SettingsUiState(
    val permissionState: PermissionState = PermissionState(),
    val isDataCollectionEnabled: Boolean = true,
    val showCity: Boolean = true,
    val nickname: String = "",
    val avatar: String? = null,
    val isUpdatingNickname: Boolean = false,
    val nicknameUpdateError: String? = null,
    val nicknameUpdateSuccess: Boolean = false,
)

@HiltViewModel
class SettingsViewModel @Inject constructor(
    private val permissionManager: PermissionManager,
    private val userPreferences: UserPreferences,
    private val tokenManager: TokenManager,
    private val userApi: UserApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(SettingsUiState())
    val uiState: StateFlow<SettingsUiState> = _uiState.asStateFlow()

    init {
        observePermissionState()
        loadUserInfo()
        loadSettings()
    }

    private fun observePermissionState() {
        viewModelScope.launch {
            permissionManager.permissionState.collect { state ->
                _uiState.update { it.copy(permissionState = state) }
            }
        }
    }

    private fun loadUserInfo() {
        viewModelScope.launch {
            userPreferences.userNickname.collect { nickname ->
                _uiState.update { it.copy(nickname = nickname ?: "") }
            }
        }
    }

    fun refreshPermissions() {
        permissionManager.refreshPermissionState()
    }

    fun openAppSettings(activity: Activity) {
        permissionManager.openAppSettings(activity)
    }

    fun openBatteryOptimizationSettings(activity: Activity) {
        permissionManager.openBatteryOptimizationSettings(activity)
    }

    private fun loadSettings() {
        viewModelScope.launch {
            try {
                val response = userApi.getUserSettings()
                val data = response.data
                if (response.isSuccess && data != null) {
                    _uiState.update { it.copy(showCity = data.showCity) }
                }
            } catch (_: Exception) {
                // 加载失败使用默认值
            }
        }
    }

    fun setShowCity(enabled: Boolean) {
        _uiState.update { it.copy(showCity = enabled) }
        viewModelScope.launch {
            try {
                userApi.updateUserSettings(UserSettingsRequest(showCity = enabled))
            } catch (_: Exception) {
                // 回滚
                _uiState.update { it.copy(showCity = !enabled) }
            }
        }
    }

    fun setDataCollectionEnabled(enabled: Boolean) {
        _uiState.update { it.copy(isDataCollectionEnabled = enabled) }
        // ✅ 数据收集改为手动触发（在 AgentDataScreen 点击分析按钮时）
        // 不再自动调度 Worker
    }

    fun updateNickname(newNickname: String) {
        val trimmed = newNickname.trim()
        if (trimmed.length < 2 || trimmed.length > 20) {
            _uiState.update { it.copy(nicknameUpdateError = "昵称需要 2-20 个字符") }
            return
        }
        _uiState.update { it.copy(isUpdatingNickname = true, nicknameUpdateError = null) }
        viewModelScope.launch {
            try {
                val response = userApi.updateCurrentUser(UpdateUserRequest(nickname = trimmed))
                if (response.isSuccess) {
                    userPreferences.updateNickname(trimmed)
                    _uiState.update { it.copy(nickname = trimmed, isUpdatingNickname = false, nicknameUpdateSuccess = true) }
                } else {
                    _uiState.update { it.copy(isUpdatingNickname = false, nicknameUpdateError = "修改失败") }
                }
            } catch (_: Exception) {
                _uiState.update { it.copy(isUpdatingNickname = false, nicknameUpdateError = "网络错误") }
            }
        }
    }

    fun clearNicknameError() {
        _uiState.update { it.copy(nicknameUpdateError = null) }
    }

    fun consumeNicknameSuccess() {
        _uiState.update { it.copy(nicknameUpdateSuccess = false) }
    }

    fun logout() {
        viewModelScope.launch {
            tokenManager.clearTokens()
            userPreferences.clear()
            // ✅ 不再需要取消 Worker（已移除自动调度）
        }
    }
}
