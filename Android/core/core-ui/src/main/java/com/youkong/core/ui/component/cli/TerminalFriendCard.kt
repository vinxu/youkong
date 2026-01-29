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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors

/**
 * Terminal Friend Card - 极简单行好友列表项
 * 格式：● 张三              🎮 在玩游戏  ▇▇▇▇▇▇▇▇▇░ 95% →
 *
 * @param friendId 好友 ID
 * @param name 好友名字
 * @param probability 有空概率 (0-100)，-1 表示无数据
 * @param reason 原因描述
 * @param emoji 活动 emoji（可选）
 * @param activity 活动描述（可选）
 * @param onClick 点击回调
 */
@Composable
fun TerminalFriendCard(
    friendId: String,
    name: String,
    probability: Int,
    reason: String,
    emoji: String? = null,
    activity: String? = null,
    onClick: () -> Unit,
) {
    val hasData = probability >= 0

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .background(CLIColors.Background)
    ) {
        // 主内容行
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // 状态点
            TerminalAvatar(isOnline = hasData)

            Spacer(modifier = Modifier.width(8.dp))

            // 名字（固定宽度）
            Box(modifier = Modifier.width(100.dp)) {
                Text(
                    text = name,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.TextPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
            }

            Spacer(modifier = Modifier.weight(1f))

            // 活动（emoji + 描述）
            val displayText = buildString {
                if (emoji != null) append("$emoji ")
                append(activity ?: reason)
            }

            Text(
                text = displayText,
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = CLIColors.TextSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            Spacer(modifier = Modifier.weight(1f))

            // 进度条或无数据
            if (hasData) {
                TerminalProgressBar(probability = probability)
            } else {
                Text(
                    text = "无数据",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 13.sp,
                    color = CLIColors.TextSecondary
                )
            }

            Spacer(modifier = Modifier.width(8.dp))

            // 箭头
            Text(
                text = ASCII.ARROW_RIGHT,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary
            )
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
