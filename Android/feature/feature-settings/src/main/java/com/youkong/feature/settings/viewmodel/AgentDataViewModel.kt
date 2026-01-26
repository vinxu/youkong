package com.youkong.feature.settings.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.agent.collector.DeviceStateCollector
import com.youkong.core.agent.collector.LocationCollector
import com.youkong.core.agent.collector.UsageStatsCollector
import com.youkong.core.agent.model.ActivityType
import com.youkong.core.agent.model.DeviceStateData
import com.youkong.core.agent.model.LocalLocationData
import com.youkong.core.agent.model.LocalScreenData
import com.youkong.core.agent.model.PlaceType
import com.youkong.core.network.api.AgentApi
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.AnalysisResult
import com.youkong.core.network.model.BatteryDataRequest
import com.youkong.core.network.model.ConnectionDataRequest
import com.youkong.core.network.model.DisplayDataRequest
import com.youkong.core.network.model.LocationDataRequest
import com.youkong.core.network.model.ModeDataRequest
import com.youkong.core.network.model.ScreenDataRequest
import com.youkong.core.permission.PermissionManager
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
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
    // LLM 分析结果
    val analysisResult: AnalysisResult? = null,
)

@HiltViewModel
class AgentDataViewModel @Inject constructor(
    private val permissionManager: PermissionManager,
    private val usageStatsCollector: UsageStatsCollector,
    private val locationCollector: LocationCollector,
    private val deviceStateCollector: DeviceStateCollector,
    private val agentApi: AgentApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AgentDataUiState())
    val uiState: StateFlow<AgentDataUiState> = _uiState.asStateFlow()

    private val dateFormat = SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault())

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, debugInfo = emptyMap()) }

            // 获取权限状态
            val permState = permissionManager.permissionState.value
            _uiState.update {
                it.copy(
                    hasUsageStatsPermission = permState.hasUsageStatsPermission,
                    hasLocationPermission = permState.hasLocationPermission,
                    hasContactsPermission = permState.hasContactsPermission,
                )
            }

            var screenData: LocalScreenData? = null
            var locationData: LocalLocationData? = null
            var deviceStateData: DeviceStateData? = null

            // 收集屏幕数据
            try {
                screenData = usageStatsCollector.collect()
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
                locationData = locationCollector.collect()
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
                deviceStateData = deviceStateCollector.collect()
                _uiState.update { it.copy(deviceStateData = deviceStateData) }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        debugInfo = it.debugInfo + ("设备状态错误" to (e.message ?: "未知错误"))
                    )
                }
            }

            // 提交数据到后端并获取 LLM 分析结果
            try {
                val request = buildRequest(screenData, locationData, deviceStateData)
                val response = agentApi.reportStatus(request)
                val responseData = response.data
                if (response.code == 0 && responseData != null) {
                    _uiState.update {
                        it.copy(
                            lastReportTime = dateFormat.format(Date()),
                            lastReportResult = "成功",
                            analysisResult = responseData.analysis,
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(
                            lastReportTime = dateFormat.format(Date()),
                            lastReportResult = "失败: ${response.message}",
                        )
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        lastReportTime = dateFormat.format(Date()),
                        lastReportResult = "失败: ${e.message}",
                        debugInfo = it.debugInfo + ("上报错误" to (e.message ?: "未知错误"))
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

    private fun buildRequest(
        screenData: LocalScreenData?,
        locationData: LocalLocationData?,
        deviceStateData: DeviceStateData?
    ): AgentStatusRequest {
        return AgentStatusRequest(
            screen = screenData?.let { convertScreenData(it) },
            location = locationData?.let { convertLocationData(it) },
            battery = deviceStateData?.let { convertBatteryData(it) },
            mode = deviceStateData?.let { convertModeData(it) },
            connection = deviceStateData?.let { convertConnectionData(it) },
            display = deviceStateData?.let { convertDisplayData(it) },
        )
    }

    private fun convertScreenData(local: LocalScreenData): ScreenDataRequest {
        val now = Clock.System.now()
        val lastActiveMinutesAgo = local.lastActiveTime?.let {
            ((now - it).inWholeMinutes).toInt()
        } ?: 0

        val sessionDurationMinutes = local.sessionStartTime?.let {
            ((now - it).inWholeMinutes).toInt()
        } ?: 0

        val activityType = when {
            !local.isScreenOn -> ActivityType.IDLE
            local.currentApp?.contains("game", ignoreCase = true) == true -> ActivityType.ENTERTAINMENT
            local.currentApp?.contains("video", ignoreCase = true) == true -> ActivityType.ENTERTAINMENT
            local.currentApp?.contains("music", ignoreCase = true) == true -> ActivityType.ENTERTAINMENT
            local.currentApp?.contains("chat", ignoreCase = true) == true -> ActivityType.COMMUNICATION
            local.currentApp?.contains("message", ignoreCase = true) == true -> ActivityType.COMMUNICATION
            local.currentApp?.contains("wechat", ignoreCase = true) == true -> ActivityType.COMMUNICATION
            else -> ActivityType.PRODUCTIVITY
        }

        return ScreenDataRequest(
            isActive = local.isScreenOn,
            activityType = activityType.name.lowercase(),
            sessionDurationMinutes = sessionDurationMinutes,
            lastActiveMinutesAgo = lastActiveMinutesAgo,
        )
    }

    private fun convertLocationData(local: LocalLocationData): LocationDataRequest {
        return LocationDataRequest(
            placeType = PlaceType.UNKNOWN.name.lowercase(),
            atPlaceSinceMinutes = 0,
        )
    }

    private fun convertBatteryData(data: DeviceStateData): BatteryDataRequest {
        val batteryState = when {
            data.isCharging && data.batteryLevel >= 100 -> "full"
            data.isCharging -> "charging"
            else -> "unplugged"
        }
        return BatteryDataRequest(
            batteryLevel = data.batteryLevel,
            batteryState = batteryState,
            isCharging = data.isCharging,
        )
    }

    private fun convertModeData(data: DeviceStateData): ModeDataRequest {
        return ModeDataRequest(
            isLowPowerMode = data.isPowerSaveMode,
            isFocusModeOn = data.isDoNotDisturbEnabled,
        )
    }

    private fun convertConnectionData(data: DeviceStateData): ConnectionDataRequest {
        return ConnectionDataRequest(
            isHeadphonesConnected = data.isHeadphonesConnected,
            networkType = data.networkType.name.lowercase(),
        )
    }

    private fun convertDisplayData(data: DeviceStateData): DisplayDataRequest {
        return DisplayDataRequest(
            screenBrightness = data.screenBrightness,
        )
    }
}
