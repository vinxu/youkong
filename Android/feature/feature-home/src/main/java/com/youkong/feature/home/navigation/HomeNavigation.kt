package com.youkong.feature.home.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.compose.composable
import com.youkong.feature.home.screen.HomeScreen

const val HOME_ROUTE = "home"

fun NavController.navigateToHome() {
    navigate(HOME_ROUTE) {
        popUpTo(graph.startDestinationId) { inclusive = true }
    }
}

fun NavGraphBuilder.homeScreen(
    onNavigateToCreateAvailability: () -> Unit,
    onNavigateToCircles: () -> Unit,
    onNavigateToMessages: () -> Unit,
    onNavigateToProfile: () -> Unit,
) {
    composable(HOME_ROUTE) {
        HomeScreen(
            onNavigateToCreateAvailability = onNavigateToCreateAvailability,
            onNavigateToCircles = onNavigateToCircles,
            onNavigateToMessages = onNavigateToMessages,
            onNavigateToProfile = onNavigateToProfile,
        )
    }
}
