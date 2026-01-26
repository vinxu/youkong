package com.youkong.feature.friends.component

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import com.youkong.core.ui.theme.ProbabilityColors

/**
 * 概率指示器组件
 * 显示5个圆点和百分比
 */
@Composable
fun ProbabilityIndicator(
    probability: Int?,
    modifier: Modifier = Modifier,
) {
    val color = ProbabilityColors.fromProbability(probability)
    val filledDots = when {
        probability == null || probability < 0 -> 0
        probability >= 80 -> 5
        probability >= 60 -> 4
        probability >= 40 -> 3
        probability >= 20 -> 2
        probability > 0 -> 1
        else -> 0
    }

    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        // 5个圆点
        repeat(5) { index ->
            Box(
                modifier = Modifier
                    .size(8.dp)
                    .clip(CircleShape)
                    .background(
                        if (index < filledDots) color else color.copy(alpha = 0.2f)
                    )
            )
        }

        // 百分比文字
        Text(
            text = if (probability != null && probability >= 0) {
                "$probability%"
            } else {
                "--"
            },
            style = MaterialTheme.typography.bodyMedium,
            color = color,
        )
    }
}

/**
 * 简单的概率徽章
 */
@Composable
fun ProbabilityBadge(
    probability: Int?,
    modifier: Modifier = Modifier,
) {
    val color = ProbabilityColors.fromProbability(probability)
    val text = if (probability != null && probability >= 0) {
        "$probability%"
    } else {
        "--"
    }

    Box(
        modifier = modifier
            .clip(MaterialTheme.shapes.small)
            .background(color.copy(alpha = 0.1f)),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelMedium,
            color = color,
        )
    }
}
