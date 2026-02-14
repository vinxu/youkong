@file:OptIn(androidx.compose.foundation.layout.ExperimentalLayoutApi::class)

package com.youkong.feature.home.component

import androidx.compose.animation.*
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.network.model.*
import com.youkong.core.ui.theme.CLIColors
import java.text.SimpleDateFormat
import java.util.*

// MARK: - Message Bubble

/**
 * 消息气泡组件
 */
@Composable
fun MessageBubble(
    message: ChatMessage,
    voiceState: VoiceScheduleState = VoiceScheduleState.IDLE,
    onConfirm: (() -> Unit)? = null,
    onCancel: (() -> Unit)? = null,
    confirmButtonText: String = "✓ 确认保存",
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = if (message.type == ChatMessageType.USER) {
            Arrangement.End
        } else {
            Arrangement.Start
        }
    ) {
        if (message.type == ChatMessageType.USER) {
            Spacer(modifier = Modifier.weight(1f, fill = false).widthIn(min = 60.dp))
        }

        Column(
            horizontalAlignment = if (message.type == ChatMessageType.USER) {
                Alignment.End
            } else {
                Alignment.Start
            }
        ) {
            // 消息内容
            MessageContent(
                message = message,
                voiceState = voiceState,
                onConfirm = onConfirm,
                onCancel = onCancel,
                confirmButtonText = confirmButtonText
            )

            // 时间
            Text(
                text = formatTime(message.timestamp),
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextWeak,
                modifier = Modifier.padding(top = 4.dp)
            )
        }

        if (message.type != ChatMessageType.USER) {
            Spacer(modifier = Modifier.weight(1f, fill = false).widthIn(min = 60.dp))
        }
    }
}

@Composable
private fun MessageContent(
    message: ChatMessage,
    voiceState: VoiceScheduleState = VoiceScheduleState.IDLE,
    onConfirm: (() -> Unit)?,
    onCancel: (() -> Unit)?,
    confirmButtonText: String = "✓ 确认保存"
) {
    when (message.type) {
        ChatMessageType.USER -> {
            Text(
                text = message.content,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.Background,
                modifier = Modifier
                    .background(CLIColors.Green)
                    .padding(12.dp)
            )
        }

        ChatMessageType.AI_TEXT, ChatMessageType.AI_THINKING -> {
            Column(
                modifier = Modifier
                    .background(CLIColors.BackgroundSecondary)
                    .border(1.dp, CLIColors.Border)
            ) {
                Text(
                    text = message.content,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.TextPrimary,
                    modifier = Modifier.padding(12.dp)
                )

                // 内联确认按钮（邀请/消息/删除预览）
                if (message.awaitingAction && (onConfirm != null || onCancel != null)) {
                    Divider(color = CLIColors.Border, thickness = 1.dp)

                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(12.dp),
                        horizontalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        // 取消按钮
                        Text(
                            text = "取消",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.TextSecondary,
                            textAlign = TextAlign.Center,
                            modifier = Modifier
                                .weight(1f)
                                .border(1.dp, CLIColors.Border)
                                .clickable { onCancel?.invoke() }
                                .padding(vertical = 8.dp)
                        )

                        // 确认按钮
                        Text(
                            text = confirmButtonText,
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.Background,
                            textAlign = TextAlign.Center,
                            modifier = Modifier
                                .weight(1f)
                                .background(CLIColors.Green)
                                .clickable { onConfirm?.invoke() }
                                .padding(vertical = 8.dp)
                        )
                    }
                }
            }
        }

        ChatMessageType.AI_SCHEDULE -> {
            message.schedule?.let { schedule ->
                SchedulePreview(
                    schedule = schedule,
                    reasoning = message.reasoning ?: emptyList(),
                    isQuery = message.isQuery,
                    voiceState = voiceState,
                    onConfirm = onConfirm,
                    onCancel = onCancel,
                    confirmButtonText = confirmButtonText
                )
            } ?: Text(
                text = message.content,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextPrimary
            )
        }

        ChatMessageType.AI_QUESTION -> {
            message.questions?.let { questions ->
                QuestionPreview(
                    questions = questions,
                    content = message.content
                )
            } ?: Text(
                text = message.content,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextPrimary
            )
        }

        ChatMessageType.SYSTEM -> {
            Text(
                text = message.content,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.Yellow,
                modifier = Modifier
                    .fillMaxWidth()
                    .background(CLIColors.Yellow.copy(alpha = 0.1f))
                    .padding(8.dp)
            )
        }
    }
}

// MARK: - Schedule Preview

/**
 * 时刻表预览组件
 */
