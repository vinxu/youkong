package com.youkong.feature.chat.component

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.youkong.core.domain.model.Message
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextOnPrimary
import com.youkong.core.ui.theme.TextPrimary

/**
 * 消息气泡组件
 */
@Composable
fun MessageBubble(
    message: Message,
    isMe: Boolean,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier = modifier.fillMaxWidth(),
        contentAlignment = if (isMe) Alignment.CenterEnd else Alignment.CenterStart,
    ) {
        Surface(
            color = if (isMe) Primary else MaterialTheme.colorScheme.surfaceVariant,
            shape = MaterialTheme.shapes.medium,
        ) {
            Text(
                text = message.content ?: "",
                style = MaterialTheme.typography.bodyMedium,
                color = if (isMe) TextOnPrimary else TextPrimary,
                modifier = Modifier.padding(
                    horizontal = 12.dp,
                    vertical = 8.dp,
                ),
            )
        }
    }
}
