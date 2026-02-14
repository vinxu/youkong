package com.youkong.feature.home.component

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.*
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.scale
import androidx.compose.ui.input.pointer.changedToUp
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.input.pointer.positionChange
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.CLIColors
import kotlinx.coroutines.launch

/**
 * 微信式长按录音按钮
 *
 * 特点：
 * - 长按开始录音
 * - 松开发送
 * - 上滑取消
 */
@Composable
fun WechatStyleVoiceButton(
    isRecording: Boolean,
    isCancelling: Boolean,
    recordingDurationMs: Long,
    onStart: () -> Unit,
    onEnd: () -> Unit,
    onCancel: () -> Unit,
    onCancellingChanged: (Boolean) -> Unit,
    modifier: Modifier = Modifier
) {
    // 取消阈值（dp）
    val cancelThreshold = -80f
    val showCancelZoneThreshold = -30f

    var dragOffsetY by remember { mutableFloatStateOf(0f) }
    var isInCancelZone by remember { mutableStateOf(false) }
    var showCancelZone by remember { mutableStateOf(false) }
    var isPressed by remember { mutableStateOf(false) }
    val coroutineScope = rememberCoroutineScope()

    // 更新取消状态
    LaunchedEffect(isInCancelZone) {
        onCancellingChanged(isInCancelZone)
    }

    Column(
        modifier = modifier,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // 取消区域（录音时显示）
        AnimatedVisibility(
            visible = isRecording && showCancelZone,
            enter = fadeIn(),
            exit = fadeOut()
        ) {
            CancelZone(
                isInCancelZone = isInCancelZone,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 16.dp)
            )
        }

        // 录音状态提示
        AnimatedVisibility(
            visible = isRecording && !isInCancelZone,
            enter = fadeIn(),
            exit = fadeOut()
        ) {
            RecordingStatus(
                durationMs = recordingDurationMs,
                showUpHint = !showCancelZone,
                modifier = Modifier.padding(bottom = 8.dp)
            )
        }

        // 录音按钮
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(44.dp)
                .background(
                    when {
                        isRecording && isInCancelZone -> CLIColors.Red.copy(alpha = 0.1f)
                        isRecording -> CLIColors.Green.copy(alpha = 0.1f)
                        else -> CLIColors.Green
                    }
                )
                .pointerInput(Unit) {
                    awaitPointerEventScope {
                        while (true) {
                            // 等待按下
                            awaitFirstDown(requireUnconsumed = false)
                            isPressed = true
                            dragOffsetY = 0f
                            isInCancelZone = false
                            showCancelZone = false
                            onStart()

                            // 追踪拖拽直到松开
                            try {
                                while (true) {
                                    val event = awaitPointerEvent()
                                    val allUp = event.changes.all { it.changedToUp() }
                                    if (allUp) {
                                        // 松开
                                        event.changes.forEach { it.consume() }
                                        break
                                    }
                                    // 追踪拖拽偏移
                                    event.changes.forEach { change ->
                                        val delta = change.positionChange()
                                        dragOffsetY += delta.y
                                        if (dragOffsetY < showCancelZoneThreshold) {
                                            showCancelZone = true
                                        }
                                        isInCancelZone = dragOffsetY < cancelThreshold
                                        change.consume()
                                    }
                                }
                            } catch (_: Exception) {
                                // 手势被取消
                                isPressed = false
                                onCancel()
                                dragOffsetY = 0f
                                isInCancelZone = false
                                showCancelZone = false
                                continue
                            }

                            // 松开后处理
                            isPressed = false
                            if (isInCancelZone) {
                                onCancel()
                            } else {
                                onEnd()
                            }
                            dragOffsetY = 0f
                            isInCancelZone = false
                            showCancelZone = false
                        }
                    }
                },
            contentAlignment = Alignment.Center
        ) {
            Text(
                text = when {
                    isRecording && isInCancelZone -> "松开取消"
                    isRecording -> "松开发送"
                    else -> "🎤 和AI说说你的行程"
                },
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                maxLines = 1,
                color = when {
                    isRecording && isInCancelZone -> CLIColors.Red
                    isRecording -> CLIColors.Green
                    else -> CLIColors.Background
                }
            )
        }
    }
}

/**
 * 取消区域
 */
@Composable
private fun CancelZone(
    isInCancelZone: Boolean,
    modifier: Modifier = Modifier
) {
    val scale by animateFloatAsState(
        targetValue = if (isInCancelZone) 1.2f else 1f,
        animationSpec = spring(
            dampingRatio = Spring.DampingRatioMediumBouncy,
            stiffness = Spring.StiffnessLow
        ),
        label = "scale"
    )

    Column(
        modifier = modifier
            .background(
                color = if (isInCancelZone) CLIColors.Red.copy(alpha = 0.15f) else CLIColors.BackgroundSecondary,
                shape = RoundedCornerShape(12.dp)
            )
            .border(
                width = if (isInCancelZone) 2.dp else 1.dp,
                color = if (isInCancelZone) CLIColors.Red else CLIColors.Border,
                shape = RoundedCornerShape(12.dp)
            )
            .padding(vertical = 16.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Icon(
            imageVector = Icons.Default.Close,
            contentDescription = "取消",
            tint = if (isInCancelZone) CLIColors.Red else CLIColors.TextWeak,
            modifier = Modifier
                .size(40.dp)
                .scale(scale)
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = if (isInCancelZone) "松开取消" else "上滑到此取消",
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = if (isInCancelZone) CLIColors.Red else CLIColors.TextWeak
        )
    }
}

/**
 * 录音状态显示
 */
@Composable
private fun RecordingStatus(
    durationMs: Long,
    showUpHint: Boolean,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier,
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        // 录音波形动画
        RecordingWaveform()

        // 录音时长
        Text(
            text = formatDuration(durationMs),
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = CLIColors.Green
        )

        // 上滑提示
        if (showUpHint) {
            Text(
                text = "↑ 上滑取消",
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextWeak
            )
        }
    }
}

/**
 * 录音波形动画
 */
@Composable
private fun RecordingWaveform() {
    val infiniteTransition = rememberInfiniteTransition(label = "waveform")
    val animationPhase by infiniteTransition.animateFloat(
        initialValue = 0f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "phase"
    )

    Row(
        horizontalArrangement = Arrangement.spacedBy(3.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        val baseHeights = listOf(8f, 16f, 20f, 14f, 10f)
        baseHeights.forEachIndexed { index, baseHeight ->
            val offset = index * 0.3f
            val height = baseHeight + kotlin.math.sin((animationPhase * Math.PI + offset).toFloat()) * 8
            Box(
                modifier = Modifier
                    .width(4.dp)
                    .height(height.dp)
                    .background(
                        color = CLIColors.Green,
                        shape = RoundedCornerShape(2.dp)
                    )
            )
        }
    }
}

/**
 * 格式化录音时长
 */
private fun formatDuration(ms: Long): String {
    val seconds = (ms / 1000).toInt()
    return "$seconds\""
}
