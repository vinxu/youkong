package com.youkong.core.agent.model

import kotlinx.datetime.Instant

/**
 * 日历数据
 */
data class CalendarData(
    val hasCurrentEvent: Boolean,
    val currentEventTitle: String? = null,
    val eventEndMinutes: Int? = null,
    val nextEventInMinutes: Int? = null,
    val todayRemainingCount: Int,
    val timestamp: Instant,
)
