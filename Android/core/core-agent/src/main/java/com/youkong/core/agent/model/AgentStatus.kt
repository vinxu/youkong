package com.youkong.core.agent.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * 屏幕使用数据
 */
@Serializable
data class ScreenData(
    @SerialName("is_active")
    val isActive: Boolean,
    @SerialName("activity_type")
    val activityType: ActivityType,
    @SerialName("session_duration_minutes")
    val sessionDurationMinutes: Int,
    @SerialName("last_active_minutes_ago")
    val lastActiveMinutesAgo: Int,
)

/**
 * 活动类型
 */
@Serializable
enum class ActivityType {
    @SerialName("entertainment")
    ENTERTAINMENT,
    @SerialName("productivity")
    PRODUCTIVITY,
    @SerialName("communication")
    COMMUNICATION,
    @SerialName("idle")
    IDLE,
}

/**
 * 位置数据
 */
@Serializable
data class LocationData(
    @SerialName("place_type")
    val placeType: PlaceType,
    @SerialName("at_place_since_minutes")
    val atPlaceSinceMinutes: Int,
)

/**
 * 地点类型
 */
@Serializable
enum class PlaceType {
    @SerialName("home")
    HOME,
    @SerialName("work")
    WORK,
    @SerialName("leisure")
    LEISURE,
    @SerialName("transit")
    TRANSIT,
    @SerialName("unknown")
    UNKNOWN,
}

/**
 * Agent 状态上报请求
 */
@Serializable
data class AgentStatusRequest(
    val screen: ScreenData? = null,
    val location: LocationData? = null,
)

/**
 * 好友有空概率推荐
 */
@Serializable
data class FriendRecommendation(
    @SerialName("friend_id")
    val friendId: String,
    val name: String,
    val avatar: String? = null,
    val probability: Int, // 0-100, -1 表示无数据
    val confidence: Confidence,
    val reason: String,
    val color: String,
    @SerialName("updated_at")
    val updatedAt: Long, // 毫秒时间戳
)

/**
 * 置信度
 */
@Serializable
enum class Confidence {
    @SerialName("high")
    HIGH,
    @SerialName("medium")
    MEDIUM,
    @SerialName("low")
    LOW,
}
