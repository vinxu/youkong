package com.youkong.feature.home.screen

import android.content.Intent
import android.provider.Settings
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.google.accompanist.swiperefresh.SwipeRefresh
import com.google.accompanist.swiperefresh.SwipeRefreshIndicator
import com.google.accompanist.swiperefresh.rememberSwipeRefreshState
import com.youkong.core.network.model.AIReadyDetails
import com.youkong.core.network.model.ScheduleGroup
import com.youkong.core.network.model.ScheduleItem
import com.youkong.core.ui.theme.CLIColors
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import com.youkong.feature.home.viewmodel.ScheduleTimelineUiState
import com.youkong.feature.home.viewmodel.ScheduleTimelineViewModel
import kotlinx.coroutines.launch

/**
 * 我的状态时刻表页面（带标题栏，用于独立展示）
 */
@Composable
fun ScheduleTimelineScreen(
    onDismiss: () -> Unit,
    viewModel: ScheduleTimelineViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val listState = rememberLazyListState()
    val coroutineScope = rememberCoroutineScope()

    val swipeRefreshState = rememberSwipeRefreshState(
        isRefreshing = uiState.isRefreshing
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
                text = "━━ 我的状态表 ━━",
                fontFamily = FontFamily.Monospace,
                fontSize = 16.sp,
                color = CLIColors.Green
            )

            Spacer(modifier = Modifier.weight(1f))

            // 占位，保持标题居中
            Box(modifier = Modifier.width(40.dp))
        }

        HorizontalDivider(color = CLIColors.Border)

        // 内容区域（无 AI 推测开关）
        ScheduleTimelineListContent(
            uiState = uiState,
            viewModel = viewModel,
            listState = listState,
            swipeRefreshState = swipeRefreshState
        )
    }

    // 编辑 Sheet
    if (uiState.showEditSheet) {
        ScheduleEditSheet(
            uiState = uiState,
            onDismiss = { viewModel.dismissEdit() },
            onEmojiChange = { viewModel.updateEditEmoji(it) },
            onStatusChange = { viewModel.updateEditStatus(it) },
            onHighlightChange = { viewModel.updateEditHighlight(it) },
            onAdjustStartTime = { viewModel.adjustStartTime(it) },
            onAdjustEndTime = { viewModel.adjustEndTime(it) },
            onRemindBeforeChange = { viewModel.updateEditRemindBefore(it) },
            onSave = { viewModel.saveEdit() },
        )
    }

    // 删除确认弹窗
    if (uiState.showDeleteConfirm) {
        DeleteConfirmDialog(
            item = uiState.deletingItem,
            onConfirm = { viewModel.deleteItem() },
            onDismiss = { viewModel.dismissDeleteConfirm() }
        )
    }
}

/**
 * 状态时刻表内容（可嵌入，无标题栏）
 */
