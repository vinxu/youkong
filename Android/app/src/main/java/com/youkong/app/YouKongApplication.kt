package com.youkong.app

import android.app.Application
import android.util.Log
import androidx.hilt.work.HiltWorkerFactory
import androidx.lifecycle.DefaultLifecycleObserver
import androidx.lifecycle.LifecycleOwner
import androidx.lifecycle.ProcessLifecycleOwner
import androidx.work.Configuration
import androidx.work.Constraints
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.NetworkType
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import com.youkong.core.agent.worker.StatusReportTrigger
import com.youkong.core.agent.worker.StatusReportWorker
import java.util.concurrent.TimeUnit
import coil.ImageLoader
import coil.ImageLoaderFactory
import com.youkong.app.push.TPNSHelper
import com.youkong.core.datastore.TokenManager
import com.youkong.core.domain.manager.AppNotificationManager
import com.youkong.core.domain.manager.UnreadMessageManager
import com.youkong.core.permission.PermissionManager
import com.youkong.core.websocket.WebSocketManager
import dagger.hilt.android.HiltAndroidApp
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient
import javax.inject.Inject

@HiltAndroidApp
class YouKongApplication : Application(), Configuration.Provider, ImageLoaderFactory {

    @Inject
    lateinit var workerFactory: HiltWorkerFactory

    @Inject
    lateinit var permissionManager: PermissionManager

    @Inject
    lateinit var okHttpClient: OkHttpClient

    @Inject
    lateinit var tpnsHelper: TPNSHelper

    @Inject
    lateinit var webSocketManager: WebSocketManager

    @Inject
    lateinit var tokenManager: TokenManager

    @Inject
    lateinit var userPreferences: com.youkong.core.datastore.UserPreferences

    @Inject
    lateinit var unreadPreferences: com.youkong.core.datastore.UnreadPreferences

    @Inject
    lateinit var workManager: WorkManager

    @Inject
    lateinit var statusReportTrigger: StatusReportTrigger

    private val applicationScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

    override val workManagerConfiguration: Configuration
        get() = Configuration.Builder()
            .setWorkerFactory(workerFactory)
            .build()

    override fun newImageLoader(): ImageLoader {
        return ImageLoader.Builder(this)
            .okHttpClient(okHttpClient)
            .crossfade(true)
            .build()
    }

    override fun onCreate() {
        super.onCreate()

        // 初始化 TPNS 推送
        tpnsHelper.init()

        // 只在主进程中初始化 WebSocket 和后台任务
        if (isMainProcess()) {
            android.util.Log.d("YouKongApplication", "Main process, initializing WebSocket and workers")

            // 初始化未读消息管理器（必须在主进程中初始化，支持持久化）
            applicationScope.launch {
                UnreadMessageManager.initialize(this@YouKongApplication, unreadPreferences)
            }

            // 监听登录状态，连接 WebSocket
            observeAuthState()

            // 后台定时上报传感器数据（登录时启动，登出时取消）
            scheduleStatusReportWorker()

            // 监听 WebSocket 消息，处理未读计数
            observeWebSocketMessages()

            // App 前台/后台切换时自动上报（60s cooldown，与 iOS 对齐）
            observeAppForeground()
        } else {
            android.util.Log.d("YouKongApplication", "Non-main process (${getCurrentProcessName()}), skipping WebSocket and workers")
        }
    }

    /**
     * 检查是否是主进程
     */
    private fun isMainProcess(): Boolean {
        val processName = getCurrentProcessName()
        return processName == packageName
    }

    /**
     * 获取当前进程名称
     */
    private fun getCurrentProcessName(): String {
        val activityManager = getSystemService(ACTIVITY_SERVICE) as android.app.ActivityManager
        return activityManager.runningAppProcesses
            ?.firstOrNull { it.pid == android.os.Process.myPid() }
            ?.processName
            ?: packageName
    }

    private fun observeAuthState() {
        applicationScope.launch {
            tokenManager.accessToken
                .distinctUntilChanged()
                .collect { token ->
                    android.util.Log.d("YouKongApplication", "Auth state changed, token: ${if (token != null) "<exists>" else "null"}")
                    if (token != null) {
                        // 已登录，连接 WebSocket
                        android.util.Log.d("YouKongApplication", "Calling webSocketManager.connect()")
                        webSocketManager.connect()
                    } else {
                        // 已登出，断开 WebSocket
                        android.util.Log.d("YouKongApplication", "Calling webSocketManager.disconnect()")
                        webSocketManager.disconnect()
                    }
                }
        }
    }

