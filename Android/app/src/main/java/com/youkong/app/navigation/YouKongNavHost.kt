package com.youkong.app.navigation

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.youkong.app.debug.ApiDebugScreen
import com.youkong.feature.auth.navigation.AUTH_GRAPH_ROUTE
import com.youkong.feature.auth.navigation.authGraph
import com.youkong.feature.auth.navigation.navigateToPhoneInput
import com.youkong.feature.chat.navigation.chatScreen
import com.youkong.feature.chat.navigation.navigateToChat
import com.youkong.feature.friends.navigation.FRIENDS_ROUTE
import com.youkong.feature.friends.navigation.friendsScreen
import com.youkong.feature.friends.navigation.navigateToFriends
import com.youkong.feature.home.navigation.HOME_ROUTE
import com.youkong.feature.home.navigation.homeScreen
import com.youkong.feature.home.navigation.navigateToHome
import com.youkong.feature.message.navigation.messagesGraph
import com.youkong.feature.message.navigation.navigateToMessages
import com.youkong.feature.profile.navigation.navigateToProfile
import com.youkong.feature.profile.navigation.profileScreen
import com.youkong.feature.settings.navigation.ONBOARDING_PERMISSION_ROUTE
import com.youkong.feature.settings.navigation.navigateToOnboardingPermission
import com.youkong.feature.settings.navigation.navigateToSettings
import com.youkong.feature.settings.navigation.settingsGraph

const val DEBUG_ROUTE = "debug"

fun NavHostController.navigateToDebug() {
    navigate(DEBUG_ROUTE) {
        launchSingleTop = true
    }
}

@Composable
fun YouKongNavHost(
    navController: NavHostController = rememberNavController(),
    viewModel: MainViewModel = hiltViewModel(),
    modifier: Modifier = Modifier,
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    val startDestination = when {
        uiState.isLoading -> AUTH_GRAPH_ROUTE
        uiState.isLoggedIn -> {
            if (uiState.hasRequiredPermissions) FRIENDS_ROUTE else ONBOARDING_PERMISSION_ROUTE
        }
        else -> AUTH_GRAPH_ROUTE
    }

    // 当登录状态或权限状态变化时，自动导航
    LaunchedEffect(uiState.isLoggedIn, uiState.isLoading, uiState.hasRequiredPermissions) {
        if (!uiState.isLoading) {
            when {
                !uiState.isLoggedIn -> {
                    navController.navigateToPhoneInput()
                }
                !uiState.hasRequiredPermissions -> {
                    navController.navigateToOnboardingPermission()
                }
                else -> {
                    navController.navigateToFriends()
                }
            }
        }
    }

    NavHost(
        navController = navController,
        startDestination = startDestination,
        modifier = modifier,
    ) {
        // 认证模块
        authGraph(
            navController = navController,
            onLoginSuccess = {
                // 登录成功后，刷新权限状态
                viewModel.refreshPermissions()
                // 导航会由 LaunchedEffect 处理
            },
        )

        // 权限引导页面 (首次登录引导)
        composable(route = ONBOARDING_PERMISSION_ROUTE) {
            com.youkong.feature.settings.screen.PermissionSetupScreen(
                onBackClick = { /* 首次引导不允许返回 */ },
                onComplete = {
                    viewModel.refreshPermissions()
                    navController.navigateToFriends()
                },
                isOnboarding = true,
            )
        }

        // 好友列表 (主页面)
        friendsScreen(
            onNavigateToChat = { userId ->
                navController.navigateToChat(userId)
            },
            onNavigateToSettings = {
                navController.navigateToSettings()
            },
        )

        // 聊天
        chatScreen(navController = navController)

        // 设置
        settingsGraph(navController = navController)

        // ========== 保留的旧模块（可选） ==========

        // 旧版首页 (保留兼容)
        homeScreen(
            onNavigateToCreateAvailability = { },
            onNavigateToCircles = { },
            onNavigateToMessages = {
                navController.navigateToMessages()
            },
            onNavigateToProfile = {
                navController.navigateToProfile()
            },
        )

        // 消息 (保留兼容)
        messagesGraph(navController = navController)

        // 个人资料 (保留兼容)
        profileScreen(
            navController = navController,
            onLogout = {
                navController.navigateToPhoneInput()
            },
            onNavigateToFriends = {
                navController.navigateToFriends()
            },
            onNavigateToInvitations = { },
        )

        // Debug 界面
        composable(DEBUG_ROUTE) {
            ApiDebugScreen()
        }
    }
}
