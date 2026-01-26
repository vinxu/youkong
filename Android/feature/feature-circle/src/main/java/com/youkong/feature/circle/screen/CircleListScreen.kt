package com.youkong.feature.circle.screen

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.KeyboardArrowRight
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.Circle
import com.youkong.core.ui.component.YouKongEmptyState
import com.youkong.core.ui.component.YouKongEmojiAvatar
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.CircleColors
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.circle.viewmodel.CircleListViewModel

@Composable
fun CircleListScreen(
    onBackClick: () -> Unit,
    onCreateCircleClick: () -> Unit,
    onCircleClick: (String) -> Unit,
    viewModel: CircleListViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = {
            YouKongTopBar(
                title = "我的圈子",
                onBackClick = onBackClick,
                actions = {
                    IconButton(onClick = onCreateCircleClick) {
                        Icon(
                            imageVector = Icons.Default.Add,
                            contentDescription = "创建圈子",
                        )
                    }
                },
            )
        },
    ) { innerPadding ->
        when {
            uiState.isLoading -> {
                YouKongLoading(message = "加载中...")
            }

            uiState.circles.isEmpty() -> {
                YouKongEmptyState(
                    emoji = "\uD83D\uDC65",
                    title = "还没有圈子",
                    subtitle = "创建一个圈子，添加你的朋友",
                    modifier = Modifier.padding(innerPadding),
                )
            }

            else -> {
                LazyColumn(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(innerPadding),
                    contentPadding = PaddingValues(vertical = YouKongTheme.spacing.md),
                ) {
                    items(
                        items = uiState.circles,
                        key = { it.id },
                    ) { circle ->
                        CircleItem(
                            circle = circle,
                            onClick = { onCircleClick(circle.id) },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun CircleItem(
    circle: Circle,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = YouKongTheme.spacing.xxl, vertical = YouKongTheme.spacing.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        YouKongEmojiAvatar(
            emoji = circle.emoji,
            backgroundColor = CircleColors.fromHex(circle.color),
            size = 48.dp,
        )

        Spacer(modifier = Modifier.width(16.dp))

        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = circle.name,
                style = MaterialTheme.typography.titleMedium,
                color = TextPrimary,
            )
        }

        Icon(
            imageVector = Icons.Default.KeyboardArrowRight,
            contentDescription = null,
            tint = TextSecondary,
        )
    }
}
