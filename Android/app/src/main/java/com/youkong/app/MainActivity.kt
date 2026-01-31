package com.youkong.app

import android.content.pm.PackageManager
import android.graphics.Color
import android.os.Build
import android.os.Bundle
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.SideEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.toArgb
import androidx.core.view.WindowCompat
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import androidx.work.WorkManager
import com.youkong.app.navigation.YouKongNavHost
import com.youkong.core.agent.worker.DataCollectWorker
import com.youkong.core.permission.NotificationPermissionManager
import com.youkong.core.permission.PermissionManager
import com.youkong.core.ui.theme.Background
import com.youkong.core.ui.theme.YouKongTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject
    lateinit var workManager: WorkManager

    @Inject
    lateinit var permissionManager: PermissionManager

    @Inject
    lateinit var notificationPermissionManager: NotificationPermissionManager

    private var hasRequestedNotificationPermission = false

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)

        // 设置系统栏为黑色终端风格
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.dark(
                scrim = Color.parseColor("#FF0D1117")
            ),
            navigationBarStyle = SystemBarStyle.dark(
                scrim = Color.parseColor("#FF0D1117")
            )
        )

        // 强制设置窗口颜色（确保生效）
        window.statusBarColor = Color.parseColor("#0D1117")
        window.navigationBarColor = Color.parseColor("#0D1117")

        setContent {
            YouKongTheme {
                // 强制设置系统栏颜色（每次重组时执行）
                SideEffect {
                    val darkColor = Color.parseColor("#0D1117")
                    window.statusBarColor = darkColor
                    window.navigationBarColor = darkColor

                    WindowCompat.getInsetsController(window, window.decorView).apply {
                        isAppearanceLightStatusBars = false
                        isAppearanceLightNavigationBars = false
                    }
                }

                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = Background,
                ) {
                    YouKongNavHost()
                }
            }
        }
    }

    override fun onResume() {
        super.onResume()
        // 每次应用回到前台时收集数据
        collectDataIfPermitted()

        // 首次请求通知权限（延迟执行，避免打扰用户）
        if (!hasRequestedNotificationPermission) {
            hasRequestedNotificationPermission = true
            window.decorView.postDelayed({
                requestNotificationPermissionIfNeeded()
            }, 2000) // 延迟 2 秒
        }
    }

    /**
     * 请求通知权限（如果需要）
     */
    private fun requestNotificationPermissionIfNeeded() {
        if (!NotificationPermissionManager.hasNotificationPermission(this)) {
            Log.d("MainActivity", "📱 请求通知权限")
            notificationPermissionManager.requestNotificationPermission(this)
        }
    }

    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray
    ) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)

        if (requestCode == NotificationPermissionManager.REQUEST_CODE_POST_NOTIFICATIONS) {
            if (grantResults.isNotEmpty() && grantResults[0] == PackageManager.PERMISSION_GRANTED) {
                Log.d("MainActivity", "📱 通知权限已授予")
            } else {
                Log.d("MainActivity", "📱 通知权限被拒绝")
                // 可以在这里提示用户去设置中开启
            }
        }
    }

    private fun collectDataIfPermitted() {
        // 检查是否有必要的权限，有任何一个权限就收集数据
        val permState = permissionManager.permissionState.value
        if (permState.hasUsageStatsPermission ||
            permState.hasLocationPermission ||
            permState.hasContactsPermission
        ) {
            DataCollectWorker.runOnce(workManager)
        }
    }
}
