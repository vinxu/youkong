package com.youkong.feature.home.screen

import android.Manifest
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.CLIColors
import kotlinx.coroutines.launch

/**
 * CLI 风格引导页（对齐 iOS：3 屏 + Chat 覆盖层）
 *
 * 0: Welcome
 * 1: PermissionRequest
 * 2: OnboardingProfilePage（渐进式滚动：Gender → Profile → MBTI → 传感器 → 状态）
 *
 * ProfilePage 完成后 → Chat 覆盖层
 * Chat 完成/跳过 → onComplete
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
fun OnboardingScreen(
    onComplete: () -> Unit
) {
    val pagerState = rememberPagerState(pageCount = { 3 })
    val scope = rememberCoroutineScope()

    // Chat overlay state (matches iOS's onboardingWowPhase pattern)
    var showChat by remember { mutableStateOf(false) }

    if (showChat) {
        // Wow Chat (full screen overlay, like iOS)
        OnboardingChatScreen(
            onComplete = onComplete,
            onSkip = onComplete
        )
    } else {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(CLIColors.Background)
        ) {
            HorizontalPager(
                state = pagerState,
                modifier = Modifier.fillMaxSize(),
                userScrollEnabled = false
            ) { page ->
                when (page) {
                    0 -> WelcomeScreen(
                        onNext = { scope.launch { pagerState.animateScrollToPage(1) } }
                    )
                    1 -> PermissionRequestScreen(
                        onComplete = { scope.launch { pagerState.animateScrollToPage(2) } }
                    )
                    2 -> OnboardingProfilePage(
                        onConfirm = { _, _ ->
                            showChat = true
                        },
                        onSkip = {
                            showChat = true
                        }
                    )
                }
            }

            // Page indicator (hidden on profile page)
            if (pagerState.currentPage < 2) {
                CLIPageIndicator(
                    currentPage = pagerState.currentPage,
                    totalPages = 3,
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .padding(bottom = 50.dp)
                )
            }
        }
    }
}

// MARK: - CLI Page Indicator

@Composable
private fun CLIPageIndicator(
    currentPage: Int,
    totalPages: Int,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically
    ) {
        repeat(totalPages) { index ->
            Text(
                text = if (index == currentPage) "●" else "○",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = if (index == currentPage) CLIColors.Green else CLIColors.TextWeak
            )
            if (index < totalPages - 1) {
                Spacer(modifier = Modifier.width(8.dp))
            }
        }
    }
}

// MARK: - Welcome Screen (第1屏)

@Composable
private fun WelcomeScreen(onNext: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Spacer(modifier = Modifier.weight(1f))

        // Logo
        Text(
            text = "有 空",
            fontFamily = FontFamily.Monospace,
            fontSize = 56.sp,
            fontWeight = FontWeight.Bold,
            color = CLIColors.Green
        )

        Spacer(modifier = Modifier.height(16.dp))

        // Slogan
        Text(
            text = "一眼谁有空，一句话约人",
            fontFamily = FontFamily.Monospace,
            fontSize = 16.sp,
            color = CLIColors.TextSecondary
        )

        Spacer(modifier = Modifier.weight(1f))

        // 开始按钮
        TextButton(
            onClick = onNext,
            modifier = Modifier
                .fillMaxWidth()
                .height(50.dp)
                .background(CLIColors.Green)
        ) {
            Text(
                text = "> 开始了解",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.Background
            )
        }

        Spacer(modifier = Modifier.height(100.dp))
    }
}

// MARK: - Permission Request Screen (第2屏)

@Composable
private fun PermissionRequestScreen(
    onComplete: () -> Unit
) {
    val allPermissionsLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestMultiplePermissions()
    ) { _ -> }

    val notificationPermissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission()
    ) { _ -> }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Spacer(modifier = Modifier.height(60.dp))

        Text(text = "🤖", fontSize = 48.sp)
        Spacer(modifier = Modifier.height(12.dp))
        Text(
            text = "━━━ 权限请求 ━━━",
            fontFamily = FontFamily.Monospace,
            fontSize = 18.sp,
            color = CLIColors.Cyan
        )

        Spacer(modifier = Modifier.height(24.dp))

        // Agent 对话框
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .border(1.dp, CLIColors.Border)
                .padding(16.dp)
        ) {
            Text(
                text = "> Agent 说：",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                fontWeight = FontWeight.Bold,
                color = CLIColors.Green
            )
            Spacer(modifier = Modifier.height(12.dp))
            Text(
                text = "\"为了更懂你，我需要一些帮助：",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextPrimary
            )
            Spacer(modifier = Modifier.height(8.dp))
            listOf(
                "1️⃣ 位置权限 - 判断你在家/公司/路上",
                "2️⃣ 日历权限 - 知道你现在有没有会议",
                "3️⃣ 运动权限 - 了解你是在运动还是静止"
            ).forEach { line ->
                Text(
                    text = line,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary
                )
                Spacer(modifier = Modifier.height(4.dp))
            }
            Text(
                text = "这些都只用于推测状态，\n不会记录具体内容。\"",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextPrimary
            )
        }

        Spacer(modifier = Modifier.weight(1f))

        // 授权按钮
        TextButton(
            onClick = {
                val permissions = mutableListOf(
                    Manifest.permission.ACCESS_FINE_LOCATION,
                    Manifest.permission.ACCESS_COARSE_LOCATION,
                    Manifest.permission.READ_CALENDAR
                )
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                    permissions.add(Manifest.permission.ACTIVITY_RECOGNITION)
                }
                allPermissionsLauncher.launch(permissions.toTypedArray())
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                    notificationPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
                }
                onComplete()
            },
            modifier = Modifier
                .fillMaxWidth()
                .height(50.dp)
                .border(1.dp, CLIColors.Green)
        ) {
            Text(
                text = "[ 授权 ]",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.Green
            )
        }

        Spacer(modifier = Modifier.height(12.dp))

        TextButton(
            onClick = onComplete,
            modifier = Modifier.fillMaxWidth()
        ) {
            Text(
                text = "[ 稍后 ]",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary
            )
        }

        Spacer(modifier = Modifier.height(50.dp))
    }
}
