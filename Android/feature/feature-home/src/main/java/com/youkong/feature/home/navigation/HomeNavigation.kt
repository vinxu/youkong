package com.youkong.feature.home.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.NavType
import androidx.navigation.compose.composable
import androidx.navigation.navArgument
import com.youkong.feature.home.screen.FriendScheduleScreen
import com.youkong.feature.home.screen.GridHomeScreen
import com.youkong.feature.home.screen.OnboardingScreen

const val HOME_ROUTE = "home"
const val ONBOARDING_ROUTE = "onboarding"
const val FRIEND_SCHEDULE_ROUTE = "friend-schedule/{userId}/{friendName}"

fun NavController.navigateToHome() {
    navigate(HOME_ROUTE) {
        popUpTo(graph.startDestinationId) { inclusive = true }
    }
}

fun NavController.navigateToOnboarding() {
    navigate(ONBOARDING_ROUTE) {
        popUpTo(graph.startDestinationId) { inclusive = true }
    }
}

fun NavController.navigateToFriendSchedule(userId: String, friendName: String) {
    navigate("friend-schedule/$userId/${java.net.URLEncoder.encode(friendName, "UTF-8")}")
}

fun NavGraphBuilder.gridHomeScreen(
    onNavigateToSettings: () -> Unit,
    onNavigateToAddFriend: () -> Unit = {},
    onNavigateToChat: (userId: String) -> Unit = {},
) {
    composable(HOME_ROUTE) {
        GridHomeScreen(
            onNavigateToSettings = onNavigateToSettings,
            onNavigateToAddFriend = onNavigateToAddFriend,
            onNavigateToChat = onNavigateToChat,
        )
    }
}

fun NavGraphBuilder.friendScheduleScreen(
    onDismiss: () -> Unit,
) {
    composable(
        route = FRIEND_SCHEDULE_ROUTE,
        arguments = listOf(
            navArgument("userId") { type = NavType.StringType },
            navArgument("friendName") { type = NavType.StringType },
        )
    ) {
        FriendScheduleScreen(onDismiss = onDismiss)
    }
}

fun NavGraphBuilder.onboardingScreen(onComplete: () -> Unit) {
    composable(ONBOARDING_ROUTE) {
        OnboardingScreen(onComplete = onComplete)
    }
}

// MARK: - Deprecated (保留旧版兼容)

@Deprecated("使用 gridHomeScreen 代替")
fun NavGraphBuilder.homeScreen(
    onNavigateToCreateAvailability: () -> Unit = {},
    onNavigateToCircles: () -> Unit = {},
    onNavigateToMessages: () -> Unit = {},
    onNavigateToProfile: () -> Unit = {},
) {
    composable(HOME_ROUTE) {
        GridHomeScreen(
            onNavigateToSettings = onNavigateToProfile
        )
    }
}
