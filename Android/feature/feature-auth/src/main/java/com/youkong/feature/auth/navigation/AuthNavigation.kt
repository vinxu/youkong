package com.youkong.feature.auth.navigation

import androidx.navigation.NavController
import androidx.navigation.NavGraphBuilder
import androidx.navigation.NavType
import androidx.navigation.compose.composable
import androidx.navigation.navArgument
import androidx.navigation.navigation
import com.youkong.feature.auth.screen.PhoneInputScreen
import com.youkong.feature.auth.screen.ProfileSetupScreen
import com.youkong.feature.auth.screen.SmsVerificationScreen

const val AUTH_GRAPH_ROUTE = "auth"
const val PHONE_INPUT_ROUTE = "phone_input"
const val SMS_VERIFICATION_ROUTE = "sms_verification/{phone}"
const val PROFILE_SETUP_ROUTE = "profile_setup"

fun NavController.navigateToPhoneInput() {
    navigate(PHONE_INPUT_ROUTE) {
        popUpTo(graph.startDestinationId) { inclusive = true }
    }
}

fun NavController.navigateToSmsVerification(phone: String) {
    navigate("sms_verification/$phone")
}

fun NavController.navigateToProfileSetup() {
    navigate(PROFILE_SETUP_ROUTE)
}

fun NavGraphBuilder.authGraph(
    navController: NavController,
    onLoginSuccess: () -> Unit,
) {
    navigation(
        route = AUTH_GRAPH_ROUTE,
        startDestination = PHONE_INPUT_ROUTE,
    ) {
        composable(PHONE_INPUT_ROUTE) {
            PhoneInputScreen(
                onNavigateToVerification = { phone ->
                    navController.navigateToSmsVerification(phone)
                }
            )
        }

        composable(
            route = SMS_VERIFICATION_ROUTE,
            arguments = listOf(
                navArgument("phone") { type = NavType.StringType }
            )
        ) { backStackEntry ->
            val phone = backStackEntry.arguments?.getString("phone") ?: ""
            SmsVerificationScreen(
                phone = phone,
                onBackClick = { navController.popBackStack() },
                onLoginSuccess = { isNewUser ->
                    if (isNewUser) {
                        navController.navigateToProfileSetup()
                    } else {
                        onLoginSuccess()
                    }
                }
            )
        }

        composable(PROFILE_SETUP_ROUTE) {
            ProfileSetupScreen(
                onSetupComplete = onLoginSuccess
            )
        }
    }
}
