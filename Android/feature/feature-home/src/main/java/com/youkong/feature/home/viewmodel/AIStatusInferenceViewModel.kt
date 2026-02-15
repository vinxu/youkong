package com.youkong.feature.home.viewmodel

import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.agent.collector.CalendarCollector
import com.youkong.core.agent.collector.DeviceStateCollector
import com.youkong.core.agent.collector.LocationCollector
import com.youkong.core.agent.collector.MovementCollector
import com.youkong.core.agent.collector.ScreenUsageCollector
import com.youkong.core.data.repository.AgentRepositoryImpl
import com.youkong.core.domain.repository.StatusFeedback
import com.youkong.core.domain.repository.StatusInferenceResult
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.BatteryDataRequest
import com.youkong.core.network.model.ConnectionDataRequest
import com.youkong.core.network.model.DisplayDataRequest
import com.youkong.core.network.model.ExtendedLocationDataRequest
import com.youkong.core.network.model.LocationDataRequest
import com.youkong.core.network.model.ModeDataRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.net.URL
import javax.inject.Inject

/**
 * AI 状态推断 ViewModel
 * 使用 V3 同步推断接口（/agent/infer-status-v2）
 */
@HiltViewModel
class AIStatusInferenceViewModel @Inject constructor(
    private val agentRepository: AgentRepositoryImpl,
    private val locationCollector: LocationCollector,
    private val movementCollector: MovementCollector,
    private val calendarCollector: CalendarCollector,
    private val deviceStateCollector: DeviceStateCollector,
    private val screenUsageCollector: ScreenUsageCollector,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AIStatusInferenceUiState())
    val uiState: StateFlow<AIStatusInferenceUiState> = _uiState.asStateFlow()

    // V3 session ID（信号矛盾时用于用户选择）
    private var v3SessionId: String? = null

    /**
     * 开始 AI 推断（V3 同步接口）
     */
    fun startInference() {
        if (_uiState.value.isInferring) return

        viewModelScope.launch {
            _uiState.update {
                it.copy(
                    isInferring = true,
                    error = null,
                    streamingPhase = "正在收集设备数据...",
                    streamingLogs = listOf(StreamingLog("> 正在收集设备数据...", LogType.PHASE)),
                    isAskingUser = false,
                )
            }
            v3SessionId = null

            try {
                // 收集传感器数据
                val sensorData = collectSensorData()

                // 展示收集到的传感器线索
                _uiState.update { state ->
                    state.copy(
                        streamingLogs = state.streamingLogs + buildSensorClues(sensorData),
                    )
                }
                kotlinx.coroutines.delay(300)

                _uiState.update { state ->
                    state.copy(
                        streamingPhase = "正在推断状态...",
                        streamingLogs = state.streamingLogs + StreamingLog("▸ 正在发送数据到 AI...", LogType.PHASE),
                    )
                }

                // 调用 V3 同步推断接口
                val result = agentRepository.inferStatusV3(sensorData)

                result.fold(
                    onSuccess = { v3Result ->
                        _uiState.update { state ->
                            state.copy(
                                streamingLogs = state.streamingLogs + StreamingLog("▸ AI 正在分析...", LogType.PHASE),
                            )
                        }

                        val completedResult = v3Result.result
                        val choiceOptions = v3Result.options
                        if (v3Result.phase == "completed" && completedResult != null) {
                            // 逐条展示推理过程
                            _uiState.update { state ->
                                state.copy(
                                    streamingLogs = state.streamingLogs + StreamingLog("▸ 推理完成", LogType.PHASE),
                                )
                            }
                            kotlinx.coroutines.delay(200)

                            if (!completedResult.reasoning.isNullOrEmpty()) {
                                _uiState.update { state ->
                                    state.copy(
                                        streamingLogs = state.streamingLogs + StreamingLog("  │ ${completedResult.reasoning}", LogType.THINKING),
                                    )
                                }
                                kotlinx.coroutines.delay(200)
                            }

                            val confLabel = when (completedResult.confidence) {
                                "high" -> "高"; "medium" -> "中"; else -> "低"
                            }
                            _uiState.update { state ->
                                state.copy(
                                    streamingLogs = state.streamingLogs +
                                        StreamingLog("  │ 置信度: $confLabel", LogType.THINKING) +
                                        StreamingLog("  ✓ ${completedResult.emoji} ${completedResult.activity}", LogType.TOOL_RESULT),
                                )
                            }

                            // 延迟 1.5s 让用户看到推理过程，再切到结果页
                            kotlinx.coroutines.delay(1500)
                            applyInferenceResult(completedResult)
                            Log.d("AIInference", "V3 推断完成: ${completedResult.emoji} ${completedResult.activity}")
                        } else if (v3Result.phase == "awaiting_choice" && !choiceOptions.isNullOrEmpty()) {
                            // 信号矛盾，需要用户选择
                            v3SessionId = v3Result.sessionId
                            val optionLogs = choiceOptions.map { opt ->
                                val reason = if (!opt.reason.isNullOrEmpty()) " - ${opt.reason}" else ""
                                StreamingLog("    ${opt.emoji} ${opt.activity}$reason", LogType.ASK_USER)
                            }
                            _uiState.update { state ->
                                state.copy(
                                    isInferring = false,
                                    isAskingUser = true,
                                    askUserQuestion = v3Result.question ?: "AI 不太确定，请选择：",
                                    askUserOptions = choiceOptions.map { "${it.emoji} ${it.activity}" },
                                    v3SessionId = v3Result.sessionId,
                                    streamingLogs = state.streamingLogs +
                                        StreamingLog("? 信号矛盾，需要你来选择", LogType.ASK_USER) +
                                        optionLogs,
                                )
                            }
                            Log.d("AIInference", "V3 等待用户选择: ${choiceOptions.size} 个选项")
                        } else {
                            _uiState.update { it.copy(isInferring = false, error = "推断返回了未知格式") }
                        }
                    },
                    onFailure = { e ->
                        Log.e("AIInference", "V3 推断失败: ${e.message}")
                        _uiState.update { it.copy(isInferring = false, error = e.message) }
                    }
                )
            } catch (e: Exception) {
                Log.e("AIInference", "Inference failed: ${e.message}")
                _uiState.update { it.copy(isInferring = false, error = e.message) }
            }
        }
    }

    /**
     * 将传感器数据转为可读的线索日志
     */
    private fun buildSensorClues(data: AgentStatusRequest): List<StreamingLog> {
        val clues = mutableListOf<StreamingLog>()
        // 优先显示地名，其次城市，最后 placeType
        val placeName = data.extendedLocation?.placeName
        val city = data.location?.city
        if (!placeName.isNullOrEmpty()) {
            clues.add(StreamingLog("  > 位置: $placeName", LogType.TOOL))
        } else if (!city.isNullOrEmpty()) {
            clues.add(StreamingLog("  > 位置: $city", LogType.TOOL))
        } else if (data.location != null) {
            clues.add(StreamingLog("  > 位置: 已定位", LogType.TOOL))
        }
        data.movement?.let {
            val movLabel = if (it.isMoving) "运动中" else "静止"
            clues.add(StreamingLog("  > 运动: $movLabel, 今日 ${it.stepsToday} 步", LogType.TOOL))
        }
        data.screen?.let {
            val typeLabel = when (it.activityType) {
                "entertainment" -> "娱乐"; "productivity" -> "工作"
                "communication" -> "通讯"; else -> "空闲"
            }
            if (it.isActive) {
                clues.add(StreamingLog("  > 屏幕: 活跃中 ($typeLabel, ${it.sessionDurationMinutes}分钟)", LogType.TOOL))
            } else {
                clues.add(StreamingLog("  > 屏幕: ${it.lastActiveMinutesAgo}分钟前活跃", LogType.TOOL))
            }
        }
        data.calendar?.let {
            if (it.hasCurrentEvent && !it.currentEventTitle.isNullOrEmpty()) {
                clues.add(StreamingLog("  > 日历: 正在进行「${it.currentEventTitle}」", LogType.TOOL))
            }
        }
        return clues
    }

    /**
     * 收集实际的传感器数据
     */
    private suspend fun collectSensorData(): AgentStatusRequest {
        // 收集位置
        val locationData = try {
            locationCollector.collect()
        } catch (e: Exception) {
            Log.w("AIInference", "Location collect failed: ${e.message}")
            null
        }

        // 收集设备状态
        val deviceState = try {
            deviceStateCollector.collect()
        } catch (e: Exception) {
            Log.w("AIInference", "DeviceState collect failed: ${e.message}")
            null
        }

        // 收集运动
        val movementData = try {
            movementCollector.collect()
        } catch (e: Exception) {
            Log.w("AIInference", "Movement collect failed: ${e.message}")
            null
        }

        // 收集日历
        val calendarData = try {
            calendarCollector.collect()
        } catch (e: Exception) {
            Log.w("AIInference", "Calendar collect failed: ${e.message}")
            null
        }

        // 收集屏幕使用
        val screenData = try {
            screenUsageCollector.collect()
        } catch (e: Exception) {
            Log.w("AIInference", "Screen usage collect failed: ${e.message}")
            null
        }

        return AgentStatusRequest(
            screen = screenData,
            location = locationData?.let {
                LocationDataRequest(
                    placeType = "unknown",
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
            battery = deviceState?.let {
                BatteryDataRequest(
                    batteryLevel = it.batteryLevel,
                    batteryState = if (it.isCharging) "charging" else "unplugged",
                    isCharging = it.isCharging,
                )
            },
            mode = deviceState?.let {
                ModeDataRequest(
                    isLowPowerMode = it.isPowerSaveMode,
                    isFocusModeOn = it.isDoNotDisturbEnabled,
                )
            },
            connection = deviceState?.let {
                ConnectionDataRequest(
                    isHeadphonesConnected = it.isHeadphonesConnected,
                    networkType = it.networkType.name,
                    wifiSSID = it.wifiSSID,
                    bluetoothDeviceType = it.bluetoothDeviceType,
                )
            },
            display = deviceState?.let {
                DisplayDataRequest(
                    screenBrightness = it.screenBrightness,
                )
            },
            calendar = calendarData?.let {
                com.youkong.core.network.model.CalendarDataRequest(
                    hasCurrentEvent = it.hasCurrentEvent,
                    currentEventTitle = it.currentEventTitle,
                    eventEndMinutes = it.eventEndMinutes,
                    nextEventInMinutes = it.nextEventInMinutes,
                    todayRemainingCount = it.todayRemainingCount,
                )
            },
            movement = movementData?.let {
                com.youkong.core.network.model.MovementDataRequest(
                    isMoving = it.isMoving,
                    movementType = it.movementType.toApiString(),
                    stepsToday = it.stepsToday,
                    stepsLastHour = it.stepsLastHour,
                    stationaryMinutes = it.stationaryMinutes,
                )
            },
            ambientLight = deviceState?.ambientLightLux?.let { lux ->
                com.youkong.core.network.model.AmbientLightDataRequest(
                    lux = lux,
                    environment = when {
                        lux < 10 -> "dark"
                        lux < 50 -> "dim"
                        lux < 300 -> "indoor"
                        lux < 1000 -> "bright"
                        lux < 10000 -> "outdoor"
                        else -> "sunlight"
                    },
                )
            },
        )
    }

    /**
     * 用户选择确认（V3 awaiting_choice 响应）
     */
    fun respondToAskUser(answer: String) {
        val sessionId = v3SessionId ?: return
        val selectedIndex = _uiState.value.askUserOptions.indexOf(answer).coerceAtLeast(0)

        viewModelScope.launch {
            _uiState.update {
                it.copy(
                    isAskingUser = false,
                    isInferring = true,
                    streamingPhase = "正在确认选择...",
                    streamingLogs = it.streamingLogs + StreamingLog("▸ 已选择: $answer", LogType.USER_ANSWER),
                )
            }

            agentRepository.inferStatusV3Respond(sessionId, selectedIndex).fold(
                onSuccess = { v3Result ->
                    val respondResult = v3Result.result
                    if (respondResult != null) {
                        applyInferenceResult(respondResult)
                        _uiState.update { state ->
                            state.copy(
                                streamingLogs = state.streamingLogs + StreamingLog(
                                    "✓ ${respondResult.emoji} ${respondResult.activity}",
                                    LogType.TOOL_RESULT
                                ),
                            )
                        }
                    } else {
                        _uiState.update { it.copy(isInferring = false, error = "确认失败") }
                    }
                },
                onFailure = { e ->
                    Log.e("AIInference", "V3 respond 失败: ${e.message}")
                    _uiState.update { it.copy(isInferring = false, error = e.message) }
                }
            )
        }
    }

    /**
     * 客户端直接搜索 Giphy API
     */
    private fun searchGiphyFromClient(query: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isSearchingGif = true) }
            Log.d("Giphy", "搜索 GIF: query='$query'")
            try {
                val gifUrl = withContext(Dispatchers.IO) {
                    val apiKey = "Ned8bTKUTSlYen6oZXmacc1sLmlLn9U8"
                    val encodedQuery = java.net.URLEncoder.encode(query, "UTF-8")
                    val urlStr = "https://api.giphy.com/v1/gifs/search?api_key=$apiKey&q=$encodedQuery&limit=1&rating=g"
                    val conn = URL(urlStr).openConnection() as java.net.HttpURLConnection
                    conn.connectTimeout = 8000
                    conn.readTimeout = 8000
                    conn.requestMethod = "GET"
                    try {
                        val body = conn.inputStream.bufferedReader().readText()
                        val json = JSONObject(body)
                        val data = json.getJSONArray("data")
                        if (data.length() > 0) {
                            data.getJSONObject(0)
                                .getJSONObject("images")
                                .getJSONObject("fixed_height")
                                .getString("url")
                        } else null
                    } finally {
                        conn.disconnect()
                    }
                }
                if (gifUrl != null) {
                    Log.d("Giphy", "GIF URL: $gifUrl")
                    _uiState.update { state ->
                        val updated = state.inference?.copy(gifUrl = gifUrl)
                        if (updated != null) state.copy(inference = updated) else state
                    }
                }
            } catch (e: Exception) {
                Log.w("Giphy", "搜索失败: ${e.message}")
            }
            _uiState.update { it.copy(isSearchingGif = false) }
        }
    }

    private fun applyInferenceResult(result: StatusInferenceResult) {
        _uiState.update { state ->
            state.copy(
                isInferring = false,
                inference = result,
                editingEmoji = result.emoji,
                editingActivity = result.activity,
                editingPlace = result.place ?: "",
                editingIsAvailable = result.isAvailable,
                streamingPhase = null,
            )
        }
    }

    /**
     * 更新编辑中的 emoji
     */
    fun updateEmoji(emoji: String) {
        _uiState.update { it.copy(editingEmoji = emoji) }
    }

    /**
     * 更新编辑中的活动描述
     */
    fun updateActivity(activity: String) {
        _uiState.update { it.copy(editingActivity = activity) }
    }

    /**
     * 更新编辑中的场所
     */
    fun updatePlace(place: String) {
        _uiState.update { it.copy(editingPlace = place) }
    }

    /**
     * 切换有空状态
     */
    fun toggleIsAvailable() {
        _uiState.update { it.copy(editingIsAvailable = !it.editingIsAvailable) }
    }

    /**
     * 切换文字编辑模式
     */
    fun startEditing() {
        _uiState.update { it.copy(isEditing = !it.isEditing) }
    }

    /**
     * 显示/隐藏 emoji 选择器
     */
    fun toggleEmojiPicker() {
        _uiState.update { it.copy(showEmojiPicker = !it.showEmojiPicker) }
    }

    /**
     * 切换到 GIF 模式
     */
    fun toggleUseGif() {
        val currentState = _uiState.value
        if (currentState.useGif) {
            _uiState.update { it.copy(useGif = false) }
            return
        }

        _uiState.update { it.copy(useGif = true) }

        if (currentState.inference?.gifUrl.isNullOrEmpty()) {
            val query = currentState.inference?.giphyQuery
                ?: currentState.inference?.activity
                ?: currentState.editingActivity
            if (query.isNotEmpty()) {
                searchGiphyFromClient(query)
            }
        }
    }

    /**
     * 切换回 emoji 模式
     */
    fun setEmojiMode() {
        _uiState.update { it.copy(useGif = false) }
    }

    /**
     * 确认状态（发布到首页）
     */
    fun confirmStatus(onSuccess: () -> Unit) {
        if (_uiState.value.isConfirming) return

        val state = _uiState.value
        val inference = state.inference

        if (inference == null) {
            _uiState.update { it.copy(error = "没有可发布的状态") }
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isConfirming = true, confirmingMessage = "发布中...") }

            // 如果使用 GIF 模式且 URL 是 Giphy 的，先上传到 COS
            val currentGifUrl = inference.gifUrl
            var finalGifUrl = currentGifUrl
            if (state.useGif && !currentGifUrl.isNullOrEmpty() && currentGifUrl.contains("giphy.com")) {
                _uiState.update { it.copy(confirmingMessage = "正在上传 GIF...") }
                val cosUrl = uploadGifToCOS(currentGifUrl)
                if (cosUrl != null) {
                    finalGifUrl = cosUrl
                    _uiState.update { s ->
                        s.copy(inference = s.inference?.copy(gifUrl = cosUrl))
                    }
                }
            }

            val feedback = StatusFeedback(
                originalEmoji = inference.emoji,
                originalActivity = inference.activity,
                correctedEmoji = state.editingEmoji,
                correctedActivity = state.editingActivity,
                correctedPlace = state.editingPlace.ifEmpty { null },
                correctedIsAvailable = state.editingIsAvailable,
                gifUrl = finalGifUrl,
                giphyQuery = inference.giphyQuery,
                useGif = state.useGif,
            )

            agentRepository.submitStatusFeedback(feedback).fold(
                onSuccess = {
                    _uiState.update { it.copy(isConfirming = false, isConfirmed = true) }
                    onSuccess()
                },
                onFailure = { e ->
                    _uiState.update { it.copy(isConfirming = false, error = e.message) }
                }
            )
        }
    }

    /**
     * 下载 Giphy GIF 并上传到后端 COS，返回 COS URL
     */
    private suspend fun uploadGifToCOS(gifUrl: String): String? {
        return withContext(Dispatchers.IO) {
            try {
                val conn = URL(gifUrl).openConnection() as java.net.HttpURLConnection
                conn.connectTimeout = 10000
                conn.readTimeout = 10000
                val gifData = try {
                    conn.inputStream.readBytes()
                } finally {
                    conn.disconnect()
                }
                agentRepository.uploadGifToCOS(gifData).getOrNull()
            } catch (e: Exception) {
                Log.w("UploadGif", "失败: ${e.message}")
                null
            }
        }
    }

    /**
     * 是否有修改
     */
    fun hasChanges(): Boolean {
        val state = _uiState.value
        val inference = state.inference ?: return false
        return state.editingEmoji != inference.emoji ||
                state.editingActivity != inference.activity ||
                state.editingIsAvailable != inference.isAvailable ||
                state.editingPlace != (inference.place ?: "") ||
                state.useGif
    }

    /**
     * 重置状态
     */
    fun reset() {
        _uiState.value = AIStatusInferenceUiState()
        v3SessionId = null
    }
}

