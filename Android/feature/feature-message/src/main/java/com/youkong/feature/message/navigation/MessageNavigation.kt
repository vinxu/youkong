package com.youkong.feature.message.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.NavType
import androidx.navigation.compose.composable
import androidx.navigation.navArgument
import androidx.navigation.navigation
import com.youkong.feature.message.screen.ChatScreen
import com.youkong.feature.message.screen.ConversationListScreen

const val MESSAGES_GRAPH_ROUTE = "messages"
const val CONVERSATION_LIST_ROUTE = "conversation_list"
const val CHAT_ROUTE = "chat?conversationId={conversationId}&partnerId={partnerId}"

fun NavController.navigateToMessages() {
    navigate(MESSAGES_GRAPH_ROUTE)
}

fun NavController.navigateToChat(conversationId: String) {
    navigate("chat?conversationId=$conversationId")
}

fun NavController.navigateToChatWithPartner(partnerId: String) {
    navigate("chat?partnerId=$partnerId")
}

fun NavGraphBuilder.messagesGraph(
    navController: NavController,
) {
    navigation(
        route = MESSAGES_GRAPH_ROUTE,
        startDestination = CONVERSATION_LIST_ROUTE,
    ) {
        composable(CONVERSATION_LIST_ROUTE) {
            ConversationListScreen(
                onBackClick = { navController.popBackStack() },
                onConversationClick = { conversationId ->
                    navController.navigateToChat(conversationId)
                },
            )
        }

        composable(
            route = CHAT_ROUTE,
            arguments = listOf(
                navArgument("conversationId") {
                    type = NavType.StringType
                    nullable = true
                    defaultValue = null
                },
                navArgument("partnerId") {
                    type = NavType.StringType
                    nullable = true
                    defaultValue = null
                }
            )
        ) {
            ChatScreen(
                onBackClick = { navController.popBackStack() },
            )
        }
    }
}
