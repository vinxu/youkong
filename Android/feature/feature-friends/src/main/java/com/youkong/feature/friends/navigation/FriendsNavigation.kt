package com.youkong.feature.friends.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.NavOptions
import androidx.navigation.compose.composable
import com.youkong.feature.friends.screen.FriendsListScreen

const val FRIENDS_ROUTE = "friends"

fun NavController.navigateToFriends(navOptions: NavOptions? = null) {
    navigate(FRIENDS_ROUTE, navOptions)
}

fun NavGraphBuilder.friendsScreen(
    onNavigateToChat: (userId: String) -> Unit,
    onNavigateToSettings: () -> Unit,
) {
    composable(route = FRIENDS_ROUTE) {
        FriendsListScreen(
            onNavigateToChat = onNavigateToChat,
            onNavigateToSettings = onNavigateToSettings,
        )
    }
}
