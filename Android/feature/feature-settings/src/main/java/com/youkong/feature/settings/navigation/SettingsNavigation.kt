package com.youkong.feature.settings.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.NavOptions
import androidx.navigation.compose.composable
import androidx.navigation.navigation
import com.youkong.feature.settings.screen.AgentDataScreen
import com.youkong.feature.settings.screen.PermissionSetupScreen
import com.youkong.feature.settings.screen.SettingsScreen

const val SETTINGS_GRAPH_ROUTE = "settings_graph"
const val SETTINGS_ROUTE = "settings"
const val PERMISSION_SETUP_ROUTE = "permission_setup"
const val ONBOARDING_PERMISSION_ROUTE = "onboarding_permission"
const val AGENT_DATA_ROUTE = "agent_data"

fun NavController.navigateToSettings(navOptions: NavOptions? = null) {
    navigate(SETTINGS_GRAPH_ROUTE, navOptions)
}

fun NavController.navigateToPermissionSetup(navOptions: NavOptions? = null) {
    navigate(PERMISSION_SETUP_ROUTE, navOptions)
}

fun NavController.navigateToOnboardingPermission(navOptions: NavOptions? = null) {
    navigate(ONBOARDING_PERMISSION_ROUTE, navOptions)
}

fun NavController.navigateToAgentData(navOptions: NavOptions? = null) {
    navigate(AGENT_DATA_ROUTE, navOptions)
}

fun NavGraphBuilder.settingsGraph(
    navController: NavController,
) {
    navigation(
        route = SETTINGS_GRAPH_ROUTE,
        startDestination = SETTINGS_ROUTE,
    ) {
        composable(route = SETTINGS_ROUTE) {
            SettingsScreen(
                onBackClick = { navController.popBackStack() },
                onPermissionSetupClick = { navController.navigateToPermissionSetup() },
                onAgentDataClick = { navController.navigateToAgentData() },
            )
        }

        composable(route = PERMISSION_SETUP_ROUTE) {
            PermissionSetupScreen(
                onBackClick = { navController.popBackStack() },
                onComplete = { navController.popBackStack() },
            )
        }

        composable(route = AGENT_DATA_ROUTE) {
            AgentDataScreen(
                onBackClick = { navController.popBackStack() },
            )
        }
    }
}
