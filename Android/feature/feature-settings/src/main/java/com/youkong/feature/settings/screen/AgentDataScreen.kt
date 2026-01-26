package com.youkong.feature.settings.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.youkong.core.agent.model.AppUsageInfo
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
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
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues),
        ) {
            LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                // LLM 分析结果（最重要，放在最上面）
                item {
                    uiState.analysisResult?.let { analysis ->
                        AnalysisResultCard(
                            emoji = analysis.lifeStatus.emoji,
                            label = analysis.lifeStatus.label,
                            probability = analysis.availability.probability,
                            status = analysis.availability.status,
                            reason = analysis.availability.reason,
                            confidence = analysis.availability.confidence,
                        )
                    } ?: run {
                        // 无分析结果时显示提示
                        Card(
                            modifier = Modifier.fillMaxWidth(),
                            colors = CardDefaults.cardColors(
                                containerColor = MaterialTheme.colorScheme.surfaceVariant,
                            ),
                        ) {
                            Column(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(24.dp),
                                horizontalAlignment = Alignment.CenterHorizontally,
                            ) {
                                Text(
                                    text = "🤔",
                                    fontSize = 48.sp,
                                )
                                Spacer(modifier = Modifier.height(8.dp))
                                Text(
                                    text = "下拉刷新获取 LLM 分析结果",
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = TextSecondary,
                                )
                            }
                        }
                    }
                }

                // 权限状态
                item {
                    DataCard(
                        title = "权限状态",
                        items = listOf(
                            "屏幕使用时间" to if (uiState.hasUsageStatsPermission) "已授权" else "未授权",
                            "位置权限" to if (uiState.hasLocationPermission) "已授权" else "未授权",
                            "通讯录权限" to if (uiState.hasContactsPermission) "已授权" else "未授权",
                        ),
                    )
                }

                // 屏幕使用数据
                item {
                    DataCard(
                        title = "屏幕使用数据",
                        items = listOf(
                            "屏幕状态" to (uiState.screenData?.let { if (it.isScreenOn) "亮屏" else "息屏" } ?: "无数据"),
                            "当前应用" to (uiState.screenData?.currentApp ?: "无"),
                            "上次活跃" to (uiState.screenData?.lastActiveTime?.toString() ?: "无"),
                            "今日使用时长" to (uiState.screenData?.let { "${it.totalScreenTimeToday / 60000} 分钟" } ?: "无数据"),
                        ),
                    )
                }

                // 今日 App 使用统计
                uiState.screenData?.appUsageList?.takeIf { it.isNotEmpty() }?.let { appList ->
                    item {
                        AppUsageCard(
                            title = "今日 App 使用",
                            appList = appList,
                        )
                    }
                }

                // 位置数据
                item {
                    DataCard(
                        title = "位置数据",
                        items = listOf(
                            "纬度" to (uiState.locationData?.latitude?.toString() ?: "无数据"),
                            "经度" to (uiState.locationData?.longitude?.toString() ?: "无数据"),
                            "精度" to (uiState.locationData?.accuracy?.let { "${it}米" } ?: "无数据"),
                            "获取时间" to (uiState.locationData?.timestamp?.toString() ?: "无数据"),
                        ),
                    )
                }

                // 设备状态数据
                item {
                    DataCard(
                        title = "设备状态",
                        items = listOf(
                            "勿扰模式" to (uiState.deviceStateData?.let { if (it.isDoNotDisturbEnabled) "开启" else "关闭" } ?: "无数据"),
                            "充电状态" to (uiState.deviceStateData?.let { if (it.isCharging) "充电中" else "未充电" } ?: "无数据"),
                            "电池电量" to (uiState.deviceStateData?.let { "${it.batteryLevel}%" } ?: "无数据"),
                            "省电模式" to (uiState.deviceStateData?.let { if (it.isPowerSaveMode) "开启" else "关闭" } ?: "无数据"),
                            "耳机连接" to (uiState.deviceStateData?.let { if (it.isHeadphonesConnected) "已连接" else "未连接" } ?: "无数据"),
                            "网络类型" to (uiState.deviceStateData?.networkType?.name ?: "无数据"),
                            "响铃模式" to (uiState.deviceStateData?.ringerMode ?: "无数据"),
                            "屏幕亮度" to (uiState.deviceStateData?.let { "${(it.screenBrightness * 100).toInt()}%" } ?: "无数据"),
                        ),
                    )
                }

                // 上报状态
                item {
                    DataCard(
                        title = "上报状态",
                        items = listOf(
                            "上次上报时间" to (uiState.lastReportTime ?: "从未上报"),
                            "上报结果" to (uiState.lastReportResult ?: "无"),
                        ),
                    )
                }

                // 调试信息
                if (uiState.debugInfo.isNotEmpty()) {
                    item {
                        DataCard(
                            title = "调试信息",
                            items = uiState.debugInfo.map { it.key to it.value },
                        )
                    }
                }

                item {
                    Spacer(modifier = Modifier.height(32.dp))
                }
            }

            // 加载指示器
            if (uiState.isLoading) {
                CircularProgressIndicator(
                    modifier = Modifier
                        .align(Alignment.TopCenter)
                        .padding(top = 16.dp),
                )
            }
        }
    }
}

