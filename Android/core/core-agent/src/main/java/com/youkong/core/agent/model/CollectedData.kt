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
    val appUsageList: List<AppUsageInfo> = emptyList(), // 今日 App 使用列表
)

/**
 * 单次使用时段
 */
data class AppUsageSession(
    val startTime: Long,      // 开始时间戳（毫秒）
    val endTime: Long,        // 结束时间戳（毫秒）
    val durationMillis: Long, // 时长（毫秒）
)

/**
 * 单个 App 使用信息
 */
data class AppUsageInfo(
    val packageName: String,
    val appName: String,
    val usageTimeMillis: Long, // 使用时长（毫秒）
    val sessions: List<AppUsageSession> = emptyList(), // 使用时段列表
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
