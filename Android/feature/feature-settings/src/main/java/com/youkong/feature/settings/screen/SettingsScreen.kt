package com.youkong.feature.settings.screen

import android.app.Activity
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.youkong.core.ui.component.cli.TerminalButton
import com.youkong.core.ui.component.cli.TerminalButtonStyle
import com.youkong.core.ui.component.cli.TerminalDivider
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.component.cli.TerminalSectionDivider
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.settings.viewmodel.SettingsViewModel

@Composable
fun SettingsScreen(
    onBackClick: () -> Unit,
    onPermissionSetupClick: () -> Unit,
    onAgentDataClick: () -> Unit = {},
    onInvitationClick: () -> Unit = {},
    viewModel: SettingsViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val context = LocalContext.current
    var showLogoutDialog by remember { mutableStateOf(false) }

    // 监听页面回来时刷新权限状态
    val lifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                viewModel.refreshPermissions()
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
            .statusBarsPadding()
    ) {
        // Terminal Header
        TerminalHeader(
            title = "$ settings --config",
            showBackButton = true,
            onBackClick = onBackClick
        )

        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .background(CLIColors.Background),
            verticalArrangement = Arrangement.spacedBy(0.dp),
        ) {
            // 权限状态
            item {
                PermissionStatusItem(
                    allPermissionsGranted = uiState.permissionState.allCorePermissionsGranted,
                    onClick = onPermissionSetupClick,
                )
            }

            item { TerminalDivider() }

            // 好友
            item {
                TerminalSectionDivider(
                    title = "FRIENDS",
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp)
                )
            }

            item {
                SettingsClickItem(
                    title = "邀请好友",
                    subtitle = "创建邀请链接，分享给朋友",
                    onClick = onInvitationClick,
                )
            }

            item { TerminalDivider() }

            // 隐私
            item {
                TerminalSectionDivider(
                    title = "PRIVACY",
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp)
                )
            }

            item {
                SettingsSwitchItem(
                    title = "数据收集",
                    subtitle = "允许收集屏幕使用和位置数据以预测有空状态",
                    checked = uiState.isDataCollectionEnabled,
                    onCheckedChange = { viewModel.setDataCollectionEnabled(it) },
                )
            }

            item { TerminalDivider() }

            item {
                SettingsClickItem(
                    title = "我的 Agent 数据",
                    subtitle = "查看当前收集到的数据",
                    onClick = onAgentDataClick,
                )
            }

            item { TerminalDivider() }

            // 系统
            item {
                TerminalSectionDivider(
                    title = "SYSTEM",
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp)
                )
            }

            item {
                SettingsClickItem(
                    title = "电池优化",
                    subtitle = "关闭电池优化以保证后台运行",
                    onClick = {
                        (context as? Activity)?.let {
                            viewModel.openBatteryOptimizationSettings(it)
                        }
                    },
                )
            }

            item { TerminalDivider() }

            // 账户
            item {
                TerminalSectionDivider(
                    title = "ACCOUNT",
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp)
                )
            }

            item {
                SettingsClickItem(
                    title = "退出登录",
                    subtitle = uiState.nickname,
                    onClick = { showLogoutDialog = true },
                    isDestructive = true,
                )
            }

            item { Spacer(modifier = Modifier.height(32.dp)) }
        }
    }

    // 退出登录确认对话框
    if (showLogoutDialog) {
        AlertDialog(
            onDismissRequest = { showLogoutDialog = false },
            title = {
                Text(
                    text = "$ logout --confirm",
                    fontFamily = FontFamily.Monospace,
                    color = CLIColors.TextPrimary
                )
            },
            text = {
                Text(
                    text = "Are you sure? [y/N]",
                    fontFamily = FontFamily.Monospace,
                    color = CLIColors.TextSecondary
                )
            },
            confirmButton = {
                TerminalButton(
                    text = "Y",
                    onClick = {
                        viewModel.logout()
                        showLogoutDialog = false
                    },
                    style = TerminalButtonStyle.DANGER
                )
            },
            dismissButton = {
                TerminalButton(
                    text = "N",
                    onClick = { showLogoutDialog = false },
                    style = TerminalButtonStyle.GHOST
                )
            },
            containerColor = CLIColors.Background
        )
    }
}

@Composable
private fun PermissionStatusItem(
    allPermissionsGranted: Boolean,
    onClick: () -> Unit,
) {
    val statusIcon = if (allPermissionsGranted) ASCII.CHECKMARK else ASCII.WARNING
    val statusColor = if (allPermissionsGranted) CLIColors.Green else CLIColors.Red
    val statusText = if (allPermissionsGranted) "GRANTED" else "REQUIRED"
    val subtitle = if (allPermissionsGranted) {
        "所有必需权限已授予"
    } else {
        "点击设置权限以启用完整功能"
    }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(CLIColors.Background)
            .clickable(onClick = onClick)
            .padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = statusIcon,
            fontFamily = FontFamily.Monospace,
            fontSize = 18.sp,
            color = statusColor,
        )

        Column(
            modifier = Modifier
                .weight(1f)
                .padding(horizontal = 12.dp),
        ) {
            Text(
                text = "权限状态: $statusText",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = statusColor,
            )
            Text(
                text = subtitle,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary,
            )
        }

        Text(
            text = ASCII.ARROW_RIGHT,
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = CLIColors.TextSecondary,
        )
    }
}

@Composable
private fun SettingsSwitchItem(
    title: String,
    subtitle: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(CLIColors.Background)
            .padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = title,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextPrimary,
            )
            Text(
                text = subtitle,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary,
            )
        }

        Text(
            text = if (checked) "[ON]" else "[OFF]",
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = if (checked) CLIColors.Green else CLIColors.TextSecondary,
            modifier = Modifier
                .padding(end = 8.dp)
                .clickable { onCheckedChange(!checked) }
        )
    }
}

@Composable
private fun SettingsClickItem(
    title: String,
    subtitle: String,
    onClick: () -> Unit,
    isDestructive: Boolean = false,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(CLIColors.Background)
            .clickable(onClick = onClick)
            .padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = title,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = if (isDestructive) CLIColors.Red else CLIColors.TextPrimary,
            )
            Text(
                text = subtitle,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary,
            )
        }

        Text(
            text = ASCII.ARROW_RIGHT,
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = if (isDestructive) CLIColors.Red else CLIColors.TextSecondary,
        )
    }
}
