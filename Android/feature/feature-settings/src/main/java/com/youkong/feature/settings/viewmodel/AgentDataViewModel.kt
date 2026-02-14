package com.youkong.feature.settings.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.agent.collector.DeviceStateCollector
import com.youkong.core.agent.collector.LocationCollector
import com.youkong.core.agent.model.DeviceStateData
import com.youkong.core.agent.model.LocalLocationData
import com.youkong.core.agent.model.PlaceType
import com.youkong.core.datastore.TokenManager
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.BatteryDataRequest
import com.youkong.core.network.model.ConnectionDataRequest
import com.youkong.core.network.model.DisplayDataRequest
import com.youkong.core.network.model.ExtendedLocationDataRequest
import com.youkong.core.network.model.HolmesFullResult
import com.youkong.core.network.model.LocationDataRequest
import com.youkong.core.network.model.ModeDataRequest
import com.youkong.core.network.api.AgentApi
import com.youkong.core.network.sse.AgentSseClient
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import javax.inject.Inject

// CLI 输出行
sealed class CliLine {
    data class Command(val text: String) : CliLine()
    data class Output(val text: String) : CliLine()
    data class Success(val text: String) : CliLine()
    data class Error(val text: String) : CliLine()
    data class Phase(val emoji: String, val text: String) : CliLine()
    data class Clue(val text: String, val isLast: Boolean = false) : CliLine()
    data class Feature(val text: String, val isLast: Boolean = false) : CliLine()
    data class Thinking(val text: String) : CliLine()
    data class Conclusion(val text: String, val isLast: Boolean = false) : CliLine()
    data class Result(val result: HolmesFullResult) : CliLine()
    object Divider : CliLine()
}

data class AgentDataUiState(
    val cliLines: List<CliLine> = emptyList(),
    val isRunning: Boolean = false,
    val lastResult: HolmesFullResult? = null,  // 保存最后的分析结果
    val isUploading: Boolean = false,          // 是否正在上报
    // 保存收集的原始数据，用于上报
    val lastLocationData: LocalLocationData? = null,
    val lastDeviceStateData: DeviceStateData? = null,
)

