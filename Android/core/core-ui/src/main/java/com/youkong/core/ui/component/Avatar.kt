package com.youkong.core.ui.component

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.component.cli.TerminalAvatar
import com.youkong.core.ui.theme.CLIColors

@Composable
fun YouKongAvatar(
    imageUrl: String?,
    name: String,
    modifier: Modifier = Modifier,
    size: Dp = 40.dp,
) {
    // CLI 风格：始终使用 TerminalAvatar（状态点）
    Box(
        modifier = modifier.size(size),
        contentAlignment = Alignment.Center,
    ) {
        TerminalAvatar(isOnline = true)
    }
}

@Composable
fun YouKongEmojiAvatar(
    emoji: String,
    backgroundColor: androidx.compose.ui.graphics.Color,
    modifier: Modifier = Modifier,
    size: Dp = 40.dp,
) {
    // CLI 风格：仅显示 emoji（无圆形背景）
    Box(
        modifier = modifier.size(size),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = emoji,
            fontSize = (size.value * 0.6).sp,
            fontFamily = FontFamily.Monospace,
            color = CLIColors.TextPrimary,
        )
    }
}
