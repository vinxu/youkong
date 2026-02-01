package com.youkong.core.ui.util

import java.time.Duration
import java.time.Instant
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter

object TimeUtils {
    /**
     * 将 ISO 8601 时间字符串转换为相对时间描述
     * 例如: "刚刚"、"3分钟前"、"2小时前"、"昨天"、"3天前"
     */
    fun getRelativeTimeText(isoTimeString: String?): String {
        if (isoTimeString == null || isoTimeString.isEmpty() || isoTimeString == "0001-01-01T00:00:00Z") {
            return "未知"
        }

        return try {
            val timestamp = ZonedDateTime.parse(isoTimeString).toInstant()
            val now = Instant.now()
            val duration = Duration.between(timestamp, now)

            val seconds = duration.seconds
            val minutes = seconds / 60
            val hours = minutes / 60
            val days = hours / 24

            when {
                seconds < 60 -> "刚刚"
                minutes < 60 -> "${minutes}分钟前"
                hours < 24 -> "${hours}小时前"
                days == 1L -> "昨天"
                days < 7 -> "${days}天前"
                days < 30 -> "${days / 7}周前"
                days < 365 -> "${days / 30}个月前"
                else -> "${days / 365}年前"
            }
        } catch (e: Exception) {
            "未知"
        }
    }
}
