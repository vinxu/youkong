package com.youkong.feature.circle.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.NavType
import androidx.navigation.compose.composable
import androidx.navigation.navArgument
import androidx.navigation.navigation
import com.youkong.feature.circle.screen.CircleDetailScreen
import com.youkong.feature.circle.screen.CircleListScreen
import com.youkong.feature.circle.screen.CreateCircleScreen

const val CIRCLES_GRAPH_ROUTE = "circles"
const val CIRCLE_LIST_ROUTE = "circle_list"
const val CREATE_CIRCLE_ROUTE = "create_circle"
const val CIRCLE_DETAIL_ROUTE = "circle_detail/{circleId}"

fun NavController.navigateToCircles() {
    navigate(CIRCLES_GRAPH_ROUTE)
}

fun NavController.navigateToCreateCircle() {
    navigate(CREATE_CIRCLE_ROUTE)
}

fun NavController.navigateToCircleDetail(circleId: String) {
    navigate("circle_detail/$circleId")
}

fun NavGraphBuilder.circlesGraph(
    navController: NavController,
    onInviteClick: (String) -> Unit = {},
) {
    navigation(
        route = CIRCLES_GRAPH_ROUTE,
        startDestination = CIRCLE_LIST_ROUTE,
    ) {
        composable(CIRCLE_LIST_ROUTE) {
            CircleListScreen(
                onBackClick = { navController.popBackStack() },
                onCreateCircleClick = { navController.navigateToCreateCircle() },
                onCircleClick = { circleId -> navController.navigateToCircleDetail(circleId) },
            )
        }

        composable(CREATE_CIRCLE_ROUTE) {
            CreateCircleScreen(
                onBackClick = { navController.popBackStack() },
                onCreateSuccess = { navController.popBackStack() },
            )
        }

        composable(
            route = CIRCLE_DETAIL_ROUTE,
            arguments = listOf(
                navArgument("circleId") { type = NavType.StringType }
            )
        ) { backStackEntry ->
            val circleId = backStackEntry.arguments?.getString("circleId") ?: ""
            CircleDetailScreen(
                onBackClick = { navController.popBackStack() },
                onInviteClick = { onInviteClick(circleId) },
            )
        }
    }
}
