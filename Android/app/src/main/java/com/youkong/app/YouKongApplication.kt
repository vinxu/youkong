package com.youkong.app

import android.app.Application
import androidx.hilt.work.HiltWorkerFactory
import androidx.work.Configuration
import androidx.work.WorkManager
import coil.ImageLoader
import coil.ImageLoaderFactory
import com.youkong.app.push.TPNSHelper
import com.youkong.core.agent.worker.DataCollectWorker
import com.youkong.core.agent.worker.StatusSyncWorker
import com.youkong.core.datastore.TokenManager
import com.youkong.core.permission.PermissionManager
import com.youkong.core.websocket.WebSocketManager
import dagger.hilt.android.HiltAndroidApp
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.distinctUntilChanged
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

        // 监听登录状态，连接 WebSocket
        observeAuthState()

        // 监听权限状态变化，自动调度后台任务
        observePermissionState()
    }

    private fun observeAuthState() {
        applicationScope.launch {
            tokenManager.accessToken
                .distinctUntilChanged()
                .collect { token ->
                    if (token != null) {
                        // 已登录，连接 WebSocket
                        webSocketManager.connect()
                    } else {
                        // 已登出，断开 WebSocket
                        webSocketManager.disconnect()
                    }
                }
        }
    }

    private fun observePermissionState() {
        applicationScope.launch {
            permissionManager.permissionState
                .map { it.allCorePermissionsGranted }
                .distinctUntilChanged()
                .collect { hasPermissions ->
                    val workManager = WorkManager.getInstance(this@YouKongApplication)
                    if (hasPermissions) {
                        // 权限已授予，启动数据收集任务
                        DataCollectWorker.schedule(workManager)
                        StatusSyncWorker.schedule(workManager)
                        // 立即执行一次数据收集
                        DataCollectWorker.runOnce(workManager)
                    } else {
                        // 权限被撤销，取消任务
                        DataCollectWorker.cancel(workManager)
                        StatusSyncWorker.cancel(workManager)
                    }
                }
        }
    }
}
