package com.youkong.feature.profile.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.cli.TerminalAvatar
import com.youkong.core.ui.component.cli.TerminalButton
import com.youkong.core.ui.component.cli.TerminalButtonStyle
import com.youkong.core.ui.component.cli.TerminalDivider
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.profile.viewmodel.ProfileViewModel

@Composable
fun ProfileScreen(
    onBackClick: () -> Unit,
    onLogout: () -> Unit,
    onNavigateToFriends: () -> Unit = {},
    onNavigateToInvitations: () -> Unit = {},
    viewModel: ProfileViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
    ) {
        // Terminal Header
        TerminalHeader(
            title = "Profile",
            showBackButton = true,
            onBackClick = onBackClick
        )

        when {
            uiState.isLoading -> {
                YouKongLoading(message = "Loading...")
            }

            uiState.user != null -> {
                val user = uiState.user!!

                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(horizontal = 16.dp),
                ) {
                    Spacer(modifier = Modifier.height(24.dp))

                    // User Info Section
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        TerminalAvatar(isOnline = true)

                        Spacer(modifier = Modifier.width(12.dp))

                        Column {
                            Text(
                                text = user.nickname,
                                fontFamily = FontFamily.Monospace,
                                fontSize = 16.sp,
                                color = CLIColors.TextPrimary,
                            )

                            Spacer(modifier = Modifier.height(4.dp))

                            Text(
                                text = user.phone.replaceRange(3, 7, "****"),
                                fontFamily = FontFamily.Monospace,
                                fontSize = 12.sp,
                                color = CLIColors.TextSecondary,
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(32.dp))

                    // Menu Items
                    ProfileMenuItem(
                        title = "My Friends",
                        onClick = onNavigateToFriends,
                    )

                    TerminalDivider(
                        modifier = Modifier.padding(vertical = 8.dp)
                    )

                    ProfileMenuItem(
                        title = "My Invitations",
                        onClick = onNavigateToInvitations,
                    )

                    Spacer(modifier = Modifier.weight(1f))

                    // Logout Button
                    TerminalButton(
                        text = if (uiState.isLoggingOut) "Logging out..." else "Logout",
                        onClick = { viewModel.logout(onLogout) },
                        enabled = !uiState.isLoggingOut,
                        style = TerminalButtonStyle.DANGER,
                        modifier = Modifier.padding(bottom = 32.dp)
                    )
                }
            }
        }
    }
}

@Composable
private fun ProfileMenuItem(
    title: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = ASCII.ARROW_RIGHT,
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = CLIColors.TextSecondary,
        )

        Spacer(modifier = Modifier.width(12.dp))

        Text(
            text = title,
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = CLIColors.TextPrimary,
            modifier = Modifier.weight(1f),
        )
    }
}
