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
import com.youkong.core.datastore.TokenManager
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.BatteryDataRequest
import com.youkong.core.network.model.ConnectionDataRequest
import com.youkong.core.network.model.DisplayDataRequest
import com.youkong.core.network.model.HolmesFullResult
import com.youkong.core.network.model.LocationDataRequest
import com.youkong.core.network.model.ModeDataRequest
import com.youkong.core.network.model.ScreenDataRequest
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
)

@HiltViewModel
class AgentDataViewModel @Inject constructor(
    private val usageStatsCollector: UsageStatsCollector,
    private val locationCollector: LocationCollector,
    private val deviceStateCollector: DeviceStateCollector,
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
                    cliLines = listOf(CliLine.Command("holmes analyze --stream"))
                )
            }
            delay(100)
            appendLine(CliLine.Output("[${dateFormat.format(Date())}] 初始化 Holmes Agent..."))
            delay(200)

            var screenData: LocalScreenData? = null
            var locationData: LocalLocationData? = null
            var deviceStateData: DeviceStateData? = null

            // Step 1: 本地数据收集
            appendLine(CliLine.Phase("📱", "采集本地设备数据..."))
            delay(150)

            try {
                screenData = usageStatsCollector.collect()
                val screenStatus = if (screenData?.isScreenOn == true) "亮屏" else "息屏"
                val currentApp = screenData?.currentApp ?: "无"
                appendLine(CliLine.Clue("屏幕: $screenStatus | 当前: $currentApp"))
            } catch (e: Exception) {
                appendLine(CliLine.Error("屏幕数据采集失败: ${e.message}"))
            }

            delay(100)
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

            // Step 2: 构建请求
            val request = buildRequest(screenData, locationData, deviceStateData)

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
                event.result?.let {
                    appendLine(CliLine.Divider)
                    appendLine(CliLine.Result(it))
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

    private fun appendLine(line: CliLine) {
        _uiState.update { it.copy(cliLines = it.cliLines + line) }
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

        // 如果屏幕是亮的，last_active_minutes_ago 应该是 0
        val lastActiveMinutesAgo = if (local.isScreenOn) {
            0
        } else {
            local.lastActiveTime?.let {
                ((now - it).inWholeMinutes).toInt()
            } ?: 0
        }

        // session_duration_minutes 应该是当前会话的持续时间（分钟）
        // 如果屏幕关闭，session 为 0
        val sessionDurationMinutes = if (local.isScreenOn) {
            local.sessionStartTime?.let {
                ((now - it).inWholeMinutes).toInt().coerceAtMost(120) // 最多 2 小时，避免异常值
            } ?: 0
        } else {
            0
        }

        // 改进 activity_type 判断
        val currentApp = local.currentApp?.lowercase() ?: ""
        val activityType = when {
            !local.isScreenOn -> ActivityType.IDLE
            // 娱乐类
            currentApp.contains("game") -> ActivityType.ENTERTAINMENT
            currentApp.contains("video") -> ActivityType.ENTERTAINMENT
            currentApp.contains("music") -> ActivityType.ENTERTAINMENT
            currentApp.contains("tiktok") -> ActivityType.ENTERTAINMENT
            currentApp.contains("douyin") -> ActivityType.ENTERTAINMENT
            currentApp.contains("bilibili") -> ActivityType.ENTERTAINMENT
            currentApp.contains("youtube") -> ActivityType.ENTERTAINMENT
            currentApp.contains("netflix") -> ActivityType.ENTERTAINMENT
            currentApp.contains("iqiyi") -> ActivityType.ENTERTAINMENT
            currentApp.contains("youku") -> ActivityType.ENTERTAINMENT
            // 通讯类
            currentApp.contains("wechat") -> ActivityType.COMMUNICATION
            currentApp.contains("weixin") -> ActivityType.COMMUNICATION
            currentApp.contains("qq") -> ActivityType.COMMUNICATION
            currentApp.contains("telegram") -> ActivityType.COMMUNICATION
            currentApp.contains("whatsapp") -> ActivityType.COMMUNICATION
            currentApp.contains("message") -> ActivityType.COMMUNICATION
            currentApp.contains("chat") -> ActivityType.COMMUNICATION
            currentApp.contains("mail") -> ActivityType.COMMUNICATION
            currentApp.contains("phone") -> ActivityType.COMMUNICATION
            currentApp.contains("dialer") -> ActivityType.COMMUNICATION
            // 系统/设置类 - 视为空闲
            currentApp.contains("launcher") -> ActivityType.IDLE
            currentApp.contains("settings") -> ActivityType.IDLE
            currentApp.contains("systemui") -> ActivityType.IDLE
            // 其他默认为工作
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
