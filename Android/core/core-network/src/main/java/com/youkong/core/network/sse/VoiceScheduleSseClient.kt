package com.youkong.core.network.sse

import android.util.Log
import com.youkong.core.network.BuildConfig
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.InferenceV2SseEvent
import com.youkong.core.network.model.VoiceScheduleEvent
import com.youkong.core.network.model.VoiceScheduleInteractionData
import com.youkong.core.network.model.VoiceScheduleInteractionRequest
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okio.BufferedSource
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Singleton

/**
 * 语音时刻表 SSE 客户端
 *
 * 支持音频上传和流式响应处理
 */
@Singleton
class VoiceScheduleSseClient @Inject constructor(
    private val json: Json
) {
    companion object {
        private const val TAG = "VoiceScheduleSSE"
    }

    private val baseUrl = BuildConfig.BASE_URL

    // SSE 专用 OkHttpClient，超时时间更长
    private val sseClient = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(5, TimeUnit.MINUTES)  // SSE 需要长时间读取
        .writeTimeout(60, TimeUnit.SECONDS)  // 音频上传可能需要更长时间
        .build()

    /**
     * 流式上传音频并接收 SSE 事件
     *
     * @param audioData WAV 格式音频数据
     * @param token 认证 Token
     * @param sessionId 可选的会话 ID（用于多轮对话）
     * @return Flow<VoiceScheduleEvent> 事件流
     */
    fun streamVoiceSchedule(
        audioData: ByteArray,
        token: String,
        sessionId: String? = null
    ): Flow<VoiceScheduleEvent> = callbackFlow {
        val multipartBuilder = MultipartBody.Builder()
            .setType(MultipartBody.FORM)

        // 添加 session_id（如果有）
        sessionId?.let {
            multipartBuilder.addFormDataPart("session_id", it)
        }

        // 添加音频文件
        multipartBuilder.addFormDataPart(
            "audio",
            "voice.wav",
            audioData.toRequestBody("audio/wav".toMediaType())
        )

        val requestBody = multipartBuilder.build()

        val request = Request.Builder()
            .url("${baseUrl}agent/voice-schedule/stream")
            .post(requestBody)
            .addHeader("Authorization", "Bearer $token")
            .addHeader("Accept", "text/event-stream")
            .addHeader("Cache-Control", "no-cache")
            .build()

        val call = sseClient.newCall(request)

        withContext(Dispatchers.IO) {
            try {
                val response = call.execute()
                Log.d(TAG, "Voice schedule response: ${response.code}")

                if (!response.isSuccessful) {
                    trySend(
                        VoiceScheduleEvent(
                            type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                            message = "HTTP ${response.code}: ${response.message}"
                        )
                    )
                    close()
                    return@withContext
                }

                val source: BufferedSource? = response.body?.source()
                if (source == null) {
                    trySend(
                        VoiceScheduleEvent(
                            type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                            message = "Empty response body"
                        )
                    )
                    close()
                    return@withContext
                }

                // 逐行读取 SSE 流
                while (isActive && !source.exhausted()) {
                    val line = source.readUtf8Line() ?: break
                    Log.d(TAG, "SSE Line: ${line.take(100)}...")

                    // 检查 [DONE] 信号
                    if (line == "data: [DONE]") {
                        Log.d(TAG, "Received DONE signal")
                        break
                    }

                    if (line.startsWith("data: ")) {
                        val jsonStr = line.removePrefix("data: ").trim()
                        if (jsonStr.isNotEmpty()) {
                            try {
                                val event = json.decodeFromString(
                                    VoiceScheduleEvent.serializer(),
                                    jsonStr
                                )
                                Log.d(TAG, "Parsed event: ${event.type}")
                                trySend(event)

                                // 终态事件时结束（防止 [DONE] 未收到时卡住）
                                val terminalTypes = setOf(
                                    com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                                    com.youkong.core.network.model.VoiceScheduleEventType.CONFIRMED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.SCHEDULE_SAVED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.STATUS_UPDATED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.MESSAGE_SENT,
                                    com.youkong.core.network.model.VoiceScheduleEventType.INVITE_SENT,
                                    com.youkong.core.network.model.VoiceScheduleEventType.DELETE_PREVIEW,
                                    com.youkong.core.network.model.VoiceScheduleEventType.SCHEDULE_DELETED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.FRIEND_REMOVED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.INVITE_RESPONDED,
                                )
                                if (event.type in terminalTypes) {
                                    break
                                }
                            } catch (e: Exception) {
                                Log.e(TAG, "Parse error: ${e.message}")
                            }
                        }
                    }
                }

                source.close()
                response.close()
            } catch (e: Exception) {
                Log.e(TAG, "SSE error: ${e.message}")
                trySend(
                    VoiceScheduleEvent(
                        type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                        message = e.message ?: "Unknown error"
                    )
                )
            }

            close()
        }

        awaitClose {
            call.cancel()
        }
    }

    /**
     * 发送交互请求（确认、取消等）
     *
     * @param sessionId 会话 ID
     * @param action 动作类型
     * @param data 交互数据
     * @param token 认证 Token
     * @return Flow<VoiceScheduleEvent> 事件流
     */
    fun streamInteraction(
        sessionId: String,
        action: String,
        data: VoiceScheduleInteractionData? = null,
        token: String
    ): Flow<VoiceScheduleEvent> = callbackFlow {
        val requestData = VoiceScheduleInteractionRequest(
            sessionId = sessionId,
            action = action,
            data = data
        )

        val requestBody = json.encodeToString(
            VoiceScheduleInteractionRequest.serializer(),
            requestData
        ).toRequestBody("application/json".toMediaType())

        val request = Request.Builder()
            .url("${baseUrl}agent/voice-schedule/interact")
            .post(requestBody)
            .addHeader("Authorization", "Bearer $token")
            .addHeader("Accept", "text/event-stream")
            .addHeader("Cache-Control", "no-cache")
            .addHeader("Content-Type", "application/json")
            .build()

        val call = sseClient.newCall(request)

        withContext(Dispatchers.IO) {
            try {
                val response = call.execute()
                Log.d(TAG, "Interaction response: ${response.code}")

                if (!response.isSuccessful) {
                    trySend(
                        VoiceScheduleEvent(
                            type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                            message = "HTTP ${response.code}: ${response.message}"
                        )
                    )
                    close()
                    return@withContext
                }

                val source: BufferedSource? = response.body?.source()
                if (source == null) {
                    trySend(
                        VoiceScheduleEvent(
                            type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                            message = "Empty response body"
                        )
                    )
                    close()
                    return@withContext
                }

                while (isActive && !source.exhausted()) {
                    val line = source.readUtf8Line() ?: break
                    Log.d(TAG, "Interaction Line: ${line.take(100)}...")

                    if (line == "data: [DONE]") {
                        break
                    }

                    if (line.startsWith("data: ")) {
                        val jsonStr = line.removePrefix("data: ").trim()
                        if (jsonStr.isNotEmpty()) {
                            try {
                                val event = json.decodeFromString(
                                    VoiceScheduleEvent.serializer(),
                                    jsonStr
                                )
                                trySend(event)

                                val terminalTypes = setOf(
                                    com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                                    com.youkong.core.network.model.VoiceScheduleEventType.CONFIRMED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.SCHEDULE_SAVED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.STATUS_UPDATED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.MESSAGE_SENT,
                                    com.youkong.core.network.model.VoiceScheduleEventType.INVITE_SENT,
                                    com.youkong.core.network.model.VoiceScheduleEventType.SCHEDULE_DELETED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.FRIEND_REMOVED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.INVITE_RESPONDED,
                                )
                                if (event.type in terminalTypes) {
                                    break
                                }
                            } catch (e: Exception) {
                                Log.e(TAG, "Parse error: ${e.message}")
                            }
                        }
                    }
                }

                source.close()
                response.close()
            } catch (e: Exception) {
                Log.e(TAG, "Interaction error: ${e.message}")
                trySend(
                    VoiceScheduleEvent(
                        type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                        message = e.message ?: "Unknown error"
                    )
                )
            }

            close()
        }

        awaitClose {
            call.cancel()
        }
    }

    /**
     * V4 文本输入（用于确认、补充等交互）
     * 通过 /voice-schedule/text 端点发送文本，走 V4 Agent 循环
     */
    fun streamTextInput(
        sessionId: String?,
        text: String,
        token: String,
        sensorData: com.youkong.core.network.model.AgentStatusRequest? = null
    ): Flow<VoiceScheduleEvent> = callbackFlow {
        val bodyJson = buildString {
            append("{")
            if (sessionId != null) append("\"session_id\":\"$sessionId\",")
            append("\"text\":\"$text\"")
            if (sensorData != null) {
                append(",\"sensor_data\":")
                append(json.encodeToString(com.youkong.core.network.model.AgentStatusRequest.serializer(), sensorData))
            }
            append("}")
        }

        val requestBody = bodyJson.toRequestBody("application/json".toMediaType())

        val request = Request.Builder()
            .url("${baseUrl}agent/voice-schedule/text")
            .post(requestBody)
            .addHeader("Authorization", "Bearer $token")
            .addHeader("Accept", "text/event-stream")
            .addHeader("Cache-Control", "no-cache")
            .addHeader("Content-Type", "application/json")
            .build()

        val call = sseClient.newCall(request)

        withContext(Dispatchers.IO) {
            try {
                val response = call.execute()
                Log.d(TAG, "Text input response: ${response.code}")

                if (!response.isSuccessful) {
                    trySend(VoiceScheduleEvent(
                        type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                        message = "HTTP ${response.code}: ${response.message}"
                    ))
                    close()
                    return@withContext
                }

                val source: BufferedSource? = response.body?.source()
                if (source == null) {
                    trySend(VoiceScheduleEvent(
                        type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                        message = "Empty response body"
                    ))
                    close()
                    return@withContext
                }

                while (isActive && !source.exhausted()) {
                    val line = source.readUtf8Line() ?: break
                    Log.d(TAG, "Text Line: ${line.take(100)}...")

                    if (line == "data: [DONE]") break

                    if (line.startsWith("data: ")) {
                        val jsonStr = line.removePrefix("data: ").trim()
                        if (jsonStr.isNotEmpty()) {
                            try {
                                val event = json.decodeFromString(
                                    VoiceScheduleEvent.serializer(),
                                    jsonStr
                                )
                                trySend(event)

                                val terminalTypes = setOf(
                                    com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                                    com.youkong.core.network.model.VoiceScheduleEventType.CONFIRMED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.SCHEDULE_SAVED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.STATUS_UPDATED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.MESSAGE_SENT,
                                    com.youkong.core.network.model.VoiceScheduleEventType.INVITE_SENT,
                                    com.youkong.core.network.model.VoiceScheduleEventType.SCHEDULE_DELETED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.FRIEND_REMOVED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.INVITE_RESPONDED,
                                )
                                if (event.type in terminalTypes) {
                                    break
                                }
                            } catch (e: Exception) {
                                Log.e(TAG, "Parse error: ${e.message}")
                            }
                        }
                    }
                }

                source.close()
                response.close()
            } catch (e: Exception) {
                Log.e(TAG, "Text input error: ${e.message}")
                trySend(VoiceScheduleEvent(
                    type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                    message = e.message ?: "Unknown error"
                ))
            }

            close()
        }

        awaitClose {
            call.cancel()
        }
    }

    /**
     * 流式状态推断（调用推断专用 SSE 端点）
     * POST /agent/infer-status-v2/stream
     */
    fun streamInferStatus(
        sensorData: AgentStatusRequest,
        token: String,
    ): Flow<InferenceV2SseEvent> = callbackFlow {
        val bodyJson = json.encodeToString(AgentStatusRequest.serializer(), sensorData)
        val requestBody = bodyJson.toRequestBody("application/json".toMediaType())

        val request = Request.Builder()
            .url("${baseUrl}agent/infer-status-v2/stream")
            .post(requestBody)
            .addHeader("Authorization", "Bearer $token")
            .addHeader("Accept", "text/event-stream")
            .addHeader("Cache-Control", "no-cache")
            .addHeader("Content-Type", "application/json")
            .build()

        val call = sseClient.newCall(request)

        withContext(Dispatchers.IO) {
            try {
                val response = call.execute()
                Log.d(TAG, "Infer status response: ${response.code}")

                if (!response.isSuccessful) {
                    trySend(InferenceV2SseEvent(type = "error", message = "HTTP ${response.code}: ${response.message}"))
                    close()
                    return@withContext
                }

                val source: BufferedSource? = response.body?.source()
                if (source == null) {
                    trySend(InferenceV2SseEvent(type = "error", message = "Empty response body"))
                    close()
                    return@withContext
                }

                while (isActive && !source.exhausted()) {
                    val line = source.readUtf8Line() ?: break
                    Log.d(TAG, "Infer Line: ${line.take(100)}...")

                    if (line == "data: [DONE]") break

                    // 跳过 keepalive 注释
                    if (line.startsWith(":")) continue

                    if (line.startsWith("data: ")) {
                        val jsonStr = line.removePrefix("data: ").trim()
                        if (jsonStr.isNotEmpty()) {
                            try {
                                val event = json.decodeFromString(
                                    InferenceV2SseEvent.serializer(),
                                    jsonStr
                                )
                                trySend(event)

                                if (event.type == "error" || event.type == "result") {
                                    // result 是最终事件，但不立即 break，等 [DONE]
                                }
                            } catch (e: Exception) {
                                Log.e(TAG, "Infer parse error: ${e.message}")
                            }
                        }
                    }
                }

                source.close()
                response.close()
            } catch (e: Exception) {
                Log.e(TAG, "Infer status error: ${e.message}")
                trySend(InferenceV2SseEvent(type = "error", message = e.message ?: "Unknown error"))
            }

            close()
        }

        awaitClose {
            call.cancel()
        }
    }

    /**
     * 发送带音频的交互请求（语音回答问题）
     */
    fun streamInteractionWithAudio(
        sessionId: String,
        action: String,
        audioData: ByteArray,
        token: String
    ): Flow<VoiceScheduleEvent> = callbackFlow {
        val multipartBuilder = MultipartBody.Builder()
            .setType(MultipartBody.FORM)
            .addFormDataPart("session_id", sessionId)
            .addFormDataPart("action", action)
            .addFormDataPart(
                "audio",
                "voice.wav",
                audioData.toRequestBody("audio/wav".toMediaType())
            )

        val requestBody = multipartBuilder.build()

        val request = Request.Builder()
            .url("${baseUrl}agent/voice-schedule/interact")
            .post(requestBody)
            .addHeader("Authorization", "Bearer $token")
            .addHeader("Accept", "text/event-stream")
            .addHeader("Cache-Control", "no-cache")
            .build()

        val call = sseClient.newCall(request)

        withContext(Dispatchers.IO) {
            try {
                val response = call.execute()
                Log.d(TAG, "Audio interaction response: ${response.code}")

                if (!response.isSuccessful) {
                    trySend(
                        VoiceScheduleEvent(
                            type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                            message = "HTTP ${response.code}: ${response.message}"
                        )
                    )
                    close()
                    return@withContext
                }

                val source: BufferedSource? = response.body?.source()
                if (source == null) {
                    trySend(
                        VoiceScheduleEvent(
                            type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                            message = "Empty response body"
                        )
                    )
                    close()
                    return@withContext
                }

                while (isActive && !source.exhausted()) {
                    val line = source.readUtf8Line() ?: break

                    if (line == "data: [DONE]") {
                        break
                    }

                    if (line.startsWith("data: ")) {
                        val jsonStr = line.removePrefix("data: ").trim()
                        if (jsonStr.isNotEmpty()) {
                            try {
                                val event = json.decodeFromString(
                                    VoiceScheduleEvent.serializer(),
                                    jsonStr
                                )
                                trySend(event)

                                val terminalTypes = setOf(
                                    com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                                    com.youkong.core.network.model.VoiceScheduleEventType.CONFIRMED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.SCHEDULE_SAVED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.STATUS_UPDATED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.MESSAGE_SENT,
                                    com.youkong.core.network.model.VoiceScheduleEventType.INVITE_SENT,
                                    com.youkong.core.network.model.VoiceScheduleEventType.SCHEDULE_DELETED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.FRIEND_REMOVED,
                                    com.youkong.core.network.model.VoiceScheduleEventType.INVITE_RESPONDED,
                                )
                                if (event.type in terminalTypes) {
                                    break
                                }
                            } catch (e: Exception) {
                                Log.e(TAG, "Parse error: ${e.message}")
                            }
                        }
                    }
                }

                source.close()
                response.close()
            } catch (e: Exception) {
                Log.e(TAG, "Audio interaction error: ${e.message}")
                trySend(
                    VoiceScheduleEvent(
                        type = com.youkong.core.network.model.VoiceScheduleEventType.ERROR,
                        message = e.message ?: "Unknown error"
                    )
                )
            }

            close()
        }

        awaitClose {
            call.cancel()
        }
    }
}
