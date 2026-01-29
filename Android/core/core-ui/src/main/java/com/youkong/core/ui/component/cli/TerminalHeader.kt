package com.youkong.core.ui.component.cli

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
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
 * Terminal Header - CLI 风格页面头部（替代 TopAppBar）
 */
@Composable
fun TerminalHeader(
    title: String,
    modifier: Modifier = Modifier,
    subtitle: String? = null,
    showBackButton: Boolean = false,
    onBackClick: (() -> Unit)? = null,
    trailingIcon: String? = null,
    onTrailingClick: (() -> Unit)? = null,
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .background(CLIColors.Background)
    ) {
        // 头部内容
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // 返回按钮
            if (showBackButton && onBackClick != null) {
                Text(
                    text = "[${ASCII.ARROW_LEFT}]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 16.sp,
                    color = CLIColors.TextPrimary,
                    modifier = Modifier
                        .clickable(onClick = onBackClick)
                        .padding(vertical = 4.dp, horizontal = 8.dp)
                )

                Spacer(modifier = Modifier.width(8.dp))
            }

            // 标题
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = title,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 16.sp,
                    color = CLIColors.TextPrimary
                )

                if (subtitle != null) {
                    Text(
                        text = subtitle,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        color = CLIColors.TextSecondary
                    )
                }
            }

            // 右侧按钮
            if (trailingIcon != null && onTrailingClick != null) {
                Text(
                    text = trailingIcon,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.TextSecondary,
                    modifier = Modifier.clickable(onClick = onTrailingClick)
                )
            }
        }

        // 分隔线
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(1.dp)
                .background(CLIColors.Border)
        )
    }
}

/**
 * Simple Terminal Header - 简化版头部（只有标题分隔线）
 */
@Composable
fun SimpleTerminalHeader(
    title: String,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .background(CLIColors.Background)
    ) {
        Text(
            text = ASCII.titleLine(title, totalLength = 50),
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = CLIColors.Border,
            modifier = Modifier.padding(vertical = 12.dp)
        )

        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(1.dp)
                .background(CLIColors.Border)
        )
    }
}
