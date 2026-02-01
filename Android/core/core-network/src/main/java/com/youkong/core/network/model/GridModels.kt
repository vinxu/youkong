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
    val relativeTime: String
)
