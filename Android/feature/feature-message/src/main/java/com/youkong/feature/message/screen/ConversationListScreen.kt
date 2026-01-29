package com.youkong.feature.message.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.Conversation
import com.youkong.core.ui.component.YouKongEmptyState
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.cli.TerminalAvatar
import com.youkong.core.ui.component.cli.TerminalDivider
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.theme.CLIColors
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.message.viewmodel.ConversationListViewModel
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

@Composable
fun ConversationListScreen(
    onBackClick: () -> Unit,
    onConversationClick: (conversationId: String, partnerId: String) -> Unit,
    viewModel: ConversationListViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = {
            TerminalHeader(
                title = "messages",
                showBackButton = true,
                onBackClick = onBackClick,
            )
        },
        containerColor = CLIColors.Background,
    ) { innerPadding ->
        when {
            uiState.isLoading -> {
                YouKongLoading(message = "加载中...")
            }

            uiState.conversations.isEmpty() -> {
                YouKongEmptyState(
                    emoji = "\uD83D\uDCAC",
                    title = "暂无消息",
                    subtitle = "和朋友约起来吧",
                    modifier = Modifier.padding(innerPadding),
                )
            }

            else -> {
                LazyColumn(
                    modifier = Modifier
                        .fillMaxSize()
                        .background(CLIColors.Background)
                        .padding(innerPadding),
                    contentPadding = PaddingValues(vertical = YouKongTheme.spacing.md),
                ) {
                    items(
                        items = uiState.conversations,
                        key = { it.id },
                    ) { conversation ->
                        ConversationItem(
                            conversation = conversation,
                            onClick = { onConversationClick(conversation.id, conversation.partner.id) },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ConversationItem(
    conversation: Conversation,
    onClick: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(CLIColors.Background)
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 12.dp)
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TerminalAvatar(isOnline = true)

            Spacer(modifier = Modifier.width(8.dp))

            Text(
                text = conversation.partner.nickname,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextPrimary,
            )

            Spacer(modifier = Modifier.weight(1f))

            // 时间戳 [HH:mm]
            if (conversation.lastMessage != null) {
                Text(
                    text = formatTimestamp(conversation.lastMessage!!.createdAt),
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary,
                )
            }

            // 未读角标
            if (conversation.unreadCount > 0) {
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    text = "(${conversation.unreadCount})",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Red,
                )
            }
        }

        if (conversation.lastMessage != null) {
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = "  ${conversation.lastMessage!!.content ?: "[消息]"}",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.padding(top = 4.dp)
            )
        }

        Spacer(modifier = Modifier.width(8.dp))

        TerminalDivider()
    }
}

private fun formatTimestamp(timestamp: Instant): String {
    val localDateTime = timestamp.toLocalDateTime(TimeZone.currentSystemDefault())
    val hour = localDateTime.hour.toString().padStart(2, '0')
    val minute = localDateTime.minute.toString().padStart(2, '0')
    return "[$hour:$minute]"
}
