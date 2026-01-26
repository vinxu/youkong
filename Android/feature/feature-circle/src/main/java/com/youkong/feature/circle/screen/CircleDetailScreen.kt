package com.youkong.feature.circle.screen

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.YouKongAvatar
import com.youkong.core.ui.component.YouKongEmojiAvatar
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.CircleColors
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.circle.viewmodel.CircleDetailViewModel

@Composable
fun CircleDetailScreen(
    onBackClick: () -> Unit,
    onInviteClick: () -> Unit = {},
    viewModel: CircleDetailViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = {
            YouKongTopBar(
                title = uiState.circle?.name ?: "圈子详情",
                onBackClick = onBackClick,
            )
        },
    ) { innerPadding ->
        when {
            uiState.isLoading -> {
                YouKongLoading(message = "加载中...")
            }

            uiState.circle != null -> {
                val circle = uiState.circle!!

                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(innerPadding),
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Spacer(modifier = Modifier.height(32.dp))

                    YouKongEmojiAvatar(
                        emoji = circle.emoji,
                        backgroundColor = CircleColors.fromHex(circle.color),
                        size = 80.dp,
                    )

                    Spacer(modifier = Modifier.height(16.dp))

                    Text(
                        text = circle.name,
                        style = MaterialTheme.typography.headlineSmall,
                        color = TextPrimary,
                    )

                    Spacer(modifier = Modifier.height(8.dp))

                    Text(
                        text = "${circle.memberCount} 位成员",
                        style = MaterialTheme.typography.bodyMedium,
                        color = TextSecondary,
                    )

                    Spacer(modifier = Modifier.height(24.dp))

                    OutlinedButton(
                        onClick = onInviteClick,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = YouKongTheme.spacing.xxl),
                    ) {
                        Text("邀请好友加入")
                    }

                    Spacer(modifier = Modifier.height(32.dp))

                    Text(
                        text = "成员",
                        style = MaterialTheme.typography.titleMedium,
                        color = TextPrimary,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = YouKongTheme.spacing.xxl),
                    )

                    Spacer(modifier = Modifier.height(16.dp))

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

@Composable
private fun MemberItem(
    nickname: String,
    avatar: String?,
) {
    androidx.compose.foundation.layout.Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = YouKongTheme.spacing.xxl, vertical = YouKongTheme.spacing.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        YouKongAvatar(
            imageUrl = avatar,
            name = nickname,
            size = 40.dp,
        )

        Spacer(modifier = Modifier.weight(0.1f).padding(start = 16.dp))

        Text(
            text = nickname,
            style = MaterialTheme.typography.bodyLarge,
            color = TextPrimary,
            modifier = Modifier.weight(1f),
        )
    }
}
