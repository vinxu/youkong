package com.youkong.feature.availability.screen

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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.LocationType
import com.youkong.core.domain.model.PresetLocations
import com.youkong.core.ui.component.cli.TerminalButton
import com.youkong.core.ui.component.cli.TerminalButtonStyle
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.availability.viewmodel.CreateAvailabilityViewModel

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun SelectLocationScreen(
    viewModel: CreateAvailabilityViewModel,
    onBackClick: () -> Unit,
    onNextClick: () -> Unit,
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = {
            TerminalHeader(
                title = "$ select --location",
                onBackClick = onBackClick
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

            // 灵活选项
            TerminalLocationOption(
                emoji = "🤷",
                title = "地点灵活",
                subtitle = "让朋友来提议见面地点",
                selected = uiState.locationType == LocationType.FLEXIBLE,
                onClick = { viewModel.selectFlexibleLocation() },
            )

            Spacer(modifier = Modifier.height(32.dp))

            // 常去的地方标题
            Text(
                text = "${ASCII.BULLET} 常去的地方",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary,
            )

            Spacer(modifier = Modifier.height(12.dp))

            // 预设地点列表
            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                PresetLocations.all.forEach { location ->
                    val isSelected = uiState.locationType == LocationType.PRESET &&
                            uiState.locationName == location.name

                    Box(
                        modifier = Modifier
                            .clickable { viewModel.selectPresetLocation(location) }
                            .border(
                                width = 1.dp,
                                color = if (isSelected) CLIColors.Green else CLIColors.Border
                            )
                            .background(
                                if (isSelected) CLIColors.Green.copy(alpha = 0.15f)
                                else CLIColors.BackgroundSecondary
                            )
                            .padding(horizontal = 16.dp, vertical = 8.dp)
                    ) {
                        Text(
                            text = "${location.emoji} ${location.name}",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 14.sp,
                            color = if (isSelected) CLIColors.Green else CLIColors.TextPrimary,
                        )
                    }
                }
            }

            // 已选地点预览
            if (uiState.locationType != LocationType.FLEXIBLE && uiState.locationName != null) {
                Spacer(modifier = Modifier.height(24.dp))

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

                        Text(
                            text = uiState.locationName!!,
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
                text = "${ASCII.ARROW_RIGHT} 下一步：选择圈子",
                onClick = {
                    viewModel.nextStep()
                    onNextClick()
                },
                enabled = true,
                modifier = Modifier.fillMaxWidth(),
                style = TerminalButtonStyle.PRIMARY
            )

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@Composable
private fun TerminalLocationOption(
    emoji: String,
    title: String,
    subtitle: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .border(
                width = if (selected) 2.dp else 1.dp,
                color = if (selected) CLIColors.Green else CLIColors.Border
            )
            .background(
                if (selected) CLIColors.Green.copy(alpha = 0.1f)
                else CLIColors.BackgroundSecondary
            )
            .padding(16.dp)
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = emoji,
                fontSize = 24.sp,
            )

            Spacer(modifier = Modifier.padding(8.dp))

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = title,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.TextPrimary,
                )
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = subtitle,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary,
                )
            }

            if (selected) {
                Text(
                    text = "✓",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 18.sp,
                    color = CLIColors.Green,
                )
            }
        }
    }
}
