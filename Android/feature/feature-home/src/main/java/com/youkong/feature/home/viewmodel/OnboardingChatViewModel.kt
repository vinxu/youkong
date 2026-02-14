package com.youkong.feature.home.viewmodel

import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.datastore.TokenManager
import com.youkong.core.network.model.VoiceScheduleEventType
import com.youkong.core.network.sse.VoiceScheduleSseClient
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import java.util.UUID
import javax.inject.Inject

/**
 * Onboarding Chat 消息
 */
data class OnboardingChatMessage(
    val id: String = UUID.randomUUID().toString(),
    val isUser: Boolean,
    val content: String,
    val timestamp: Long = System.currentTimeMillis()
)

/**
 * Chat UI State
 */
data class OnboardingChatUiState(
    val messages: List<OnboardingChatMessage> = emptyList(),
    val isStreaming: Boolean = false,
    val isCompleted: Boolean = false,
    val error: String? = null,
    val greeting: String = ""
)

@HiltViewModel
class OnboardingChatViewModel @Inject constructor(
    private val sseClient: VoiceScheduleSseClient,
    private val tokenManager: TokenManager
) : ViewModel() {

    companion object {
        private const val TAG = "OnboardingChat"
    }

    private val _uiState = MutableStateFlow(OnboardingChatUiState())
    val uiState: StateFlow<OnboardingChatUiState> = _uiState.asStateFlow()

    private var sessionId: String? = null
    private var streamingContent = StringBuilder()

    init {
        _uiState.value = _uiState.value.copy(greeting = buildGreeting())
    }

    fun sendMessage(text: String) {
        if (text.isBlank() || _uiState.value.isStreaming) return

        // 添加用户消息
        val userMessage = OnboardingChatMessage(isUser = true, content = text)
        _uiState.value = _uiState.value.copy(
            messages = _uiState.value.messages + userMessage,
            isStreaming = true,
            error = null
        )

        viewModelScope.launch {
            val token = tokenManager.accessToken.first()
            if (token.isNullOrEmpty()) {
                _uiState.value = _uiState.value.copy(
                    isStreaming = false,
                    error = "未登录"
                )
                return@launch
            }

            streamingContent.clear()
            try {
                sseClient.streamTextInput(
                    sessionId = sessionId,
                    text = text,
                    token = token,
                ).collect { event ->
                    Log.d(TAG, "Event: ${event.type}")

                    // 保存 session_id
                    event.sessionId?.let { sessionId = it }

                    when (event.type) {
                        VoiceScheduleEventType.CHAT_STREAM -> {
                            val chunk = event.text ?: ""
                            streamingContent.append(chunk)
                            updateStreamingMessage(streamingContent.toString())
                        }
                        VoiceScheduleEventType.CHAT_STREAM_END -> {
                            finalizeStreamingMessage()
                        }
                        VoiceScheduleEventType.CHAT -> {
                            val content = event.text ?: event.message ?: ""
                            if (content.isNotBlank()) {
                                addAiMessage(content)
                            }
                        }
                        VoiceScheduleEventType.SCHEDULE_PREVIEW -> {
                            val previewText = buildSchedulePreviewText(event)
                            if (previewText.isNotBlank()) {
                                addAiMessage(previewText)
                            }
                        }
                        VoiceScheduleEventType.SCHEDULE_SAVED -> {
                            addAiMessage("日程已保存 ✓")
                            _uiState.value = _uiState.value.copy(isCompleted = true)
                        }
                        VoiceScheduleEventType.STATUS_UPDATED -> {
                            val statusText = event.text ?: "状态已更新"
                            addAiMessage(statusText)
                        }
                        VoiceScheduleEventType.ERROR -> {
                            val errorMsg = event.message ?: "发生错误"
                            _uiState.value = _uiState.value.copy(
                                isStreaming = false,
                                error = errorMsg
                            )
                        }
                        else -> {}
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Stream error: ${e.message}")
                _uiState.value = _uiState.value.copy(
                    isStreaming = false,
                    error = e.message ?: "网络错误"
                )
            }

            // 流结束
            if (streamingContent.isNotEmpty()) {
                finalizeStreamingMessage()
            }
            _uiState.value = _uiState.value.copy(isStreaming = false)
        }
    }

    private fun updateStreamingMessage(content: String) {
        val messages = _uiState.value.messages.toMutableList()
        val lastMsg = messages.lastOrNull()
        if (lastMsg != null && !lastMsg.isUser && lastMsg.id == "streaming") {
            messages[messages.lastIndex] = lastMsg.copy(content = content)
        } else {
            messages.add(OnboardingChatMessage(id = "streaming", isUser = false, content = content))
        }
        _uiState.value = _uiState.value.copy(messages = messages)
    }

    private fun finalizeStreamingMessage() {
        val messages = _uiState.value.messages.toMutableList()
        val lastMsg = messages.lastOrNull()
        if (lastMsg != null && lastMsg.id == "streaming") {
            messages[messages.lastIndex] = lastMsg.copy(id = UUID.randomUUID().toString())
        }
        _uiState.value = _uiState.value.copy(messages = messages)
        streamingContent.clear()
    }

    private fun addAiMessage(content: String) {
        val aiMessage = OnboardingChatMessage(isUser = false, content = content)
        _uiState.value = _uiState.value.copy(
            messages = _uiState.value.messages + aiMessage
        )
    }

    private fun buildSchedulePreviewText(event: com.youkong.core.network.model.VoiceScheduleEvent): String {
        val items = event.items
        if (items.isNullOrEmpty()) return event.text ?: ""

        val sb = StringBuilder("📋 日程预览：\n")
        items.forEach { item ->
            sb.append("${item.emoji} ${item.startTime}-${item.endTime} ${item.status}\n")
        }
        return sb.toString().trim()
    }

    fun buildSuggestionChips(): List<String> {
        val hour = java.util.Calendar.getInstance().get(java.util.Calendar.HOUR_OF_DAY)
        return when {
            hour < 12 -> listOf(
                "帮我安排今天的时间表",
                "下午两点到四点想去健身"
            )
            hour < 18 -> listOf(
                "帮我安排今天剩下的时间",
                "晚上七点到九点想去运动"
            )
            else -> listOf(
                "帮我安排明天的时间表",
                "明天上午十点有个会议"
            )
        }
    }

    private fun buildGreeting(): String {
        val hour = java.util.Calendar.getInstance().get(java.util.Calendar.HOUR_OF_DAY)
        val timeGreeting = when (hour) {
            in 0..5 -> "这么晚还没睡呀"
            in 6..8 -> "早上好"
            in 9..11 -> "上午好"
            in 12..13 -> "中午好"
            in 14..16 -> "下午好"
            in 17..18 -> "傍晚好"
            in 19..21 -> "晚上好"
            else -> "夜深了"
        }
        return "${timeGreeting}！我是你的 AI 助手 👋\n告诉我接下来的安排，我帮你生成今天的时间表"
    }
}
