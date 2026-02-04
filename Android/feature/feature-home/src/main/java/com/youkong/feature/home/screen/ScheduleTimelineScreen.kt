package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.ExperimentalMaterialApi
import androidx.compose.material.pullrefresh.PullRefreshIndicator
import androidx.compose.material.pullrefresh.pullRefresh
import androidx.compose.material.pullrefresh.rememberPullRefreshState
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.network.model.ScheduleGroup
import com.youkong.core.network.model.ScheduleItem
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.home.viewmodel.ScheduleTimelineViewModel
import kotlinx.coroutines.launch

/**
 * 我的状态时刻表页面
 *
 * CLI 风格，按日期分组显示历史时刻表
 */
@OptIn(ExperimentalMaterialApi::class)
@Composable
fun ScheduleTimelineScreen(
    onDismiss: () -> Unit,
    viewModel: ScheduleTimelineViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val listState = rememberLazyListState()
    val coroutineScope = rememberCoroutineScope()

    val pullRefreshState = rememberPullRefreshState(
        refreshing = uiState.isRefreshing,
        onRefresh = { viewModel.refresh() }
    )

    // 首次加载
    LaunchedEffect(Unit) {
        viewModel.loadInitialData()
    }

    // 滚动到底部触发加载更多
    LaunchedEffect(listState) {
        snapshotFlow { listState.layoutInfo.visibleItemsInfo.lastOrNull()?.index }
            .collect { lastVisibleIndex ->
                if (lastVisibleIndex != null && lastVisibleIndex >= uiState.scheduleGroups.size - 2) {
                    viewModel.loadMore()
                }
            }
    }

    // 首次加载完成后滚动到底部（最新）
    LaunchedEffect(uiState.scheduleGroups.size) {
        if (uiState.scheduleGroups.isNotEmpty() && !uiState.isLoading) {
            coroutineScope.launch {
                listState.animateScrollToItem(uiState.scheduleGroups.size - 1)
            }
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
    ) {
        // CLI 标题栏
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            TextButton(onClick = onDismiss) {
                Text(
                    text = "[X]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary
                )
            }

            Spacer(modifier = Modifier.weight(1f))

            Text(
                text = "━━ 我的状态时刻表 ━━",
                fontFamily = FontFamily.Monospace,
                fontSize = 16.sp,
                color = CLIColors.Green
            )

            Spacer(modifier = Modifier.weight(1f))

            // 占位，保持标题居中
            Box(modifier = Modifier.width(40.dp))
        }

        Divider(color = CLIColors.Border)

        // 内容区域
        Box(
            modifier = Modifier
                .weight(1f)
                .pullRefresh(pullRefreshState)
        ) {
            when {
                uiState.isLoading && uiState.scheduleGroups.isEmpty() -> {
                    LoadingView()
                }

                uiState.isEmpty -> {
                    EmptyView()
                }

                else -> {
                    LazyColumn(
                        state = listState,
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(0.dp)
                    ) {
                        // 加载更多指示器（顶部）
                        if (uiState.hasMore) {
                            item(key = "load_more_top") {
                                LoadMoreIndicator(
                                    isLoading = uiState.isLoadingMore
                                )
                            }
                        }

                        // 时刻表分组（倒序，最新在底部）
                        items(
                            items = uiState.scheduleGroups.reversed(),
                            key = { it.id }
                        ) { group ->
                            ScheduleGroupView(
                                group = group,
                                isItemExecuted = { viewModel.isItemExecuted(it, group) },
                                isItemActive = { viewModel.isItemActive(it, group) }
                            )
                        }
                    }
                }
            }

            PullRefreshIndicator(
                refreshing = uiState.isRefreshing,
                state = pullRefreshState,
                modifier = Modifier.align(Alignment.TopCenter),
                backgroundColor = CLIColors.BackgroundSecondary,
                contentColor = CLIColors.Green
            )
        }
    }
}

@Composable
private fun LoadingView() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "[",
                    fontFamily = FontFamily.Monospace,
                    color = CLIColors.Border
                )
                CircularProgressIndicator(
                    modifier = Modifier.size(16.dp),
                    strokeWidth = 2.dp,
                    color = CLIColors.Green
                )
                Text(
                    text = "]",
                    fontFamily = FontFamily.Monospace,
                    color = CLIColors.Border
                )
            }

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = "加载中...",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.Yellow
            )
        }
    }
}

