package com.youkong.feature.home.component

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.DateRange
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.youkong.core.domain.model.Availability
import com.youkong.core.domain.model.LocationType
import com.youkong.core.ui.component.YouKongAvatar
import com.youkong.core.ui.component.YouKongCard
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongTheme
import kotlinx.datetime.Clock
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

@Composable
fun AvailabilityCard(
    availability: Availability,
    isMine: Boolean = false,
    onCancelClick: (() -> Unit)? = null,
    onClick: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    YouKongCard(
        modifier = modifier,
        onClick = onClick,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(YouKongTheme.spacing.lg),
        ) {
            // 头部：用户信息
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                YouKongAvatar(
                    imageUrl = availability.user.avatar,
                    name = availability.user.nickname,
                    size = 40.dp,
                )

                Spacer(modifier = Modifier.width(12.dp))

                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = availability.user.nickname,
                        style = MaterialTheme.typography.titleMedium,
                        color = TextPrimary,
                    )

                    Text(
                        text = formatRelativeTime(availability),
                        style = MaterialTheme.typography.bodySmall,
                        color = TextSecondary,
                    )
                }

                if (isMine && onCancelClick != null) {
                    IconButton(onClick = onCancelClick) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = "取消",
                            tint = TextSecondary,
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            // 时间信息
            Row(
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    imageVector = Icons.Default.DateRange,
                    contentDescription = null,
                    tint = Primary,
                    modifier = Modifier.padding(end = 8.dp),
                )
                Text(
                    text = formatTimeRange(availability),
                    style = MaterialTheme.typography.bodyMedium,
                    color = TextPrimary,
                )
            }

            Spacer(modifier = Modifier.height(8.dp))

            // 地点信息
            Row(
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    imageVector = Icons.Default.LocationOn,
                    contentDescription = null,
                    tint = Primary,
                    modifier = Modifier.padding(end = 8.dp),
                )
                Text(
                    text = formatLocation(availability),
                    style = MaterialTheme.typography.bodyMedium,
                    color = TextPrimary,
                )
            }
        }
    }
}

private fun formatRelativeTime(availability: Availability): String {
    val now = Clock.System.now()
    val diff = availability.startTime - now
    val hours = diff.inWholeHours
    val minutes = diff.inWholeMinutes

    return when {
        hours < 0 -> "进行中"
        hours == 0L -> "${minutes}分钟后"
        hours < 24 -> "${hours}小时后"
        hours < 48 -> "明天"
        else -> {
            val date = availability.startTime.toLocalDateTime(TimeZone.currentSystemDefault())
            "${date.monthNumber}月${date.dayOfMonth}日"
        }
    }
}

private fun formatTimeRange(availability: Availability): String {
    val tz = TimeZone.currentSystemDefault()
    val start = availability.startTime.toLocalDateTime(tz)
    val end = availability.endTime.toLocalDateTime(tz)

    val startTimeStr = "${start.hour.toString().padStart(2, '0')}:${start.minute.toString().padStart(2, '0')}"
    val endTimeStr = "${end.hour.toString().padStart(2, '0')}:${end.minute.toString().padStart(2, '0')}"

    val dateStr = if (start.dayOfMonth != Clock.System.now().toLocalDateTime(tz).dayOfMonth) {
        "${start.monthNumber}月${start.dayOfMonth}日 "
    } else {
        "今天 "
    }

    return "$dateStr$startTimeStr - $endTimeStr"
}

private fun formatLocation(availability: Availability): String {
    return when (availability.location.type) {
        LocationType.FLEXIBLE -> "地点灵活"
        LocationType.PRESET -> availability.location.name ?: "未知地点"
        LocationType.CUSTOM -> availability.location.name ?: "自定义地点"
    }
}
