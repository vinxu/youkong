package com.youkong.feature.circle.screen

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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Scaffold
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
import com.youkong.core.ui.component.cli.TerminalButton
import com.youkong.core.ui.component.cli.TerminalButtonStyle
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.component.cli.TerminalTextField
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
import com.youkong.core.ui.theme.CircleColors
import com.youkong.feature.circle.viewmodel.CreateCircleViewModel

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun CreateCircleScreen(
    onBackClick: () -> Unit,
    onCreateSuccess: () -> Unit,
    viewModel: CreateCircleViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    LaunchedEffect(uiState.isCreated) {
        if (uiState.isCreated) {
            onCreateSuccess()
        }
    }

    Scaffold(
        topBar = {
            TerminalHeader(
                title = "$ create --circle",
                showBackButton = true,
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
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Spacer(modifier = Modifier.height(24.dp))

            // 预览 - CLI 风格emoji显示
            Box(
                modifier = Modifier
                    .size(80.dp)
                    .border(2.dp, CLIColors.Green)
                    .background(CLIColors.BackgroundSecondary),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = uiState.emoji,
                    fontSize = 40.sp,
                )
            }

            Spacer(modifier = Modifier.height(24.dp))

            // 名称输入
            Text(
                text = "${ASCII.ARROW_RIGHT} 圈子名称",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary,
                modifier = Modifier.align(Alignment.Start)
            )

            Spacer(modifier = Modifier.height(8.dp))

            TerminalTextField(
                value = uiState.name,
                onValueChange = viewModel::onNameChange,
                placeholder = "给圈子起个名字",
                modifier = Modifier.fillMaxWidth(),
            )

            if (uiState.error != null) {
                Text(
                    text = "${ASCII.CROSS} ${uiState.error}",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Red,
                    modifier = Modifier
                        .align(Alignment.Start)
                        .padding(top = 4.dp),
                )
            } else {
                Text(
                    text = "${ASCII.PROMPT} 最多10个字符",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextWeak,
                    modifier = Modifier
                        .align(Alignment.Start)
                        .padding(top = 4.dp),
                )
            }

            Spacer(modifier = Modifier.height(24.dp))

            // Emoji 选择
            Text(
                text = "${ASCII.ARROW_RIGHT} 选择图标",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary,
                modifier = Modifier.align(Alignment.Start),
            )

            Spacer(modifier = Modifier.height(12.dp))

            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                viewModel.availableEmojis.forEach { emoji ->
                    TerminalEmojiOption(
                        emoji = emoji,
                        selected = emoji == uiState.emoji,
                        onClick = { viewModel.onEmojiSelect(emoji) },
                    )
                }
            }

            Spacer(modifier = Modifier.height(24.dp))

            // 颜色选择
            Text(
                text = "${ASCII.ARROW_RIGHT} 选择颜色",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary,
                modifier = Modifier.align(Alignment.Start),
            )

            Spacer(modifier = Modifier.height(12.dp))

            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                viewModel.availableColors.forEach { color ->
                    TerminalColorOption(
                        color = color,
                        selected = color == uiState.selectedColor,
                        onClick = { viewModel.onColorSelect(color) },
                    )
                }
            }

            Spacer(modifier = Modifier.height(32.dp))

            // 创建按钮
            TerminalButton(
                text = if (uiState.isLoading) "CREATING..." else "${ASCII.ARROW_RIGHT} CREATE_CIRCLE",
                onClick = viewModel::createCircle,
                enabled = uiState.name.trim().isNotEmpty() && !uiState.isLoading,
                modifier = Modifier.fillMaxWidth(),
                style = TerminalButtonStyle.PRIMARY
            )

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@Composable
private fun TerminalEmojiOption(
    emoji: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Box(
        modifier = Modifier
            .size(48.dp)
            .border(
                width = if (selected) 2.dp else 1.dp,
                color = if (selected) CLIColors.Green else CLIColors.Border
            )
            .background(
                if (selected) CLIColors.Green.copy(alpha = 0.15f)
                else CLIColors.BackgroundSecondary
            )
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = emoji,
            fontSize = 24.sp,
        )
    }
}

@Composable
private fun TerminalColorOption(
    color: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Box(
        modifier = Modifier
            .size(48.dp)
            .border(
                width = if (selected) 2.dp else 1.dp,
                color = if (selected) CLIColors.Green else CLIColors.Border
            )
            .background(CircleColors.fromHex(color))
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        if (selected) {
            Text(
                text = "✓",
                fontFamily = FontFamily.Monospace,
                fontSize = 24.sp,
                color = CLIColors.TextPrimary,
            )
        }
    }
}
