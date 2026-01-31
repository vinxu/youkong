package com.youkong.core.permission

import android.Manifest
import android.app.Activity
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.provider.Settings
import android.util.Log
import androidx.core.app.ActivityCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import javax.inject.Inject
import javax.inject.Singleton

/**
 * 通知权限管理器
 * 负责请求和检查通知权限状态
 */
@Singleton
class NotificationPermissionManager @Inject constructor() {

    companion object {
        private const val TAG = "NotificationPermission"
        const val REQUEST_CODE_POST_NOTIFICATIONS = 1001

        /**
         * 检查是否有通知权限
         */
        fun hasNotificationPermission(context: Context): Boolean {
            return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                // Android 13+: 检查 POST_NOTIFICATIONS 权限
                ContextCompat.checkSelfPermission(
                    context,
                    Manifest.permission.POST_NOTIFICATIONS
                ) == PackageManager.PERMISSION_GRANTED
            } else {
                // Android 13 以下: 检查通知是否启用
                NotificationManagerCompat.from(context).areNotificationsEnabled()
            }
        }
    }

    /**
     * 请求通知权限
     * @param activity 当前 Activity
     * @return true 如果已有权限或成功请求，false 如果需要引导用户去设置
     */
    fun requestNotificationPermission(activity: Activity): Boolean {
        if (hasNotificationPermission(activity)) {
            Log.d(TAG, "📱 已有通知权限")
            return true
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            // Android 13+: 请求运行时权限
            ActivityCompat.requestPermissions(
                activity,
                arrayOf(Manifest.permission.POST_NOTIFICATIONS),
                REQUEST_CODE_POST_NOTIFICATIONS
            )
            Log.d(TAG, "📱 请求通知权限（Android 13+）")
            return true
        } else {
            // Android 13 以下: 引导用户去系统设置
            Log.d(TAG, "📱 需要引导用户去系统设置开启通知")
            return false
        }
    }

    /**
     * 打开应用通知设置页面
     */
    fun openNotificationSettings(context: Context) {
        try {
            val intent = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                // Android 8.0+: 打开应用通知设置
                Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).apply {
                    putExtra(Settings.EXTRA_APP_PACKAGE, context.packageName)
                }
            } else {
                // Android 8.0 以下: 打开应用详情页
                Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS).apply {
                    data = Uri.fromParts("package", context.packageName, null)
                }
            }
            intent.flags = Intent.FLAG_ACTIVITY_NEW_TASK
            context.startActivity(intent)
            Log.d(TAG, "📱 打开通知设置页面")
        } catch (e: Exception) {
            Log.e(TAG, "📱 打开通知设置失败", e)
        }
    }

    /**
     * 是否应该显示权限说明（用户之前拒绝过）
     */
    fun shouldShowRequestPermissionRationale(activity: Activity): Boolean {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            ActivityCompat.shouldShowRequestPermissionRationale(
                activity,
                Manifest.permission.POST_NOTIFICATIONS
            )
        } else {
            false
        }
    }
}
