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
import com.youkong.feature.message.navigation.navigateToChatWithPartner
import com.youkong.feature.friends.navigation.addFriendScreen
import com.youkong.feature.friends.navigation.friendsScreen
import com.youkong.feature.friends.navigation.navigateToAddFriend
import com.youkong.feature.friends.navigation.navigateToFriends
import com.youkong.feature.home.navigation.HOME_ROUTE
import com.youkong.feature.home.navigation.gridHomeScreen
import com.youkong.feature.home.navigation.friendScheduleScreen
import com.youkong.feature.home.navigation.navigateToFriendSchedule
import com.youkong.feature.home.navigation.navigateToHome
import com.youkong.feature.home.navigation.onboardingScreen
import com.youkong.feature.message.navigation.messagesGraph
import com.youkong.feature.message.navigation.navigateToMessages
import com.youkong.feature.profile.navigation.navigateToProfile
import com.youkong.feature.profile.navigation.profileScreen
import com.youkong.feature.settings.navigation.ONBOARDING_PERMISSION_ROUTE
import com.youkong.feature.settings.navigation.navigateToOnboardingPermission
import com.youkong.feature.settings.navigation.navigateToAgentData
import com.youkong.feature.settings.navigation.navigateToInvitation
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
            if (uiState.hasRequiredPermissions) HOME_ROUTE else ONBOARDING_PERMISSION_ROUTE
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
                    navController.navigateToHome()
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
                viewModel.refreshPermissions()
            },
        )

        // 引导流程（3 屏 + Chat 覆盖层，对齐 iOS）
        composable(route = ONBOARDING_PERMISSION_ROUTE) {
            com.youkong.feature.home.screen.OnboardingScreen(
                onComplete = {
                    viewModel.refreshPermissions()
                    navController.navigateToHome()
                }
            )
        }

        // 宫格首页 (主页面)
        gridHomeScreen(
            onNavigateToSettings = {
                navController.navigateToSettings()
            },
            onNavigateToAddFriend = {
                navController.navigateToAddFriend()
            },
            onNavigateToChat = { userId ->
                navController.navigateToChatWithPartner(userId)
            },
        )

        // 好友时刻表页面
        friendScheduleScreen(
            onDismiss = {
                navController.popBackStack()
            },
        )

        // 好友列表
        friendsScreen(
            onNavigateToChat = { userId ->
                navController.navigateToChatWithPartner(userId)
            },
            onNavigateToSettings = {
                navController.navigateToSettings()
            },
            onNavigateToAgentData = {
                navController.navigateToAgentData()
            },
            onNavigateToInvitation = {
                navController.navigateToInvitation()
            },
            onNavigateToAddFriend = {
                navController.navigateToAddFriend()
            },
        )

        // 添加好友页面
        addFriendScreen(
            onBackClick = {
                navController.popBackStack()
            },
        )

        // 聊天和消息
        messagesGraph(
            navController = navController,
            onNavigateToFriendSchedule = { userId, friendName ->
                navController.navigateToFriendSchedule(userId, friendName)
            },
        )

        // 设置
        settingsGraph(navController = navController)

        // 个人资料
        profileScreen(
            navController = navController,
            onLogout = {
                navController.navigateToPhoneInput()
            },
            onNavigateToFriends = {
                navController.navigateToHome()
            },
            onNavigateToInvitations = { },
        )

        // Debug 界面
        composable(DEBUG_ROUTE) {
            ApiDebugScreen()
        }
    }
}
