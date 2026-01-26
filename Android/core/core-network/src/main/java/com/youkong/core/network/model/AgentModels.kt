package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * 屏幕使用数据
 */
@Serializable
data class ScreenDataRequest(
    @SerialName("is_active")
    val isActive: Boolean,
    @SerialName("activity_type")
    val activityType: String,
    @SerialName("session_duration_minutes")
    val sessionDurationMinutes: Int,
    @SerialName("last_active_minutes_ago")
    val lastActiveMinutesAgo: Int,
)

/**
 * 位置数据
 */
@Serializable
data class LocationDataRequest(
    @SerialName("place_type")
    val placeType: String,
    @SerialName("at_place_since_minutes")
    val atPlaceSinceMinutes: Int,
)

/**
 * 电池数据
 */
@Serializable
data class BatteryDataRequest(
    @SerialName("battery_level")
    val batteryLevel: Int,
    @SerialName("battery_state")
    val batteryState: String,
    @SerialName("is_charging")
    val isCharging: Boolean,
)

/**
 * 模式数据
 */
@Serializable
data class ModeDataRequest(
    @SerialName("is_low_power_mode")
    val isLowPowerMode: Boolean,
    @SerialName("is_focus_mode_on")
    val isFocusModeOn: Boolean,
)

/**
 * 连接数据
 */
@Serializable
data class ConnectionDataRequest(
    @SerialName("is_headphones_connected")
    val isHeadphonesConnected: Boolean,
    @SerialName("network_type")
    val networkType: String,
)

/**
 * 显示数据
 */
@Serializable
data class DisplayDataRequest(
    @SerialName("screen_brightness")
    val screenBrightness: Float,
)

/**
 * Agent 状态上报请求（扩展版，匹配后端 ExtendedStatusReportRequest）
 */
@Serializable
data class AgentStatusRequest(
    val screen: ScreenDataRequest? = null,
    val location: LocationDataRequest? = null,
    val battery: BatteryDataRequest? = null,
    val mode: ModeDataRequest? = null,
    val connection: ConnectionDataRequest? = null,
    val display: DisplayDataRequest? = null,
)

// ========== 响应数据结构 ==========

/**
 * 生活状态
 */
@Serializable
data class LifeStatus(
    val emoji: String,
    val label: String,
    val description: String? = null,
)

/**
 * 有空分析结果
 */
@Serializable
data class AvailabilityAnalysis(
    val status: String,
    val probability: Int,
    val reason: String,
    val confidence: String,
)

/**
 * LLM 分析结果
 */
@Serializable
data class AnalysisResult(
    val availability: AvailabilityAnalysis,
    @SerialName("life_status")
    val lifeStatus: LifeStatus,
)

/**
 * 状态上报响应
 */
@Serializable
data class StatusReportResponse(
    val success: Boolean,
    @SerialName("next_report_in")
    val nextReportIn: Int,
    val analysis: AnalysisResult? = null,
)

/**
 * 好友有空概率响应
 */
@Serializable
data class FriendProbabilityResponse(
    @SerialName("friend_id")
    val friendId: String,
    val name: String,
    val avatar: String? = null,
    val probability: Int, // 0-100, -1 表示无数据
    val confidence: String, // "high", "medium", "low"
    val reason: String,
    val color: String,
    @SerialName("updated_at")
    val updatedAt: Long, // 毫秒时间戳
)

/**
 * 有空概率列表响应
 */
@Serializable
data class FreeProbabilityResponse(
    val friends: List<FriendProbabilityResponse>,
    @SerialName("generated_at")
    val generatedAt: Long, // 毫秒时间戳
)
