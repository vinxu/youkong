package com.youkong.feature.home.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.compose.composable
import com.youkong.feature.home.screen.GridHomeScreen

const val HOME_ROUTE = "home"

fun NavController.navigateToHome() {
    navigate(HOME_ROUTE) {
        popUpTo(graph.startDestinationId) { inclusive = true }
    }
}

fun NavGraphBuilder.gridHomeScreen(
    onNavigateToSettings: () -> Unit,
) {
    composable(HOME_ROUTE) {
        GridHomeScreen(
            onNavigateToSettings = onNavigateToSettings
        )
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