@Composable
fun SchedulePreview(
    schedule: List<ScheduleItem>,
    reasoning: List<String> = emptyList(),
    isQuery: Boolean = false,
    voiceState: VoiceScheduleState = VoiceScheduleState.IDLE,
    onConfirm: (() -> Unit)? = null,
    onCancel: (() -> Unit)? = null,
    confirmButtonText: String = "✓ 确认保存",
    modifier: Modifier = Modifier
) {
    var showReasoning by remember { mutableStateOf(false) }
    var actionTaken by remember { mutableStateOf<String?>(null) } // "confirming", "confirmed", "cancelled"

    Column(
        modifier = modifier
            .background(CLIColors.BackgroundSecondary)
            .border(1.dp, CLIColors.Cyan.copy(alpha = 0.3f))
            .padding(12.dp)
    ) {
        // 标题
        Text(
            text = if (isQuery) "📋 当前时刻表：" else "📋 生成的时刻表：",
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = CLIColors.Cyan
        )

        Spacer(modifier = Modifier.height(8.dp))

        // 时刻表条目
        schedule.forEach { item ->
            ScheduleItemRow(item = item)
            Spacer(modifier = Modifier.height(4.dp))
        }

        // 推理依据（可展开）
        if (reasoning.isNotEmpty()) {
            Divider(
                color = CLIColors.Border,
                modifier = Modifier.padding(vertical = 8.dp)
            )

            TextButton(
                onClick = { showReasoning = !showReasoning }
            ) {
                Text(
                    text = if (isQuery) "说明 ${if (showReasoning) "▲" else "▼"}" else "推理依据 ${if (showReasoning) "▲" else "▼"}",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextWeak
                )
            }

            AnimatedVisibility(visible = showReasoning) {
                Column {
                    reasoning.forEach { reason ->
                        Row(modifier = Modifier.padding(start = 4.dp)) {
                            Text(
                                text = "•",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 12.sp,
                                color = CLIColors.TextWeak
                            )
                            Spacer(modifier = Modifier.width(6.dp))
                            Text(
                                text = reason,
                                fontFamily = FontFamily.Monospace,
                                fontSize = 12.sp,
                                color = CLIColors.TextSecondary
                            )
                        }
                    }
                }
            }
        }

        // 操作区域（非查询模式）
        if (!isQuery && (onConfirm != null || onCancel != null)) {
            Divider(
                color = CLIColors.Border,
                modifier = Modifier.padding(vertical = 8.dp)
            )

            val isConfirmed = actionTaken == "confirmed" || voiceState == VoiceScheduleState.COMPLETED
            val isConfirming = actionTaken == "confirming" || voiceState == VoiceScheduleState.CONFIRMING
            val isCancelled = actionTaken == "cancelled"

            when {
                isConfirmed -> {
                    Text(
                        text = "✓ 已保存",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.Green,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 8.dp),
                        textAlign = TextAlign.Center
                    )
                }
                isConfirming -> {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 8.dp),
                        horizontalArrangement = Arrangement.Center,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(14.dp),
                            strokeWidth = 2.dp,
                            color = CLIColors.Green
                        )
                        Spacer(modifier = Modifier.width(6.dp))
                        Text(
                            text = "确认中...",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.TextSecondary
                        )
                    }
                }
                isCancelled -> {
                    Text(
                        text = "已放弃",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.TextWeak,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 8.dp),
                        textAlign = TextAlign.Center
                    )
                }
                else -> {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        TextButton(
                            onClick = {
                                actionTaken = "cancelled"
                                onCancel?.invoke()
                            },
                            modifier = Modifier
                                .weight(1f)
                                .border(1.dp, CLIColors.Border)
                        ) {
                            Text(
                                text = "放弃",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 12.sp,
                                color = CLIColors.TextSecondary
                            )
                        }

                        TextButton(
                            onClick = {
                                actionTaken = "confirming"
                                onConfirm?.invoke()
                            },
                            modifier = Modifier
                                .weight(1f)
                                .background(CLIColors.Green)
                        ) {
                            Text(
                                text = confirmButtonText,
                                fontFamily = FontFamily.Monospace,
                                fontSize = 12.sp,
                                color = CLIColors.Background
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(4.dp))

                    Text(
                        text = "继续说话可修改内容",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 10.sp,
                        color = CLIColors.TextWeak,
                        modifier = Modifier.padding(start = 4.dp)
                    )
                }
            }
        }
    }
}

