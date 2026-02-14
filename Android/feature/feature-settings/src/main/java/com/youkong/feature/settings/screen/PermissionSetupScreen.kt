package com.youkong.feature.settings.screen

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.DateRange
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.ui.unit.sp
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.youkong.core.permission.RequiredPermission
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.Success
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.feature.settings.viewmodel.PermissionSetupViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PermissionSetupScreen(
    onBackClick: () -> Unit,
    onComplete: () -> Unit,
    isOnboarding: Boolean = false,
    viewModel: PermissionSetupViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val currentPermission = viewModel.getCurrentPermission()

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

    // 完成时回调
    LaunchedEffect(uiState.isComplete) {
        if (uiState.isComplete) {
            onComplete()
        }
    }

    // 位置权限请求（多个权限）
    val locationPermissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestMultiplePermissions()
    ) { _ ->
        viewModel.refreshPermissions()
    }

    // 运动数据权限请求
    val activityRecognitionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission()
    ) { _ ->
        viewModel.refreshPermissions()
    }

    // 日历权限请求
    val calendarPermissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission()
    ) { _ ->
        viewModel.refreshPermissions()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(if (isOnboarding) "开始使用" else "权限设置") },
                navigationIcon = {
                    if (!isOnboarding) {
                        IconButton(onClick = onBackClick) {
                            Icon(
                                imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                                contentDescription = "返回",
                            )
                        }
                    }
                },
                actions = {
                    if (!isOnboarding) {
                        TextButton(onClick = { viewModel.skipToComplete() }) {
                            Text("跳过")
                        }
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
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            if (currentPermission == null || uiState.isComplete) {
                // 所有权限已授予或已跳过
                PermissionCompleteContent(
                    onContinue = onComplete,
                )
            } else {
                // 显示当前需要的权限
                PermissionRequestContent(
                    permission = currentPermission,
                    onRequest = {
                        when (currentPermission) {
                            RequiredPermission.LOCATION -> {
                                locationPermissionLauncher.launch(viewModel.getLocationPermissions())
                            }
                            RequiredPermission.ACTIVITY_RECOGNITION -> {
                                viewModel.getActivityRecognitionPermission()?.let {
                                    activityRecognitionLauncher.launch(it)
                                } ?: viewModel.refreshPermissions()
                            }
                            RequiredPermission.CALENDAR -> {
                                calendarPermissionLauncher.launch(viewModel.getCalendarPermission())
                            }
                        }
                    },
                    onSkip = { viewModel.skipCurrentPermission() },
                    isOnboarding = isOnboarding,
                )
            }
        }
    }
}

@Composable
private fun PermissionRequestContent(
    permission: RequiredPermission,
    onRequest: () -> Unit,
    onSkip: () -> Unit,
    isOnboarding: Boolean = false,
) {
    val emoji = when (permission) {
        RequiredPermission.LOCATION -> "📍"
        RequiredPermission.ACTIVITY_RECOGNITION -> "🏃"
        RequiredPermission.CALENDAR -> "📅"
    }

    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = emoji,
            fontSize = 64.sp,
        )

        Spacer(modifier = Modifier.height(24.dp))

        Text(
            text = permission.title,
            style = MaterialTheme.typography.headlineMedium,
            textAlign = TextAlign.Center,
        )

        Spacer(modifier = Modifier.height(12.dp))

        Text(
            text = permission.description,
            style = MaterialTheme.typography.bodyLarge,
            color = TextSecondary,
            textAlign = TextAlign.Center,
        )

        Spacer(modifier = Modifier.height(16.dp))

        // 详细说明为什么需要这个权限
        Text(
            text = permission.whyNeed,
            style = MaterialTheme.typography.bodyMedium,
            color = TextSecondary,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(horizontal = 16.dp),
        )

        Spacer(modifier = Modifier.height(24.dp))

        // 数据安全提示
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                imageVector = Icons.Default.CheckCircle,
                contentDescription = null,
                modifier = Modifier.size(16.dp),
                tint = Success,
            )
            Text(
                text = " 数据仅在本地处理，不会上传到服务器",
                style = MaterialTheme.typography.bodySmall,
                color = Success,
            )
        }

        Spacer(modifier = Modifier.height(32.dp))

        Button(
            onClick = onRequest,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(text = "授予权限")
        }

        if (!isOnboarding) {
            Spacer(modifier = Modifier.height(12.dp))

            OutlinedButton(
                onClick = onSkip,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text("稍后再说")
            }
        }
    }
}

@Composable
private fun PermissionCompleteContent(
    onContinue: () -> Unit,
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(
            imageVector = Icons.Default.CheckCircle,
            contentDescription = null,
            modifier = Modifier.size(80.dp),
            tint = Success,
        )

        Spacer(modifier = Modifier.height(24.dp))

        Text(
            text = "权限设置完成",
            style = MaterialTheme.typography.headlineMedium,
            textAlign = TextAlign.Center,
        )

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = "所有必需权限已授予，您可以开始使用了",
            style = MaterialTheme.typography.bodyLarge,
            color = TextSecondary,
            textAlign = TextAlign.Center,
        )

        Spacer(modifier = Modifier.height(48.dp))

        Button(
            onClick = onContinue,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text("开始使用")
        }
    }
}
