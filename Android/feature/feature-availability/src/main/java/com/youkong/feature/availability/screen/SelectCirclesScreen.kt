package com.youkong.feature.availability.screen

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
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
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Check
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.Circle
import com.youkong.core.ui.component.YouKongButton
import com.youkong.core.ui.component.YouKongEmptyState
import com.youkong.core.ui.component.YouKongEmojiAvatar
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.Border
import com.youkong.core.ui.theme.CircleColors
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongShapes
import com.youkong.core.ui.theme.YouKongTheme
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
            YouKongTopBar(
                title = "选择圈子",
                onBackClick = onBackClick,
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            Text(
                text = "选择可以看到这条有空的圈子",
                style = MaterialTheme.typography.bodyMedium,
                color = TextSecondary,
                modifier = Modifier.padding(horizontal = YouKongTheme.spacing.xxl, vertical = YouKongTheme.spacing.md),
            )

            when {
                uiState.circles.isEmpty() -> {
                    YouKongEmptyState(
                        emoji = "\uD83D\uDC65",
                        title = "还没有圈子",
                        subtitle = "先创建一个圈子，添加你的朋友",
                    )
                }

                else -> {
                    LazyColumn(
                        modifier = Modifier.weight(1f),
                    ) {
                        items(
                            items = uiState.circles,
                            key = { it.id },
                        ) { circle ->
                            CircleItem(
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
                    text = uiState.error!!,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.padding(horizontal = YouKongTheme.spacing.xxl),
                )
                Spacer(modifier = Modifier.height(8.dp))
            }

            // 发布按钮
            YouKongButton(
                text = if (uiState.selectedCircleIds.isEmpty()) {
                    "请选择至少一个圈子"
                } else {
                    "发布有空 (${uiState.selectedCircleIds.size}个圈子可见)"
                },
                onClick = { viewModel.createAvailability() },
                enabled = uiState.selectedCircleIds.isNotEmpty() && !uiState.isLoading,
                loading = uiState.isLoading,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = YouKongTheme.spacing.xxl)
                    .padding(bottom = 32.dp),
            )
        }
    }
}

@Composable
private fun CircleItem(
    circle: Circle,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = YouKongTheme.spacing.lg, vertical = YouKongTheme.spacing.sm),
        shape = YouKongShapes.medium,
        color = if (selected) Primary.copy(alpha = 0.1f) else androidx.compose.ui.graphics.Color.Transparent,
    ) {
        Row(
            modifier = Modifier.padding(YouKongTheme.spacing.md),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            YouKongEmojiAvatar(
                emoji = circle.emoji,
                backgroundColor = CircleColors.fromHex(circle.color),
                size = 44.dp,
            )

            Spacer(modifier = Modifier.width(16.dp))

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = circle.name,
                    style = MaterialTheme.typography.titleMedium,
                    color = TextPrimary,
                )
            }

            if (selected) {
                Icon(
                    imageVector = Icons.Default.CheckCircle,
                    contentDescription = null,
                    tint = Primary,
                    modifier = Modifier.size(24.dp),
                )
            }
        }
    }
}
