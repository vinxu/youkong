package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.youkong.core.domain.repository.StatusCardOption
import com.youkong.core.ui.emoji.EmojiStateView
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.home.viewmodel.AIStatusInferenceViewModel
import com.youkong.feature.home.viewmodel.InferPhase
import com.youkong.feature.home.viewmodel.LogType
import com.youkong.feature.home.viewmodel.StreamingLog

/**
 * AI 状态推断界面 — 4选1 模式
 */
@Composable
fun AIStatusInferenceScreen(
    onDismiss: () -> Unit,
    onConfirmed: () -> Unit,
    viewModel: AIStatusInferenceViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    // 首次进入时开始推断
    LaunchedEffect(Unit) {
        viewModel.startInference()
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
    ) {
        Column(modifier = Modifier.fillMaxSize()) {
            // Header
            AIInferenceHeader(
                phase = uiState.currentPhase,
                onDismiss = onDismiss,
                onBack = { viewModel.backToOptions() }
            )

            // 分隔线
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(1.dp)
                    .background(CLIColors.Border)
            )

            // 内容 — 根据阶段切换
            when {
                uiState.error != null && uiState.currentPhase != InferPhase.EDITING -> {
                    ErrorView(
                        error = uiState.error ?: "推断失败",
                        onRetry = { viewModel.startInference() }
                    )
                }

                else -> when (uiState.currentPhase) {
                    InferPhase.LOADING -> {
                        InferringView(
                            streamingPhase = uiState.streamingPhase,
                            streamingLogs = uiState.streamingLogs,
                        )
                    }

                    InferPhase.OPTIONS -> {
                        OptionsGridView(
                            options = uiState.statusOptions,
                            selectedIndex = uiState.selectedIndex,
                            isRefreshing = uiState.isRefreshing,
                            gifLoadingIndices = uiState.gifLoadingIndices,
                            onSelectOption = { viewModel.selectOption(it) },
                            onConfirmSelection = { viewModel.confirmSelection() },
                            onRefresh = { viewModel.refreshOptions() },
                        )
                    }

                    InferPhase.EDITING -> {
                        ResultView(
                            uiState = uiState,
                            viewModel = viewModel,
                            onConfirmed = {
                                viewModel.confirmStatus {
                                    onConfirmed()
                                    onDismiss()
                                }
                            }
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun AIInferenceHeader(
    phase: InferPhase,
    onDismiss: () -> Unit,
    onBack: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        // 左侧：编辑阶段显示返回，其他阶段显示关闭
        if (phase == InferPhase.EDITING) {
            TextButton(onClick = onBack) {
                Text(
                    text = "[返回]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary
                )
            }
        } else {
            TextButton(onClick = onDismiss) {
                Text(
                    text = "[X]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary
                )
            }
        }

        Text(
            text = when (phase) {
                InferPhase.LOADING -> "━━ AI 推断中 ━━"
                InferPhase.OPTIONS -> "━━ 选择你的状态 ━━"
                InferPhase.EDITING -> "━━ 编辑发布 ━━"
            },
            fontFamily = FontFamily.Monospace,
            fontSize = 18.sp,
            fontWeight = FontWeight.Bold,
            color = CLIColors.Cyan
        )

        // 占位
        Box(modifier = Modifier.width(48.dp))
    }
}

// MARK: - 4选1 选项网格

@Composable
private fun OptionsGridView(
    options: List<StatusCardOption>,
    selectedIndex: Int?,
    isRefreshing: Boolean,
    gifLoadingIndices: Set<Int>,
    onSelectOption: (Int) -> Unit,
    onConfirmSelection: () -> Unit,
    onRefresh: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // 提示文字
        Text(
            text = "AI 为你推断了以下可能的状态",
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = CLIColors.TextSecondary,
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Center,
        )

        // 2×2 网格
        LazyVerticalGrid(
            columns = GridCells.Fixed(2),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            modifier = Modifier.weight(1f),
        ) {
            items(options, key = { it.index }) { option ->
                StatusOptionCard(
                    option = option,
                    isSelected = selectedIndex == option.index,
                    isGifLoading = option.index in gifLoadingIndices,
                    onClick = { onSelectOption(option.index) }
                )
            }
        }

        // 底部按钮区
        Column(
            modifier = Modifier.fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // 确认选择
            Button(
                onClick = onConfirmSelection,
                enabled = selectedIndex != null,
                modifier = Modifier.fillMaxWidth(),
                colors = ButtonDefaults.buttonColors(
                    containerColor = CLIColors.Green,
                    contentColor = CLIColors.Background,
                    disabledContainerColor = CLIColors.Border,
                    disabledContentColor = CLIColors.TextWeak,
                )
            ) {
                Text(
                    text = "确认选择",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp
                )
            }

            // 换一批
            TextButton(
                onClick = onRefresh,
                enabled = !isRefreshing,
            ) {
                if (isRefreshing) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(14.dp),
                        color = CLIColors.Yellow,
                        strokeWidth = 2.dp
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        text = "生成中...",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = CLIColors.TextWeak
                    )
                } else {
                    Text(
                        text = "换一批",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = CLIColors.Cyan
                    )
                }
            }
        }
    }
}

@Composable
private fun StatusOptionCard(
    option: StatusCardOption,
    isSelected: Boolean,
    isGifLoading: Boolean,
    onClick: () -> Unit,
) {
    val shape = RoundedCornerShape(16.dp)

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(1f)
            .clickable(onClick = onClick)
            .then(
                if (isSelected) Modifier.border(3.dp, CLIColors.Green, shape)
                else Modifier
            ),
        shape = shape,
        colors = CardDefaults.cardColors(containerColor = Color.Transparent),
        elevation = CardDefaults.cardElevation(
            defaultElevation = if (isSelected) 8.dp else 0.dp,
        ),
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            // 背景层
            if (!option.gifUrl.isNullOrEmpty()) {
                // GIF 背景（使用 ImageDecoderDecoder 播放动画，和首页一致）
                val context = androidx.compose.ui.platform.LocalContext.current
                val gifLoader = remember {
                    coil.ImageLoader.Builder(context)
                        .components {
                            if (android.os.Build.VERSION.SDK_INT >= 28) {
                                add(coil.decode.ImageDecoderDecoder.Factory())
                            }
                        }
                        .build()
                }
                coil.compose.AsyncImage(
                    model = coil.request.ImageRequest.Builder(context)
                        .data(option.gifUrl)
                        .crossfade(false)
                        .build(),
                    imageLoader = gifLoader,
                    contentDescription = option.activity,
                    modifier = Modifier
                        .fillMaxSize()
                        .clip(shape),
                    contentScale = ContentScale.Crop,
                )
            } else {
                // 无 GIF 时用渐变背景（对齐 iOS）
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .background(
                            brush = androidx.compose.ui.graphics.Brush.linearGradient(
                                colors = listOf(CLIColors.BackgroundSecondary, CLIColors.Background)
                            ),
                            shape = shape,
                        )
                )
            }

            // 暗化遮罩（始终有，对齐 iOS）
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Color.Black.copy(alpha = 0.35f))
            )

            // 内容
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(12.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                // 动画 Emoji（从 COS 加载 animated WebP）
                EmojiStateView(
                    emoji = option.emoji,
                    modifier = Modifier.size(56.dp),
                )
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = option.activity,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Bold,
                    color = Color.White,
                    textAlign = TextAlign.Center,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }

            // GIF 加载中指示器（右下角）
            if (isGifLoading && option.gifUrl.isNullOrEmpty()) {
                Box(
                    modifier = Modifier
                        .align(Alignment.BottomEnd)
                        .padding(8.dp)
                        .background(
                            Color.Black.copy(alpha = 0.5f),
                            RoundedCornerShape(12.dp)
                        )
                        .padding(horizontal = 8.dp, vertical = 4.dp),
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(4.dp),
                    ) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(10.dp),
                            color = Color.White,
                            strokeWidth = 1.5.dp,
                        )
                        Text(
                            text = "GIF",
                            fontSize = 9.sp,
                            color = Color.White.copy(alpha = 0.8f),
                            fontFamily = FontFamily.Monospace,
                        )
                    }
                }
            }
        }
    }
}

