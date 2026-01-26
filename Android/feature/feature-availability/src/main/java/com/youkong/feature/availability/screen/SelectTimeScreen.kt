package com.youkong.feature.availability.screen

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.YouKongButton
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongTheme
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
            YouKongTopBar(
                title = "选择时间",
                onCloseClick = onCloseClick,
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = YouKongTheme.spacing.xxl),
        ) {
            Spacer(modifier = Modifier.height(24.dp))

            Text(
                text = "快捷选择",
                style = MaterialTheme.typography.titleMedium,
                color = TextPrimary,
            )

            Spacer(modifier = Modifier.height(16.dp))

            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                QuickTimeChip(
                    label = "现在",
                    emoji = "\u23F0",
                    selected = uiState.quickTimeOption == "now",
                    onClick = { viewModel.selectQuickTime("now") },
                )
                QuickTimeChip(
                    label = "今晚",
                    emoji = "\uD83C\uDF19",
                    selected = uiState.quickTimeOption == "tonight",
                    onClick = { viewModel.selectQuickTime("tonight") },
                )
                QuickTimeChip(
                    label = "明天",
                    emoji = "\u2600\uFE0F",
                    selected = uiState.quickTimeOption == "tomorrow",
                    onClick = { viewModel.selectQuickTime("tomorrow") },
                )
            }

            Spacer(modifier = Modifier.height(32.dp))

            Text(
                text = "自定义时间",
                style = MaterialTheme.typography.titleMedium,
                color = TextPrimary,
            )

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = "点击上方快捷选项，或者使用日期时间选择器",
                style = MaterialTheme.typography.bodyMedium,
                color = TextSecondary,
            )

            // 已选时间预览
            if (uiState.selectedDate != null && uiState.startTime != null && uiState.endTime != null) {
                Spacer(modifier = Modifier.height(24.dp))

                Text(
                    text = "已选择",
                    style = MaterialTheme.typography.titleSmall,
                    color = TextSecondary,
                )

                Spacer(modifier = Modifier.height(8.dp))

                val dateStr = "${uiState.selectedDate!!.monthNumber}月${uiState.selectedDate!!.dayOfMonth}日"
                val startStr = "${uiState.startTime!!.hour}:${uiState.startTime!!.minute.toString().padStart(2, '0')}"
                val endStr = "${uiState.endTime!!.hour}:${uiState.endTime!!.minute.toString().padStart(2, '0')}"

                Text(
                    text = "$dateStr $startStr - $endStr",
                    style = MaterialTheme.typography.headlineSmall,
                    color = Primary,
                )
            }

            Spacer(modifier = Modifier.weight(1f))

            YouKongButton(
                text = "下一步：选择地点",
                onClick = {
                    viewModel.nextStep()
                    onNextClick()
                },
                enabled = viewModel.canProceedToNextStep(),
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 32.dp),
            )
        }
    }
}

@Composable
private fun QuickTimeChip(
    label: String,
    emoji: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    FilterChip(
        selected = selected,
        onClick = onClick,
        label = {
            Text("$emoji $label")
        },
        colors = FilterChipDefaults.filterChipColors(
            selectedContainerColor = Primary.copy(alpha = 0.15f),
            selectedLabelColor = Primary,
        ),
        border = if (selected) {
            BorderStroke(1.dp, Primary)
        } else {
            FilterChipDefaults.filterChipBorder(enabled = true, selected = false)
        },
    )
}