@Composable
private fun ScheduleItemRow(item: ScheduleItem) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = "${item.startTime}-${item.endTime}",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = CLIColors.TextWeak,
            modifier = Modifier.width(90.dp)
        )

        Text(
            text = item.emoji,
            fontSize = 16.sp
        )

        Text(
            text = item.status,
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = CLIColors.TextPrimary,
            modifier = Modifier.weight(1f)
        )

        if (item.executed == true) {
            Text(
                text = "✓",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.Green
            )
        }
    }
}

// MARK: - Question Preview

/**
 * 问题预览组件
 */
@Composable
fun QuestionPreview(
    questions: List<ClarifyQuestion>,
    content: String,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .background(CLIColors.BackgroundSecondary)
            .border(1.dp, CLIColors.Yellow.copy(alpha = 0.3f))
            .padding(12.dp)
    ) {
        Text(
            text = content,
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = CLIColors.TextPrimary
        )

        Spacer(modifier = Modifier.height(8.dp))

        questions.forEach { question ->
            Column {
                Text(
                    text = "• ${question.question}",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Yellow
                )

                // 选项
                if (question.options.isNotEmpty()) {
                    FlowRow(
                        modifier = Modifier.padding(start = 12.dp, top = 4.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        question.options.forEach { option ->
                            Text(
                                text = option,
                                fontFamily = FontFamily.Monospace,
                                fontSize = 10.sp,
                                color = CLIColors.TextSecondary,
                                modifier = Modifier
                                    .background(CLIColors.Background)
                                    .border(1.dp, CLIColors.Border)
                                    .padding(horizontal = 8.dp, vertical = 4.dp)
                            )
                        }
                    }
                }

                Spacer(modifier = Modifier.height(4.dp))
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "请语音回答以上问题",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = CLIColors.TextWeak
        )
    }
}

// MARK: - Progress Feedback

/**
 * 进度反馈组件
 */
@Composable
fun ProgressFeedbackView(
    progressItems: List<ProgressItem>,
    isProcessing: Boolean,
    processingStatus: String,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .background(CLIColors.BackgroundSecondary)
            .border(1.dp, CLIColors.Cyan.copy(alpha = 0.3f))
            .padding(12.dp)
    ) {
        // 已完成的进度项
        progressItems.forEachIndexed { index, item ->
            val isLatest = index == progressItems.lastIndex && isProcessing

            Row(
                verticalAlignment = Alignment.Top,
                modifier = Modifier.padding(vertical = 2.dp)
            ) {
                // 状态图标
                if (isLatest && !item.isCompleted) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(16.dp),
                        strokeWidth = 2.dp,
                        color = CLIColors.Cyan
                    )
                } else {
                    Text(
                        text = "✓",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.Green,
                        modifier = Modifier.width(16.dp)
                    )
                }

                Spacer(modifier = Modifier.width(8.dp))

                Column {
                    Text(
                        text = item.message,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = if (isLatest) CLIColors.TextPrimary else CLIColors.TextSecondary
                    )

                    item.detail?.let { detail ->
                        Text(
                            text = "→ $detail",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.Cyan
                        )
                    }
                }
            }
        }

        // 当前处理状态（如果没有进度项）
        if (progressItems.isEmpty() && isProcessing) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                CircularProgressIndicator(
                    modifier = Modifier.size(16.dp),
                    strokeWidth = 2.dp,
                    color = CLIColors.Cyan
                )

                Text(
                    text = processingStatus,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Cyan
                )
            }
        }
    }
}

// MARK: - Simple Processing View

/**
 * 简化的处理中视图（有消息后使用）
 */
@Composable
fun SimpleProcessingView(
    isProcessing: Boolean,
    modifier: Modifier = Modifier
) {
    if (isProcessing) {
        Row(
            modifier = modifier
                .fillMaxWidth()
                .background(CLIColors.BackgroundSecondary)
                .border(1.dp, CLIColors.Cyan.copy(alpha = 0.3f))
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            CircularProgressIndicator(
                modifier = Modifier.size(16.dp),
                strokeWidth = 2.dp,
                color = CLIColors.Cyan
            )

            Text(
                text = "处理中...",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary
            )
        }
    }
}

// MARK: - Visibility Selection

/**
 * 可见性选择组件
 */
