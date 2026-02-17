package com.youkong.core.ui.rive

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.CLIColors
import kotlin.math.min

/** 互动者简要信息（core-ui 本地定义，避免跨模块依赖） */
data class InteractorBrief(
    val emoji: String = "",
    val nickname: String = "",
)

/**
 * 多人拥挤布局 — 主角 Emoji + 互动者 Emoji，全员做同一动作
 * 使用 Microsoft Fluent Animated Emoji（APNG）展示活动状态
 */
@Composable
fun CrowdedRiveView(
    emoji: String,
    riveCharacter: String = "",
    mainGender: String = "male",
    interactors: List<InteractorBrief>,
    modifier: Modifier = Modifier,
) {
    val displayInteractors = interactors.take(8)
    val overflow = (interactors.size - 8).coerceAtLeast(0)
    val totalPeople = 1 + displayInteractors.size
    BoxWithConstraints(modifier = modifier) {
        val w = maxWidth
        val h = maxHeight
        val containerSize = min(w.value, h.value)
        val positions = layoutPositions(totalPeople)
        val sizes = characterSizes(totalPeople, containerSize)

        Box(modifier = Modifier.size(w, h)) {
            // 主角色（第一个位置，最大）— Emoji 状态动画
            val mainPos = positions[0]
            EmojiStateView(
                emoji = emoji,
                modifier = Modifier
                    .size(sizes.first.dp)
                    .offset(
                        x = (mainPos.first * w.value - sizes.first / 2).dp,
                        y = (mainPos.second * h.value - sizes.first / 2).dp,
                    ),
            )

            // 互动者角色（每人一个 Emoji 状态实例）
            displayInteractors.forEachIndexed { index, interactor ->
                val posIndex = index + 1
                if (posIndex < positions.size) {
                    val pos = positions[posIndex]
                    val subSize = sizes.second

                    Box(
                        modifier = Modifier
                            .offset(
                                x = (pos.first * w.value - subSize / 2).dp,
                                y = (pos.second * h.value - subSize / 2).dp,
                            ),
                        contentAlignment = Alignment.Center,
                    ) {
                        EmojiStateView(
                            emoji = interactor.emoji.ifEmpty { emoji },
                            modifier = Modifier.size(subSize.dp),
                        )

                        // 昵称标签（自适应大小）
                        val name = interactor.nickname
                        val nameFontSize = when {
                            name.length <= 2 -> maxOf(subSize * 0.18f, 7f)
                            name.length <= 4 -> maxOf(subSize * 0.14f, 6f)
                            name.length <= 6 -> maxOf(subSize * 0.11f, 5f)
                            else -> maxOf(subSize * 0.09f, 4.5f)
                        }
                        Text(
                            text = name,
                            fontFamily = FontFamily.Monospace,
                            fontWeight = FontWeight.Bold,
                            fontSize = nameFontSize.sp,
                            color = CLIColors.Green,
                            maxLines = 1,
                            modifier = Modifier.offset(y = (subSize * 0.45f).dp),
                        )
                    }
                }
            }

            // 溢出标记（右下角）
            if (overflow > 0) {
                Text(
                    text = "+$overflow",
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Bold,
                    fontSize = 9.sp,
                    color = CLIColors.Yellow,
                    modifier = Modifier
                        .offset(
                            x = (w.value * 0.78f).dp,
                            y = (h.value * 0.82f).dp,
                        )
                        .background(
                            CLIColors.BackgroundSecondary.copy(alpha = 0.8f),
                            RoundedCornerShape(4.dp),
                        ),
                )
            }
        }
    }
}

// 尺寸：Pair(主角色大小, 互动者大小)
private fun characterSizes(total: Int, containerSize: Float): Pair<Float, Float> = when (total) {
    1 -> containerSize * 0.85f to 0f
    2 -> containerSize * 0.55f to containerSize * 0.45f
    3 -> containerSize * 0.50f to containerSize * 0.38f
    4 -> containerSize * 0.45f to containerSize * 0.35f
    5 -> containerSize * 0.40f to containerSize * 0.32f
    else -> containerSize * 0.35f to containerSize * 0.27f
}

// 预定义位置表：返回 (x%, y%) 列表
private fun layoutPositions(total: Int): List<Pair<Float, Float>> = when (total) {
    1 -> listOf(0.5f to 0.45f)
    2 -> listOf(
        0.35f to 0.45f,
        0.70f to 0.50f,
    )
    3 -> listOf(
        0.50f to 0.30f,
        0.28f to 0.68f,
        0.72f to 0.68f,
    )
    4 -> listOf(
        0.30f to 0.30f,
        0.70f to 0.30f,
        0.30f to 0.68f,
        0.70f to 0.68f,
    )
    5 -> listOf(
        0.50f to 0.25f,
        0.20f to 0.48f,
        0.80f to 0.48f,
        0.30f to 0.75f,
        0.70f to 0.75f,
    )
    else -> {
        // 3x3 网格取前 N 个
        val grid = listOf(
            0.22f to 0.22f,
            0.50f to 0.22f,
            0.78f to 0.22f,
            0.22f to 0.50f,
            0.50f to 0.50f,
            0.78f to 0.50f,
            0.22f to 0.78f,
            0.50f to 0.78f,
            0.78f to 0.78f,
        )
        grid.take(minOf(total, 9))
    }
}
