package com.youkong.feature.invitation.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.cli.TerminalButton
import com.youkong.core.ui.component.cli.TerminalButtonStyle
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
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

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
    ) {
        TerminalHeader(
            title = "CREATE_INVITATION",
            showBackButton = true,
            onBackClick = onBackClick,
        )

        when {
            uiState.isLoading -> {
                YouKongLoading(message = "Loading...")
            }

            else -> {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(16.dp),
                ) {
                    // 分隔线
                    Text(
                        text = ASCII.titleLine("INVITE_FRIEND", totalLength = 40),
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.Border,
                    )

                    val circle = uiState.circle
                    if (circle != null) {
                        Spacer(modifier = Modifier.height(16.dp))

                        // 圈子信息
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .background(CLIColors.BackgroundSecondary)
                                .border(width = 1.dp, color = CLIColors.Border)
                                .padding(12.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(
                                text = ASCII.BULLET,
                                fontFamily = FontFamily.Monospace,
                                fontSize = 14.sp,
                                color = CLIColors.Green,
                            )

                            Spacer(modifier = Modifier.width(8.dp))

                            Text(
                                text = circle.emoji,
                                fontSize = 16.sp,
                            )

                            Spacer(modifier = Modifier.width(8.dp))

                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = circle.name,
                                    fontFamily = FontFamily.Monospace,
                                    fontSize = 14.sp,
                                    color = CLIColors.TextPrimary,
                                )
                                Text(
                                    text = "${circle.memberCount} members",
                                    fontFamily = FontFamily.Monospace,
                                    fontSize = 12.sp,
                                    color = CLIColors.TextSecondary,
                                )
                            }
                        }
                    }

                    Spacer(modifier = Modifier.height(24.dp))

                    // 有效期标题
                    Text(
                        text = "${ASCII.ARROW_RIGHT} Expiration (days)",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 13.sp,
                        color = CLIColors.TextPrimary,
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

                    Spacer(modifier = Modifier.height(24.dp))

                    // 最大使用次数标题
                    Text(
                        text = "${ASCII.ARROW_RIGHT} Max uses",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 13.sp,
                        color = CLIColors.TextPrimary,
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

                    // 错误信息
                    val error = uiState.error
                    if (error != null) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(bottom = 16.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(
                                text = ASCII.CROSS,
                                fontFamily = FontFamily.Monospace,
                                fontSize = 14.sp,
                                color = CLIColors.Red,
                            )
                            Spacer(modifier = Modifier.width(8.dp))
                            Text(
                                text = error,
                                fontFamily = FontFamily.Monospace,
                                fontSize = 13.sp,
                                color = CLIColors.Red,
                            )
                        }
                    }

                    // 生成按钮
                    TerminalButton(
                        text = if (uiState.isCreating) "GENERATING..." else "GENERATE_INVITATION",
                        onClick = viewModel::createInvitation,
                        enabled = !uiState.isCreating,
                        style = TerminalButtonStyle.PRIMARY,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(bottom = 16.dp),
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
                .background(
                    if (isSelected) CLIColors.Green.copy(alpha = 0.15f)
                    else CLIColors.BackgroundSecondary
                )
                .border(
                    width = 1.dp,
                    color = if (isSelected) CLIColors.Green else CLIColors.Border
                )
                .clickable { onSelect(day) }
                .padding(horizontal = 16.dp, vertical = 8.dp),
        ) {
            Text(
                text = if (isSelected) "${ASCII.CHECKMARK} $day" else "$day",
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = if (isSelected) CLIColors.Green else CLIColors.TextPrimary,
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
                .background(
                    if (isSelected) CLIColors.Green.copy(alpha = 0.15f)
                    else CLIColors.BackgroundSecondary
                )
                .border(
                    width = 1.dp,
                    color = if (isSelected) CLIColors.Green else CLIColors.Border
                )
                .clickable { onSelect(use) }
                .padding(horizontal = 16.dp, vertical = 8.dp),
        ) {
            Text(
                text = if (isSelected) "${ASCII.CHECKMARK} $use" else "$use",
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = if (isSelected) CLIColors.Green else CLIColors.TextPrimary,
            )
        }
    }
}