/**
 * 流式日志条目
 */
data class StreamingLog(
    val text: String,
    val type: LogType,
)

enum class LogType {
    PHASE, TOOL, TOOL_RESULT, THINKING, ASK_USER, USER_ANSWER, ERROR,
}

/**
 * AI 状态推断 UI 状态
 */
data class AIStatusInferenceUiState(
    val isInferring: Boolean = false,
    val inference: StatusInferenceResult? = null,
    val error: String? = null,
    // 编辑状态
    val isEditing: Boolean = false,
    val editingEmoji: String = "",
    val editingActivity: String = "",
    val editingPlace: String = "",
    val editingIsAvailable: Boolean = false,
    val showEmojiPicker: Boolean = false,
    // 确认状态
    val isConfirming: Boolean = false,
    val confirmingMessage: String = "",
    val isConfirmed: Boolean = false,
    // 流式推断状态
    val streamingPhase: String? = null,
    val streamingLogs: List<StreamingLog> = emptyList(),
    // V3 用户选择状态
    val isAskingUser: Boolean = false,
    val askUserQuestion: String = "",
    val askUserOptions: List<String> = emptyList(),
    val v3SessionId: String? = null,
    // GIF 模式切换
    val useGif: Boolean = false,
    val isSearchingGif: Boolean = false,
)
