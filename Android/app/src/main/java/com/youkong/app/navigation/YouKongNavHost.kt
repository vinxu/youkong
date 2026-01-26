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
import com.youkong.feature.availability.navigation.createAvailabilityGraph
import com.youkong.feature.availability.navigation.navigateToCreateAvailability
import com.youkong.feature.circle.navigation.circlesGraph
import com.youkong.feature.circle.navigation.navigateToCircles
import com.youkong.feature.friend.navigation.friendGraph
import com.youkong.feature.friend.navigation.navigateToFriendList
import com.youkong.feature.home.navigation.HOME_ROUTE
import com.youkong.feature.home.navigation.homeScreen
import com.youkong.feature.home.navigation.navigateToHome
import com.youkong.feature.invitation.navigation.invitationGraph
import com.youkong.feature.invitation.navigation.navigateToCreateInvitation
import com.youkong.feature.invitation.navigation.navigateToInvitationList
import com.youkong.feature.message.navigation.messagesGraph
import com.youkong.feature.message.navigation.navigateToMessages
import com.youkong.feature.profile.navigation.navigateToProfile
import com.youkong.feature.profile.navigation.profileScreen

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
        uiState.isLoggedIn -> HOME_ROUTE
        else -> AUTH_GRAPH_ROUTE
    }

    // 当登录状态变化时，自动导航
    LaunchedEffect(uiState.isLoggedIn, uiState.isLoading) {
        if (!uiState.isLoading) {
            if (uiState.isLoggedIn) {
                navController.navigateToHome()
            } else {
                navController.navigateToPhoneInput()
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
                navController.navigateToHome()
            },
        )

        // 首页
        homeScreen(
            onNavigateToCreateAvailability = {
                navController.navigateToCreateAvailability()
            },
            onNavigateToCircles = {
                navController.navigateToCircles()
            },
            onNavigateToMessages = {
                navController.navigateToMessages()
            },
            onNavigateToProfile = {
                navController.navigateToProfile()
            },
        )

        // 发布有空
        createAvailabilityGraph(
            navController = navController,
            onCreateSuccess = {
                // 返回首页并刷新
            },
        )

        // 圈子管理
        circlesGraph(
            navController = navController,
            onInviteClick = { circleId ->
                navController.navigateToCreateInvitation(circleId)
            },
        )

        // 消息
        messagesGraph(navController = navController)

        // 邀请
        invitationGraph(navController = navController)

        // 好友
        friendGraph(navController = navController)

        // 个人资料
        profileScreen(
            navController = navController,
            onLogout = {
                navController.navigateToPhoneInput()
            },
            onNavigateToFriends = {
                navController.navigateToFriendList()
            },
            onNavigateToInvitations = {
                navController.navigateToInvitationList()
            },
        )

        // Debug 界面
        composable(DEBUG_ROUTE) {
            ApiDebugScreen()
        }
    }
}
