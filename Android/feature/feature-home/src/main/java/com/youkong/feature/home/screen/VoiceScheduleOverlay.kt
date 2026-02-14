package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.network.model.VoiceScheduleState
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.home.component.*
import com.youkong.feature.home.viewmodel.VoiceScheduleUiState
import kotlinx.coroutines.launch

/**
 * 语音时刻表覆盖层
 *
 * 显示对话过程、时刻表预览、确认按钮等
 */
@Composable
fun VoiceScheduleOverlay(
    uiState: VoiceScheduleUiState,
    onDismiss: () -> Unit,
    onConfirm: () -> Unit,
    onCancel: () -> Unit,
    onVisibilitySelected: (com.youkong.core.network.model.ScheduleVisibility) -> Unit,
    onCircleToggled: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    val listState = rememberLazyListState()
    val coroutineScope = rememberCoroutineScope()

    // 自动滚动到最新消息
    LaunchedEffect(uiState.messages.size, uiState.progressItems.size) {
        if (uiState.messages.isNotEmpty() || uiState.progressItems.isNotEmpty()) {
            coroutineScope.launch {
                listState.animateScrollToItem(
                    index = maxOf(0, uiState.messages.size - 1 + (if (uiState.state == VoiceScheduleState.PROCESSING) 1 else 0))
                )
            }
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(CLIColors.Background)
    ) {
        // 顶部关闭按钮
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 16.dp),
            horizontalArrangement = Arrangement.End
        ) {
            IconButton(
                onClick = {
                    onCancel()
                    onDismiss()
                }
            ) {
                Icon(
                    imageVector = Icons.Default.Close,
                    contentDescription = "关闭",
                    tint = CLIColors.TextSecondary
                )
            }
        }

        // 内容区域
        LazyColumn(
            state = listState,
            modifier = Modifier
                .weight(1f)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // 消息列表
            items(
                items = uiState.messages,
                key = { it.id }
            ) { message ->
                val confirmText = when {
                    uiState.pendingInvite != null -> "✓ 发送邀请"
                    uiState.pendingMessage != null -> "✓ 发送消息"
                    uiState.pendingDeletion != null -> "✓ 确认删除"
                    else -> "✓ 确认保存"
                }
                MessageBubble(
                    message = message,
                    voiceState = uiState.state,
                    onConfirm = onConfirm,
                    onCancel = onCancel,
                    confirmButtonText = confirmText
                )
            }

            // 处理中状态 - 始终显示详细进度反馈
            if (uiState.state == VoiceScheduleState.PROCESSING ||
                uiState.state == VoiceScheduleState.CONFIRMING ||
                uiState.progressItems.isNotEmpty()) {
                item(key = "progressFeedback") {
                    ProgressFeedbackView(
                        progressItems = uiState.progressItems,
                        isProcessing = uiState.state == VoiceScheduleState.PROCESSING ||
                                      uiState.state == VoiceScheduleState.CONFIRMING,
                        processingStatus = uiState.processingStatus
                    )
                }
            }

            // 可见性选择
            if (uiState.showVisibilitySelection) {
                item(key = "visibilitySelection") {
                    VisibilitySelectionView(
                        selectedVisibility = uiState.selectedVisibility,
                        selectedCircleIds = uiState.selectedCircleIds,
                        availableCircles = uiState.availableCircles,
                        onVisibilitySelected = onVisibilitySelected,
                        onCircleToggled = onCircleToggled,
                        onConfirm = onConfirm
                    )
                }
            }

            // 底部留白
            item {
                Spacer(modifier = Modifier.height(16.dp))
            }
        }

        // 底部浮框确认已移除，确认操作统一在消息气泡内完成
    }
}

@Composable
private fun ApprovalButtons(
    onCancel: () -> Unit,
    onConfirm: () -> Unit,
    isConfirming: Boolean,
    confirmButtonText: String = "✓ 确认保存",
    modifier: Modifier = Modifier
) {
    Column(modifier = modifier) {
        Divider(color = CLIColors.Border.copy(alpha = 0.5f))

        Spacer(modifier = Modifier.height(12.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // 取消按钮
            OutlinedButton(
                onClick = onCancel,
                modifier = Modifier.weight(1f),
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = CLIColors.TextSecondary
                )
            ) {
                Text(
                    text = "取消",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp
                )
            }

            // 确认按钮
            Button(
                onClick = onConfirm,
                enabled = !isConfirming,
                modifier = Modifier.weight(1f),
                colors = ButtonDefaults.buttonColors(
                    containerColor = CLIColors.Green,
                    contentColor = CLIColors.Background
                )
            ) {
                Text(
                    text = confirmButtonText,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp
                )
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "继续按住首页按钮说话可修改",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = CLIColors.TextWeak,
            modifier = Modifier.align(Alignment.CenterHorizontally)
        )
    }
}
