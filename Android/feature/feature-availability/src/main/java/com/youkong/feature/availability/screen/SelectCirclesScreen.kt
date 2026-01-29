package com.youkong.feature.availability.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.MaterialTheme
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
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.Circle
import com.youkong.core.ui.component.cli.TerminalButton
import com.youkong.core.ui.component.cli.TerminalButtonStyle
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.availability.viewmodel.CreateAvailabilityViewModel

@Composable
fun SelectCirclesScreen(
    viewModel: CreateAvailabilityViewModel,
    onBackClick: () -> Unit,
    onCreateSuccess: () -> Unit,
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
                title = "$ select --circles",
                onBackClick = onBackClick
            )
        },
        containerColor = CLIColors.Background
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            // 提示文本
            Text(
                text = "${ASCII.BULLET} 选择可以看到这条有空的圈子",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 16.dp),
            )

            when {
                uiState.circles.isEmpty() -> {
                    // 空状态
                    Box(
                        modifier = Modifier
                            .weight(1f)
                            .fillMaxWidth(),
                        contentAlignment = Alignment.Center
                    ) {
                        Column(
                            horizontalAlignment = Alignment.CenterHorizontally
                        ) {
                            Text(
                                text = "👥",
                                fontSize = 48.sp,
                            )
                            Spacer(modifier = Modifier.height(16.dp))
                            Text(
                                text = "还没有圈子",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 16.sp,
                                color = CLIColors.TextPrimary,
                            )
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(
                                text = "先创建一个圈子，添加你的朋友",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 12.sp,
                                color = CLIColors.TextSecondary,
                            )
                        }
                    }
                }

                else -> {
                    LazyColumn(
                        modifier = Modifier.weight(1f),
                    ) {
                        items(
                            items = uiState.circles,
                            key = { it.id },
                        ) { circle ->
                            TerminalCircleItem(
                                circle = circle,
                                selected = circle.id in uiState.selectedCircleIds,
                                onClick = { viewModel.toggleCircle(circle.id) },
                            )
                        }
                    }
                }
            }

            // 错误提示
            if (uiState.error != null) {
                Text(
                    text = "${ASCII.CROSS} ${uiState.error}",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Red,
                    modifier = Modifier.padding(horizontal = 16.dp),
                )
                Spacer(modifier = Modifier.height(8.dp))
            }

            // 发布按钮
            Column(
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 16.dp)
            ) {
                if (uiState.isLoading) {
                    Text(
                        text = "${ASCII.LOADING} 发布中...",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = CLIColors.Yellow,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 12.dp)
                    )
                }

                TerminalButton(
                    text = if (uiState.selectedCircleIds.isEmpty()) {
                        "请选择至少一个圈子"
                    } else {
                        "${ASCII.ARROW_RIGHT} 发布有空 (${uiState.selectedCircleIds.size}个圈子可见)"
                    },
                    onClick = { viewModel.createAvailability() },
                    enabled = uiState.selectedCircleIds.isNotEmpty() && !uiState.isLoading,
                    modifier = Modifier.fillMaxWidth(),
                    style = TerminalButtonStyle.PRIMARY
                )
            }
        }
    }
}

@Composable
private fun TerminalCircleItem(
    circle: Circle,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 4.dp)
            .background(
                if (selected) CLIColors.Green.copy(alpha = 0.1f)
                else CLIColors.BackgroundSecondary
            )
            .border(
                width = 1.dp,
                color = if (selected) CLIColors.Green else CLIColors.Border
            )
            .padding(12.dp)
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // Emoji 头像
            Text(
                text = circle.emoji,
                fontSize = 24.sp,
            )

            Spacer(modifier = Modifier.width(12.dp))

            // 圈子名称
            Text(
                text = circle.name,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextPrimary,
                modifier = Modifier.weight(1f)
            )

            // 选择标记
            if (selected) {
                Text(
                    text = "[✓]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.Green,
                )
            } else {
                Text(
                    text = "[ ]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.TextWeak,
                )
            }
        }
    }
}
