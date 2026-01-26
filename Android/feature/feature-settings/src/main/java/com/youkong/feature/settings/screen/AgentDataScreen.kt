package com.youkong.feature.settings.screen

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.Success
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.feature.settings.viewmodel.AgentDataViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AgentDataScreen(
    onBackClick: () -> Unit,
    viewModel: AgentDataViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("我的 Agent 数据") },
                navigationIcon = {
                    IconButton(onClick = onBackClick) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "返回",
                        )
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.refresh() }) {
                        Icon(
                            imageVector = Icons.Default.Refresh,
                            contentDescription = "刷新",
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                ),
            )
        },
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(16.dp)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            // 权限状态
            DataCard(
                title = "权限状态",
                items = listOf(
                    "屏幕使用时间" to if (uiState.hasUsageStatsPermission) "已授权" else "未授权",
                    "位置权限" to if (uiState.hasLocationPermission) "已授权" else "未授权",
                    "通讯录权限" to if (uiState.hasContactsPermission) "已授权" else "未授权",
                ),
            )

            // 屏幕使用数据
            DataCard(
                title = "屏幕使用数据",
                items = listOf(
                    "屏幕状态" to (uiState.screenData?.let { if (it.isScreenOn) "亮屏" else "息屏" } ?: "无数据"),
                    "当前应用" to (uiState.screenData?.currentApp ?: "无"),
                    "上次活跃" to (uiState.screenData?.lastActiveTime?.toString() ?: "无"),
                    "今日使用时长" to (uiState.screenData?.let { "${it.totalScreenTimeToday / 60000} 分钟" } ?: "无数据"),
                ),
            )

            // 位置数据
            DataCard(
                title = "位置数据",
                items = listOf(
                    "纬度" to (uiState.locationData?.latitude?.toString() ?: "无数据"),
                    "经度" to (uiState.locationData?.longitude?.toString() ?: "无数据"),
                    "精度" to (uiState.locationData?.accuracy?.let { "${it}米" } ?: "无数据"),
                    "获取时间" to (uiState.locationData?.timestamp?.toString() ?: "无数据"),
                ),
            )

            // 设备状态数据
            DataCard(
                title = "设备状态",
                items = listOf(
                    "勿扰模式" to (uiState.deviceStateData?.let { if (it.isDoNotDisturbEnabled) "开启 (忙碌)" else "关闭" } ?: "无数据"),
                    "充电状态" to (uiState.deviceStateData?.let { if (it.isCharging) "充电中" else "未充电" } ?: "无数据"),
                    "电池电量" to (uiState.deviceStateData?.let { "${it.batteryLevel}%" } ?: "无数据"),
                    "省电模式" to (uiState.deviceStateData?.let { if (it.isPowerSaveMode) "开启 (可能在外)" else "关闭" } ?: "无数据"),
                    "耳机连接" to (uiState.deviceStateData?.let { if (it.isHeadphonesConnected) "已连接" else "未连接" } ?: "无数据"),
                    "网络类型" to (uiState.deviceStateData?.networkType?.let {
                        when (it.name) {
                            "WIFI" -> "WiFi (可能在室内)"
                            "CELLULAR" -> "蜂窝 (可能移动中)"
                            "NONE" -> "无网络"
                            else -> it.name
                        }
                    } ?: "无数据"),
                    "响铃模式" to (uiState.deviceStateData?.ringerMode?.let {
                        when (it) {
                            "silent" -> "静音"
                            "vibrate" -> "振动"
                            "normal" -> "正常"
                            else -> it
                        }
                    } ?: "无数据"),
                ),
            )

            // 上报状态
            DataCard(
                title = "上报状态",
                items = listOf(
                    "上次上报时间" to (uiState.lastReportTime ?: "从未上报"),
                    "上报结果" to (uiState.lastReportResult ?: "无"),
                ),
            )

            // 调试信息
            if (uiState.debugInfo.isNotEmpty()) {
                DataCard(
                    title = "调试信息",
                    items = uiState.debugInfo.map { it.key to it.value },
                )
            }

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@Composable
private fun DataCard(
    title: String,
    items: List<Pair<String, String>>,
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surface,
        ),
        elevation = CardDefaults.cardElevation(defaultElevation = 1.dp),
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
        ) {
            Text(
                text = title,
                style = MaterialTheme.typography.titleMedium,
                color = Primary,
            )
            Spacer(modifier = Modifier.height(12.dp))
            items.forEach { (label, value) ->
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 4.dp),
                ) {
                    Text(
                        text = label,
                        style = MaterialTheme.typography.bodyMedium,
                        color = TextSecondary,
                        modifier = Modifier.weight(1f),
                    )
                    Text(
                        text = value,
                        style = MaterialTheme.typography.bodyMedium,
                        color = if (value == "已授权") Success else MaterialTheme.colorScheme.onSurface,
                    )
                }
            }
        }
    }
}
