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
 * Agent 状态上报请求
 */
@Serializable
data class AgentStatusRequest(
    val screen: ScreenDataRequest? = null,
    val location: LocationDataRequest? = null,
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
