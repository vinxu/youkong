package com.youkong.feature.invitation.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.YouKongButton
import com.youkong.core.ui.component.YouKongEmojiAvatar
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.CircleColors
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.invitation.viewmodel.CreateInvitationViewModel

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun CreateInvitationScreen(
    onBackClick: () -> Unit,
    onCreateSuccess: (String) -> Unit,
    viewModel: CreateInvitationViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    LaunchedEffect(uiState.createdInvitation) {
        uiState.createdInvitation?.let { invitation ->
            onCreateSuccess(invitation.id)
        }
    }

    Scaffold(
        topBar = {
            YouKongTopBar(
                title = "创建邀请",
                onBackClick = onBackClick,
            )
        },
    ) { innerPadding ->
        when {
            uiState.isLoading -> {
                YouKongLoading(message = "加载中...")
            }

            else -> {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(innerPadding)
                        .padding(horizontal = YouKongTheme.spacing.xxl),
                ) {
                    Spacer(modifier = Modifier.height(24.dp))

                    Text(
                        text = "邀请好友加入",
                        style = MaterialTheme.typography.titleMedium,
                        color = TextPrimary,
                    )

                    val circle = uiState.circle
                    if (circle != null) {
                        Spacer(modifier = Modifier.height(16.dp))

                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clip(RoundedCornerShape(12.dp))
                                .background(MaterialTheme.colorScheme.surfaceVariant)
                                .padding(16.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            YouKongEmojiAvatar(
                                emoji = circle.emoji,
                                backgroundColor = CircleColors.fromHex(circle.color),
                                size = 48.dp,
                            )

                            Column(
                                modifier = Modifier
                                    .padding(start = 12.dp)
                                    .weight(1f),
                            ) {
                                Text(
                                    text = circle.name,
                                    style = MaterialTheme.typography.bodyLarge,
                                    color = TextPrimary,
                                )
                                Text(
                                    text = "${circle.memberCount} 位成员",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = TextSecondary,
                                )
                            }
                        }
                    }

                    Spacer(modifier = Modifier.height(32.dp))

                    Text(
                        text = "有效期",
                        style = MaterialTheme.typography.titleMedium,
                        color = TextPrimary,
                    )

                    Spacer(modifier = Modifier.height(12.dp))

                    FlowRow(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        ExpireDaysOption(
                            days = listOf(1, 3, 7, 14, 30),
                            selected = uiState.expireDays,
                            onSelect = viewModel::setExpireDays,
                        )
                    }

                    Spacer(modifier = Modifier.height(32.dp))

                    Text(
                        text = "最大使用次数",
                        style = MaterialTheme.typography.titleMedium,
                        color = TextPrimary,
                    )

                    Spacer(modifier = Modifier.height(12.dp))

                    FlowRow(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        MaxUsesOption(
                            uses = listOf(10, 50, 100, 500),
                            selected = uiState.maxUses,
                            onSelect = viewModel::setMaxUses,
                        )
                    }

                    Spacer(modifier = Modifier.weight(1f))

                    val error = uiState.error
                    if (error != null) {
                        Text(
                            text = error,
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.error,
                            modifier = Modifier.padding(bottom = 16.dp),
                        )
                    }

                    YouKongButton(
                        text = "生成邀请链接",
                        onClick = viewModel::createInvitation,
                        enabled = !uiState.isCreating,
                        loading = uiState.isCreating,
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
private fun ExpireDaysOption(
    days: List<Int>,
    selected: Int,
    onSelect: (Int) -> Unit,
) {
    days.forEach { day ->
        val isSelected = day == selected
        Box(
            modifier = Modifier
                .clip(RoundedCornerShape(8.dp))
                .background(
                    if (isSelected) Primary else MaterialTheme.colorScheme.surfaceVariant
                )
                .clickable { onSelect(day) }
                .padding(horizontal = 16.dp, vertical = 10.dp),
        ) {
            Text(
                text = "${day}天",
                style = MaterialTheme.typography.bodyMedium,
                color = if (isSelected) MaterialTheme.colorScheme.onPrimary else TextPrimary,
            )
        }
    }
}

@Composable
private fun MaxUsesOption(
    uses: List<Int>,
    selected: Int,
    onSelect: (Int) -> Unit,
) {
    uses.forEach { use ->
        val isSelected = use == selected
        Box(
            modifier = Modifier
                .clip(RoundedCornerShape(8.dp))
                .background(
                    if (isSelected) Primary else MaterialTheme.colorScheme.surfaceVariant
                )
                .clickable { onSelect(use) }
                .padding(horizontal = 16.dp, vertical = 10.dp),
        ) {
            Text(
                text = "${use}次",
                style = MaterialTheme.typography.bodyMedium,
                color = if (isSelected) MaterialTheme.colorScheme.onPrimary else TextPrimary,
            )
        }
    }
}
