package com.youkong.feature.settings.viewmodel

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
    val permissionState: PermissionState = PermissionState(),
    val isComplete: Boolean = false,
    val skippedPermissions: Set<RequiredPermission> = emptySet(),
)

@HiltViewModel
class PermissionSetupViewModel @Inject constructor(
    private val permissionManager: PermissionManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(PermissionSetupUiState())
    val uiState: StateFlow<PermissionSetupUiState> = _uiState.asStateFlow()

    // 权限请求顺序：位置 → 运动数据 → 日历
    private val permissionOrder = listOf(
        RequiredPermission.LOCATION,
        RequiredPermission.ACTIVITY_RECOGNITION,
        RequiredPermission.CALENDAR,
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
        val state = _uiState.value
        val missingPermissions = state.permissionState.getMissingPermissions()
        return permissionOrder.firstOrNull { it in missingPermissions && it !in state.skippedPermissions }
    }

    fun getLocationPermissions(): Array<String> {
        return permissionManager.getLocationPermissions()
    }

    fun getActivityRecognitionPermission(): String? {
        return permissionManager.getActivityRecognitionPermission()
    }

    fun getCalendarPermission(): String {
        return permissionManager.getCalendarPermission()
    }

    fun skipCurrentPermission() {
        val current = getCurrentPermission() ?: return
        _uiState.update { it.copy(skippedPermissions = it.skippedPermissions + current) }
    }

    fun skipToComplete() {
        _uiState.update { it.copy(isComplete = true) }
    }
}
