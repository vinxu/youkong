package com.youkong.feature.home.viewmodel

import android.content.ContentValues
import android.content.Context
import android.provider.CalendarContract
import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.agent.voice.VoiceRecordingManager
import com.youkong.core.datastore.TokenManager
import com.youkong.core.network.model.*
import com.youkong.core.network.sse.VoiceScheduleSseClient
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Locale
import java.util.TimeZone
import javax.inject.Inject

@HiltViewModel
class VoiceScheduleViewModel @Inject constructor(
    private val voiceRecordingManager: VoiceRecordingManager,
    private val voiceScheduleSseClient: VoiceScheduleSseClient,
    private val tokenManager: TokenManager,
    @ApplicationContext private val appContext: Context
) : ViewModel() {

    // UI State
    private val _uiState = MutableStateFlow(VoiceScheduleUiState())
    val uiState: StateFlow<VoiceScheduleUiState> = _uiState.asStateFlow()

    // 录音状态（来自 VoiceRecordingManager）
    val isRecording: StateFlow<Boolean> = voiceRecordingManager.isRecording
    val recordingDuration: StateFlow<Long> = voiceRecordingManager.recordingDuration
    val isCancelling: StateFlow<Boolean> = voiceRecordingManager.isCancelled

    // 会话 ID
    private var sessionId: String? = null

    // 权限
    val hasPermission: Boolean get() = voiceRecordingManager.hasPermission

    init {
        // 监听录音状态变化
        viewModelScope.launch {
            voiceRecordingManager.isRecording.collect { recording ->
                if (recording) {
                    _uiState.update { it.copy(state = VoiceScheduleState.RECORDING) }
                }
            }
        }
    }

    // MARK: - Recording

    /**
     * 开始录音
     */
    fun startRecording() {
        val currentState = _uiState.value.state
        if (currentState == VoiceScheduleState.PROCESSING ||
            currentState == VoiceScheduleState.CONFIRMING
        ) {
            return
        }

        if (voiceRecordingManager.startRecording()) {
            _uiState.update { it.copy(state = VoiceScheduleState.RECORDING) }
        } else {
            addMessage(ChatMessageType.SYSTEM, "录音失败，请检查麦克风权限")
        }
    }

    /**
     * 取消录音
     */
    fun cancelRecording() {
        voiceRecordingManager.cancelRecording()
        if (_uiState.value.state == VoiceScheduleState.RECORDING) {
            _uiState.update { it.copy(state = VoiceScheduleState.IDLE) }
        }
    }

    /**
     * 设置取消状态（UI 反馈）
     */
    fun setCancelling(cancelling: Boolean) {
        voiceRecordingManager.setCancelling(cancelling)
    }

    /**
     * 提交录音
     */
    fun submitRecording() {
        viewModelScope.launch {
            val result = voiceRecordingManager.stopRecording()
            if (result == null) {
                // 录音太短或取消
                addMessage(ChatMessageType.SYSTEM, "录音时间太短，请长按说话")
                _uiState.update { it.copy(state = VoiceScheduleState.IDLE) }
                return@launch
            }

            processAudio(result.data)
        }
    }

    // MARK: - Text Input

    /**
     * 提交文本输入（走 V4 Agent 流程，跳过语音转文字）
     */
    fun submitTextInput(text: String) {
        if (text.isBlank()) return
        val currentState = _uiState.value.state
        if (currentState == VoiceScheduleState.PROCESSING ||
            currentState == VoiceScheduleState.CONFIRMING
        ) {
            return
        }

        viewModelScope.launch {
            _uiState.update {
                it.copy(
                    state = VoiceScheduleState.PROCESSING,
                    processingStatus = "处理中...",
                    progressItems = emptyList()
                )
            }

            val token = tokenManager.accessToken.first()
            if (token == null) {
                addMessage(ChatMessageType.SYSTEM, "请先登录")
                _uiState.update { it.copy(state = VoiceScheduleState.IDLE) }
                return@launch
            }

            try {
                voiceScheduleSseClient.streamTextInput(
                    sessionId = sessionId,
                    text = text,
                    token = token
                ).collect { event ->
                    handleEvent(event)
                }
            } catch (e: Exception) {
                Log.e(TAG, "文本处理失败: ${e.message}")
                addMessage(ChatMessageType.SYSTEM, "处理失败: ${e.message}")
                _uiState.update { it.copy(state = VoiceScheduleState.IDLE) }
            }
        }
    }

    // MARK: - Process Audio

    private suspend fun processAudio(audioData: ByteArray) {
        _uiState.update {
            it.copy(
                state = VoiceScheduleState.PROCESSING,
                processingStatus = "上传中...",
                progressItems = emptyList()
            )
        }

        val token = tokenManager.accessToken.first()
        if (token == null) {
            addMessage(ChatMessageType.SYSTEM, "请先登录")
            _uiState.update { it.copy(state = VoiceScheduleState.IDLE) }
            return
        }

        try {
            voiceScheduleSseClient.streamVoiceSchedule(
                audioData = audioData,
                token = token,
                sessionId = sessionId  // 多轮对话保持上下文
            ).collect { event ->
                handleEvent(event)
            }
        } catch (e: Exception) {
            Log.e(TAG, "处理失败: ${e.message}")
            addMessage(ChatMessageType.SYSTEM, "处理失败: ${e.message}")
            _uiState.update { it.copy(state = VoiceScheduleState.IDLE) }
        }
    }

    // MARK: - Handle SSE Events

    private fun handleEvent(event: VoiceScheduleEvent) {
        Log.d(TAG, "Event: ${event.type}")

        when (event.type) {
            VoiceScheduleEventType.SESSION_START -> {
                event.sessionId?.let { sessionId = it }
                _uiState.update { it.copy(processingStatus = "已连接") }
            }

            VoiceScheduleEventType.RECOGNIZING -> {
                _uiState.update {
                    it.copy(processingStatus = event.status ?: "识别中...")
                }
            }

            VoiceScheduleEventType.PROGRESS -> {
                handleProgressEvent(event)
            }

            VoiceScheduleEventType.TRANSCRIPT -> {
                // 标记所有进度项为完成
                _uiState.update {
                    it.copy(
                        progressItems = it.progressItems.map { item ->
                            item.copy(isCompleted = true)
                        },
                        transcript = event.message ?: "",
                        processingStatus = "分析中..."
                    )
                }
                // 添加用户消息
                event.message?.takeIf { it.isNotEmpty() }?.let { text ->
                    addMessage(ChatMessageType.USER, text)
                }
            }

            VoiceScheduleEventType.THINKING -> {
                _uiState.update {
                    it.copy(processingStatus = event.status ?: "AI 思考中...")
                }
            }

            VoiceScheduleEventType.CLARIFY -> {
                event.questions?.let { questions ->
                    _uiState.update { it.copy(clarifyQuestions = questions) }
                    addMessage(
                        ChatMessageType.AI_QUESTION,
                        "需要确认一些信息：",
                        questions = questions
                    )
                }
                _uiState.update { it.copy(state = VoiceScheduleState.IDLE) }
            }

            VoiceScheduleEventType.SCHEDULE -> {
                event.items?.let { items ->
                    _uiState.update {
                        it.copy(
                            schedule = items,
                            reasoning = event.reasoning ?: emptyList(),
                            progressItems = emptyList()
                        )
                    }
                    val isQueryMode = event.isQuery ?: false
                    addMessage(
                        ChatMessageType.AI_SCHEDULE,
                        if (isQueryMode) "当前时刻表" else "已生成时刻表",
                        schedule = items,
                        reasoning = event.reasoning,
                        isQuery = isQueryMode
                    )
                }
                _uiState.update { it.copy(state = VoiceScheduleState.IDLE) }
            }

            VoiceScheduleEventType.CURRENT_STATUS -> {
                val emoji = event.emoji ?: "🤔"
                val status = event.statusText ?: "未知状态"
                val reason = event.reason ?: ""

                val item = ScheduleItem(
                    startTime = java.text.SimpleDateFormat("HH:mm", java.util.Locale.getDefault())
                        .format(java.util.Date()),
                    endTime = "--:--",
                    emoji = emoji,
                    status = status,
                    executed = false
                )
                _uiState.update {
                    it.copy(
                        schedule = listOf(item),
                        reasoning = event.reasoning ?: emptyList(),
                        progressItems = emptyList()
                    )
                }

                var content = "$emoji $status"
                if (reason.isNotEmpty()) {
                    content += "\n$reason"
                }
                addMessage(ChatMessageType.AI_TEXT, content)
                _uiState.update { it.copy(state = VoiceScheduleState.IDLE) }
            }

            VoiceScheduleEventType.VISIBILITY_PROMPT -> {
                _uiState.update { it.copy(showVisibilitySelection = true) }
            }

            VoiceScheduleEventType.CIRCLE_LIST -> {
                event.circles?.let { circles ->
                    _uiState.update { it.copy(availableCircles = circles) }
                }
            }

            VoiceScheduleEventType.CHAT -> {
                val message = event.message ?: "我在这里，有什么可以帮你的？"
                addMessage(ChatMessageType.AI_TEXT, message)
                _uiState.update {
                    it.copy(
                        progressItems = emptyList(),
                        // 不覆盖终态（COMPLETED），避免 chat 事件把已完成的状态冲掉
                        state = if (it.state == VoiceScheduleState.COMPLETED) VoiceScheduleState.COMPLETED
                                else VoiceScheduleState.IDLE
                    )
                }
            }

            VoiceScheduleEventType.CONFIRMED -> {
                addMessage(ChatMessageType.SYSTEM, "✓ 已保存！状态将按时刻表自动更新")
                _uiState.update { it.copy(state = VoiceScheduleState.COMPLETED) }
            }

            VoiceScheduleEventType.ERROR -> {
                val msg = event.message ?: "未知错误"
                addMessage(ChatMessageType.SYSTEM, "错误: $msg")
                _uiState.update {
                    it.copy(
                        state = VoiceScheduleState.IDLE,
                        progressItems = emptyList()
                    )
                }
            }

            // ========== V4 Agent 架构事件 ==========
            VoiceScheduleEventType.PHASE -> {
                val phase = event.phase ?: ""
                _uiState.update {
                    it.copy(
                        v4Phase = phase,
                        processingStatus = event.message ?: "处理中..."
                    )
                }
                Log.d(TAG, "V4 phase=$phase loop=${event.loop ?: 0}")
            }

            VoiceScheduleEventType.TOOL_START -> {
                val toolName = event.toolName ?: ""
                val toolDesc = event.message ?: "执行 $toolName..."
                val item = ProgressItem(action = "tool", message = toolDesc)
                _uiState.update {
                    it.copy(
                        currentToolName = toolName,
                        progressItems = it.progressItems + item,
                        processingStatus = toolDesc
                    )
                }
            }

            VoiceScheduleEventType.TOOL_END -> {
                _uiState.update {
                    val items = it.progressItems.toMutableList()
                    if (items.isNotEmpty()) {
                        items[items.lastIndex] = items.last().copy(isCompleted = true)
                    }
                    it.copy(
                        currentToolName = "",
                        progressItems = items
                    )
                }
            }

            VoiceScheduleEventType.SCHEDULE_PREVIEW -> {
                event.items?.let { items ->
                    _uiState.update {
                        it.copy(
                            schedule = items,
                            progressItems = emptyList(),
                            canApprove = true,
                            state = VoiceScheduleState.AWAITING_APPROVAL
                        )
                    }
                    val msg = event.message ?: "已生成时刻表预览"
                    addMessage(
                        ChatMessageType.AI_SCHEDULE,
                        msg,
                        schedule = items,
                        reasoning = null,
                        isQuery = false
                    )
                }
            }

            VoiceScheduleEventType.SCHEDULE_SAVED -> {
                event.items?.let { items ->
                    _uiState.update { it.copy(schedule = items) }
                    val dateStr = event.date ?: SimpleDateFormat("yyyy-MM-dd", Locale.getDefault()).format(java.util.Date())
                    scheduleCalendarReminders(items, dateStr)
                }
                addMessage(ChatMessageType.SYSTEM, "✓ 时刻表已保存！状态将按时刻表自动更新")
                _uiState.update { it.copy(state = VoiceScheduleState.COMPLETED) }
                // 通知首页刷新 Grid
                GridRefreshEvent.trigger()
            }

            VoiceScheduleEventType.STATUS_UPDATED -> {
                val emoji = event.emoji ?: "🤔"
                val status = event.status ?: "状态已更新"
                val msg = event.message ?: "$emoji $status"
                addMessage(ChatMessageType.AI_TEXT, msg)
                _uiState.update {
                    it.copy(
                        progressItems = emptyList(),
                        state = VoiceScheduleState.COMPLETED
                    )
                }
                // 通知首页刷新 Grid（更新 GIF 等）
                GridRefreshEvent.trigger()
            }

            VoiceScheduleEventType.PREFERENCE_UPDATED -> {
                val msg = event.message ?: "偏好设置已更新"
                addMessage(ChatMessageType.SYSTEM, "✓ $msg")
                _uiState.update {
                    it.copy(
                        progressItems = emptyList(),
                        state = VoiceScheduleState.IDLE
                    )
                }
            }

            VoiceScheduleEventType.CHAT_STREAM -> {
                event.message?.let { text ->
                    _uiState.update { state ->
                        val newText = state.streamingText + text
                        val messages = state.messages.toMutableList()
                        val lastAiIdx = messages.indexOfLast { it.type == ChatMessageType.AI_TEXT }
                        if (lastAiIdx >= 0) {
                            messages[lastAiIdx] = messages[lastAiIdx].copy(content = newText)
                        } else {
                            messages.add(ChatMessage(type = ChatMessageType.AI_TEXT, content = newText))
                        }
                        state.copy(streamingText = newText, messages = messages)
                    }
                }
            }

            VoiceScheduleEventType.CHAT_STREAM_END -> {
                _uiState.update {
                    it.copy(
                        streamingText = "",
                        progressItems = emptyList(),
                        state = VoiceScheduleState.IDLE
                    )
                }
            }

            VoiceScheduleEventType.FRIENDS_RESULT -> {
                _uiState.update {
                    it.copy(
                        friendsResult = event.friends ?: emptyList(),
                        progressItems = emptyList(),
                        state = VoiceScheduleState.IDLE
                    )
                }
                val msg = event.message ?: "查询完成"
                addMessage(ChatMessageType.AI_TEXT, msg)
            }

            VoiceScheduleEventType.MESSAGE_PREVIEW -> {
                _uiState.update {
                    it.copy(
                        pendingMessage = event.pendingMessage,
                        progressItems = emptyList(),
                        canApprove = true,
                        state = VoiceScheduleState.AWAITING_APPROVAL
                    )
                }
                val msg = event.message ?: "消息预览"
                addMessage(ChatMessageType.AI_TEXT, msg, awaitingAction = true)
            }

            VoiceScheduleEventType.MESSAGE_SENT -> {
                _uiState.update {
                    it.copy(
                        pendingMessage = null,
                        state = VoiceScheduleState.IDLE
                    )
                }
                val sentTo = event.sentTo ?: ""
                val msg = event.message ?: "消息已发送给 $sentTo"
                addMessage(ChatMessageType.SYSTEM, "✓ $msg")
            }

            VoiceScheduleEventType.INVITE_PREVIEW -> {
                _uiState.update {
                    it.copy(
                        pendingInvite = event.pendingInvite,
                        progressItems = emptyList(),
                        canApprove = true,
                        state = VoiceScheduleState.AWAITING_APPROVAL
                    )
                }
                val msg = event.message ?: "邀请预览"
                addMessage(ChatMessageType.AI_TEXT, msg, awaitingAction = true)
            }

            VoiceScheduleEventType.INVITE_SENT -> {
                _uiState.update {
                    it.copy(
                        pendingInvite = null,
                        state = VoiceScheduleState.IDLE
                    )
                }
                val sentTo = event.sentTo ?: ""
                val msg = event.message ?: "邀请已发送给 $sentTo"
                addMessage(ChatMessageType.SYSTEM, "✓ $msg")
            }

            VoiceScheduleEventType.DELETE_PREVIEW -> {
                _uiState.update {
                    it.copy(
                        pendingDeletion = event.pendingDeletion,
                        progressItems = emptyList(),
                        canApprove = true,
                        state = VoiceScheduleState.AWAITING_APPROVAL
                    )
                }
                val msg = event.message ?: "将删除安排，等待确认"
                addMessage(ChatMessageType.AI_TEXT, msg, awaitingAction = true)
            }

            VoiceScheduleEventType.SCHEDULE_DELETED -> {
                _uiState.update {
                    it.copy(
                        pendingDeletion = null,
                        schedule = event.items ?: emptyList(),
                        state = VoiceScheduleState.COMPLETED
                    )
                }
                val msg = event.message ?: "已删除"
                addMessage(ChatMessageType.SYSTEM, "✓ $msg")
            }

            VoiceScheduleEventType.FRIEND_REMOVED -> {
                _uiState.update {
                    it.copy(
                        pendingDeletion = null,
                        state = VoiceScheduleState.COMPLETED
                    )
                }
                val msg = event.message ?: "好友已删除"
                addMessage(ChatMessageType.SYSTEM, "✓ $msg")
            }

            VoiceScheduleEventType.INVITE_RESPONDED -> {
                val msg = event.message ?: "已响应邀请"
                addMessage(ChatMessageType.SYSTEM, "✓ $msg")
                _uiState.update { it.copy(state = VoiceScheduleState.COMPLETED) }
            }

            // ========== Legacy 多阶段对话状态机事件 ==========
            VoiceScheduleEventType.PHASE_CHANGE -> {
                event.phase?.let { phaseStr ->
                    val phase = try { ConversationPhase.valueOf(phaseStr.uppercase()) } catch (_: Exception) { null }
                    phase?.let {
                        _uiState.update {
                            it.copy(
                                currentPhase = phase,
                                processingStatus = event.message ?: "阶段: ${phase.name}"
                            )
                        }
                    }
                    Log.d(TAG, "Legacy 阶段变更: ${event.previousPhase} -> $phaseStr")
                }
            }

            VoiceScheduleEventType.INTENT_SUMMARY -> {
                event.intentSummary?.let { summary ->
                    _uiState.update { it.copy(intentSummary = summary) }
                    val activities = summary.activities?.joinToString("、") ?: ""
                    val message = event.message ?: "理解到：$activities"
                    addMessage(ChatMessageType.AI_TEXT, message)
                    summary.reasoning?.takeIf { it.isNotEmpty() }?.let { reasoning ->
                        _uiState.update { it.copy(reasoning = reasoning) }
                    }
                }
            }

            VoiceScheduleEventType.DISCUSSION -> {
                val message = event.message ?: "需要确认一些信息"
                event.clarifications?.let { clarifications ->
                    _uiState.update { it.copy(clarifications = clarifications) }
                    val questions = clarifications
                        .filter { it.answered != true }
                        .map { item ->
                            ClarifyQuestion(
                                id = item.id,
                                question = item.question,
                                options = emptyList(),
                                allowVoice = true
                            )
                        }
                    if (questions.isNotEmpty()) {
                        addMessage(ChatMessageType.AI_QUESTION, message, questions = questions)
                    } else {
                        addMessage(ChatMessageType.AI_TEXT, message)
                    }
                } ?: addMessage(ChatMessageType.AI_TEXT, message)
                _uiState.update { it.copy(state = VoiceScheduleState.DISCUSSING) }
            }

            VoiceScheduleEventType.DRAFT_PLAN -> {
                event.draftPlan?.let { plan ->
                    _uiState.update { it.copy(draftPlan = plan) }
                    plan.schedule?.let { items ->
                        _uiState.update { it.copy(schedule = items) }
                    }
                    plan.reasoning?.let { reasoning ->
                        _uiState.update { it.copy(reasoning = reasoning) }
                    }
                    val summary = plan.summary ?: "已生成时刻表"
                    addMessage(
                        ChatMessageType.AI_SCHEDULE,
                        summary,
                        schedule = plan.schedule,
                        reasoning = plan.reasoning,
                        isQuery = false
                    )
                    plan.changes?.takeIf { it.isNotEmpty() }?.let { changes ->
                        val changesText = buildString {
                            appendLine("变更：")
                            changes.forEach { change ->
                                val typeEmoji = when (change.type) {
                                    "add" -> "➕"
                                    "delete" -> "➖"
                                    else -> "✏️"
                                }
                                appendLine("$typeEmoji ${change.description} (${change.timeRange})")
                            }
                        }
                        addMessage(ChatMessageType.SYSTEM, changesText)
                    }
                }
                _uiState.update {
                    it.copy(
                        canApprove = event.canApprove ?: true,
                        state = VoiceScheduleState.AWAITING_APPROVAL
                    )
                }
            }

            VoiceScheduleEventType.APPROVAL_PROMPT -> {
                val message = event.message ?: "确认后将保存时刻表"
                _uiState.update { it.copy(canApprove = event.canApprove ?: true) }
                event.draftPlan?.schedule?.let { items ->
                    _uiState.update { it.copy(schedule = items) }
                    addMessage(
                        ChatMessageType.AI_SCHEDULE,
                        message,
                        schedule = items,
                        reasoning = null,
                        isQuery = false
                    )
                } ?: run {
                    if (_uiState.value.schedule.isNotEmpty()) {
                        addMessage(ChatMessageType.SYSTEM, message)
                    }
                }
                _uiState.update { it.copy(state = VoiceScheduleState.AWAITING_APPROVAL) }
            }

            null -> {
                // 未知类型（coerceInputValues 会将未识别的枚举设为 null）
            }
        }
    }

    private fun handleProgressEvent(event: VoiceScheduleEvent) {
        val action = event.action
        val message = event.message

        if (action != null && message != null) {
            val item = ProgressItem(
                action = action,
                message = message,
                detail = event.detail
            )
            _uiState.update {
                it.copy(
                    progressItems = it.progressItems + item,
                    processingStatus = message
                )
            }
        } else if (event.detail != null && _uiState.value.progressItems.isNotEmpty()) {
            val items = _uiState.value.progressItems.toMutableList()
            items[items.lastIndex] = items.last().copy(detail = event.detail)
            _uiState.update { it.copy(progressItems = items) }
        }
    }

    // MARK: - Confirm

    /**
     * 确认保存时刻表
     */
    fun confirmSchedule() {
        confirmScheduleWithVisibility(
            visibility = _uiState.value.selectedVisibility,
            circleIds = _uiState.value.selectedCircleIds.toList()
        )
    }

    /**
     * 带可见性设置的确认
     */
    fun confirmScheduleWithVisibility(
        visibility: ScheduleVisibility,
        circleIds: List<String>
    ) {
        val sid = sessionId
        if (sid == null && _uiState.value.schedule.isNotEmpty()) {
            // 没有 session，模拟确认
            viewModelScope.launch {
                _uiState.update { it.copy(state = VoiceScheduleState.CONFIRMING) }
                kotlinx.coroutines.delay(500)
                addMessage(ChatMessageType.SYSTEM, "✓ 已保存！")
                _uiState.update { it.copy(state = VoiceScheduleState.COMPLETED) }
            }
            return
        }

        if (sid == null) return

        _uiState.update { it.copy(state = VoiceScheduleState.CONFIRMING) }

        viewModelScope.launch {
            val token = tokenManager.accessToken.first() ?: return@launch

            try {
                // 通过 V4 text 端点发送"确认"，让 LLM 调用 confirm 工具
                voiceScheduleSseClient.streamTextInput(
                    sessionId = sid,
                    text = "确认",
                    token = token
                ).collect { event ->
                    handleEvent(event)
                }
            } catch (e: Exception) {
                addMessage(ChatMessageType.SYSTEM, "确认失败: ${e.message}")
                _uiState.update { it.copy(state = VoiceScheduleState.IDLE) }
            }
        }
    }

    // MARK: - Cancel Pending

    /**
     * 放弃当前待确认操作（预览/草稿），保留对话继续修改
     */
    fun cancelPending() {
        _uiState.update {
            it.copy(
                schedule = emptyList(),
                pendingMessage = null,
                pendingInvite = null,
                pendingDeletion = null,
                canApprove = false,
                showVisibilitySelection = false,
                state = VoiceScheduleState.IDLE
            )
        }
        addMessage(ChatMessageType.SYSTEM, "已放弃，你可以继续说")
    }

    // MARK: - Cancel Session

    /**
     * 取消会话
     */
    fun cancelSession() {
        val sid = sessionId
        if (sid != null) {
            viewModelScope.launch {
                val token = tokenManager.accessToken.first() ?: return@launch
                try {
                    voiceScheduleSseClient.streamInteraction(
                        sessionId = sid,
                        action = VoiceScheduleAction.CANCEL.value,
                        token = token
                    ).collect { /* 忽略 */ }
                } catch (e: Exception) {
                    // 忽略取消错误
                }
            }
        }
        reset()
    }

    // MARK: - Visibility

    /**
     * 选择可见性
     */
    fun selectVisibility(visibility: ScheduleVisibility) {
        _uiState.update { it.copy(selectedVisibility = visibility) }
    }

    /**
     * 切换圈子选择
     */
    fun toggleCircleSelection(circleId: String) {
        val current = _uiState.value.selectedCircleIds.toMutableSet()
        if (current.contains(circleId)) {
            current.remove(circleId)
        } else {
            current.add(circleId)
        }
        _uiState.update { it.copy(selectedCircleIds = current) }
    }

    // MARK: - Reset

    /**
     * 重置状态
     */
    fun reset() {
        sessionId = null
        _uiState.update { VoiceScheduleUiState() }
    }

    // MARK: - Helper

    private fun addMessage(
        type: ChatMessageType,
        content: String,
        schedule: List<ScheduleItem>? = null,
        questions: List<ClarifyQuestion>? = null,
        reasoning: List<String>? = null,
        isQuery: Boolean = false,
        awaitingAction: Boolean = false
    ) {
        val message = ChatMessage(
            type = type,
            content = content,
            schedule = schedule,
            questions = questions,
            reasoning = reasoning,
            isQuery = isQuery,
            awaitingAction = awaitingAction
        )
        _uiState.update { it.copy(messages = it.messages + message) }
    }

    // MARK: - Calendar Reminders

    private fun scheduleCalendarReminders(items: List<ScheduleItem>, date: String) {
        scheduleReminders(appContext, items, date)
    }

    companion object {
        private const val TAG = "VoiceScheduleVM"

        fun scheduleReminders(context: Context, items: List<ScheduleItem>, date: String) {
            val reminderItems = items.filter { (it.remindBefore ?: 0) > 0 }
            if (reminderItems.isEmpty()) return

            try {
                val calendarId = getCalendarId(context) ?: return
                for (item in reminderItems) {
                    createCalendarReminderStatic(context, calendarId, item, date)
                }
            } catch (e: SecurityException) {
                Log.w(TAG, "日历权限未授予: ${e.message}")
            } catch (e: Exception) {
                Log.e(TAG, "创建日历提醒失败: ${e.message}")
            }
        }

        fun removeCalendarReminder(context: Context, date: String, startTime: String) {
            val prefs = context.getSharedPreferences("reminders", Context.MODE_PRIVATE)
            val eventId = prefs.getLong("reminder_${date}_${startTime}", -1L)
            if (eventId == -1L) return

            try {
                val deleteUri = android.content.ContentUris.withAppendedId(CalendarContract.Events.CONTENT_URI, eventId)
                context.contentResolver.delete(deleteUri, null, null)
                Log.d(TAG, "已删除日历提醒: $date $startTime")
            } catch (e: Exception) {
                Log.e(TAG, "删除日历提醒失败: ${e.message}")
            }
            prefs.edit().remove("reminder_${date}_${startTime}").apply()
        }

        private fun getCalendarId(context: Context): Long? {
            val projection = arrayOf(CalendarContract.Calendars._ID)
            val selection = "${CalendarContract.Calendars.IS_PRIMARY} = 1"
            context.contentResolver.query(
                CalendarContract.Calendars.CONTENT_URI, projection, selection, null, null
            )?.use { if (it.moveToFirst()) return it.getLong(0) }
            context.contentResolver.query(
                CalendarContract.Calendars.CONTENT_URI, projection, null, null, null
            )?.use { if (it.moveToFirst()) return it.getLong(0) }
            return null
        }

        private fun createCalendarReminderStatic(context: Context, calendarId: Long, item: ScheduleItem, date: String) {
            val sdf = SimpleDateFormat("yyyy-MM-dd HH:mm", Locale.getDefault())
            val startMillis = sdf.parse("$date ${item.startTime}")?.time ?: return
            val endMillis = sdf.parse("$date ${item.endTime}")?.time ?: return
            val remindMinutes = item.remindBefore ?: 15

            val eventValues = ContentValues().apply {
                put(CalendarContract.Events.CALENDAR_ID, calendarId)
                put(CalendarContract.Events.TITLE, "${item.emoji} ${item.status}")
                put(CalendarContract.Events.DTSTART, startMillis)
                put(CalendarContract.Events.DTEND, endMillis)
                put(CalendarContract.Events.EVENT_TIMEZONE, java.util.TimeZone.getDefault().id)
            }
            val eventUri = context.contentResolver.insert(CalendarContract.Events.CONTENT_URI, eventValues) ?: return
            val eventId = eventUri.lastPathSegment?.toLongOrNull() ?: return

            val reminderValues = ContentValues().apply {
                put(CalendarContract.Reminders.EVENT_ID, eventId)
                put(CalendarContract.Reminders.MINUTES, remindMinutes)
                put(CalendarContract.Reminders.METHOD, CalendarContract.Reminders.METHOD_ALERT)
            }
            context.contentResolver.insert(CalendarContract.Reminders.CONTENT_URI, reminderValues)

            val prefs = context.getSharedPreferences("reminders", Context.MODE_PRIVATE)
            prefs.edit().putLong("reminder_${date}_${item.startTime}", eventId).apply()
            Log.d(TAG, "已创建日历提醒: ${item.emoji} ${item.status}, 提前${remindMinutes}分钟")
        }
    }
}

