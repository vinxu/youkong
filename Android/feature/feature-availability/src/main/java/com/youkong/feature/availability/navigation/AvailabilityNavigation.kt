package com.youkong.feature.availability.navigation

import androidx.compose.runtime.remember
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.compose.composable
import androidx.navigation.navigation
import com.youkong.feature.availability.screen.SelectCirclesScreen
import com.youkong.feature.availability.screen.SelectLocationScreen
import com.youkong.feature.availability.screen.SelectTimeScreen
import com.youkong.feature.availability.viewmodel.CreateAvailabilityViewModel

const val CREATE_AVAILABILITY_GRAPH_ROUTE = "create_availability"
const val SELECT_TIME_ROUTE = "select_time"
const val SELECT_LOCATION_ROUTE = "select_location"
const val SELECT_CIRCLES_ROUTE = "select_circles"

fun NavController.navigateToCreateAvailability() {
    navigate(CREATE_AVAILABILITY_GRAPH_ROUTE)
}

fun NavGraphBuilder.createAvailabilityGraph(
    navController: NavController,
    onCreateSuccess: () -> Unit,
) {
    navigation(
        route = CREATE_AVAILABILITY_GRAPH_ROUTE,
        startDestination = SELECT_TIME_ROUTE,
    ) {
        composable(SELECT_TIME_ROUTE) { backStackEntry ->
            val parentEntry = remember(backStackEntry) {
                navController.getBackStackEntry(CREATE_AVAILABILITY_GRAPH_ROUTE)
            }
            val viewModel: CreateAvailabilityViewModel = hiltViewModel(parentEntry)

            SelectTimeScreen(
                viewModel = viewModel,
                onCloseClick = { navController.popBackStack(CREATE_AVAILABILITY_GRAPH_ROUTE, true) },
                onNextClick = { navController.navigate(SELECT_LOCATION_ROUTE) },
            )
        }

        composable(SELECT_LOCATION_ROUTE) { backStackEntry ->
            val parentEntry = remember(backStackEntry) {
                navController.getBackStackEntry(CREATE_AVAILABILITY_GRAPH_ROUTE)
            }
            val viewModel: CreateAvailabilityViewModel = hiltViewModel(parentEntry)

            SelectLocationScreen(
                viewModel = viewModel,
                onBackClick = { navController.popBackStack() },
                onNextClick = { navController.navigate(SELECT_CIRCLES_ROUTE) },
            )
        }

        composable(SELECT_CIRCLES_ROUTE) { backStackEntry ->
            val parentEntry = remember(backStackEntry) {
                navController.getBackStackEntry(CREATE_AVAILABILITY_GRAPH_ROUTE)
            }
            val viewModel: CreateAvailabilityViewModel = hiltViewModel(parentEntry)

            SelectCirclesScreen(
                viewModel = viewModel,
                onBackClick = { navController.popBackStack() },
                onCreateSuccess = {
                    navController.popBackStack(CREATE_AVAILABILITY_GRAPH_ROUTE, true)
                    onCreateSuccess()
                },
            )
        }
    }
}