// MARK: - Loading 阶段

@Composable
private fun InferringView(
    streamingPhase: String?,
    streamingLogs: List<StreamingLog>,
) {
    val scrollState = rememberScrollState()

    // 自动滚动到底部
    LaunchedEffect(streamingLogs.size) {
        scrollState.animateScrollTo(scrollState.maxValue)
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp)
    ) {
        // 顶部状态
        Row(
            modifier = Modifier.padding(bottom = 16.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            CircularProgressIndicator(
                modifier = Modifier.size(16.dp),
                color = CLIColors.Yellow,
                strokeWidth = 2.dp
            )
            Text(
                text = streamingPhase ?: "分析中...",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.Yellow
            )
        }

        // 流式日志
        Column(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth()
                .background(CLIColors.BackgroundSecondary)
                .padding(12.dp)
                .verticalScroll(scrollState),
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            if (streamingLogs.isEmpty()) {
                Text(
                    text = "> 正在收集设备数据...",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextWeak
                )
            }
            streamingLogs.forEach { log ->
                val color = when (log.type) {
                    LogType.PHASE -> CLIColors.Cyan
                    LogType.TOOL -> CLIColors.Yellow
                    LogType.TOOL_RESULT -> CLIColors.Green
                    LogType.THINKING -> CLIColors.TextSecondary
                    LogType.ASK_USER -> CLIColors.Yellow
                    LogType.USER_ANSWER -> CLIColors.Green
                    LogType.ERROR -> CLIColors.Red
                }
                if (log.type == LogType.THINKING) {
                    Text(
                        text = log.text,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 11.sp,
                        color = color,
                    )
                } else {
                    Text(
                        text = log.text,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 11.sp,
                        color = color,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
            }
        }
    }
}

// MARK: - 错误页

@Composable
private fun ErrorView(error: String, onRetry: () -> Unit) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(20.dp)
        ) {
            Text(
                text = "❌",
                fontSize = 64.sp
            )

            Text(
                text = "推断失败",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.Red
            )

            Text(
                text = error,
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextSecondary,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(horizontal = 32.dp)
            )

            TextButton(
                onClick = onRetry,
                modifier = Modifier.border(1.dp, CLIColors.Cyan)
            ) {
                Text(
                    text = "[重试]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.Cyan
                )
            }
        }
    }
}

