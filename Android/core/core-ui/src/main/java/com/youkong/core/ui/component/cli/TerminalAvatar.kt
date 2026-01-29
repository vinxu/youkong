package com.youkong.core.ui.component.cli

import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors

/**
 * Terminal Avatar - 用状态点 (●/○) 替代头像图片
 * @param isOnline 是否在线/有数据
 */
@Composable
fun TerminalAvatar(isOnline: Boolean = true) {
    Text(
        text = if (isOnline) ASCII.BULLET else ASCII.BULLET_EMPTY,
        fontFamily = FontFamily.Monospace,
        fontSize = 14.sp,
        color = if (isOnline) CLIColors.Green else CLIColors.TextSecondary
    )
}
