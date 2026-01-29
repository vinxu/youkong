package com.youkong.feature.circle.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
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
import com.youkong.core.ui.component.YouKongAvatar
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.cli.TerminalButton
import com.youkong.core.ui.component.cli.TerminalButtonStyle
import com.youkong.core.ui.component.cli.TerminalDivider
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.component.cli.TerminalSectionDivider
import com.youkong.core.ui.theme.CLIColors
import com.youkong.core.ui.theme.CircleColors
import com.youkong.feature.circle.viewmodel.CircleDetailViewModel

@Composable
fun CircleDetailScreen(
    onBackClick: () -> Unit,
    onInviteClick: () -> Unit = {},
    viewModel: CircleDetailViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
    ) {
        when {
            uiState.isLoading -> {
                YouKongLoading(message = "Loading...")
            }

            uiState.circle != null -> {
                val circle = uiState.circle!!

                Column(modifier = Modifier.fillMaxSize()) {
                    TerminalHeader(
                        title = circle.name,
                        subtitle = "${circle.memberCount} members",
                        showBackButton = true,
                        onBackClick = onBackClick
                    )

                    Column(
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(16.dp)
                    ) {
                        // Circle Info
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(vertical = 16.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            // Emoji without background
                            Text(
                                text = circle.emoji,
                                fontSize = 48.sp,
                                modifier = Modifier.padding(end = 16.dp)
                            )

                            Column {
                                Text(
                                    text = circle.name,
                                    fontFamily = FontFamily.Monospace,
                                    fontSize = 18.sp,
                                    color = CircleColors.fromHex(circle.color)
                                )

                                Spacer(modifier = Modifier.height(4.dp))

                                Text(
                                    text = "members: ${circle.memberCount}",
                                    fontFamily = FontFamily.Monospace,
                                    fontSize = 12.sp,
                                    color = CLIColors.TextSecondary
                                )
                            }
                        }

                        Spacer(modifier = Modifier.height(8.dp))

                        // Invite button
                        TerminalButton(
                            text = "[INVITE_FRIENDS]",
                            onClick = onInviteClick,
                            style = TerminalButtonStyle.SECONDARY
                        )

                        Spacer(modifier = Modifier.height(24.dp))

                        // Members section
                        TerminalSectionDivider(title = "MEMBERS")

                        Spacer(modifier = Modifier.height(8.dp))

                        if (circle.members != null) {
                            LazyColumn {
                                items(circle.members!!) { member ->
                                    MemberItem(
                                        nickname = member.nickname,
                                        avatar = member.avatar,
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun MemberItem(
    nickname: String,
    avatar: String?,
) {
    Column(modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            YouKongAvatar(
                imageUrl = avatar,
                name = nickname,
                size = 32.dp,
            )

            Spacer(modifier = Modifier.padding(start = 12.dp))

            Text(
                text = nickname,
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = CLIColors.TextPrimary,
                modifier = Modifier.weight(1f),
            )
        }

        TerminalDivider()
    }
}
