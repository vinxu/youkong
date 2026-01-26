package com.youkong.feature.settings.viewmodel

import android.app.Activity
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.permission.PermissionManager
import com.youkong.core.permission.PermissionState
import com.youkong.core.permission.RequiredPermission
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class PermissionSetupUiState(
    val currentStep: Int = 0,
    val permissionState: PermissionState = PermissionState(),
    val isComplete: Boolean = false,
)

@HiltViewModel
class PermissionSetupViewModel @Inject constructor(
    private val permissionManager: PermissionManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(PermissionSetupUiState())
    val uiState: StateFlow<PermissionSetupUiState> = _uiState.asStateFlow()

    // 权限请求顺序
    private val permissionOrder = listOf(
        RequiredPermission.USAGE_STATS,
        RequiredPermission.LOCATION,
        RequiredPermission.CONTACTS,
        RequiredPermission.NOTIFICATION,
        RequiredPermission.BACKGROUND_LOCATION,
    )

    init {
        observePermissionState()
    }

    private fun observePermissionState() {
        viewModelScope.launch {
            permissionManager.permissionState.collect { state ->
                _uiState.update {
                    it.copy(
                        permissionState = state,
                        isComplete = state.allCorePermissionsGranted,
                    )
                }
            }
        }
    }

    fun refreshPermissions() {
        permissionManager.refreshPermissionState()
    }

    fun getCurrentPermission(): RequiredPermission? {
        val state = _uiState.value.permissionState
        val missingPermissions = state.getMissingPermissions()
        return permissionOrder.firstOrNull { it in missingPermissions }
    }

    fun openUsageStatsSettings(activity: Activity) {
        permissionManager.openUsageStatsSettings(activity)
    }

    fun getLocationPermissions(): Array<String> {
        return permissionManager.getLocationPermissions()
    }

    fun getBackgroundLocationPermission(): String? {
        return permissionManager.getBackgroundLocationPermission()
    }

    fun getContactsPermission(): String {
        return permissionManager.getContactsPermission()
    }

    fun getNotificationPermission(): String? {
        return permissionManager.getNotificationPermission()
    }

    fun nextStep() {
        _uiState.update { it.copy(currentStep = it.currentStep + 1) }
    }

    fun skipToComplete() {
        _uiState.update { it.copy(isComplete = true) }
    }
}
