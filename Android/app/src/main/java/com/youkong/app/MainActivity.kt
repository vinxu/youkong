package com.youkong.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import androidx.work.WorkManager
import com.youkong.app.navigation.YouKongNavHost
import com.youkong.core.agent.worker.DataCollectWorker
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

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)

        enableEdgeToEdge()

        setContent {
            YouKongTheme {
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
