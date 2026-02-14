package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * 宫格响应
 */
@Serializable
data class GridResponse(
    @SerialName("grid_size")
    val gridSize: Int,
    val friends: List<FriendGridItem>
)

/**
 * 宫格中的好友项
 */
@Serializable
data class FriendGridItem(
    @SerialName("user_id")
    val userId: String,
    val nickname: String,
    val avatar: String? = null,
    val emoji: String,
    val status: String,
    @SerialName("updated_at")
    val updatedAt: String,
    @SerialName("relative_time")
    val relativeTime: String,
    val city: String? = null,
    @SerialName("is_available")
    val isAvailable: Boolean = false,
    @SerialName("is_visiting")
    val isVisiting: Boolean = false,
    @SerialName("gif_url")
    val gifUrl: String? = null,
    @SerialName("giphy_query")
    val giphyQuery: String? = null,
    @SerialName("use_gif")
    val useGif: Boolean = false,
    @SerialName("needs_schedule")
    val needsSchedule: Boolean = false,
)

/**
 * AI 自动推测就绪详情
 */
@Serializable
data class AIReadyDetails(
    @SerialName("perm_location")
    val permLocation: Boolean = false,
    @SerialName("perm_motion")
    val permMotion: Boolean = false,
    @SerialName("perm_calendar")
    val permCalendar: Boolean = false,
    @SerialName("has_invited_friend")
    val hasInvitedFriend: Boolean = false,
    @SerialName("has_voice_schedule")
    val hasVoiceSchedule: Boolean = false,
)
