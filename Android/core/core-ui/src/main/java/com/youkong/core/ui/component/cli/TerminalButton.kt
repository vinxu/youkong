package com.youkong.core.ui.component.cli

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.CLIColors

/**
 * Terminal Button Style
 */
enum class TerminalButtonStyle {
    PRIMARY,   // 绿色主按钮
    SECONDARY, // 蓝色次按钮
    DANGER,    // 红色危险按钮
    GHOST      // 无背景文本按钮
}

/**
 * Terminal Button - CLI 风格按钮（无圆角，纯文本）
 */
@Composable
fun TerminalButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    style: TerminalButtonStyle = TerminalButtonStyle.PRIMARY,
    enabled: Boolean = true,
) {
    val (backgroundColor, textColor, borderColor) = when (style) {
        TerminalButtonStyle.PRIMARY -> Triple(
            CLIColors.Green.copy(alpha = 0.15f),
            CLIColors.Green,
            CLIColors.Green
        )
        TerminalButtonStyle.SECONDARY -> Triple(
            CLIColors.Blue.copy(alpha = 0.15f),
            CLIColors.Blue,
            CLIColors.Blue
        )
        TerminalButtonStyle.DANGER -> Triple(
            CLIColors.Red.copy(alpha = 0.15f),
            CLIColors.Red,
            CLIColors.Red
        )
        TerminalButtonStyle.GHOST -> Triple(
            Color.Transparent,
            CLIColors.TextSecondary,
            CLIColors.Border
        )
    }

    Box(
        modifier = modifier
            .fillMaxWidth()
            .background(if (enabled) backgroundColor else CLIColors.BackgroundSecondary)
            .border(width = 1.dp, color = if (enabled) borderColor else CLIColors.Border)
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 10.dp),
        contentAlignment = Alignment.Center
    ) {
        Text(
            text = text,
            fontFamily = FontFamily.Monospace,
            fontWeight = FontWeight.Medium,
            fontSize = 14.sp,
            color = if (enabled) textColor else CLIColors.TextWeak
        )
    }
}
