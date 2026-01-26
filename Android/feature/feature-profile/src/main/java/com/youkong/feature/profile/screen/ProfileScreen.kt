package com.youkong.feature.profile.screen

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.YouKongAvatar
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.YouKongOutlinedButton
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.Error
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongTheme
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

    Scaffold(
        topBar = {
            YouKongTopBar(
                title = "我的",
                onBackClick = onBackClick,
            )
        },
    ) { innerPadding ->
        when {
            uiState.isLoading -> {
                YouKongLoading(message = "加载中...")
            }

            uiState.user != null -> {
                val user = uiState.user!!

                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(innerPadding)
                        .padding(horizontal = YouKongTheme.spacing.xxl),
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Spacer(modifier = Modifier.height(48.dp))

                    YouKongAvatar(
                        imageUrl = user.avatar,
                        name = user.nickname,
                        size = 80.dp,
                    )

                    Spacer(modifier = Modifier.height(16.dp))

                    Text(
                        text = user.nickname,
                        style = MaterialTheme.typography.headlineSmall,
                        color = TextPrimary,
                    )

                    Spacer(modifier = Modifier.height(8.dp))

                    Text(
                        text = user.phone.replaceRange(3, 7, "****"),
                        style = MaterialTheme.typography.bodyMedium,
                        color = TextSecondary,
                    )

                    Spacer(modifier = Modifier.height(48.dp))

                    ProfileMenuItem(
                        icon = Icons.Filled.Person,
                        title = "我的好友",
                        onClick = onNavigateToFriends,
                    )

                    HorizontalDivider(
                        modifier = Modifier.padding(horizontal = 16.dp),
                        color = MaterialTheme.colorScheme.outline.copy(alpha = 0.2f),
                    )

                    ProfileMenuItem(
                        icon = Icons.AutoMirrored.Filled.Send,
                        title = "我的邀请",
                        onClick = onNavigateToInvitations,
                    )

                    Spacer(modifier = Modifier.weight(1f))

                    YouKongOutlinedButton(
                        text = if (uiState.isLoggingOut) "退出中..." else "退出登录",
                        onClick = { viewModel.logout(onLogout) },
                        enabled = !uiState.isLoggingOut,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(bottom = 32.dp),
                    )
                }
            }
        }
    }
}

@Composable
private fun ProfileMenuItem(
    icon: ImageVector,
    title: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            modifier = Modifier.size(24.dp),
            tint = TextSecondary,
        )

        Spacer(modifier = Modifier.width(16.dp))

        Text(
            text = title,
            style = MaterialTheme.typography.bodyLarge,
            color = TextPrimary,
            modifier = Modifier.weight(1f),
        )

        Icon(
            imageVector = Icons.AutoMirrored.Filled.KeyboardArrowRight,
            contentDescription = null,
            modifier = Modifier.size(24.dp),
            tint = TextSecondary,
        )
    }
}
