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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import com.youkong.core.domain.model.Invitation
import com.youkong.core.domain.model.InvitationStatus
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
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

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
    ) {
        TerminalHeader(
            title = "MY_INVITATIONS",
            showBackButton = true,
            onBackClick = onBackClick,
        )

        when {
            uiState.isLoading -> {
                YouKongLoading(message = "Loading invitations...")
            }

            uiState.invitations.isEmpty() -> {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center,
                ) {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally,
                    ) {
                        Text(
                            text = ASCII.INFO,
                            fontFamily = FontFamily.Monospace,
                            fontSize = 32.sp,
                            color = CLIColors.TextSecondary,
                        )
                        Spacer(modifier = Modifier.height(16.dp))
                        Text(
                            text = "No invitations found",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 14.sp,
                            color = CLIColors.TextSecondary,
                        )
                    }
                }
            }

            else -> {
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                    contentPadding = PaddingValues(16.dp),
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
    val statusSymbol = when (invitation.status) {
        InvitationStatus.ACTIVE -> ASCII.BULLET
        InvitationStatus.DISABLED -> ASCII.CROSS
        InvitationStatus.EXPIRED -> ASCII.BULLET_EMPTY
    }
    val statusColor = when (invitation.status) {
        InvitationStatus.ACTIVE -> CLIColors.Green
        InvitationStatus.DISABLED -> CLIColors.Red
        InvitationStatus.EXPIRED -> CLIColors.TextSecondary
    }
    val statusText = when (invitation.status) {
        InvitationStatus.ACTIVE -> "ACTIVE"
        InvitationStatus.DISABLED -> "DISABLED"
        InvitationStatus.EXPIRED -> "EXPIRED"
    }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
    ) {
        // 顶部边框
        Text(
            text = ASCII.BOX_TOP_LEFT + ASCII.BOX_HORIZONTAL.repeat(48) + ASCII.BOX_TOP_RIGHT,
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = CLIColors.Border,
        )

        // 内容区域
        Row(
            modifier = Modifier.padding(vertical = 8.dp),
            verticalAlignment = Alignment.Top,
        ) {
            // 左边框
            Text(
                text = ASCII.BOX_VERTICAL,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.Border,
            )

            Spacer(modifier = Modifier.width(8.dp))

            Column(modifier = Modifier.weight(1f)) {
                // 标题行
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = statusSymbol,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = statusColor,
                    )

                    Spacer(modifier = Modifier.width(8.dp))

                    Text(
                        text = circle?.name ?: "GENERAL_INVITE",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = CLIColors.TextPrimary,
                    )

                    if (circle != null) {
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = circle.emoji,
                            fontSize = 14.sp,
                        )
                    }
                }

                Spacer(modifier = Modifier.height(8.dp))

                // 使用信息
                Text(
                    text = "Used: ${invitation.useCount}/${invitation.maxUses}  ${ASCII.BULLET}  Status: $statusText",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary,
                )

                // 有效期
                val expiresAt = invitation.expiresAt
                if (expiresAt != null) {
                    Spacer(modifier = Modifier.height(4.dp))
                    val localDateTime = expiresAt.toLocalDateTime(TimeZone.currentSystemDefault())
                    Text(
                        text = "Expires: ${localDateTime.year}-${localDateTime.monthNumber.toString().padStart(2, '0')}-${localDateTime.dayOfMonth.toString().padStart(2, '0')}",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.TextWeak,
                    )
                }

                // 邀请码
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = ASCII.boxedText(invitation.code, 42),
                    fontFamily = FontFamily.Monospace,
                    fontSize = 11.sp,
                    color = CLIColors.Green,
                    lineHeight = 14.sp,
                )
            }

            Spacer(modifier = Modifier.width(8.dp))

            // 右边框
            Text(
                text = ASCII.BOX_VERTICAL,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.Border,
            )
        }

        // 底部边框
        Text(
            text = ASCII.BOX_BOTTOM_LEFT + ASCII.BOX_HORIZONTAL.repeat(48) + ASCII.BOX_BOTTOM_RIGHT,
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = CLIColors.Border,
        )
    }
}
