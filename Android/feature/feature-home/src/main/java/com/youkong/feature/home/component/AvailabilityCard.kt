package com.youkong.feature.home.component

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
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
import com.youkong.core.domain.model.Availability
import com.youkong.core.domain.model.LocationType
import com.youkong.core.ui.component.cli.TerminalAvatar
import com.youkong.core.ui.component.cli.TerminalDivider
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
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
    Column(
        modifier = modifier
            .fillMaxWidth()
            .background(CLIColors.Background)
            .then(
                if (onClick != null) Modifier.clickable(onClick = onClick)
                else Modifier
            )
            .padding(horizontal = 16.dp, vertical = 12.dp)
    ) {
        // 头部：用户信息
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TerminalAvatar(isOnline = true)

            Spacer(modifier = Modifier.width(8.dp))

            Text(
                text = availability.user.nickname,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextPrimary,
            )

            Spacer(modifier = Modifier.width(8.dp))

            Text(
                text = "[${formatRelativeTimeShort(availability)}]",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary,
            )

            Spacer(modifier = Modifier.weight(1f))

            if (isMine && onCancelClick != null) {
                Text(
                    text = "[X]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Red,
                    modifier = Modifier.clickable(onClick = onCancelClick)
                )
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        // 时间信息
        Row(
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "${ASCII.ARROW_RIGHT} ",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.Green,
            )
            Text(
                text = formatTimeRange(availability),
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextPrimary,
            )
        }

        Spacer(modifier = Modifier.height(4.dp))

        // 地点信息
        Row(
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "${ASCII.ARROW_RIGHT} ",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.Blue,
            )
            Text(
                text = formatLocation(availability),
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextPrimary,
            )
        }

        Spacer(modifier = Modifier.height(8.dp))

        TerminalDivider()
    }
}

private fun formatRelativeTimeShort(availability: Availability): String {
    val now = Clock.System.now()
    val diff = availability.startTime - now
    val hours = diff.inWholeHours
    val minutes = diff.inWholeMinutes

    return when {
        hours < 0 -> "NOW"
        hours == 0L -> "${minutes}m"
        hours < 24 -> "${hours}h"
        hours < 48 -> "TMR"
        else -> {
            val date = availability.startTime.toLocalDateTime(TimeZone.currentSystemDefault())
            "${date.monthNumber}/${date.dayOfMonth}"
        }
    }
}

private fun formatTimeRange(availability: Availability): String {
    val tz = TimeZone.currentSystemDefault()
    val start = availability.startTime.toLocalDateTime(tz)
    val end = availability.endTime.toLocalDateTime(tz)

    val startTimeStr = "[${start.hour.toString().padStart(2, '0')}:${start.minute.toString().padStart(2, '0')}]"
    val endTimeStr = "[${end.hour.toString().padStart(2, '0')}:${end.minute.toString().padStart(2, '0')}]"

    val dateStr = if (start.dayOfMonth != Clock.System.now().toLocalDateTime(tz).dayOfMonth) {
        "${start.monthNumber}/${start.dayOfMonth} "
    } else {
        ""
    }

    return "$dateStr$startTimeStr-$endTimeStr"
}

private fun formatLocation(availability: Availability): String {
    return when (availability.location.type) {
        LocationType.FLEXIBLE -> "地点灵活"
        LocationType.PRESET -> availability.location.name ?: "未知地点"
        LocationType.CUSTOM -> availability.location.name ?: "自定义地点"
    }
}
