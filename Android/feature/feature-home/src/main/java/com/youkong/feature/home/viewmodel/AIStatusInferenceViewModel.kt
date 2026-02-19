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
import com.youkong.core.domain.repository.StatusCardOption
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
 * 支持 4选1 推断模式 + 编辑发布
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

    // 已展示过的活动（用于换一批排除）
    private var shownActivities = mutableListOf<String>()

    // 缓存的传感器数据（换一批复用）
    private var cachedSensorData: AgentStatusRequest? = null

    /**
     * 开始 4选1 推断
     */
    fun startInference() {
        if (_uiState.value.isInferring) return

        viewModelScope.launch {
            _uiState.update {
                it.copy(
                    currentPhase = InferPhase.LOADING,
                    isInferring = true,
                    error = null,
                    statusOptions = emptyList(),
                    selectedIndex = null,
                    inferenceSessionId = "",
                    streamingPhase = "正在收集设备数据...",
                    streamingLogs = listOf(StreamingLog("> 正在收集设备数据...", LogType.PHASE)),
                )
            }
            shownActivities.clear()

            try {
                // 收集传感器数据
                val sensorData = collectSensorData()
                cachedSensorData = sensorData

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
                        streamingLogs = state.streamingLogs +
                            StreamingLog("▸ 正在发送数据到 AI...", LogType.PHASE) +
                            StreamingLog("▸ AI 正在生成 4 个选项...", LogType.PHASE),
                    )
                }

                // 调用 4选1 推断接口
                agentRepository.inferOptions(sensorData).fold(
                    onSuccess = { result ->
                        // 记录已展示的活动
                        result.options.forEach { shownActivities.add(it.activity) }

                        _uiState.update { state ->
                            state.copy(
                                isInferring = false,
                                currentPhase = InferPhase.OPTIONS,
                                statusOptions = result.options,
                                inferenceSessionId = result.sessionId,
                                streamingLogs = state.streamingLogs +
                                    StreamingLog("  ✓ 已生成 ${result.options.size} 个选项", LogType.TOOL_RESULT),
                            )
                        }
                        Log.d("AIInference", "4选1 推断完成: ${result.options.size} 个选项")

                        // 客户端补充 GIF（复用已有 Giphy 搜索）
                        fetchGifsForOptions(result.options)
                    },
                    onFailure = { e ->
                        Log.e("AIInference", "4选1 推断失败: ${e.message}")
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
     * 换一批
     */
    fun refreshOptions() {
        if (_uiState.value.isRefreshing) return

        viewModelScope.launch {
            _uiState.update { it.copy(isRefreshing = true, selectedIndex = null) }

            try {
                val sensorData = cachedSensorData ?: collectSensorData()
                val sessionId = _uiState.value.inferenceSessionId

                agentRepository.inferOptions(
                    request = sensorData,
                    excludeActivities = shownActivities.toList(),
                    sessionId = sessionId.ifEmpty { null },
                ).fold(
                    onSuccess = { result ->
                        result.options.forEach { shownActivities.add(it.activity) }
                        _uiState.update {
                            it.copy(
                                isRefreshing = false,
                                statusOptions = result.options,
                                inferenceSessionId = result.sessionId,
                            )
                        }
                        Log.d("AIInference", "换一批完成: ${result.options.size} 个选项")

                        // 客户端补充 GIF
                        fetchGifsForOptions(result.options)
                    },
                    onFailure = { e ->
                        Log.e("AIInference", "换一批失败: ${e.message}")
                        _uiState.update { it.copy(isRefreshing = false) }
                    }
                )
            } catch (e: Exception) {
                _uiState.update { it.copy(isRefreshing = false) }
            }
        }
    }

    /**
     * 选择/取消选择某个选项
     */
    fun selectOption(index: Int) {
        _uiState.update {
            it.copy(selectedIndex = if (it.selectedIndex == index) null else index)
        }
    }

    /**
     * 确认选择 → 进入编辑页
     */
    fun confirmSelection() {
        val state = _uiState.value
        val idx = state.selectedIndex ?: return
        val option = state.statusOptions.find { it.index == idx } ?: return

        // 构建 StatusInferenceResult
        val inference = StatusInferenceResult(
            emoji = option.emoji,
            activity = option.activity,
            place = option.place,
            isAvailable = option.isAvailable,
            confidence = option.confidence,
            gifUrl = option.gifUrl,
            giphyQuery = option.giphyQuery,
        )

        _uiState.update {
            it.copy(
                currentPhase = InferPhase.EDITING,
                inference = inference,
                editingEmoji = option.emoji,
                editingActivity = option.activity,
                editingPlace = option.place ?: "",
                editingIsAvailable = option.isAvailable,
                useGif = !option.gifUrl.isNullOrEmpty(),
            )
        }
    }

    /**
     * 从编辑页返回选项页
     */
    fun backToOptions() {
        _uiState.update { it.copy(currentPhase = InferPhase.OPTIONS) }
    }

    /**
     * 将传感器数据转为可读的线索日志
     */
    private fun buildSensorClues(data: AgentStatusRequest): List<StreamingLog> {
        val clues = mutableListOf<StreamingLog>()
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
        val locationData = try { locationCollector.collect() } catch (e: Exception) { null }
        val deviceState = try { deviceStateCollector.collect() } catch (e: Exception) { null }
        val movementData = try { movementCollector.collect() } catch (e: Exception) { null }
        val calendarData = try { calendarCollector.collect() } catch (e: Exception) { null }
        val screenData = try { screenUsageCollector.collect() } catch (e: Exception) { null }

        return AgentStatusRequest(
            screen = screenData,
            location = locationData?.let {
                LocationDataRequest(placeType = "unknown", atPlaceSinceMinutes = 0, city = it.city)
            },
            extendedLocation = locationData?.let {
                ExtendedLocationDataRequest(
                    placeType = "unknown", placeName = it.placeName,
                    atPlaceSinceMinutes = 0, latitude = it.latitude, longitude = it.longitude,
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
                ModeDataRequest(isLowPowerMode = it.isPowerSaveMode, isFocusModeOn = it.isDoNotDisturbEnabled)
            },
            connection = deviceState?.let {
                ConnectionDataRequest(
                    isHeadphonesConnected = it.isHeadphonesConnected,
                    networkType = it.networkType.name,
                    wifiSSID = it.wifiSSID,
                    bluetoothDeviceType = it.bluetoothDeviceType,
                )
            },
            display = deviceState?.let { DisplayDataRequest(screenBrightness = it.screenBrightness) },
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
                        lux < 10 -> "dark"; lux < 50 -> "dim"; lux < 300 -> "indoor"
                        lux < 1000 -> "bright"; lux < 10000 -> "outdoor"; else -> "sunlight"
                    },
                )
            },
        )
    }

    /**
     * 为 4 个选项串行获取 GIF（通过 gif.playa.cn 代理，和首页一致）
     */
    private fun fetchGifsForOptions(options: List<StatusCardOption>) {
        // 标记所有需要加载 GIF 的选项
        val needGif = options.filter { it.gifUrl.isNullOrEmpty() && (it.giphyQuery.isNotEmpty() || it.activity.isNotEmpty()) }
        _uiState.update { it.copy(gifLoadingIndices = needGif.map { o -> o.index }.toSet()) }

        // 串行请求避免代理并发超时（Vercel 冷启动）
        viewModelScope.launch {
            for (option in needGif) {
                val query = option.giphyQuery.ifEmpty { option.activity }
                Log.d("AIInference", "GIF搜索: index=${option.index} query=$query")
                val gifUrl = fetchGifViaProxy(query, "opt_${option.index}")
                Log.d("AIInference", "GIF结果: index=${option.index} url=${gifUrl?.take(60)}")

                _uiState.update { state ->
                    state.copy(
                        statusOptions = if (gifUrl != null) {
                            state.statusOptions.map {
                                if (it.index == option.index) it.copy(gifUrl = gifUrl) else it
                            }
                        } else state.statusOptions,
                        // 移除已完成加载的 index
                        gifLoadingIndices = state.gifLoadingIndices - option.index,
                    )
                }
            }
        }
    }

    /**
     * 通过 gif.playa.cn 代理获取 GIF cos_url（和首页 GridHomeViewModel 一致）
     * 失败自动重试 1 次
     */
    private suspend fun fetchGifViaProxy(query: String, seed: String): String? = withContext(Dispatchers.IO) {
        repeat(2) { attempt ->
            try {
                val encoded = java.net.URLEncoder.encode(query, "UTF-8")
                val seedHash = md5("$seed$query${java.time.LocalDate.now()}")
                val conn = URL("https://gif.playa.cn/api/giphy?q=$encoded&seed=$seedHash").openConnection() as java.net.HttpURLConnection
                conn.connectTimeout = 10000
                conn.readTimeout = 10000
                val body = conn.inputStream.bufferedReader().readText()
                conn.disconnect()
                // 检查是否是错误响应
                if (body.contains("FUNCTION_INVOCATION_TIMEOUT") || body.contains("error")) {
                    Log.w("Giphy", "代理超时 attempt=$attempt query=$query")
                    if (attempt == 0) {
                        kotlinx.coroutines.delay(1000) // 等 1 秒重试
                        return@repeat
                    }
                    return@withContext null
                }
                val json = JSONObject(body)
                val url = json.optJSONObject("result")?.optString("cos_url")?.takeIf { it.isNotEmpty() }
                if (url != null) return@withContext url
            } catch (e: Exception) {
                Log.w("Giphy", "GIF 代理请求失败 attempt=$attempt: ${e.message}")
                if (attempt == 0) kotlinx.coroutines.delay(1000)
            }
        }
        null
    }

    private fun md5(input: String): String {
        val bytes = java.security.MessageDigest.getInstance("MD5").digest(input.toByteArray())
        return bytes.joinToString("") { "%02x".format(it) }
    }

    /**
     * 客户端搜索 GIF（编辑页用，同样走 gif.playa.cn 代理）
     */
    private fun searchGiphyFromClient(query: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isSearchingGif = true) }
            try {
                val gifUrl = fetchGifViaProxy(query, "edit_${System.currentTimeMillis()}")
                if (gifUrl != null) {
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

    // MARK: - 编辑操作

    fun updateEmoji(emoji: String) { _uiState.update { it.copy(editingEmoji = emoji) } }
    fun updateActivity(activity: String) { _uiState.update { it.copy(editingActivity = activity) } }
    fun updatePlace(place: String) { _uiState.update { it.copy(editingPlace = place) } }
    fun toggleIsAvailable() { _uiState.update { it.copy(editingIsAvailable = !it.editingIsAvailable) } }
    fun startEditing() { _uiState.update { it.copy(isEditing = !it.isEditing) } }
    fun toggleEmojiPicker() { _uiState.update { it.copy(showEmojiPicker = !it.showEmojiPicker) } }

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
            if (query.isNotEmpty()) searchGiphyFromClient(query)
        }
    }

    fun setEmojiMode() { _uiState.update { it.copy(useGif = false) } }

    /**
     * 确认状态（发布到首页）
     */
    fun confirmStatus(onSuccess: () -> Unit) {
        if (_uiState.value.isConfirming) return
        val state = _uiState.value
        val inference = state.inference ?: run {
            _uiState.update { it.copy(error = "没有可发布的状态") }
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isConfirming = true, confirmingMessage = "发布中...") }

            var finalGifUrl = inference.gifUrl
            if (state.useGif && !finalGifUrl.isNullOrEmpty() && finalGifUrl.contains("giphy.com")) {
                _uiState.update { it.copy(confirmingMessage = "正在上传 GIF...") }
                val cosUrl = uploadGifToCOS(finalGifUrl)
                if (cosUrl != null) {
                    finalGifUrl = cosUrl
                    _uiState.update { s -> s.copy(inference = s.inference?.copy(gifUrl = cosUrl)) }
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
                inferenceSessionId = state.inferenceSessionId.ifEmpty { null },
                selectedOptionIdx = state.selectedIndex,
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

    private suspend fun uploadGifToCOS(gifUrl: String): String? {
        return withContext(Dispatchers.IO) {
            try {
                val conn = URL(gifUrl).openConnection() as java.net.HttpURLConnection
                conn.connectTimeout = 10000
                conn.readTimeout = 10000
                val gifData = try { conn.inputStream.readBytes() } finally { conn.disconnect() }
                agentRepository.uploadGifToCOS(gifData).getOrNull()
            } catch (e: Exception) {
                Log.w("UploadGif", "失败: ${e.message}")
                null
            }
        }
    }

    fun hasChanges(): Boolean {
        val state = _uiState.value
        val inference = state.inference ?: return false
        return state.editingEmoji != inference.emoji ||
                state.editingActivity != inference.activity ||
                state.editingIsAvailable != inference.isAvailable ||
                state.editingPlace != (inference.place ?: "") ||
                state.useGif
    }
}

/**
 * 推断阶段
 */
enum class InferPhase {
    LOADING,  // 加载中
    OPTIONS,  // 4选1 选项展示
    EDITING,  // 编辑发布
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
    val currentPhase: InferPhase = InferPhase.LOADING,
    val isInferring: Boolean = false,
    // 4选1 选项
    val statusOptions: List<StatusCardOption> = emptyList(),
    val selectedIndex: Int? = null,
    val inferenceSessionId: String = "",
    val isRefreshing: Boolean = false,
    val gifLoadingIndices: Set<Int> = emptySet(), // 正在加载 GIF 的选项 index
    // 编辑状态
    val inference: StatusInferenceResult? = null,
    val error: String? = null,
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
    // GIF 模式切换
    val useGif: Boolean = false,
    val isSearchingGif: Boolean = false,
    // Legacy（保留兼容）
    val isAskingUser: Boolean = false,
    val askUserQuestion: String = "",
    val askUserOptions: List<String> = emptyList(),
    val v3SessionId: String? = null,
)