@Composable
private fun AnalysisResultCard(
    emoji: String,
    label: String,
    probability: Int,
    status: String,
    reason: String,
    confidence: String,
) {
    val probabilityColor = when {
        probability >= 80 -> Color(0xFF22C55E) // 绿色
        probability >= 60 -> Color(0xFF86EFAC) // 浅绿
        probability >= 40 -> Color(0xFFFBBF24) // 黄色
        probability >= 20 -> Color(0xFFF97316) // 橙色
        else -> Color(0xFFEF4444) // 红色
    }

    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surface,
        ),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(20.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = "LLM 分析结果",
                style = MaterialTheme.typography.titleMedium,
                color = Primary,
            )

            Spacer(modifier = Modifier.height(16.dp))

            // Emoji + 状态标签
            Text(
                text = emoji,
                fontSize = 64.sp,
            )

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = label,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
            )

            Spacer(modifier = Modifier.height(16.dp))

            // 有空概率
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.Center,
            ) {
                Text(
                    text = "有空概率：",
                    style = MaterialTheme.typography.bodyLarge,
                    color = TextSecondary,
                )
                Box(
                    modifier = Modifier
                        .background(
                            color = probabilityColor,
                            shape = RoundedCornerShape(8.dp)
                        )
                        .padding(horizontal = 12.dp, vertical = 4.dp)
                ) {
                    Text(
                        text = "$probability%",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = Color.White,
                    )
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            // 状态和原因
            Text(
                text = status,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
            )

            Spacer(modifier = Modifier.height(4.dp))

            Text(
                text = reason,
                style = MaterialTheme.typography.bodyMedium,
                color = TextSecondary,
                textAlign = TextAlign.Center,
            )

            Spacer(modifier = Modifier.height(8.dp))

            // 置信度
            Text(
                text = "置信度: $confidence",
                style = MaterialTheme.typography.bodySmall,
                color = TextSecondary,
            )
        }
    }
}

@Composable
private fun AppUsageCard(
    title: String,
    appList: List<AppUsageInfo>,
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
            appList.forEach { app ->
                // App 名称和总时长
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 4.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text(
                        text = app.appName,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onSurface,
                        modifier = Modifier.weight(1f),
                    )
                    Text(
                        text = "总计: ${formatDuration(app.usageTimeMillis)}",
                        style = MaterialTheme.typography.bodyMedium,
                        color = TextSecondary,
                    )
                }
                // 显示使用时段列表
                if (app.sessions.isNotEmpty()) {
                    app.sessions.forEachIndexed { index, session ->
                        val isLast = index == app.sessions.lastIndex
                        val prefix = if (isLast) "└── " else "├── "
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(start = 16.dp, top = 2.dp, bottom = 2.dp),
                        ) {
                            Text(
                                text = prefix,
                                style = MaterialTheme.typography.bodySmall,
                                color = TextSecondary,
                            )
                            Text(
                                text = "${formatTime(session.startTime)} - ${formatTime(session.endTime)}",
                                style = MaterialTheme.typography.bodySmall,
                                color = TextSecondary,
                            )
                            Text(
                                text = " (${formatDuration(session.durationMillis)})",
                                style = MaterialTheme.typography.bodySmall,
                                color = TextSecondary.copy(alpha = 0.7f),
                            )
                        }
                    }
                }
                Spacer(modifier = Modifier.height(8.dp))
            }
        }
    }
}

private fun formatDuration(millis: Long): String {
    val totalMinutes = millis / 60000
    val hours = totalMinutes / 60
    val minutes = totalMinutes % 60
    return if (hours > 0) {
        "${hours}小时${minutes}分钟"
    } else {
        "${minutes}分钟"
    }
}

private fun formatTime(timestamp: Long): String {
    val sdf = SimpleDateFormat("HH:mm", Locale.getDefault())
    return sdf.format(Date(timestamp))
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
