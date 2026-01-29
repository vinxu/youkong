package com.youkong.core.ui.component.cli

import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors

/**
 * Terminal Progress Bar - 用 ASCII 字符显示概率进度条
 * @param probability 概率值 (0-100)，-1 表示无数据
 * @param totalBlocks 总块数（默认 10）
 */
@Composable
fun TerminalProgressBar(
    probability: Int,
    totalBlocks: Int = 10,
) {
    val color = CLIColors.probabilityColor(probability)

    Row {
        // 进度条
        Text(
            text = ASCII.progressBar(probability, totalBlocks),
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = color
        )

        Spacer(modifier = Modifier.width(4.dp))

        // 百分比文本
        Text(
            text = "$probability%",
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = color
        )
    }
}
