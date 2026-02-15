package com.youkong.feature.home.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.agent.collector.CalendarCollector
import com.youkong.core.agent.collector.DeviceStateCollector
import com.youkong.core.agent.collector.LocationCollector
import com.youkong.core.agent.collector.MovementCollector
import com.youkong.core.network.api.AgentApi
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.*
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class StatusAnalysisViewModel @Inject constructor(
    private val agentApi: AgentApi,
    private val locationCollector: LocationCollector,
    private val calendarCollector: CalendarCollector,
    private val movementCollector: MovementCollector,
    private val deviceStateCollector: DeviceStateCollector
) : ViewModel() {

    private val _uiState = MutableStateFlow(StatusAnalysisUiState())
    val uiState: StateFlow<StatusAnalysisUiState> = _uiState.asStateFlow()

    // MARK: - Start Analysis

    fun startAnalysis() {
        viewModelScope.launch {
            _uiState.update { it.copy(isAnalyzing = true, errorMessage = null) }

            // 标题
            appendLine("🔍 有空 Agent - 状态分析", LineType.TITLE)
            appendLine("━━━━━━━━━━━━━━━━━━━━━━━━━━", LineType.TITLE)
            appendLine("", LineType.NORMAL)

            // 收集设备数据
            appendLine("📡 正在收集设备数据...", LineType.PHASE)
            collectDeviceData()

            // 构建请求
            val request = buildStatusRequest()

            // 调用 API
            appendLine("", LineType.NORMAL)
            appendLine("🤖 正在分析状态...", LineType.PHASE)

            try {
                val response = agentApi.reportStatus(request)

                val analysis = response.data?.analysis
                if (analysis != null) {
                    displayAnalysisResult(analysis)
                    _uiState.update {
                        it.copy(
                            isAnalyzing = false,
                            analysisCompleted = true
                        )
                    }
                } else {
                    appendLine("⚠️ 未获取到分析结果", LineType.ERROR)
                    _uiState.update { it.copy(isAnalyzing = false) }
                }
            } catch (e: Exception) {
                appendLine("❌ 分析失败: ${e.message}", LineType.ERROR)
                _uiState.update {
                    it.copy(
                        isAnalyzing = false,
                        errorMessage = e.message
                    )
                }
            }
        }
    }

    // MARK: - Collect Device Data

    private suspend fun collectDeviceData() {
        val deviceState = deviceStateCollector.collect()
        val locationData = locationCollector.collect()
        val movementData = movementCollector.collect()

        // 位置信息
        val placeText = if (locationData != null) {
            "已获取 (${String.format("%.4f", locationData.latitude)}, ${String.format("%.4f", locationData.longitude)})"
        } else {
            "未知"
        }
        appendLine("├─ 位置: $placeText", LineType.CLUE)

        // 电池状态
        val batteryPercent = deviceState.batteryLevel
        val batteryText = if (deviceState.isCharging) "$batteryPercent% (充电中)" else "$batteryPercent%"
        appendLine("├─ 电量: $batteryText", LineType.CLUE)

        // 网络状态
        appendLine("├─ 网络: ${deviceState.networkType}", LineType.CLUE)

        // 日历
        val calendarData = calendarCollector.collect()
        if (calendarData != null) {
            if (calendarData.hasCurrentEvent) {
                appendLine("├─ 日历: 有会议进行中", LineType.CLUE)
            } else if (calendarData.nextEventInMinutes != null && calendarData.nextEventInMinutes!! > 0) {
                appendLine("├─ 日历: ${calendarData.nextEventInMinutes}分钟后有会议", LineType.CLUE)
            }
        }

        // 运动
        if (movementData != null) {
            val movementText = if (movementData.isMoving) movementData.movementType else "静止"
            appendLine("└─ 运动: $movementText", LineType.CLUE)
        } else {
            appendLine("└─ 运动: 无权限", LineType.CLUE)
        }

        // 模拟分析延迟
        delay(500)
    }

    // MARK: - Build Request

    private suspend fun buildStatusRequest(): AgentStatusRequest {
        val deviceState = deviceStateCollector.collect()
        val locationData = locationCollector.collect()
        val calendarData = calendarCollector.collect()
        val movementData = movementCollector.collect()

        return AgentStatusRequest(
            screen = null, // 简化处理，屏幕状态暂不上报
            location = locationData?.let {
                LocationDataRequest(
                    placeType = "unknown", // 简化处理，暂不推断位置类型
                    atPlaceSinceMinutes = 0,
                    city = it.city,
                )
            },
            extendedLocation = locationData?.let {
                ExtendedLocationDataRequest(
                    placeType = "unknown",
                    placeName = it.placeName,
                    atPlaceSinceMinutes = 0,
                    latitude = it.latitude,
                    longitude = it.longitude,
                )
            },
            battery = BatteryDataRequest(
                batteryLevel = deviceState.batteryLevel,
                batteryState = if (deviceState.isCharging) "charging" else "unplugged",
                isCharging = deviceState.isCharging
            ),
            mode = ModeDataRequest(
                isLowPowerMode = deviceState.isPowerSaveMode,
                isFocusModeOn = deviceState.isDoNotDisturbEnabled
            ),
            connection = ConnectionDataRequest(
                isHeadphonesConnected = deviceState.isHeadphonesConnected,
                networkType = deviceState.networkType.name
            ),
            display = DisplayDataRequest(
                screenBrightness = deviceState.screenBrightness
            )
        )
    }

    // MARK: - Display Result

    private fun displayAnalysisResult(analysis: AnalysisResult) {
        appendLine("", LineType.NORMAL)
        appendLine("✨ 分析结果", LineType.PHASE)
        appendLine("━━━━━━━━━━━━━━━━━━━━━━━━━━", LineType.PHASE)
        appendLine("", LineType.NORMAL)

        // 生活状态
        appendLine("${analysis.lifeStatus.emoji} ${analysis.lifeStatus.label}", LineType.RESULT)
        analysis.lifeStatus.description?.let {
            appendLine(it, LineType.THINKING)
        }

        appendLine("", LineType.NORMAL)

        // 有空概率
        val probability = analysis.availability.probability
        val status = analysis.availability.status
        appendLine("有空概率: $probability% ($status)", LineType.CONCLUSION)
        appendLine("理由: ${analysis.availability.reason}", LineType.THINKING)
        appendLine("置信度: ${analysis.availability.confidence}", LineType.THINKING)

        appendLine("", LineType.NORMAL)
        appendLine("✅ 状态已更新", LineType.RESULT)
    }

    // MARK: - Helper

    private fun appendLine(text: String, type: LineType) {
        _uiState.update {
            it.copy(outputLines = it.outputLines + OutputLine(text, type))
        }
    }
}

// MARK: - UI State

data class StatusAnalysisUiState(
    val outputLines: List<OutputLine> = emptyList(),
    val isAnalyzing: Boolean = false,
    val analysisCompleted: Boolean = false,
    val errorMessage: String? = null
)

data class OutputLine(
    val text: String,
    val type: LineType
)

enum class LineType {
    TITLE,      // 标题
    PHASE,      // 阶段
    CLUE,       // 线索
    THINKING,   // 推理
    CONCLUSION, // 结论
    RESULT,     // 结果
    ERROR,      // 错误
    NORMAL      // 普通
}
