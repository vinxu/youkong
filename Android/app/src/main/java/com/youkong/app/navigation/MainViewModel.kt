package com.youkong.app.navigation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.repository.AuthRepository
import com.youkong.core.permission.PermissionManager
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class MainUiState(
    val isLoading: Boolean = true,
    val isLoggedIn: Boolean = false,
    val hasRequiredPermissions: Boolean = false,
)

@HiltViewModel
class MainViewModel @Inject constructor(
    private val authRepository: AuthRepository,
    private val permissionManager: PermissionManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(MainUiState())
    val uiState: StateFlow<MainUiState> = _uiState.asStateFlow()

    init {
        observeLoginState()
        observePermissionState()
    }

    private fun observeLoginState() {
        viewModelScope.launch {
            authRepository.isLoggedIn.collect { isLoggedIn ->
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        isLoggedIn = isLoggedIn,
                    )
                }
            }
        }
    }

    private fun observePermissionState() {
        viewModelScope.launch {
            permissionManager.permissionState.collect { permissionState ->
                _uiState.update {
                    it.copy(
                        hasRequiredPermissions = permissionState.allCorePermissionsGranted,
                    )
                }
            }
        }
    }

    fun refreshPermissions() {
        permissionManager.refreshPermissionState()
    }
}
