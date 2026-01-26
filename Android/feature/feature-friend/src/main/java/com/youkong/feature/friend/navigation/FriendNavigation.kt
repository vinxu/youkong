package com.youkong.feature.friend.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.compose.composable
import androidx.navigation.navigation
import com.youkong.feature.friend.screen.FriendListScreen

const val FRIEND_GRAPH_ROUTE = "friend"
const val FRIEND_LIST_ROUTE = "friend_list"

fun NavController.navigateToFriendList() {
    navigate(FRIEND_GRAPH_ROUTE)
}

fun NavGraphBuilder.friendGraph(
    navController: NavController,
) {
    navigation(
        route = FRIEND_GRAPH_ROUTE,
        startDestination = FRIEND_LIST_ROUTE,
    ) {
        composable(FRIEND_LIST_ROUTE) {
            FriendListScreen(
                onBackClick = { navController.popBackStack() },
            )
        }
    }
}