@HiltViewModel
class AgentDataViewModel @Inject constructor(
    private val locationCollector: LocationCollector,
    private val deviceStateCollector: DeviceStateCollector,
    private val agentApi: AgentApi,
    private val agentSseClient: AgentSseClient,
    private val tokenManager: TokenManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AgentDataUiState())
    val uiState: StateFlow<AgentDataUiState> = _uiState.asStateFlow()

    private val dateFormat = SimpleDateFormat("HH:mm:ss", Locale.getDefault())

    // 用于累积 thinking 内容的缓冲区
    private val thinkingBuffer = StringBuilder()

    init {
        // 🔧 调试模式：手动点击按钮才上报，不自动运行
        // refresh()
    }

    fun refresh() {
        if (_uiState.value.isRunning) return

        viewModelScope.launch {
            // 清空缓冲区
            thinkingBuffer.clear()

            _uiState.update {
                it.copy(
                    isRunning = true,
                    cliLines = listOf(CliLine.Command("holmes analyze --stream")),
                    lastResult = null  // 清空上次的分析结果
                )
            }
            delay(100)
            appendLine(CliLine.Output("[${dateFormat.format(Date())}] 初始化 Holmes Agent..."))
            delay(200)

            var locationData: LocalLocationData? = null
            var deviceStateData: DeviceStateData? = null

            // Step 1: 本地数据收集
            appendLine(CliLine.Phase("📱", "采集本地设备数据..."))
            delay(150)
            try {
                locationData = locationCollector.collect()
                if (locationData != null) {
                    appendLine(CliLine.Clue("位置: (${String.format("%.4f", locationData.latitude)}, ${String.format("%.4f", locationData.longitude)})"))
                } else {
                    appendLine(CliLine.Clue("位置: 无法获取"))
                }
            } catch (e: Exception) {
                appendLine(CliLine.Clue("位置: 无法获取"))
            }

            delay(100)
            try {
                deviceStateData = deviceStateCollector.collect()
                appendLine(CliLine.Clue("电量: ${deviceStateData.batteryLevel}% ${if (deviceStateData.isCharging) "⚡充电中" else ""}", isLast = true))
            } catch (e: Exception) {
                appendLine(CliLine.Clue("设备状态: 无法获取", isLast = true))
            }

            delay(200)

            // Step 2: 构建请求并保存收集的数据
            val request = buildRequest(locationData, deviceStateData)

            // 保存收集的数据，供后续上报使用
            _uiState.update {
                it.copy(
                    lastLocationData = locationData,
                    lastDeviceStateData = deviceStateData
                )
            }

            appendLine(CliLine.Divider)
            appendLine(CliLine.Phase("📡", "连接 Holmes 推理服务..."))
            delay(100)
            appendLine(CliLine.Output("POST /api/v1/agent/status/stream"))

            // Step 3: 获取 token 并发起 SSE 请求
            val token = tokenManager.accessToken.first()
            if (token.isNullOrBlank()) {
                appendLine(CliLine.Error("未登录，无法进行推理"))
                finishRun()
                return@launch
            }

            delay(200)
            appendLine(CliLine.Success("✓ 连接成功，开始流式推理"))
            appendLine(CliLine.Divider)

            // Step 4: 处理 SSE 流
            try {
                agentSseClient.streamStatus(request, token).collect { event ->
                    handleSseEvent(event)
                }
            } catch (e: Exception) {
                appendLine(CliLine.Error("流式请求失败: ${e.message}"))
            }

            finishRun()
        }
    }

    private fun handleSseEvent(event: com.youkong.core.network.model.SseEvent) {
        when (event.type) {
            "phase" -> {
                // 切换到新阶段前，先输出缓冲区中的 thinking 内容
                flushThinkingBuffer()

                val emoji = when (event.phase) {
                    "collecting" -> "📡"
                    "extracting" -> "🧠"
                    "reasoning" -> "💭"
                    "done" -> "✅"
                    else -> "▸"
                }
                event.content?.let {
                    appendLine(CliLine.Phase(emoji, it))
                }
            }
            "clue" -> {
                flushThinkingBuffer()
                event.content?.let {
                    val isLast = it.startsWith("└")
                    appendLine(CliLine.Clue(it.removePrefix("├─ ").removePrefix("└─ "), isLast))
                }
            }
            "feature" -> {
                flushThinkingBuffer()
                event.content?.let {
                    val isLast = it.startsWith("└")
                    appendLine(CliLine.Feature(it.removePrefix("├─ ").removePrefix("└─ "), isLast))
                }
            }
            "thinking" -> {
                event.content?.let { content ->
                    // 累积 thinking 内容到缓冲区
                    thinkingBuffer.append(content)

                    // 检查是否包含换行符，如果有就输出完整的行
                    if (content.contains("\n")) {
                        val text = thinkingBuffer.toString()
                        val lines = text.split("\n")

                        // 输出除最后一行外的所有行（完整的行）
                        for (i in 0 until lines.size - 1) {
                            if (lines[i].isNotEmpty()) {
                                appendLine(CliLine.Thinking(lines[i]))
                            }
                        }

                        // 将最后一行（可能是不完整的）保留在缓冲区
                        thinkingBuffer.clear()
                        thinkingBuffer.append(lines.last())
                    }
                }
            }
            "conclusion" -> {
                flushThinkingBuffer()
                event.content?.let {
                    val isLast = it.startsWith("└")
                    appendLine(CliLine.Conclusion(it.removePrefix("├─ ").removePrefix("└─ "), isLast))
                }
            }
            "done" -> {
                flushThinkingBuffer()
                event.result?.let { result ->
                    appendLine(CliLine.Divider)
                    appendLine(CliLine.Result(result))
                    // 保存分析结果，供后续上报使用
                    _uiState.update { it.copy(lastResult = result) }
                }
            }
            "error" -> {
                flushThinkingBuffer()
                event.content?.let {
                    appendLine(CliLine.Error(it))
                }
            }
        }
    }

    /**
     * 输出缓冲区中剩余的 thinking 内容
     */
    private fun flushThinkingBuffer() {
        if (thinkingBuffer.isNotEmpty()) {
            appendLine(CliLine.Thinking(thinkingBuffer.toString()))
            thinkingBuffer.clear()
        }
    }

    private fun finishRun() {
        flushThinkingBuffer()
        appendLine(CliLine.Divider)
        appendLine(CliLine.Output("[${dateFormat.format(Date())}] Done."))
        _uiState.update { it.copy(isRunning = false) }
    }

    /**
     * 上报状态到服务器
     */
    fun uploadStatus() {
        val currentState = _uiState.value
        val result = currentState.lastResult
        if (result == null) {
            appendLine(CliLine.Error("没有分析结果，无法上报"))
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isUploading = true) }
            appendLine(CliLine.Divider)
            appendLine(CliLine.Output("[${dateFormat.format(Date())}] 上报状态到服务器..."))

            try {
                // ✅ 使用已收集的数据，不重新收集（避免耗时）
                val locationData = currentState.lastLocationData
                val deviceStateData = currentState.lastDeviceStateData

                // 构建请求
                val request = buildRequest(locationData, deviceStateData)

                appendLine(CliLine.Output("POST /api/v1/agent/status"))
                val response = agentApi.reportStatus(request)

                if (response.code == 0) {
                    val msg = response.data?.message ?: "分析已触发"
                    appendLine(CliLine.Success("✓ 上报成功: $msg"))
                    appendLine(CliLine.Output("概率: ${result.result?.probability}%, 状态: ${result.result?.summary}"))
                } else {
                    appendLine(CliLine.Error("上报失败: ${response.message}"))
                }
            } catch (e: Exception) {
                appendLine(CliLine.Error("上报失败: ${e.message}"))
            } finally {
                _uiState.update { it.copy(isUploading = false) }
            }
        }
    }

    private fun appendLine(line: CliLine) {
        _uiState.update { it.copy(cliLines = it.cliLines + line) }
    }

    private fun buildRequest(
        locationData: LocalLocationData?,
        deviceStateData: DeviceStateData?
    ): AgentStatusRequest {
        return AgentStatusRequest(
            screen = null,
            location = locationData?.let { convertLocationData(it) },
            extendedLocation = locationData?.let { convertExtendedLocationData(it) },
            battery = deviceStateData?.let { convertBatteryData(it) },
            mode = deviceStateData?.let { convertModeData(it) },
            connection = deviceStateData?.let { convertConnectionData(it) },
            display = deviceStateData?.let { convertDisplayData(it) },
        )
    }

    private fun convertLocationData(local: LocalLocationData): LocationDataRequest {
        return LocationDataRequest(
            placeType = PlaceType.UNKNOWN.name.lowercase(),
            atPlaceSinceMinutes = 0,
            city = local.city,
        )
    }

    private fun convertExtendedLocationData(local: LocalLocationData): ExtendedLocationDataRequest {
        return ExtendedLocationDataRequest(
            placeType = PlaceType.UNKNOWN.name.lowercase(),
            placeName = local.placeName,
            atPlaceSinceMinutes = 0,
            latitude = local.latitude,
            longitude = local.longitude,
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