/**
 * 语音时刻表 UI 状态
 */
data class VoiceScheduleUiState(
    val state: VoiceScheduleState = VoiceScheduleState.IDLE,
    val messages: List<ChatMessage> = emptyList(),
    val processingStatus: String = "",
    val transcript: String = "",
    val schedule: List<ScheduleItem> = emptyList(),
    val clarifyQuestions: List<ClarifyQuestion> = emptyList(),
    val reasoning: List<String> = emptyList(),
    val progressItems: List<ProgressItem> = emptyList(),
    // 可见性
    val showVisibilitySelection: Boolean = false,
    val selectedVisibility: ScheduleVisibility = ScheduleVisibility.ALL_FRIENDS,
    val selectedCircleIds: Set<String> = emptySet(),
    val availableCircles: List<CircleInfoCompact> = emptyList(),
    // V4 Agent 状态
    val v4Phase: String = "",
    val currentToolName: String = "",
    val friendsResult: List<V4FriendInfo> = emptyList(),
    val pendingMessage: V4PendingMessage? = null,
    val pendingInvite: V4PendingInvite? = null,
    val pendingDeletion: V4PendingDeletion? = null,
    val streamingText: String = "",
    // Legacy 多阶段对话
    val currentPhase: ConversationPhase = ConversationPhase.UNDERSTANDING,
    val intentSummary: IntentSummary? = null,
    val draftPlan: DraftPlan? = null,
    val clarifications: List<ClarificationItem> = emptyList(),
    val canApprove: Boolean = false
) {
    val hasSchedule: Boolean get() = schedule.isNotEmpty()
    val showOverlay: Boolean get() = messages.isNotEmpty() || state != VoiceScheduleState.IDLE
}
