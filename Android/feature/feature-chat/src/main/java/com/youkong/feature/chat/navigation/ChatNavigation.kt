package com.youkong.feature.chat.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.NavOptions
import androidx.navigation.NavType
import androidx.navigation.compose.composable
import androidx.navigation.navArgument
import com.youkong.feature.chat.screen.ChatScreen

const val CHAT_ROUTE = "chat/{userId}"

fun NavController.navigateToChat(userId: String, navOptions: NavOptions? = null) {
    navigate("chat/$userId", navOptions)
}

fun NavGraphBuilder.chatScreen(
    navController: NavController,
) {
    composable(
        route = CHAT_ROUTE,
        arguments = listOf(
            navArgument("userId") { type = NavType.StringType }
        )
    ) {
        ChatScreen(
            onBackClick = { navController.popBackStack() },
        )
    }
}
