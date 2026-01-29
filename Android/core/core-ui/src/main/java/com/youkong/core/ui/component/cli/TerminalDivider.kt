package com.youkong.core.ui.component.cli

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors

/**
 * Terminal Divider Style
 */
enum class TerminalDividerStyle {
    SOLID,  // 实线 ─
    HEAVY,  // 粗线 ═
    DOTTED  // 点线 ┄
}

/**
 * Terminal Divider - CLI 风格分隔线
 */
@Composable
fun TerminalDivider(
    modifier: Modifier = Modifier,
    style: TerminalDividerStyle = TerminalDividerStyle.SOLID,
) {
    Box(
        modifier = modifier
            .fillMaxWidth()
            .height(1.dp)
            .background(CLIColors.Border)
    )
}

/**
 * Terminal Section Divider - 带标题的分隔线
 */
@Composable
fun TerminalSectionDivider(
    title: String,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // 左侧短横线
        Text(
            text = ASCII.SEPARATOR,
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = CLIColors.Border,
            modifier = Modifier.width(20.dp)
        )

        // 标题
        Text(
            text = title,
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = CLIColors.TextWeak,
            modifier = Modifier.padding(horizontal = 8.dp)
        )

        // 右侧填充横线
        Box(
            modifier = Modifier
                .weight(1f)
                .height(1.dp)
                .background(CLIColors.Border)
        )
    }
}
