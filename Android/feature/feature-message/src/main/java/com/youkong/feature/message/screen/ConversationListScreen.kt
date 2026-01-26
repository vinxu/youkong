package com.youkong.feature.message.screen

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
import androidx.compose.material3.Badge
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.Conversation
import com.youkong.core.ui.component.YouKongAvatar
import com.youkong.core.ui.component.YouKongEmptyState
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.message.viewmodel.ConversationListViewModel

@Composable
fun ConversationListScreen(
    onBackClick: () -> Unit,
    onConversationClick: (String) -> Unit,
    viewModel: ConversationListViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = {
            YouKongTopBar(
                title = "消息",
                onBackClick = onBackClick,
            )
        },
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
                        .padding(innerPadding),
                    contentPadding = PaddingValues(vertical = YouKongTheme.spacing.md),
                ) {
                    items(
                        items = uiState.conversations,
                        key = { it.id },
                    ) { conversation ->
                        ConversationItem(
                            conversation = conversation,
                            onClick = { onConversationClick(conversation.id) },
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
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = YouKongTheme.spacing.xxl, vertical = YouKongTheme.spacing.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        YouKongAvatar(
            imageUrl = conversation.partner.avatar,
            name = conversation.partner.nickname,
            size = 48.dp,
        )

        Spacer(modifier = Modifier.width(16.dp))

        Column(modifier = Modifier.weight(1f)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = conversation.partner.nickname,
                    style = MaterialTheme.typography.titleMedium,
                    color = TextPrimary,
                    modifier = Modifier.weight(1f),
                )

                if (conversation.unreadCount > 0) {
                    Badge {
                        Text(text = conversation.unreadCount.toString())
                    }
                }
            }

            if (conversation.lastMessage != null) {
                Text(
                    text = conversation.lastMessage!!.content ?: "[消息]",
                    style = MaterialTheme.typography.bodyMedium,
                    color = TextSecondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}
