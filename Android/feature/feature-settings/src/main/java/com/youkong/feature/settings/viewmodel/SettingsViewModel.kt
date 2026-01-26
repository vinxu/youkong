package com.youkong.feature.settings.viewmodel

import android.app.Activity
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.work.WorkManager
import com.youkong.core.agent.worker.DataCollectWorker
import com.youkong.core.agent.worker.StatusSyncWorker
import com.youkong.core.datastore.TokenManager
import com.youkong.core.datastore.UserPreferences
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
    val nickname: String = "",
    val avatar: String? = null,
)

@HiltViewModel
class SettingsViewModel @Inject constructor(
    private val permissionManager: PermissionManager,
    private val userPreferences: UserPreferences,
    private val tokenManager: TokenManager,
    private val workManager: WorkManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(SettingsUiState())
    val uiState: StateFlow<SettingsUiState> = _uiState.asStateFlow()

    init {
        observePermissionState()
        loadUserInfo()
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

    fun openUsageStatsSettings(activity: Activity) {
        permissionManager.openUsageStatsSettings(activity)
    }

    fun openAppSettings(activity: Activity) {
        permissionManager.openAppSettings(activity)
    }

    fun openBatteryOptimizationSettings(activity: Activity) {
        permissionManager.openBatteryOptimizationSettings(activity)
    }

    fun setDataCollectionEnabled(enabled: Boolean) {
        _uiState.update { it.copy(isDataCollectionEnabled = enabled) }
        if (enabled) {
            DataCollectWorker.schedule(workManager)
            StatusSyncWorker.schedule(workManager)
        } else {
            DataCollectWorker.cancel(workManager)
            StatusSyncWorker.cancel(workManager)
        }
    }

    fun logout() {
        viewModelScope.launch {
            tokenManager.clearTokens()
            userPreferences.clear()
            DataCollectWorker.cancel(workManager)
            StatusSyncWorker.cancel(workManager)
        }
    }
}
