package com.youkong.core.domain.model

import kotlinx.datetime.Instant

data class Circle(
    val id: String,
    val name: String,
    val emoji: String,
    val color: String,
    val ownerId: String,
    val createdAt: Instant,
)

data class CircleDetail(
    val id: String,
    val name: String,
    val emoji: String,
    val color: String,
    val ownerId: String,
    val memberCount: Int,
    val members: List<UserProfile>? = null,
    val createdAt: Instant,
)

// 预设圈子颜色
object CircleColorPresets {
    const val PINK = "#EC4899"
    const val ORANGE = "#F97316"
    const val BLUE = "#3B82F6"
    const val GREEN = "#22C55E"
    const val PURPLE = "#8B5CF6"
    const val YELLOW = "#EAB308"

    val all = listOf(PINK, ORANGE, BLUE, GREEN, PURPLE, YELLOW)
}