@Composable
private fun EmptyView() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Text(
                text = """
                    ┌─────────────────┐
                    │                 │
                    │      📋        │
                    │                 │
                    └─────────────────┘
                """.trimIndent(),
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.Border,
                textAlign = TextAlign.Center
            )

            Text(
                text = "> 暂无时刻表",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary
            )

            Text(
                text = "  用语音创建你的状态时刻表吧",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextWeak
            )
        }
    }
}

@Composable
private fun LoadMoreIndicator(isLoading: Boolean) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 16.dp),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically
    ) {
        if (isLoading) {
            Text(
                text = "[",
                fontFamily = FontFamily.Monospace,
                color = CLIColors.Border
            )
            CircularProgressIndicator(
                modifier = Modifier.size(14.dp),
                strokeWidth = 2.dp,
                color = CLIColors.TextSecondary
            )
            Text(
                text = "]",
                fontFamily = FontFamily.Monospace,
                color = CLIColors.Border
            )
            Spacer(modifier = Modifier.width(4.dp))
            Text(
                text = "加载更多...",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary
            )
        } else {
            Text(
                text = "[ 向上滚动加载更多历史 ]",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextWeak
            )
        }
    }
}

@Composable
private fun ScheduleGroupView(
    group: ScheduleGroup,
    isItemExecuted: (ScheduleItem) -> Boolean,
    isItemActive: (ScheduleItem) -> Boolean
) {
    Column(
        modifier = Modifier.padding(vertical = 8.dp)
    ) {
        // 日期分隔线
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = "──",
                fontFamily = FontFamily.Monospace,
                color = CLIColors.Border
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = group.displayDate,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = if (group.isCurrentOrFuture) CLIColors.Green else CLIColors.TextSecondary
            )
            if (group.status == "active") {
                Spacer(modifier = Modifier.width(4.dp))
                Text(
                    text = "(进行中)",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 10.sp,
                    color = CLIColors.Yellow
                )
            }
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = "──",
                fontFamily = FontFamily.Monospace,
                color = CLIColors.Border
            )
        }

        Spacer(modifier = Modifier.height(8.dp))

        // 时刻表条目
        group.items.forEachIndexed { index, item ->
            val executed = isItemExecuted(item)
            val active = isItemActive(item)

            ScheduleItemView(
                item = item,
                isExecuted = executed,
                isActive = active
            )

            // 连接线
            if (index < group.items.size - 1) {
                ConnectorLine(isExecuted = executed)
            }
        }
    }
}

@Composable
private fun ScheduleItemView(
    item: ScheduleItem,
    isExecuted: Boolean,
    isActive: Boolean
) {
    val borderColor = when {
        isActive -> CLIColors.Green
        isExecuted -> CLIColors.TextWeak
        else -> CLIColors.Border
    }

    val textColor = when {
        isActive -> CLIColors.TextPrimary
        isExecuted -> CLIColors.TextWeak
        else -> CLIColors.TextSecondary
    }

    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        // 时间
        Text(
            text = "${item.startTime}-${item.endTime}",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = textColor,
            modifier = Modifier.width(80.dp),
            textAlign = TextAlign.End
        )

        // 状态卡片
        Row(
            modifier = Modifier
                .weight(1f)
                .background(
                    color = CLIColors.BackgroundSecondary,
                    shape = RoundedCornerShape(4.dp)
                )
                .border(
                    width = if (isActive) 2.dp else 1.dp,
                    color = borderColor,
                    shape = RoundedCornerShape(4.dp)
                )
                .padding(horizontal = 12.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Text(
                text = item.emoji,
                fontSize = 20.sp
            )

            Text(
                text = item.status,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = textColor,
                modifier = Modifier.weight(1f),
                maxLines = 1
            )

            // 状态指示
            when {
                isActive -> {
                    Text(
                        text = "[NOW]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 10.sp,
                        color = CLIColors.Green
                    )
                }
                isExecuted -> {
                    Text(
                        text = "[DONE]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 10.sp,
                        color = CLIColors.TextWeak
                    )
                }
            }
        }
    }
}

@Composable
private fun ConnectorLine(isExecuted: Boolean) {
    Column(
        modifier = Modifier.padding(start = 80.dp + 8.dp + 45.dp)
    ) {
        Text(
            text = "│",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = if (isExecuted) CLIColors.TextWeak else CLIColors.Border
        )
        Text(
            text = "│",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = if (isExecuted) CLIColors.TextWeak else CLIColors.Border
        )
    }
}
