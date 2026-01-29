package com.youkong.feature.availability.screen

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.cli.TerminalButton
import com.youkong.core.ui.component.cli.TerminalButtonStyle
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.availability.viewmodel.CreateAvailabilityViewModel

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun SelectTimeScreen(
    viewModel: CreateAvailabilityViewModel,
    onCloseClick: () -> Unit,
    onNextClick: () -> Unit,
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = {
            TerminalHeader(
                title = "$ select --time",
                onCloseClick = onCloseClick
            )
        },
        containerColor = CLIColors.Background
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp),
        ) {
            Spacer(modifier = Modifier.height(24.dp))

            // 快捷选择标题
            Text(
                text = "${ASCII.BULLET} 快捷选择",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary,
            )

            Spacer(modifier = Modifier.height(12.dp))

            // 快捷时间选项
            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                TerminalQuickTimeChip(
                    label = "现在",
                    emoji = "⏰",
                    selected = uiState.quickTimeOption == "now",
                    onClick = { viewModel.selectQuickTime("now") },
                )
                TerminalQuickTimeChip(
                    label = "今晚",
                    emoji = "🌙",
                    selected = uiState.quickTimeOption == "tonight",
                    onClick = { viewModel.selectQuickTime("tonight") },
                )
                TerminalQuickTimeChip(
                    label = "明天",
                    emoji = "☀️",
                    selected = uiState.quickTimeOption == "tomorrow",
                    onClick = { viewModel.selectQuickTime("tomorrow") },
                )
            }

            Spacer(modifier = Modifier.height(32.dp))

            // 自定义时间标题
            Text(
                text = "${ASCII.BULLET} 自定义时间",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary,
            )

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = "点击上方快捷选项，或使用日期时间选择器",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextWeak,
            )

            // 已选时间预览
            if (uiState.selectedDate != null && uiState.startTime != null && uiState.endTime != null) {
                Spacer(modifier = Modifier.height(24.dp))

                // CLI 风格时间显示框
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .border(1.dp, CLIColors.Green)
                        .background(CLIColors.BackgroundSecondary)
                        .padding(16.dp)
                ) {
                    Column {
                        Text(
                            text = "${ASCII.ARROW_RIGHT} 已选择",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.Green,
                        )

                        Spacer(modifier = Modifier.height(8.dp))

                        val dateStr = "${uiState.selectedDate!!.monthNumber}月${uiState.selectedDate!!.dayOfMonth}日"
                        val startStr = "${uiState.startTime!!.hour}:${uiState.startTime!!.minute.toString().padStart(2, '0')}"
                        val endStr = "${uiState.endTime!!.hour}:${uiState.endTime!!.minute.toString().padStart(2, '0')}"

                        Text(
                            text = "$dateStr $startStr - $endStr",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 16.sp,
                            color = CLIColors.TextPrimary,
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(32.dp))

            // 下一步按钮
            TerminalButton(
                text = "${ASCII.ARROW_RIGHT} 下一步：选择地点",
                onClick = {
                    viewModel.nextStep()
                    onNextClick()
                },
                enabled = viewModel.canProceedToNextStep(),
                modifier = Modifier.fillMaxWidth(),
                style = TerminalButtonStyle.PRIMARY
            )

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@Composable
private fun TerminalQuickTimeChip(
    label: String,
    emoji: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Box(
        modifier = Modifier
            .clickable(onClick = onClick)
            .border(
                width = 1.dp,
                color = if (selected) CLIColors.Green else CLIColors.Border
            )
            .background(
                if (selected) CLIColors.Green.copy(alpha = 0.15f)
                else CLIColors.BackgroundSecondary
            )
            .padding(horizontal = 16.dp, vertical = 8.dp)
    ) {
        Text(
            text = "$emoji $label",
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = if (selected) CLIColors.Green else CLIColors.TextPrimary,
        )
    }
}
