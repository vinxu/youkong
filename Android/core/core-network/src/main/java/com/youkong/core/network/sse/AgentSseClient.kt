package com.youkong.core.network.sse

import android.util.Log
import com.youkong.core.network.BuildConfig
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.SseEvent
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okio.BufferedSource
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AgentSseClient @Inject constructor(
    private val json: Json,
) {
    private val baseUrl = BuildConfig.BASE_URL

    // SSE 专用 OkHttpClient，超时时间更长
    private val sseClient = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(5, TimeUnit.MINUTES)  // SSE 需要长时间读取
        .writeTimeout(30, TimeUnit.SECONDS)
        .build()

    /**
     * 发起流式推理请求
     */
    fun streamStatus(request: AgentStatusRequest, token: String): Flow<SseEvent> = callbackFlow {
        val requestBody = json.encodeToString(AgentStatusRequest.serializer(), request)
            .toRequestBody("application/json".toMediaType())

        val httpRequest = Request.Builder()
            .url("${baseUrl}agent/status/stream")
            .post(requestBody)
            .addHeader("Authorization", "Bearer $token")
            .addHeader("Accept", "text/event-stream")
            .addHeader("Cache-Control", "no-cache")
            .build()

        val call = sseClient.newCall(httpRequest)

        withContext(Dispatchers.IO) {
            try {
                val response = call.execute()
                Log.d("SSE", "Response code: ${response.code}")

                if (!response.isSuccessful) {
                    trySend(
                        SseEvent(
                            type = "error",
                            content = "HTTP ${response.code}: ${response.message}"
                        )
                    )
                    close()
                    return@withContext
                }

                val source: BufferedSource? = response.body?.source()
                if (source == null) {
                    trySend(SseEvent(type = "error", content = "Empty response body"))
                    close()
                    return@withContext
                }

                // 逐行读取 SSE 流
                while (isActive && !source.exhausted()) {
                    val line = source.readUtf8Line() ?: break
                    Log.d("SSE", "Received line: $line")

                    if (line.startsWith("data: ")) {
                        val jsonStr = line.removePrefix("data: ").trim()
                        if (jsonStr.isNotEmpty()) {
                            try {
                                val event = json.decodeFromString(SseEvent.serializer(), jsonStr)
                                Log.d("SSE", "Parsed event: ${event.type}")
                                trySend(event)

                                // done 或 error 时结束
                                if (event.type == "done" || event.type == "error") {
                                    break
                                }
                            } catch (e: Exception) {
                                Log.e("SSE", "Parse error: ${e.message}")
                            }
                        }
                    }
                }

                source.close()
                response.close()
            } catch (e: Exception) {
                Log.e("SSE", "SSE error: ${e.message}")
                trySend(SseEvent(type = "error", content = e.message ?: "Unknown error"))
            }

            close()
        }

        awaitClose {
            call.cancel()
        }
    }
}