@Composable
fun ScheduleTimelineContent(
    viewModel: ScheduleTimelineViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val listState = rememberLazyListState()
    val coroutineScope = rememberCoroutineScope()

    val swipeRefreshState = rememberSwipeRefreshState(
        isRefreshing = uiState.isRefreshing
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

    ScheduleTimelineListContent(
        uiState = uiState,
        viewModel = viewModel,
        listState = listState,
        swipeRefreshState = swipeRefreshState
    )

    // 编辑 Sheet
    if (uiState.showEditSheet) {
        ScheduleEditSheet(
            uiState = uiState,
            onDismiss = { viewModel.dismissEdit() },
            onEmojiChange = { viewModel.updateEditEmoji(it) },
            onStatusChange = { viewModel.updateEditStatus(it) },
            onHighlightChange = { viewModel.updateEditHighlight(it) },
            onAdjustStartTime = { viewModel.adjustStartTime(it) },
            onAdjustEndTime = { viewModel.adjustEndTime(it) },
            onRemindBeforeChange = { viewModel.updateEditRemindBefore(it) },
            onSave = { viewModel.saveEdit() },
        )
    }

    // 删除确认弹窗
    if (uiState.showDeleteConfirm) {
        DeleteConfirmDialog(
            item = uiState.deletingItem,
            onConfirm = { viewModel.deleteItem() },
            onDismiss = { viewModel.dismissDeleteConfirm() }
        )
    }
}

@Composable
private fun ScheduleTimelineListContent(
    uiState: com.youkong.feature.home.viewmodel.ScheduleTimelineUiState,
    viewModel: ScheduleTimelineViewModel,
    listState: androidx.compose.foundation.lazy.LazyListState,
    swipeRefreshState: com.google.accompanist.swiperefresh.SwipeRefreshState
) {
    SwipeRefresh(
        state = swipeRefreshState,
        onRefresh = { viewModel.refresh() },
        modifier = Modifier.fillMaxSize(),
        indicator = { state, trigger ->
            SwipeRefreshIndicator(
                state = state,
                refreshTriggerDistance = trigger,
                backgroundColor = CLIColors.BackgroundSecondary,
                contentColor = CLIColors.Green
            )
        }
    ) {
        when {
            uiState.isLoading && uiState.scheduleGroups.isEmpty() -> {
                ScheduleLoadingView()
            }

            uiState.isEmpty -> {
                ScheduleEmptyView()
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
                            isItemActive = { viewModel.isItemActive(it, group) },
                            onEditItem = { item -> viewModel.startEditing(item, group) },
                            onToggleHighlight = { item -> viewModel.toggleHighlight(item, group) },
                            onDeleteItem = { item -> viewModel.confirmDelete(item, group) }
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ScheduleLoadingView() {
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
private fun ScheduleEmptyView() {
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
                text = "> 暂无状态表",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary
            )

            Text(
                text = "按住下方【按住说话生成】按钮",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextWeak
            )

            Text(
                text = "AI 直接帮你生成状态时刻表",
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
    isItemActive: (ScheduleItem) -> Boolean,
    onEditItem: (ScheduleItem) -> Unit = {},
    onToggleHighlight: (ScheduleItem) -> Unit = {},
    onDeleteItem: (ScheduleItem) -> Unit = {},
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
            // 只有今天且有正在进行的条目才显示"进行中"
            val todayStr = LocalDate.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd"))
            if (group.date == todayStr && group.items.any { isItemActive(it) }) {
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

            MainScheduleItemView(
                item = item,
                isExecuted = executed,
                isActive = active,
                // 今天和未来的所有条目都可以点击编辑
                onTap = if (group.isCurrentOrFuture) {
                    { onEditItem(item) }
                } else null,
                // 今天和未来的所有条目都可以切换有空/没空
                onToggleHighlight = if (group.isCurrentOrFuture) {
                    { onToggleHighlight(item) }
                } else null,
                // 所有条目都可以长按删除
                onLongPress = { onDeleteItem(item) }
            )

            // 连接线
            if (index < group.items.size - 1) {
                ConnectorLine(isExecuted = executed)
            }
        }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun MainScheduleItemView(
    item: ScheduleItem,
    isExecuted: Boolean,
    isActive: Boolean,
    onTap: (() -> Unit)? = null,
    onToggleHighlight: (() -> Unit)? = null,
    onLongPress: (() -> Unit)? = null,
) {
    val isHighlight = item.highlight == true
    val isBooking = item.bookingId != null

    // 颜色优先级：booking(cyan) > active(黄) > highlight(绿) > executed(灰) > default
    val borderColor = when {
        isBooking && !isExecuted -> CLIColors.Cyan
        isActive -> CLIColors.Yellow
        isHighlight && !isExecuted -> CLIColors.Green
        isExecuted -> CLIColors.TextWeak
        else -> CLIColors.Border
    }

    val textColor = when {
        isBooking && !isExecuted -> CLIColors.TextPrimary
        isActive -> CLIColors.TextPrimary
        isHighlight && !isExecuted -> CLIColors.TextPrimary
        isExecuted -> CLIColors.TextWeak
        else -> CLIColors.TextSecondary
    }

    val bgColor = when {
        isBooking && !isExecuted -> CLIColors.Cyan.copy(alpha = 0.08f)
        isActive -> CLIColors.Yellow.copy(alpha = 0.08f)
        isHighlight && !isExecuted -> CLIColors.Green.copy(alpha = 0.08f)
        else -> CLIColors.BackgroundSecondary
    }

    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp)
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

        // 状态卡片（点击编辑）
        Row(
            modifier = Modifier
                .weight(1f)
                .background(
                    color = bgColor,
                    shape = RoundedCornerShape(4.dp)
                )
                .border(
                    width = if (isActive || isHighlight) 2.dp else 1.dp,
                    color = borderColor,
                    shape = RoundedCornerShape(4.dp)
                )
                .then(
                    if (onTap != null || onLongPress != null) {
                        Modifier.combinedClickable(
                            onClick = { onTap?.invoke() },
                            onLongClick = { onLongPress?.invoke() }
                        )
                    } else Modifier
                )
                .padding(horizontal = 10.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp)
        ) {
            // Emoji 或 GIF
            if (!item.gifUrl.isNullOrEmpty()) {
                coil.compose.AsyncImage(
                    model = item.gifUrl,
                    contentDescription = item.status,
                    modifier = Modifier.size(28.dp),
                    contentScale = androidx.compose.ui.layout.ContentScale.Crop,
                )
            } else {
                Text(
                    text = item.emoji,
                    fontSize = 20.sp
                )
            }

            Text(
                text = buildString {
                    append(item.status)
                    if (item.isAIGuess == true) append(" (AI)")
                    if (!item.withUsers.isNullOrEmpty()) append(" 👥 ${item.withUsers}")
                },
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = textColor,
                modifier = Modifier.weight(1f),
                maxLines = 1
            )

            // 提醒图标
            if ((item.remindBefore ?: 0) > 0) {
                Text(
                    text = "🔔",
                    fontSize = 10.sp
                )
            }

            // 状态指示
            when {
                isBooking && !isExecuted -> {
                    Text(
                        text = "[约]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 10.sp,
                        color = CLIColors.Cyan
                    )
                }
                isActive -> {
                    Text(
                        text = "[NOW]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 10.sp,
                        color = CLIColors.Yellow
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

            // 有空/没空 toggle（与 [NOW]/[DONE] 同一设计语言）
            if (onToggleHighlight != null && !isBooking) {
                Text(
                    text = if (isHighlight) "[有空]" else "[没空]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 10.sp,
                    color = if (isHighlight) CLIColors.Green else CLIColors.TextWeak,
                    modifier = Modifier.clickable { onToggleHighlight() }
                )
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

// ========== 编辑 Sheet ==========

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ScheduleEditSheet(
    uiState: ScheduleTimelineUiState,
    onDismiss: () -> Unit,
    onEmojiChange: (String) -> Unit,
    onStatusChange: (String) -> Unit,
    onHighlightChange: (Boolean) -> Unit,
    onAdjustStartTime: (Int) -> Unit,
    onAdjustEndTime: (Int) -> Unit,
    onRemindBeforeChange: (Int) -> Unit,
    onSave: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = CLIColors.Background,
        contentColor = CLIColors.TextPrimary,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 32.dp)
        ) {
            // 标题栏
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                TextButton(onClick = onDismiss) {
                    Text(
                        text = "[取消]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.TextSecondary
                    )
                }

                Spacer(modifier = Modifier.weight(1f))

                Text(
                    text = "━━ 编辑状态 ━━",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 16.sp,
                    color = CLIColors.Green
                )

                Spacer(modifier = Modifier.weight(1f))

                val canSave = uiState.editEmoji.isNotEmpty() && uiState.editStatus.isNotEmpty() && !uiState.isSavingEdit && uiState.editConflictItems.isEmpty()
                TextButton(
                    onClick = onSave,
                    enabled = canSave,
                ) {
                    Text(
                        text = if (uiState.isSavingEdit) "[...]" else "[保存]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = if (canSave) CLIColors.Green else CLIColors.TextWeak
                    )
                }
            }

            HorizontalDivider(color = CLIColors.Border)

            // 时段调整
            Column(
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                // 开始时间
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "> 开始:",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.TextSecondary
                    )
                    Spacer(modifier = Modifier.weight(1f))
                    TextButton(onClick = { onAdjustStartTime(-30) }) {
                        Text(
                            text = "◀",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 14.sp,
                            color = CLIColors.Green
                        )
                    }
                    Text(
                        text = uiState.editStartTime,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = CLIColors.TextPrimary,
                        textAlign = TextAlign.Center,
                        modifier = Modifier.width(60.dp)
                    )
                    TextButton(onClick = { onAdjustStartTime(30) }) {
                        Text(
                            text = "▶",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 14.sp,
                            color = CLIColors.Green
                        )
                    }
                }

                // 结束时间
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "> 结束:",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.TextSecondary
                    )
                    Spacer(modifier = Modifier.weight(1f))
                    TextButton(onClick = { onAdjustEndTime(-30) }) {
                        Text(
                            text = "◀",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 14.sp,
                            color = CLIColors.Green
                        )
                    }
                    Text(
                        text = uiState.editEndTime,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = CLIColors.TextPrimary,
                        textAlign = TextAlign.Center,
                        modifier = Modifier.width(60.dp)
                    )
                    TextButton(onClick = { onAdjustEndTime(30) }) {
                        Text(
                            text = "▶",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 14.sp,
                            color = CLIColors.Green
                        )
                    }
                }

                // 冲突警告
                if (uiState.editConflictItems.isNotEmpty()) {
                    Column(
                        modifier = Modifier.padding(top = 4.dp),
                        verticalArrangement = Arrangement.spacedBy(2.dp)
                    ) {
                        Text(
                            text = "⚠ 与以下时段冲突：",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.Red
                        )
                        uiState.editConflictItems.forEach { item ->
                            Text(
                                text = "  ${item.startTime}-${item.endTime} ${item.status}",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 10.sp,
                                color = CLIColors.Red
                            )
                        }
                    }
                }
            }

            // Emoji 选择
            Column(
                modifier = Modifier.padding(horizontal = 16.dp)
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = "> Emoji:",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.TextSecondary
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    if (uiState.editEmoji.isEmpty()) {
                        Text(
                            text = "点击下方选择",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 10.sp,
                            color = CLIColors.TextWeak
                        )
                    } else {
                        // 已选 emoji，点击可移除
                        splitEmojis(uiState.editEmoji).forEach { emoji ->
                            TextButton(
                                onClick = {
                                    onEmojiChange(uiState.editEmoji.replace(emoji, ""))
                                },
                                contentPadding = PaddingValues(horizontal = 4.dp, vertical = 0.dp),
                            ) {
                                Text(text = emoji, fontSize = 24.sp)
                                Text(
                                    text = "×",
                                    fontFamily = FontFamily.Monospace,
                                    fontSize = 10.sp,
                                    color = CLIColors.TextWeak
                                )
                            }
                        }
                    }
                }

                Spacer(modifier = Modifier.height(8.dp))

                EmojiGridPicker(
                    selected = uiState.editEmoji,
                    onSelectionChange = onEmojiChange,
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            // 状态文字
            Column(
                modifier = Modifier.padding(horizontal = 16.dp)
            ) {
                Text(
                    text = "> 状态:",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary
                )

                Spacer(modifier = Modifier.height(8.dp))

                OutlinedTextField(
                    value = uiState.editStatus,
                    onValueChange = onStatusChange,
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = {
                        Text(
                            text = "输入状态文字",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 14.sp,
                            color = CLIColors.TextWeak
                        )
                    },
                    textStyle = androidx.compose.ui.text.TextStyle(
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = CLIColors.TextPrimary,
                    ),
                    colors = OutlinedTextFieldDefaults.colors(
                        focusedBorderColor = CLIColors.Green,
                        unfocusedBorderColor = CLIColors.Border,
                        cursorColor = CLIColors.Green,
                        focusedContainerColor = CLIColors.BackgroundSecondary,
                        unfocusedContainerColor = CLIColors.BackgroundSecondary,
                    ),
                    shape = RoundedCornerShape(4.dp),
                    singleLine = true,
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            // 有空/没空开关
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "> 可用:",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary
                )

                Spacer(modifier = Modifier.weight(1f))

                Row(
                    modifier = Modifier
                        .background(
                            color = CLIColors.BackgroundSecondary,
                            shape = RoundedCornerShape(4.dp)
                        )
                        .border(
                            width = 1.dp,
                            color = CLIColors.Border,
                            shape = RoundedCornerShape(4.dp)
                        ),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    // 没空
                    Text(
                        text = " 没空 ",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = if (!uiState.editHighlight) CLIColors.TextPrimary else CLIColors.TextWeak,
                        modifier = Modifier
                            .clickable { onHighlightChange(false) }
                            .background(
                                if (!uiState.editHighlight) CLIColors.Border else CLIColors.BackgroundSecondary,
                                shape = RoundedCornerShape(topStart = 4.dp, bottomStart = 4.dp)
                            )
                            .padding(horizontal = 12.dp, vertical = 8.dp)
                    )
                    // 有空
                    Text(
                        text = " 有空 ",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = if (uiState.editHighlight) CLIColors.Green else CLIColors.TextWeak,
                        modifier = Modifier
                            .clickable { onHighlightChange(true) }
                            .background(
                                if (uiState.editHighlight) CLIColors.Green.copy(alpha = 0.15f) else CLIColors.BackgroundSecondary,
                                shape = RoundedCornerShape(topEnd = 4.dp, bottomEnd = 4.dp)
                            )
                            .padding(horizontal = 12.dp, vertical = 8.dp)
                    )
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            // 提醒选择
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "> 提醒:",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary
                )

                Spacer(modifier = Modifier.weight(1f))

                var expanded by remember { mutableStateOf(false) }
                val remindOptions = listOf(
                    0 to "不提醒",
                    5 to "5分钟前",
                    10 to "10分钟前",
                    15 to "15分钟前",
                    30 to "30分钟前",
                    60 to "1小时前",
                )
                val currentLabel = remindOptions.firstOrNull { it.first == uiState.editRemindBefore }?.second ?: "不提醒"

                Box {
                    Text(
                        text = "[ $currentLabel ▼ ]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = if (uiState.editRemindBefore > 0) CLIColors.Yellow else CLIColors.TextSecondary,
                        modifier = Modifier
                            .clickable { expanded = true }
                            .background(
                                color = CLIColors.BackgroundSecondary,
                                shape = RoundedCornerShape(4.dp)
                            )
                            .border(
                                width = 1.dp,
                                color = CLIColors.Border,
                                shape = RoundedCornerShape(4.dp)
                            )
                            .padding(horizontal = 12.dp, vertical = 8.dp)
                    )

                    DropdownMenu(
                        expanded = expanded,
                        onDismissRequest = { expanded = false },
                        modifier = Modifier.background(CLIColors.BackgroundSecondary),
                    ) {
                        remindOptions.forEach { (minutes, label) ->
                            DropdownMenuItem(
                                text = {
                                    Text(
                                        text = label,
                                        fontFamily = FontFamily.Monospace,
                                        fontSize = 12.sp,
                                        color = if (minutes == uiState.editRemindBefore) CLIColors.Green else CLIColors.TextPrimary
                                    )
                                },
                                onClick = {
                                    onRemindBeforeChange(minutes)
                                    expanded = false
                                },
                            )
                        }
                    }
                }
            }
        }
    }
}

// ========== 删除确认弹窗 ==========

@Composable
private fun DeleteConfirmDialog(
    item: ScheduleItem?,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
) {
    if (item == null) return

    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = CLIColors.Background,
        titleContentColor = CLIColors.TextPrimary,
        textContentColor = CLIColors.TextSecondary,
        title = {
            Text(
                text = "删除状态",
                fontFamily = FontFamily.Monospace,
            )
        },
        text = {
            Text(
                text = "确定删除「${item.emoji} ${item.status}」？",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
            )
        },
        confirmButton = {
            TextButton(onClick = onConfirm) {
                Text(
                    text = "删除",
                    fontFamily = FontFamily.Monospace,
                    color = CLIColors.Red,
                )
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(
                    text = "取消",
                    fontFamily = FontFamily.Monospace,
                    color = CLIColors.TextSecondary,
                )
            }
        },
    )
}

// ========== Emoji 选择器 ==========

internal val emojiPages = listOf(
    "日常" to listOf(
        "💼", "📚", "💻", "🎮", "🎬", "🎵", "🏃", "🧘",
        "🛌", "🧹", "🛒", "📱", "✍️", "🎨", "📷", "🎧",
    ),
    "饮食" to listOf(
        "☕", "🍵", "🍺", "🍷", "🥤", "🧋", "🍔", "🍜",
        "🍣", "🍕", "🥗", "🍰", "🍳", "🥘", "🍱", "🧁",
    ),
    "出行" to listOf(
        "🚗", "🚇", "🚶", "🏠", "🏢", "🏫", "🏥", "✈️",
        "🚴", "🛵", "🚕", "🚌", "⛽", "🅿️", "🛫", "🧳",
    ),
    "社交" to listOf(
        "👥", "🤝", "💬", "📞", "🎉", "🍻", "👋", "🤗",
        "💌", "🎁", "🥂", "👫", "🫂", "💑", "👨‍👩‍👧", "🧑‍🤝‍🧑",
    ),
    "心情" to listOf(
        "😊", "😴", "🥱", "😤", "😎", "🤔", "😌", "🥰",
        "😂", "😢", "😡", "🤩", "😇", "🫠", "🤯", "😶‍🌫️",
    ),
)

internal const val MAX_EMOJI_SELECTION = 2

/** 按 grapheme cluster 正确拆分 emoji 字符串（处理代理对、ZWJ 序列等） */
internal fun splitEmojis(str: String): List<String> {
    if (str.isEmpty()) return emptyList()
    val result = mutableListOf<String>()
    val it = java.text.BreakIterator.getCharacterInstance()
    it.setText(str)
    var start = it.first()
    var end = it.next()
    while (end != java.text.BreakIterator.DONE) {
        result.add(str.substring(start, end))
        start = end
        end = it.next()
    }
    return result
}

@Composable
internal fun EmojiGridPicker(
    selected: String,
    onSelectionChange: (String) -> Unit,
) {
    var currentPage by remember { mutableIntStateOf(0) }
    val selectedList = splitEmojis(selected)

    Column {
        // 分类标签
        Row(verticalAlignment = Alignment.CenterVertically) {
            emojiPages.forEachIndexed { index, (label, _) ->
                TextButton(
                    onClick = { currentPage = index },
                    contentPadding = PaddingValues(horizontal = 8.dp, vertical = 4.dp),
                    modifier = Modifier.height(32.dp),
                ) {
                    Text(
                        text = label,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 11.sp,
                        color = if (currentPage == index) CLIColors.Green else CLIColors.TextWeak,
                    )
                }
            }

            Spacer(modifier = Modifier.weight(1f))

            Text(
                text = "${selectedList.size}/$MAX_EMOJI_SELECTION",
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextWeak,
            )
        }

        Spacer(modifier = Modifier.height(4.dp))

        // Emoji 网格
        val emojis = emojiPages[currentPage].second
        LazyVerticalGrid(
            columns = GridCells.Fixed(8),
            modifier = Modifier.height(120.dp),
            horizontalArrangement = Arrangement.spacedBy(2.dp),
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            items(emojis) { emoji ->
                val isSelected = selectedList.contains(emoji)
                val isDisabled = !isSelected && selectedList.size >= MAX_EMOJI_SELECTION

                Box(
                    modifier = Modifier
                        .aspectRatio(1f)
                        .background(
                            color = if (isSelected) CLIColors.Green.copy(alpha = 0.2f)
                            else androidx.compose.ui.graphics.Color.Transparent,
                            shape = RoundedCornerShape(4.dp)
                        )
                        .then(
                            if (isSelected) Modifier.border(1.dp, CLIColors.Green, RoundedCornerShape(4.dp))
                            else Modifier
                        )
                        .clickable(enabled = !isDisabled) {
                            if (isSelected) {
                                onSelectionChange(selected.replace(emoji, ""))
                            } else {
                                onSelectionChange(selected + emoji)
                            }
                        },
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = emoji,
                        fontSize = 22.sp,
                        modifier = Modifier.then(
                            if (isDisabled) Modifier.graphicsLayer { alpha = 0.4f }
                            else Modifier
                        ),
                    )
                }
            }
        }
    }
}

// ========== AI 自动推测 Toggle Bar ==========

@Composable
fun AutoPredictToggleBar(
    enabled: Boolean,
    aiReady: Boolean,
    isToggling: Boolean,
    onToggle: () -> Unit,
    onShowReadiness: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = "AI 自动推测",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 13.sp,
                    fontWeight = FontWeight.Bold,
                    color = CLIColors.TextPrimary
                )
                Spacer(modifier = Modifier.width(4.dp))
                Text(
                    text = "ⓘ",
                    fontSize = 14.sp,
                    color = CLIColors.TextWeak,
                    modifier = Modifier.clickable { onShowReadiness() }
                )
            }
            Text(
                text = "根据你的习惯和数据，自动补全空缺状态",
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextSecondary
            )
        }

        Switch(
            checked = enabled,
            onCheckedChange = {
                if (!aiReady && it) {
                    // 条件不满足试图打开 → 弹 readiness sheet
                    onShowReadiness()
                } else {
                    onToggle()
                }
            },
            enabled = !isToggling,
            colors = SwitchDefaults.colors(
                checkedThumbColor = CLIColors.Green,
                checkedTrackColor = CLIColors.Green.copy(alpha = 0.3f),
                uncheckedThumbColor = CLIColors.TextWeak,
                uncheckedTrackColor = CLIColors.Border,
            )
        )
    }

    HorizontalDivider(color = CLIColors.Border)
}