    private fun scheduleStatusReportWorker() {
        applicationScope.launch {
            tokenManager.isLoggedIn
                .distinctUntilChanged()
                .collect { isLoggedIn ->
                    if (isLoggedIn) {
                        val constraints = Constraints.Builder()
                            .setRequiredNetworkType(NetworkType.CONNECTED)
                            .build()
                        val workRequest = PeriodicWorkRequestBuilder<StatusReportWorker>(
                            15, TimeUnit.MINUTES
                        ).setConstraints(constraints).build()
                        workManager.enqueueUniquePeriodicWork(
                            StatusReportWorker.WORK_NAME,
                            ExistingPeriodicWorkPolicy.KEEP,
                            workRequest,
                        )
                        // 登录时立即触发一次上报
                        statusReportTrigger.triggerNow()
                        Log.d("YouKongApplication", "StatusReportWorker scheduled (15min) + immediate")
                    } else {
                        workManager.cancelUniqueWork(StatusReportWorker.WORK_NAME)
                        Log.d("YouKongApplication", "StatusReportWorker cancelled")
                    }
                }
        }
    }

    /**
     * 监听 App 前台/后台切换，自动上报状态（60s cooldown）
     * 对齐 iOS RootView.onChange(of: scenePhase)
     */
    private fun observeAppForeground() {
        ProcessLifecycleOwner.get().lifecycle.addObserver(object : DefaultLifecycleObserver {
            override fun onStart(owner: LifecycleOwner) {
                // App 进入前台
                applicationScope.launch {
                    val isLoggedIn = tokenManager.isLoggedIn.first()
                    if (isLoggedIn) {
                        statusReportTrigger.triggerIfNeeded()
                        Log.d("YouKongApplication", "App foreground → triggerIfNeeded")
                    }
                }
            }

            override fun onStop(owner: LifecycleOwner) {
                // App 进入后台（与 iOS 一致，后台也触发一次）
                applicationScope.launch {
                    val isLoggedIn = tokenManager.isLoggedIn.first()
                    if (isLoggedIn) {
                        statusReportTrigger.triggerIfNeeded()
                        Log.d("YouKongApplication", "App background → triggerIfNeeded")
                    }
                }
            }
        })
    }

    /**
     * 监听 WebSocket 消息，处理未读计数
     * 参考需求文档第二章节：增加未读的时机
     */
    private fun observeWebSocketMessages() {
        applicationScope.launch {
            webSocketManager.newMessages.collect { newMessageData ->
                val conversationId = newMessageData.conversationId
                val message = newMessageData.message

                android.util.Log.d("YouKongApplication", "🔔 [WebSocket] 收到新消息")
                android.util.Log.d("YouKongApplication", "🔔 发送者ID: ${message.sender.id}")
                android.util.Log.d("YouKongApplication", "🔔 会话ID: $conversationId")

                // 获取当前用户 ID
                val currentUserId = userPreferences.userId.first()
                android.util.Log.d("YouKongApplication", "🔔 当前用户ID: $currentUserId")

                // 条件 1：不是自己发的消息
                if (message.sender.id == currentUserId) {
                    android.util.Log.d("YouKongApplication", "🔔 ⚠️ 是自己发的消息，跳过未读计数")
                    return@collect
                }

                // 条件 2：不在当前聊天页面
                val currentConversationId = AppNotificationManager.currentConversationId
                android.util.Log.d("YouKongApplication", "🔔 当前聊天页面会话ID: ${currentConversationId ?: "nil"}")

                if (currentConversationId == conversationId) {
                    android.util.Log.d("YouKongApplication", "🔔 ⚠️ 当前正在此聊天页面，跳过未读计数")
                    return@collect
                }

                // 满足条件，增加未读
                android.util.Log.d("YouKongApplication", "🔔 ✅ 增加会话 $conversationId 的未读计数")
                UnreadMessageManager.incrementUnread(conversationId)
            }
        }
    }
}
