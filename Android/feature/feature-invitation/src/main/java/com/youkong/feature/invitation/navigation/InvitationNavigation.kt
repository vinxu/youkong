package com.youkong.feature.invitation.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.NavType
import androidx.navigation.compose.composable
import androidx.navigation.navArgument
import androidx.navigation.navigation
import com.youkong.feature.invitation.screen.CreateInvitationScreen
import com.youkong.feature.invitation.screen.InvitationDetailScreen
import com.youkong.feature.invitation.screen.InvitationListScreen

const val INVITATION_GRAPH_ROUTE = "invitation"
const val CREATE_INVITATION_ROUTE = "create_invitation?circleId={circleId}"
const val INVITATION_LIST_ROUTE = "invitation_list"
const val INVITATION_DETAIL_ROUTE = "invitation_detail/{invitationId}"

fun NavController.navigateToCreateInvitation(circleId: String? = null) {
    val route = if (circleId != null) {
        "create_invitation?circleId=$circleId"
    } else {
        "create_invitation"
    }
    navigate(route)
}

fun NavController.navigateToInvitationList() {
    navigate(INVITATION_LIST_ROUTE)
}

fun NavController.navigateToInvitationDetail(invitationId: String) {
    navigate("invitation_detail/$invitationId")
}

fun NavGraphBuilder.invitationGraph(
    navController: NavController,
) {
    navigation(
        route = INVITATION_GRAPH_ROUTE,
        startDestination = INVITATION_LIST_ROUTE,
    ) {
        composable(INVITATION_LIST_ROUTE) {
            InvitationListScreen(
                onBackClick = { navController.popBackStack() },
                onInvitationClick = { invitationId ->
                    navController.navigateToInvitationDetail(invitationId)
                },
            )
        }

        composable(
            route = CREATE_INVITATION_ROUTE,
            arguments = listOf(
                navArgument("circleId") {
                    type = NavType.StringType
                    nullable = true
                    defaultValue = null
                }
            )
        ) {
            CreateInvitationScreen(
                onBackClick = { navController.popBackStack() },
                onCreateSuccess = { invitationId ->
                    navController.popBackStack()
                    navController.navigateToInvitationDetail(invitationId)
                },
            )
        }

        composable(
            route = INVITATION_DETAIL_ROUTE,
            arguments = listOf(
                navArgument("invitationId") { type = NavType.StringType }
            )
        ) {
            InvitationDetailScreen(
                onBackClick = { navController.popBackStack() },
            )
        }
    }
}