// ========== AI Readiness Sheet ==========

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AIReadinessSheet(
    details: AIReadyDetails?,
    onDismiss: () -> Unit,
    onNavigateToAddFriend: () -> Unit,
) {
    val context = LocalContext.current
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val d = details ?: AIReadyDetails()

    val allReady = d.permLocation && d.permMotion && d.permCalendar
            && d.hasInvitedFriend && d.hasVoiceSchedule

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = CLIColors.Background,
        contentColor = CLIColors.TextPrimary,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 32.dp)
        ) {
            // 标题栏
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "━━ AI 自动推测 ━━",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 16.sp,
                    color = CLIColors.Green,
                    modifier = Modifier.weight(1f),
                    textAlign = TextAlign.Center
                )

                TextButton(onClick = onDismiss) {
                    Text(
                        text = "完成",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.Green
                    )
                }
            }

            HorizontalDivider(color = CLIColors.Border)

            // 说明
            Text(
                text = "AI 将根据你授权的数据和社交信息，自动推测并补全空缺的状态时段。",
                fontFamily = FontFamily.Monospace,
                fontSize = 11.sp,
                color = CLIColors.TextSecondary,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp)
            )

            // 数据权限授权
            SectionHeader("数据权限授权")

            ReadinessRow(
                icon = "📍",
                title = "位置权限",
                ready = d.permLocation,
                action = {
                    val intent = Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS).apply {
                        data = android.net.Uri.fromParts("package", context.packageName, null)
                        flags = Intent.FLAG_ACTIVITY_NEW_TASK
                    }
                    context.startActivity(intent)
                }
            )

            ReadinessRow(
                icon = "🏃",
                title = "运动数据权限",
                ready = d.permMotion,
                action = {
                    val intent = Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS).apply {
                        data = android.net.Uri.fromParts("package", context.packageName, null)
                        flags = Intent.FLAG_ACTIVITY_NEW_TASK
                    }
                    context.startActivity(intent)
                }
            )

            ReadinessRow(
                icon = "📅",
                title = "日历权限",
                ready = d.permCalendar,
                action = {
                    val intent = Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS).apply {
                        data = android.net.Uri.fromParts("package", context.packageName, null)
                        flags = Intent.FLAG_ACTIVITY_NEW_TASK
                    }
                    context.startActivity(intent)
                }
            )

            Spacer(modifier = Modifier.height(8.dp))

            // 社交
            SectionHeader("社交")

            ReadinessRow(
                icon = "👥",
                title = "邀请一个朋友",
                ready = d.hasInvitedFriend,
                action = {
                    onNavigateToAddFriend()
                    onDismiss()
                }
            )

            Spacer(modifier = Modifier.height(8.dp))

            // 行程
            SectionHeader("行程")

            ReadinessRow(
                icon = "🎤",
                title = "成功建立一次行程",
                ready = d.hasVoiceSchedule,
                action = null // 需要用语音或文本建行程，无直接跳转
            )

            // 底部提示
            if (!allReady) {
                Spacer(modifier = Modifier.height(16.dp))
                Text(
                    text = "完成以上条件，即可开启 AI 自动推测",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 11.sp,
                    color = CLIColors.Yellow,
                    textAlign = TextAlign.Center,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp)
                )
            }
        }
    }
}

@Composable
private fun SectionHeader(title: String) {
    Text(
        text = "> $title",
        fontFamily = FontFamily.Monospace,
        fontSize = 11.sp,
        color = CLIColors.TextWeak,
        modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp)
    )
}

@Composable
private fun ReadinessRow(
    icon: String,
    title: String,
    ready: Boolean,
    action: (() -> Unit)?,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .then(
                if (!ready && action != null) Modifier.clickable { action() }
                else Modifier
            )
            .padding(horizontal = 16.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(text = icon, fontSize = 16.sp)

        Spacer(modifier = Modifier.width(12.dp))

        Text(
            text = title,
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = if (ready) CLIColors.TextPrimary else CLIColors.TextSecondary,
            modifier = Modifier.weight(1f)
        )

        if (ready) {
            Text(
                text = "✓",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.Green
            )
        } else if (action != null) {
            Text(
                text = "去完成 >",
                fontFamily = FontFamily.Monospace,
                fontSize = 11.sp,
                color = CLIColors.Green
            )
        } else {
            Text(
                text = "○",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextWeak
            )
        }
    }
}
