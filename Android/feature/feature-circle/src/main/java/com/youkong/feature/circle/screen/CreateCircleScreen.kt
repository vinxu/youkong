package com.youkong.feature.circle.screen

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material3.Icon
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
import com.youkong.core.ui.component.YouKongTextField
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.CircleColors
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.White
import com.youkong.core.ui.theme.YouKongTheme
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
            YouKongTopBar(
                title = "创建圈子",
                onBackClick = onBackClick,
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = YouKongTheme.spacing.xxl),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Spacer(modifier = Modifier.height(32.dp))

            // 预览
            YouKongEmojiAvatar(
                emoji = uiState.emoji,
                backgroundColor = CircleColors.fromHex(uiState.selectedColor),
                size = 80.dp,
            )

            Spacer(modifier = Modifier.height(32.dp))

            // 名称输入
            YouKongTextField(
                value = uiState.name,
                onValueChange = viewModel::onNameChange,
                label = "圈子名称",
                placeholder = "给圈子起个名字",
                error = uiState.error,
                modifier = Modifier.fillMaxWidth(),
            )

            Text(
                text = "最多10个字符",
                style = MaterialTheme.typography.bodySmall,
                color = TextSecondary,
                modifier = Modifier
                    .align(Alignment.Start)
                    .padding(top = 4.dp),
            )

            Spacer(modifier = Modifier.height(24.dp))

            // Emoji 选择
            Text(
                text = "选择图标",
                style = MaterialTheme.typography.titleMedium,
                color = TextPrimary,
                modifier = Modifier.align(Alignment.Start),
            )

            Spacer(modifier = Modifier.height(12.dp))

            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                viewModel.availableEmojis.forEach { emoji ->
                    EmojiOption(
                        emoji = emoji,
                        selected = emoji == uiState.emoji,
                        onClick = { viewModel.onEmojiSelect(emoji) },
                    )
                }
            }

            Spacer(modifier = Modifier.height(24.dp))

            // 颜色选择
            Text(
                text = "选择颜色",
                style = MaterialTheme.typography.titleMedium,
                color = TextPrimary,
                modifier = Modifier.align(Alignment.Start),
            )

            Spacer(modifier = Modifier.height(12.dp))

            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                viewModel.availableColors.forEach { color ->
                    ColorOption(
                        color = color,
                        selected = color == uiState.selectedColor,
                        onClick = { viewModel.onColorSelect(color) },
                    )
                }
            }

            Spacer(modifier = Modifier.weight(1f))

            YouKongButton(
                text = "创建圈子",
                onClick = viewModel::createCircle,
                enabled = uiState.name.trim().isNotEmpty() && !uiState.isLoading,
                loading = uiState.isLoading,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 32.dp),
            )
        }
    }
}

@Composable
private fun EmojiOption(
    emoji: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Box(
        modifier = Modifier
            .size(48.dp)
            .clip(CircleShape)
            .background(if (selected) Primary.copy(alpha = 0.15f) else MaterialTheme.colorScheme.surface)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = emoji,
            style = MaterialTheme.typography.titleLarge,
        )
    }
}

@Composable
private fun ColorOption(
    color: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Box(
        modifier = Modifier
            .size(48.dp)
            .clip(CircleShape)
            .background(CircleColors.fromHex(color))
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        if (selected) {
            Icon(
                imageVector = Icons.Default.Check,
                contentDescription = null,
                tint = White,
                modifier = Modifier.size(24.dp),
            )
        }
    }
}
