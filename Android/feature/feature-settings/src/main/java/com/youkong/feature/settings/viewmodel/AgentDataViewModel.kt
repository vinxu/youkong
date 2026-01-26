package com.youkong.feature.settings.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.agent.collector.DeviceStateCollector
import com.youkong.core.agent.collector.LocationCollector
import com.youkong.core.agent.collector.UsageStatsCollector
import com.youkong.core.agent.model.DeviceStateData
import com.youkong.core.agent.model.LocalLocationData
import com.youkong.core.agent.model.LocalScreenData
import com.youkong.core.permission.PermissionManager
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import javax.inject.Inject

data class AgentDataUiState(
    val hasUsageStatsPermission: Boolean = false,
    val hasLocationPermission: Boolean = false,
    val hasContactsPermission: Boolean = false,
    val screenData: LocalScreenData? = null,
    val locationData: LocalLocationData? = null,
    val deviceStateData: DeviceStateData? = null,
    val lastReportTime: String? = null,
    val lastReportResult: String? = null,
    val debugInfo: Map<String, String> = emptyMap(),
    val isLoading: Boolean = false,
)

@HiltViewModel
class AgentDataViewModel @Inject constructor(
    private val permissionManager: PermissionManager,
    private val usageStatsCollector: UsageStatsCollector,
    private val locationCollector: LocationCollector,
    private val deviceStateCollector: DeviceStateCollector,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AgentDataUiState())
    val uiState: StateFlow<AgentDataUiState> = _uiState.asStateFlow()

    private val dateFormat = SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault())

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }

            // 获取权限状态
            val permState = permissionManager.permissionState.value
            _uiState.update {
                it.copy(
                    hasUsageStatsPermission = permState.hasUsageStatsPermission,
                    hasLocationPermission = permState.hasLocationPermission,
                    hasContactsPermission = permState.hasContactsPermission,
                )
            }

            // 收集屏幕数据
            try {
                val screenData = usageStatsCollector.collect()
                _uiState.update { it.copy(screenData = screenData) }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        debugInfo = it.debugInfo + ("屏幕数据错误" to (e.message ?: "未知错误"))
                    )
                }
            }

            // 收集位置数据
            try {
                val locationData = locationCollector.collect()
                _uiState.update { it.copy(locationData = locationData) }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        debugInfo = it.debugInfo + ("位置数据错误" to (e.message ?: "未知错误"))
                    )
                }
            }

            // 收集设备状态数据
            try {
                val deviceStateData = deviceStateCollector.collect()
                _uiState.update { it.copy(deviceStateData = deviceStateData) }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        debugInfo = it.debugInfo + ("设备状态错误" to (e.message ?: "未知错误"))
                    )
                }
            }

            // 添加刷新时间
            _uiState.update {
                it.copy(
                    isLoading = false,
                    debugInfo = it.debugInfo + ("刷新时间" to dateFormat.format(Date()))
                )
            }
        }
    }
}
