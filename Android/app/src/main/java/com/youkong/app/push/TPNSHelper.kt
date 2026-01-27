package com.youkong.app.push

import android.content.Context
import android.os.Build
import android.util.Log
import com.tencent.android.tpush.XGIOperateCallback
import com.tencent.android.tpush.XGPushConfig
import com.tencent.android.tpush.XGPushManager
import com.youkong.app.BuildConfig
import com.youkong.core.domain.push.PushManager
import com.youkong.core.network.api.DeviceApi
import com.youkong.core.network.model.RegisterDeviceTokenRequest
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.suspendCancellableCoroutine
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.coroutines.resume

/**
 * 腾讯 TPNS 推送管理器
 */
@Singleton
class TPNSHelper @Inject constructor(
    @ApplicationContext private val context: Context,
    private val deviceApi: DeviceApi,
) : PushManager {

    companion object {
        private const val TAG = "TPNSHelper"
    }

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    /**
     * 初始化 TPNS 推送服务
     * 在 Application.onCreate() 中调用
     */
    fun init() {
        // 启用调试模式（仅 Debug 构建）
        XGPushConfig.enableDebug(context, BuildConfig.DEBUG)

        // 注册推送服务
        XGPushManager.registerPush(context, object : XGIOperateCallback {
            override fun onSuccess(data: Any?, flag: Int) {
                val token = data as? String
                Log.d(TAG, "TPNS 注册成功，token: $token")
                // 上报 Token 到后端
                if (!token.isNullOrEmpty()) {
                    scope.launch {
                        uploadToken(token)
                    }
                }
            }

            override fun onFail(data: Any?, errCode: Int, msg: String?) {
                Log.e(TAG, "TPNS 注册失败，errCode: $errCode, msg: $msg")
            }
        })
    }

    /**
     * 上报推送 Token 到后端服务器
     */
    private suspend fun uploadToken(token: String) {
        try {
            val request = RegisterDeviceTokenRequest(
                token = token,
                platform = "android",
                deviceName = "${Build.MANUFACTURER} ${Build.MODEL}",
                appVersion = BuildConfig.VERSION_NAME,
            )
            val response = deviceApi.registerToken(request)
            if (response.isSuccess) {
                Log.d(TAG, "Token 上报成功")
            } else {
                Log.e(TAG, "Token 上报失败: ${response.message}")
            }
        } catch (e: Exception) {
            Log.e(TAG, "Token 上报异常", e)
        }
    }

    /**
     * 绑定用户账号到推送服务
     * 登录成功后调用，用于接收定向推送
     */
    override suspend fun bindAccount(userId: String) {
        suspendCancellableCoroutine { continuation ->
            XGPushManager.bindAccount(context, userId, object : XGIOperateCallback {
                override fun onSuccess(data: Any?, flag: Int) {
                    Log.d(TAG, "绑定账号成功，userId: $userId")
                    if (continuation.isActive) {
                        continuation.resume(Unit)
                    }
                }

                override fun onFail(data: Any?, errCode: Int, msg: String?) {
                    Log.e(TAG, "绑定账号失败，errCode: $errCode, msg: $msg")
                    // 绑定失败不阻塞登录流程
                    if (continuation.isActive) {
                        continuation.resume(Unit)
                    }
                }
            })
        }
    }

    /**
     * 解绑用户账号
     * 退出登录时调用
     */
    override suspend fun unbindAccount() {
        suspendCancellableCoroutine { continuation ->
            XGPushManager.delAccount(context, null, object : XGIOperateCallback {
                override fun onSuccess(data: Any?, flag: Int) {
                    Log.d(TAG, "解绑账号成功")
                    if (continuation.isActive) {
                        continuation.resume(Unit)
                    }
                }

                override fun onFail(data: Any?, errCode: Int, msg: String?) {
                    Log.e(TAG, "解绑账号失败，errCode: $errCode, msg: $msg")
                    // 解绑失败不阻塞退出流程
                    if (continuation.isActive) {
                        continuation.resume(Unit)
                    }
                }
            })
        }
    }
}
