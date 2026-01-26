package com.youkong.feature.profile.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.compose.composable
import com.youkong.feature.profile.screen.ProfileScreen

const val PROFILE_ROUTE = "profile"

fun NavController.navigateToProfile() {
    navigate(PROFILE_ROUTE)
}

fun NavGraphBuilder.profileScreen(
    navController: NavController,
    onLogout: () -> Unit,
    onNavigateToFriends: () -> Unit = {},
    onNavigateToInvitations: () -> Unit = {},
) {
    composable(PROFILE_ROUTE) {
        ProfileScreen(
            onBackClick = { navController.popBackStack() },
            onLogout = onLogout,
            onNavigateToFriends = onNavigateToFriends,
            onNavigateToInvitations = onNavigateToInvitations,
        )
    }
}