// MARK: - 编辑发布页

@Composable
private fun ResultView(
    uiState: com.youkong.feature.home.viewmodel.AIStatusInferenceUiState,
    viewModel: AIStatusInferenceViewModel,
    onConfirmed: () -> Unit,
) {
    val scrollState = rememberScrollState()

    // 根据 emoji 数量计算字体大小
    fun emojiSize(count: Int): Int = when {
        count <= 1 -> 72
        count == 2 -> 56
        else -> 48
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(scrollState)
            .padding(16.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(20.dp)
    ) {
        // GIF 或 Emoji 显示区域
        val inference = uiState.inference
        val emojiList = splitEmojis(uiState.editingEmoji)

        Box(
            modifier = Modifier
                .fillMaxWidth(),
            contentAlignment = Alignment.Center
        ) {
            // Emoji / GIF 展示
            Box(
                modifier = Modifier
                    .height(120.dp)
                    .fillMaxWidth(),
                contentAlignment = Alignment.Center
            ) {
                if (uiState.useGif && !inference?.gifUrl.isNullOrEmpty()) {
                    AsyncImage(
                        model = inference!!.gifUrl,
                        contentDescription = inference.activity,
                        modifier = Modifier.fillMaxHeight(),
                        contentScale = ContentScale.Fit,
                    )
                } else {
                    Text(
                        text = uiState.editingEmoji,
                        fontSize = emojiSize(emojiList.size).sp,
                    )
                    if (uiState.useGif && uiState.isSearchingGif) {
                        Row(
                            modifier = Modifier.align(Alignment.BottomCenter),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(4.dp)
                        ) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(12.dp),
                                color = CLIColors.Yellow,
                                strokeWidth = 1.5.dp
                            )
                            Text(
                                text = "搜索 GIF...",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 10.sp,
                                color = CLIColors.TextWeak
                            )
                        }
                    }
                }
            }

            // 右上角编辑按钮
            Text(
                text = if (uiState.showEmojiPicker) "[收起]" else "[编辑]",
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextSecondary,
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .clickable { viewModel.toggleEmojiPicker() }
                    .padding(4.dp)
            )
        }

        // Emoji / GIF 切换按钮
        EmojiGifToggle(
            useGif = uiState.useGif,
            isSearchingGif = uiState.isSearchingGif,
            onEmojiClick = { viewModel.setEmojiMode() },
            onGifClick = { viewModel.toggleUseGif() }
        )

        // Emoji 网格选择器（点击编辑后展开）
        if (uiState.showEmojiPicker) {
            Column(
                modifier = Modifier.fillMaxWidth()
            ) {
                // 已选 emoji 展示 + 可移除
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = "> Emoji:",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.TextSecondary
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    if (emojiList.isEmpty()) {
                        Text(
                            text = "点击下方选择",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 10.sp,
                            color = CLIColors.TextWeak
                        )
                    } else {
                        emojiList.forEach { emoji ->
                            TextButton(
                                onClick = {
                                    viewModel.updateEmoji(uiState.editingEmoji.replace(emoji, ""))
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
                    selected = uiState.editingEmoji,
                    onSelectionChange = { viewModel.updateEmoji(it) },
                )
            }
        }

        // 活动描述
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically
        ) {
            if (uiState.isEditing) {
                BasicTextField(
                    value = uiState.editingActivity,
                    onValueChange = { viewModel.updateActivity(it) },
                    textStyle = androidx.compose.ui.text.TextStyle(
                        fontFamily = FontFamily.Monospace,
                        fontSize = 18.sp,
                        fontWeight = FontWeight.Bold,
                        color = CLIColors.TextPrimary,
                        textAlign = TextAlign.Center
                    ),
                    modifier = Modifier
                        .weight(1f)
                        .border(1.dp, CLIColors.Border)
                        .padding(horizontal = 16.dp, vertical = 8.dp)
                )
            } else {
                Text(
                    text = uiState.editingActivity,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 18.sp,
                    fontWeight = FontWeight.Bold,
                    color = CLIColors.TextPrimary
                )
            }

            Spacer(modifier = Modifier.width(8.dp))

            Text(
                text = if (uiState.isEditing) "[完成]" else "[编辑]",
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextSecondary,
                modifier = Modifier
                    .clickable { viewModel.startEditing() }
                    .padding(4.dp)
            )
        }

        // 场所
        uiState.inference?.place?.takeIf { it.isNotEmpty() }?.let { place ->
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(text = "📍", fontSize = 14.sp)
                Text(
                    text = if (uiState.isEditing) uiState.editingPlace else place,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.TextSecondary
                )
            }
        }

        // 预计持续时长
        uiState.inference?.durationHint?.takeIf { it.isNotEmpty() }?.let { hint ->
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(text = "⏱️", fontSize = 12.sp)
                Text(
                    text = hint,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextWeak
                )
            }
        }

        // 有空状态开关
        AvailabilityToggle(
            isAvailable = uiState.editingIsAvailable,
            onClick = { viewModel.toggleIsAvailable() }
        )

        // 置信度
        uiState.inference?.confidence?.let { confidence ->
            ConfidenceView(confidence = confidence)
        }

        // 推理依据
        uiState.inference?.reasoning?.takeIf { it.isNotEmpty() }?.let { reasoning ->
            ReasoningView(reasoning = reasoning)
        }

        Spacer(modifier = Modifier.height(16.dp))

        // 确认发布按钮
        Button(
            onClick = onConfirmed,
            enabled = !uiState.isConfirming,
            modifier = Modifier.fillMaxWidth(),
            colors = ButtonDefaults.buttonColors(
                containerColor = CLIColors.Green,
                contentColor = CLIColors.Background
            )
        ) {
            if (uiState.isConfirming) {
                CircularProgressIndicator(
                    modifier = Modifier.size(16.dp),
                    color = CLIColors.Background,
                    strokeWidth = 2.dp
                )
                Spacer(modifier = Modifier.width(8.dp))
            } else {
                Text(text = "✓", fontSize = 14.sp)
                Spacer(modifier = Modifier.width(8.dp))
            }
            Text(
                text = if (uiState.isConfirming && uiState.confirmingMessage.isNotEmpty()) {
                    uiState.confirmingMessage
                } else if (viewModel.hasChanges()) {
                    "确认修改并发布"
                } else {
                    "确认发布"
                },
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp
            )
        }

        Spacer(modifier = Modifier.height(32.dp))
    }
}

