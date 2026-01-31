package com.youkong.core.domain.manager

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.pm.ShortcutManager
import android.os.Build
import android.service.notification.StatusBarNotification
import android.util.Log
import androidx.annotation.RequiresApi

/**
 * Badge Manager - 管理应用图标角标
 *
 * 使用 Android 原生 Badge API（不需要通知权限）：
 * - Android 8.0+: 使用 NotificationManager.notify() + setNumber()
 * - Android 13+: 优先使用 NotificationManager.setNotificationCount()
 *
 * 注意：部分设备（小米、华为等）需要用户手动开启角标权限
 */
object BadgeManager {
    private const val TAG = "BadgeManager"
    private const val CHANNEL_ID = "youkong_badge"
    private const val CHANNEL_NAME = "未读消息"
    private const val NOTIFICATION_ID = 999

    @Volatile
    private var isInitialized = false

    /**
     * 初始化（应用启动时调用一次）
     */
    fun initialize(context: Context) {
        if (isInitialized) return

        try {
            // Android 8.0+ 需要创建通知通道
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                createNotificationChannel(context)
            }

            isInitialized = true
            Log.d(TAG, "📱 [BadgeManager] 初始化成功")
        } catch (e: Exception) {
            Log.e(TAG, "📱 [BadgeManager] 初始化失败", e)
        }
    }

    @RequiresApi(Build.VERSION_CODES.O)
    private fun createNotificationChannel(context: Context) {
        val notificationManager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        val channel = NotificationChannel(
            CHANNEL_ID,
            CHANNEL_NAME,
            NotificationManager.IMPORTANCE_MIN // 最低优先级（不会弹出、不会发声）
        ).apply {
            setShowBadge(true) // 启用角标
            enableVibration(false)
            enableLights(false)
            setSound(null, null)
            description = "用于显示未读消息数量"
        }
        notificationManager.createNotificationChannel(channel)
        Log.d(TAG, "📱 [BadgeManager] 通知通道创建成功")
    }

    /**
     * 更新应用角标
     * @param context 上下文
     * @param count 未读数（0 会清除角标）
     */
    fun updateBadge(context: Context, count: Int) {
        if (!isInitialized) {
            Log.w(TAG, "📱 [BadgeManager] 未初始化，跳过更新")
            return
        }

        try {
            when {
                Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
                    // Android 12+ (API 31+): 使用 Notification Badge API
                    updateBadgeApi31(context, count)
                }
                Build.VERSION.SDK_INT >= Build.VERSION_CODES.O -> {
                    // Android 8.0+ (API 26+): 使用通知 + setNumber
                    updateBadgeApi26(context, count)
                }
                else -> {
                    Log.w(TAG, "📱 [BadgeManager] Android 版本过低，不支持 Badge")
                }
            }
        } catch (e: Exception) {
            Log.e(TAG, "📱 [BadgeManager] 更新角标失败", e)
        }
    }

    /**
     * Android 12+ 使用 Notification Badge API
     */
    @RequiresApi(Build.VERSION_CODES.S)
    private fun updateBadgeApi31(context: Context, count: Int) {
        val notificationManager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

        if (count <= 0) {
            // 清除所有通知和角标
            notificationManager.cancelAll()
            Log.d(TAG, "📱 [BadgeManager] 清除角标 (API 31+)")
        } else {
            // 创建一个最小化的通知（只为了显示角标）
            val notification = Notification.Builder(context, CHANNEL_ID)
                .setSmallIcon(android.R.drawable.ic_dialog_email)
                .setContentTitle("有空")
                .setContentText("$count 条未读消息")
                .setNumber(count) // 设置角标数字
                .setOnlyAlertOnce(true)
                .setAutoCancel(false) // 不自动消失（保持角标）
                .build()

            notificationManager.notify(NOTIFICATION_ID, notification)
            Log.d(TAG, "📱 [BadgeManager] 更新角标: $count (API 31+)")
        }
    }

    /**
     * Android 8.0+ 使用通知 + setNumber
     */
    @RequiresApi(Build.VERSION_CODES.O)
    private fun updateBadgeApi26(context: Context, count: Int) {
        val notificationManager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

        if (count <= 0) {
            notificationManager.cancel(NOTIFICATION_ID)
            Log.d(TAG, "📱 [BadgeManager] 清除角标 (API 26+)")
        } else {
            val notification = Notification.Builder(context, CHANNEL_ID)
                .setSmallIcon(android.R.drawable.ic_dialog_email)
                .setContentTitle("有空")
                .setContentText("$count 条未读消息")
                .setNumber(count)
                .setOnlyAlertOnce(true)
                .setAutoCancel(false)
                .build()

            notificationManager.notify(NOTIFICATION_ID, notification)
            Log.d(TAG, "📱 [BadgeManager] 更新角标: $count (API 26+)")
        }
    }
}
