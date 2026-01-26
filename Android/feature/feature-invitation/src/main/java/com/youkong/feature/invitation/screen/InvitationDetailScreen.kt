package com.youkong.feature.invitation.screen

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.youkong.core.domain.model.InvitationStatus
import com.youkong.core.ui.component.YouKongButton
import com.youkong.core.ui.component.YouKongEmojiAvatar
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.CircleColors
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.invitation.viewmodel.InvitationDetailViewModel
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

@Composable
fun InvitationDetailScreen(
    onBackClick: () -> Unit,
    viewModel: InvitationDetailViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val context = LocalContext.current

    LaunchedEffect(uiState.disableSuccess) {
        if (uiState.disableSuccess) {
            Toast.makeText(context, "邀请已禁用", Toast.LENGTH_SHORT).show()
            onBackClick()
        }
    }

    Scaffold(
        topBar = {
            YouKongTopBar(
                title = "邀请详情",
                onBackClick = onBackClick,
            )
        },
    ) { innerPadding ->
        when {
            uiState.isLoading -> {
                YouKongLoading(message = "加载中...")
            }

            uiState.invitation != null -> {
                val invitation = uiState.invitation!!
                val circle = invitation.circle
                val expiresAt = invitation.expiresAt

                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(innerPadding)
                        .verticalScroll(rememberScrollState())
                        .padding(horizontal = YouKongTheme.spacing.xxl),
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Spacer(modifier = Modifier.height(24.dp))

                    // QR Code Card
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(16.dp),
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.surface,
                        ),
                        elevation = CardDefaults.cardElevation(defaultElevation = 4.dp),
                    ) {
                        Column(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(24.dp),
                            horizontalAlignment = Alignment.CenterHorizontally,
                        ) {
                            if (circle != null) {
                                YouKongEmojiAvatar(
                                    emoji = circle.emoji,
                                    backgroundColor = CircleColors.fromHex(
                                        circle.color ?: "#3B82F6"
                                    ),
                                    size = 64.dp,
                                )
                                Spacer(modifier = Modifier.height(12.dp))
                                Text(
                                    text = circle.name,
                                    style = MaterialTheme.typography.titleMedium,
                                    color = TextPrimary,
                                )
                                Spacer(modifier = Modifier.height(4.dp))
                            }

                            Text(
                                text = invitation.inviter?.nickname ?: "邀请你加入",
                                style = MaterialTheme.typography.bodyMedium,
                                color = TextSecondary,
                            )

                            Spacer(modifier = Modifier.height(20.dp))

                            // QR Code
                            if (uiState.qrCodeUrl != null) {
                                Box(
                                    modifier = Modifier
                                        .size(180.dp)
                                        .clip(RoundedCornerShape(12.dp))
                                        .border(
                                            1.dp,
                                            MaterialTheme.colorScheme.outline.copy(alpha = 0.3f),
                                            RoundedCornerShape(12.dp)
                                        ),
                                    contentAlignment = Alignment.Center,
                                ) {
                                    AsyncImage(
                                        model = uiState.qrCodeUrl,
                                        contentDescription = "邀请二维码",
                                        modifier = Modifier.size(160.dp),
                                    )
                                }
                            } else {
                                Box(
                                    modifier = Modifier
                                        .size(180.dp)
                                        .clip(RoundedCornerShape(12.dp))
                                        .background(MaterialTheme.colorScheme.surfaceVariant),
                                    contentAlignment = Alignment.Center,
                                ) {
                                    Text(
                                        text = "加载中...",
                                        style = MaterialTheme.typography.bodyMedium,
                                        color = TextSecondary,
                                    )
                                }
                            }

                            Spacer(modifier = Modifier.height(16.dp))

                            Text(
                                text = "扫描二维码加入",
                                style = MaterialTheme.typography.bodySmall,
                                color = TextSecondary,
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(24.dp))

                    // Info Card
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(12.dp),
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.surfaceVariant,
                        ),
                    ) {
                        Column(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(16.dp),
                        ) {
                            InfoRow(label = "邀请码", value = invitation.code)

                            HorizontalDivider(
                                modifier = Modifier.padding(vertical = 12.dp),
                                color = MaterialTheme.colorScheme.outline.copy(alpha = 0.2f),
                            )

                            InfoRow(
                                label = "已使用",
                                value = "${invitation.useCount}/${invitation.maxUses}"
                            )

                            HorizontalDivider(
                                modifier = Modifier.padding(vertical = 12.dp),
                                color = MaterialTheme.colorScheme.outline.copy(alpha = 0.2f),
                            )

                            val statusText = when (invitation.status) {
                                InvitationStatus.ACTIVE -> "有效"
                                InvitationStatus.DISABLED -> "已禁用"
                                InvitationStatus.EXPIRED -> "已过期"
                            }
                            InfoRow(label = "状态", value = statusText)

                            if (expiresAt != null) {
                                HorizontalDivider(
                                    modifier = Modifier.padding(vertical = 12.dp),
                                    color = MaterialTheme.colorScheme.outline.copy(alpha = 0.2f),
                                )
                                val localDateTime = expiresAt.toLocalDateTime(
                                    TimeZone.currentSystemDefault()
                                )
                                InfoRow(
                                    label = "有效期至",
                                    value = "${localDateTime.year}/${localDateTime.monthNumber}/${localDateTime.dayOfMonth}"
                                )
                            }
                        }
                    }

                    Spacer(modifier = Modifier.height(24.dp))

                    // Action Buttons
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        OutlinedButton(
                            onClick = {
                                copyToClipboard(context, invitation.inviteUrl)
                                Toast.makeText(context, "链接已复制", Toast.LENGTH_SHORT).show()
                            },
                            modifier = Modifier.weight(1f),
                        ) {
                            Text(text = "复制链接")
                        }

                        YouKongButton(
                            text = "分享",
                            onClick = {
                                shareInvitation(context, invitation.inviteUrl)
                            },
                            modifier = Modifier.weight(1f),
                        )
                    }

                    Spacer(modifier = Modifier.height(16.dp))

                    if (invitation.status == InvitationStatus.ACTIVE) {
                        TextButton(
                            onClick = viewModel::disableInvitation,
                            enabled = !uiState.isDisabling,
                        ) {
                            Text(
                                text = if (uiState.isDisabling) "禁用中..." else "禁用此邀请",
                                color = MaterialTheme.colorScheme.error,
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(32.dp))
                }
            }

            uiState.error != null -> {
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(innerPadding),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = uiState.error!!,
                        style = MaterialTheme.typography.bodyLarge,
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            }
        }
    }
}

@Composable
private fun InfoRow(
    label: String,
    value: String,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = TextSecondary,
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            color = TextPrimary,
        )
    }
}

private fun copyToClipboard(context: Context, text: String) {
    val clipboardManager = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
    val clip = ClipData.newPlainText("邀请链接", text)
    clipboardManager.setPrimaryClip(clip)
}

private fun shareInvitation(context: Context, url: String) {
    val intent = Intent(Intent.ACTION_SEND).apply {
        type = "text/plain"
        putExtra(Intent.EXTRA_TEXT, "邀请你加入有空，一起约个时间吧！\n$url")
    }
    context.startActivity(Intent.createChooser(intent, "分享邀请"))
}
