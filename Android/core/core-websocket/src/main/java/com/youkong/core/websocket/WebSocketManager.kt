package com.youkong.core.websocket

import android.util.Log
import com.youkong.core.datastore.TokenManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class WebSocketManager @Inject constructor(
    private val tokenManager: TokenManager,
) {
    companion object {
        private const val TAG = "WebSocketManager"
        private const val WS_URL = "ws://49.232.13.41:8080/ws"
        private const val RECONNECT_DELAY_MS = 5000L
        private const val PING_INTERVAL_MS = 30000L

        @Volatile
        private var instanceCount = 0
    }

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val json = Json { ignoreUnknownKeys = true }

    private var webSocket: WebSocket? = null

    @Volatile
    private var isConnected = false

    @Volatile
    private var isConnecting = false

    private var shouldReconnect = true
    private var pingJob: Job? = null

    init {
        instanceCount++
        Log.d(TAG, "WebSocketManager instance created, total instances: $instanceCount")
    }

    private val _newMessages = MutableSharedFlow<NewMessageData>(extraBufferCapacity = 64)
    val newMessages: SharedFlow<NewMessageData> = _newMessages.asSharedFlow()

    private val client = OkHttpClient.Builder()
        .readTimeout(0, TimeUnit.MILLISECONDS)  // 禁用读取超时，让 WebSocket 保持连接
        .build()

    fun connect() {
        Log.d(TAG, "connect() called, isConnected=$isConnected, isConnecting=$isConnecting")
        if (isConnected || isConnecting) {
            Log.d(TAG, "Skipping connect - already connected or connecting")
            return
        }

        scope.launch {
            val token = tokenManager.accessToken.first()
            if (token.isNullOrEmpty()) {
                Log.w(TAG, "No token available, skipping connect")
                return@launch
            }

            Log.d(TAG, "Token available (length=${token.length}), proceeding to connect")
            shouldReconnect = true
            connectInternal(token)
        }
    }

    private fun connectInternal(token: String) {
        synchronized(this) {
            if (isConnecting) {
                Log.w(TAG, "Already connecting, skipping")
                return
            }
            isConnecting = true
        }

        Log.d(TAG, "Starting WebSocket connection... [instance #$instanceCount]")

        // 关闭旧连接
        webSocket?.let {
            Log.d(TAG, "Closing old WebSocket connection")
            it.close(1000, "New connection")
        }
        webSocket = null

        val url = "$WS_URL?token=$token"
        Log.d(TAG, "Connecting to: ${WS_URL}?token=<${token.take(10)}...>")

        val request = Request.Builder()
            .url(url)
            .addHeader("User-Agent", "YouKong-Android")
            .build()

        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                Log.d(TAG, "WebSocket connected successfully, response code: ${response.code}")
                isConnecting = false
                isConnected = true
                startPingLoop()
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                Log.d(TAG, "WebSocket message: $text")
                handleMessage(text)
            }

            override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                webSocket.close(1000, null)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                pingJob?.cancel()
                isConnecting = false
                isConnected = false
                scheduleReconnect()
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                Log.e(TAG, "WebSocket failure: ${t.message}, response code: ${response?.code}, body: ${response?.body?.string()}", t)
                pingJob?.cancel()
                isConnecting = false
                isConnected = false
                scheduleReconnect()
            }
        })
    }

    private fun handleMessage(text: String) {
        try {
            val message = json.decodeFromString<WebSocketMessage>(text)
            when (message.type) {
                WebSocketMessageType.PONG -> { }
                WebSocketMessageType.NEW_MESSAGE -> {
                    message.data?.let { data ->
                        val newMessageData = json.decodeFromString<NewMessageData>(data.toString())
                        scope.launch {
                            _newMessages.emit(newMessageData)
                        }
                    }
                }
                else -> { }
            }
        } catch (e: Exception) {
            Log.e(TAG, "Failed to parse message", e)
        }
    }

    private fun scheduleReconnect() {
        if (!shouldReconnect) return

        scope.launch {
            delay(RECONNECT_DELAY_MS)
            val token = tokenManager.accessToken.first()
            if (!token.isNullOrEmpty() && shouldReconnect) {
                connectInternal(token)
            }
        }
    }

    fun disconnect() {
        shouldReconnect = false
        webSocket?.close(1000, "User disconnect")
        webSocket = null
        isConnected = false
    }

    private fun startPingLoop() {
        pingJob?.cancel()
        pingJob = scope.launch {
            while (isActive && isConnected) {
                delay(PING_INTERVAL_MS)
                if (isConnected) {
                    webSocket?.send("""{"type":"PING"}""")
                }
            }
        }
    }
}