// MARK: - 子组件

@Composable
private fun AvailabilityToggle(
    isAvailable: Boolean,
    onClick: () -> Unit,
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() }
            .border(
                width = 1.dp,
                color = if (isAvailable) CLIColors.Yellow else CLIColors.Border
            ),
        color = if (isAvailable) CLIColors.Yellow.copy(alpha = 0.1f) else CLIColors.BackgroundSecondary
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = if (isAvailable) "👑" else "💤",
                fontSize = 24.sp
            )

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = if (isAvailable) "有空" else "忙碌",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = if (isAvailable) CLIColors.Yellow else CLIColors.TextSecondary
                )
                Text(
                    text = if (isAvailable) "好友可以看到你有空" else "暂时不方便被打扰",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 10.sp,
                    color = CLIColors.TextWeak
                )
            }

            Text(
                text = if (isAvailable) "[ON]" else "[OFF]",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = if (isAvailable) CLIColors.Green else CLIColors.TextWeak
            )
        }
    }
}

@Composable
private fun ConfidenceView(confidence: String) {
    val text = when (confidence) {
        "high" -> "高"
        "medium" -> "中"
        "low" -> "低"
        else -> confidence
    }
    val color = when (confidence) {
        "high" -> CLIColors.Green
        "medium" -> CLIColors.Yellow
        else -> CLIColors.TextWeak
    }

    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = "置信度:",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = CLIColors.TextWeak
        )
        Text(
            text = text,
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = color
        )
    }
}

