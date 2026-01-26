package com.youkong.feature.invitation.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.Invitation
import com.youkong.core.domain.model.InvitationStatus
import com.youkong.core.ui.component.YouKongEmojiAvatar
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.CircleColors
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.invitation.viewmodel.InvitationListViewModel
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

@Composable
fun InvitationListScreen(
    onBackClick: () -> Unit,
    onInvitationClick: (String) -> Unit,
    viewModel: InvitationListViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = {
            YouKongTopBar(
                title = "我的邀请",
                onBackClick = onBackClick,
            )
        },
    ) { innerPadding ->
        when {
            uiState.isLoading -> {
                YouKongLoading(message = "加载中...")
            }

            uiState.invitations.isEmpty() -> {
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(innerPadding),
                    contentAlignment = Alignment.Center,
                ) {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally,
                    ) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.Send,
                            contentDescription = null,
                            modifier = Modifier.size(64.dp),
                            tint = TextSecondary,
                        )
                        Spacer(modifier = Modifier.height(16.dp))
                        Text(
                            text = "暂无邀请链接",
                            style = MaterialTheme.typography.bodyLarge,
                            color = TextSecondary,
                        )
                    }
                }
            }

            else -> {
                LazyColumn(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(innerPadding),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                    contentPadding = PaddingValues(
                        horizontal = YouKongTheme.spacing.xxl,
                        vertical = 16.dp,
                    ),
                ) {
                    items(uiState.invitations, key = { it.id }) { invitation ->
                        InvitationCard(
                            invitation = invitation,
                            onClick = { onInvitationClick(invitation.id) },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun InvitationCard(
    invitation: Invitation,
    onClick: () -> Unit,
) {
    val circle = invitation.circle

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surface,
        ),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (circle != null) {
                YouKongEmojiAvatar(
                    emoji = circle.emoji,
                    backgroundColor = CircleColors.fromHex(circle.color ?: "#3B82F6"),
                    size = 48.dp,
                )
            } else {
                Box(
                    modifier = Modifier
                        .size(48.dp)
                        .clip(CircleShape)
                        .background(Primary.copy(alpha = 0.2f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = Icons.AutoMirrored.Filled.Send,
                        contentDescription = null,
                        tint = Primary,
                        modifier = Modifier.size(24.dp),
                    )
                }
            }

            Spacer(modifier = Modifier.width(12.dp))

            Column(
                modifier = Modifier.weight(1f),
            ) {
                Text(
                    text = circle?.name ?: "通用邀请",
                    style = MaterialTheme.typography.bodyLarge,
                    color = TextPrimary,
                )

                Spacer(modifier = Modifier.height(4.dp))

                Row(
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = "已使用 ${invitation.useCount}/${invitation.maxUses}",
                        style = MaterialTheme.typography.bodySmall,
                        color = TextSecondary,
                    )

                    Spacer(modifier = Modifier.width(8.dp))

                    val statusText = when (invitation.status) {
                        InvitationStatus.ACTIVE -> "有效"
                        InvitationStatus.DISABLED -> "已禁用"
                        InvitationStatus.EXPIRED -> "已过期"
                    }
                    val statusColor = when (invitation.status) {
                        InvitationStatus.ACTIVE -> Primary
                        InvitationStatus.DISABLED -> MaterialTheme.colorScheme.error
                        InvitationStatus.EXPIRED -> TextSecondary
                    }

                    Text(
                        text = statusText,
                        style = MaterialTheme.typography.bodySmall,
                        color = statusColor,
                    )
                }

                val expiresAt = invitation.expiresAt
                if (expiresAt != null) {
                    Spacer(modifier = Modifier.height(4.dp))
                    val localDateTime = expiresAt.toLocalDateTime(TimeZone.currentSystemDefault())
                    Text(
                        text = "有效期至 ${localDateTime.monthNumber}/${localDateTime.dayOfMonth}",
                        style = MaterialTheme.typography.bodySmall,
                        color = TextSecondary,
                    )
                }
            }
        }
    }
}
