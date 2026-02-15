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
    // 像素场景
    @SerialName("scene_pose")
    val scenePose: String? = null,
    @SerialName("scene_arms")
    val sceneArms: String? = null,
    @SerialName("scene_expression")
    val sceneExpression: String? = null,
    @SerialName("scene_prop")
    val sceneProp: String? = null,
    @SerialName("scene_surface")
    val sceneSurface: String? = null,
    // Rive 动画状态
    @SerialName("rive_state")
    val riveState: String? = null,
    // AI 互动选项
    val interactions: List<InteractionOptionItem>? = null,
    // 今日互动计数
    @SerialName("interaction_count")
    val interactionCount: Int = 0,
)

/**
 * AI 互动选项
 */
@Serializable
data class InteractionOptionItem(
    val emoji: String,
    val label: String,
    @SerialName("push_text")
    val pushText: String,
)

/**
 * 发送互动请求
 */
@Serializable
data class SendInteractionRequest(
    @SerialName("receiver_id")
    val receiverId: String,
    @SerialName("action_emoji")
    val actionEmoji: String,
    @SerialName("action_label")
    val actionLabel: String,
    @SerialName("action_push_text")
    val actionPushText: String,
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