@Composable
private fun ReasoningView(reasoning: String) {
    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        Text(
            text = "推理依据:",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = CLIColors.TextWeak
        )
        Surface(
            modifier = Modifier.fillMaxWidth(),
            color = CLIColors.BackgroundSecondary
        ) {
            Text(
                text = reasoning,
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextSecondary,
                modifier = Modifier.padding(12.dp)
            )
        }
    }
}

@Composable
private fun EmojiGifToggle(
    useGif: Boolean,
    isSearchingGif: Boolean,
    onEmojiClick: () -> Unit,
    onGifClick: () -> Unit,
) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Emoji 按钮
        Surface(
            modifier = Modifier
                .clickable { onEmojiClick() }
                .border(
                    width = 1.dp,
                    color = if (!useGif) CLIColors.Green else CLIColors.Border
                ),
            color = if (!useGif) CLIColors.Green.copy(alpha = 0.1f) else CLIColors.BackgroundSecondary
        ) {
            Text(
                text = "😀",
                fontSize = 20.sp,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
            )
        }

        // GIF 按钮
        Surface(
            modifier = Modifier
                .clickable { onGifClick() }
                .border(
                    width = 1.dp,
                    color = if (useGif) CLIColors.Green else CLIColors.Border
                ),
            color = if (useGif) CLIColors.Green.copy(alpha = 0.1f) else CLIColors.BackgroundSecondary
        ) {
            Row(
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                Text(
                    text = "GIF",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Bold,
                    color = if (useGif) CLIColors.Green else CLIColors.TextSecondary
                )
                if (isSearchingGif) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(12.dp),
                        color = CLIColors.Yellow,
                        strokeWidth = 1.5.dp
                    )
                }
            }
        }
    }
}
