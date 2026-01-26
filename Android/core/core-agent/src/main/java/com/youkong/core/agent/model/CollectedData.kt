package com.youkong.core.agent.model

import kotlinx.datetime.Instant

/**
 * 收集的本地数据
 */
data class CollectedData(
    val screenData: LocalScreenData? = null,
    val locationData: LocalLocationData? = null,
    val collectedAt: Instant,
)

/**
 * 本地屏幕使用数据
 */
data class LocalScreenData(
    val isScreenOn: Boolean,
    val lastActiveTime: Instant?,
    val currentApp: String?,
    val sessionStartTime: Instant?,
    val totalScreenTimeToday: Long, // 毫秒
)

/**
 * 本地位置数据
 */
data class LocalLocationData(
    val latitude: Double,
    val longitude: Double,
    val accuracy: Float,
    val timestamp: Instant,
)
