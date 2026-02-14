package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.home.viewmodel.AIStatusInferenceViewModel
import com.youkong.feature.home.viewmodel.LogType
import com.youkong.feature.home.viewmodel.StreamingLog

/**
 * AI 状态推断界面
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
            AIInferenceHeader(onDismiss = onDismiss)

            // 分隔线
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(1.dp)
                    .background(CLIColors.Border)
            )

            // 内容
            when {
                uiState.isInferring -> {
                    InferringView(
                        streamingPhase = uiState.streamingPhase,
                        streamingLogs = uiState.streamingLogs,
                    )
                }

                uiState.error != null -> {
                    ErrorView(
                        error = uiState.error ?: "推断失败",
                        onRetry = { viewModel.startInference() }
                    )
                }

                uiState.inference != null -> {
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

                else -> {
                    InferringView(
                        streamingPhase = "正在连接...",
                        streamingLogs = emptyList(),
                    )
                }
            }
        }

        // 底部浮框：用户确认选择题
        if (uiState.isAskingUser) {
            AskUserBottomPanel(
                question = uiState.askUserQuestion,
                options = uiState.askUserOptions,
                onOptionSelected = { answer -> viewModel.respondToAskUser(answer) },
                modifier = Modifier.align(Alignment.BottomCenter)
            )
        }
    }
}

@Composable
private fun AIInferenceHeader(onDismiss: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
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

        Text(
            text = "━━ AI 推断当下状态 ━━",
            fontFamily = FontFamily.Monospace,
            fontSize = 18.sp,
            fontWeight = FontWeight.Bold,
            color = CLIColors.Cyan
        )

        // 占位
        Box(modifier = Modifier.width(48.dp))
    }
}

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
                    // thinking 允许视觉换行
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

@Composable
private fun AskUserBottomPanel(
    question: String,
    options: List<String>,
    onOptionSelected: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier
            .fillMaxWidth()
            .padding(16.dp),
        color = CLIColors.BackgroundSecondary,
        shadowElevation = 0.dp,
        border = androidx.compose.foundation.BorderStroke(1.dp, CLIColors.Border),
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Text(
                text = question,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextPrimary
            )

            options.forEach { option ->
                TextButton(
                    onClick = { onOptionSelected(option) },
                    modifier = Modifier
                        .fillMaxWidth()
                        .border(1.dp, CLIColors.Green)
                ) {
                    Text(
                        text = option,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = CLIColors.Green
                    )
                }
            }
        }
    }
}

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