@Composable
fun VisibilitySelectionView(
    selectedVisibility: ScheduleVisibility,
    selectedCircleIds: Set<String>,
    availableCircles: List<CircleInfoCompact>,
    onVisibilitySelected: (ScheduleVisibility) -> Unit,
    onCircleToggled: (String) -> Unit,
    onConfirm: () -> Unit,
    modifier: Modifier = Modifier
) {
    var showCircleSelection by remember { mutableStateOf(false) }

    Column(
        modifier = modifier
            .background(CLIColors.BackgroundSecondary)
            .border(1.dp, CLIColors.Yellow.copy(alpha = 0.3f))
            .padding(12.dp)
    ) {
        Text(
            text = "这些状态谁可以看到？",
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = CLIColors.TextPrimary
        )

        Spacer(modifier = Modifier.height(12.dp))

        // 可见性选项
        ScheduleVisibility.entries.forEach { visibility ->
            VisibilityOptionRow(
                visibility = visibility,
                isSelected = selectedVisibility == visibility,
                onClick = {
                    onVisibilitySelected(visibility)
                    if (visibility == ScheduleVisibility.CIRCLES) {
                        showCircleSelection = true
                    }
                }
            )
            Spacer(modifier = Modifier.height(8.dp))
        }

        // 已选择的圈子
        if (selectedVisibility == ScheduleVisibility.CIRCLES && selectedCircleIds.isNotEmpty()) {
            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = "已选择的圈子：",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextWeak
            )

            Spacer(modifier = Modifier.height(4.dp))

            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                availableCircles.filter { selectedCircleIds.contains(it.id) }.forEach { circle ->
                    Text(
                        text = "${circle.emoji} ${circle.name}",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.TextPrimary,
                        modifier = Modifier
                            .background(CLIColors.Cyan.copy(alpha = 0.2f))
                            .padding(horizontal = 8.dp, vertical = 4.dp)
                    )
                }
            }

            Spacer(modifier = Modifier.height(4.dp))

            TextButton(onClick = { showCircleSelection = true }) {
                Text(
                    text = "[修改]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Cyan
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // 确认按钮
        val canConfirm = selectedVisibility != ScheduleVisibility.CIRCLES || selectedCircleIds.isNotEmpty()
        TextButton(
            onClick = onConfirm,
            enabled = canConfirm,
            modifier = Modifier
                .fillMaxWidth()
                .background(if (canConfirm) CLIColors.Green else CLIColors.TextWeak)
        ) {
            Text(
                text = "✓ 确认并执行",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.Background
            )
        }
    }

    // 圈子选择弹窗
    if (showCircleSelection) {
        CircleSelectionDialog(
            circles = availableCircles,
            selectedIds = selectedCircleIds,
            onCircleToggled = onCircleToggled,
            onDismiss = { showCircleSelection = false }
        )
    }
}

@Composable
private fun VisibilityOptionRow(
    visibility: ScheduleVisibility,
    isSelected: Boolean,
    onClick: () -> Unit
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(CLIColors.Background)
            .border(1.dp, if (isSelected) CLIColors.Cyan else CLIColors.Border)
            .clickable { onClick() }
            .padding(10.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = visibility.label,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextPrimary
            )
            Text(
                text = visibility.description,
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextWeak
            )
        }

        if (isSelected) {
            Text(
                text = "✓",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.Cyan
            )
        }
    }
}

@Composable
private fun CircleSelectionDialog(
    circles: List<CircleInfoCompact>,
    selectedIds: Set<String>,
    onCircleToggled: (String) -> Unit,
    onDismiss: () -> Unit
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Text(
                text = "━━ 选择可见的圈子 ━━",
                fontFamily = FontFamily.Monospace,
                color = CLIColors.Cyan
            )
        },
        text = {
            Column {
                circles.forEach { circle ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { onCircleToggled(circle.id) }
                            .padding(vertical = 8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = circle.emoji,
                            fontSize = 20.sp
                        )

                        Spacer(modifier = Modifier.width(8.dp))

                        Text(
                            text = circle.name,
                            fontFamily = FontFamily.Monospace,
                            fontSize = 14.sp,
                            color = CLIColors.TextPrimary,
                            modifier = Modifier.weight(1f)
                        )

                        Text(
                            text = "${circle.memberCount}人",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.TextWeak
                        )

                        Spacer(modifier = Modifier.width(8.dp))

                        if (selectedIds.contains(circle.id)) {
                            Text(
                                text = "✓",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 14.sp,
                                color = CLIColors.Green
                            )
                        }
                    }

                    Divider(color = CLIColors.Border)
                }
            }
        },
        confirmButton = {
            TextButton(onClick = onDismiss) {
                Text(
                    text = "完成",
                    fontFamily = FontFamily.Monospace,
                    color = CLIColors.Green
                )
            }
        },
        containerColor = CLIColors.Background
    )
}

// MARK: - Helpers

private fun formatTime(timestamp: Long): String {
    val sdf = SimpleDateFormat("HH:mm", Locale.getDefault())
    return sdf.format(Date(timestamp))
}
