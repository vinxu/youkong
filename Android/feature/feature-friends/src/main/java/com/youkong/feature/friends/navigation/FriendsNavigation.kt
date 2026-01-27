package com.youkong.feature.friends.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.NavOptions
import androidx.navigation.compose.composable
import com.youkong.feature.friends.screen.AddFriendScreen
import com.youkong.feature.friends.screen.FriendsListScreen

const val FRIENDS_ROUTE = "friends"
const val ADD_FRIEND_ROUTE = "add_friend"

fun NavController.navigateToFriends(navOptions: NavOptions? = null) {
    navigate(FRIENDS_ROUTE, navOptions)
}

fun NavController.navigateToAddFriend(navOptions: NavOptions? = null) {
    navigate(ADD_FRIEND_ROUTE, navOptions)
}

fun NavGraphBuilder.friendsScreen(
    onNavigateToChat: (userId: String) -> Unit,
    onNavigateToSettings: () -> Unit,
    onNavigateToAgentData: () -> Unit,
    onNavigateToInvitation: () -> Unit,
    onNavigateToAddFriend: () -> Unit,
) {
    composable(route = FRIENDS_ROUTE) {
        FriendsListScreen(
            onNavigateToChat = onNavigateToChat,
            onNavigateToSettings = onNavigateToSettings,
            onNavigateToAgentData = onNavigateToAgentData,
            onNavigateToInvitation = onNavigateToInvitation,
            onNavigateToAddFriend = onNavigateToAddFriend,
        )
    }
}

fun NavGraphBuilder.addFriendScreen(
    onBackClick: () -> Unit,
) {
    composable(route = ADD_FRIEND_ROUTE) {
        AddFriendScreen(
            onBackClick = onBackClick,
        )
    }
}
