package com.youkong.core.agent.model

import kotlinx.datetime.Instant

/**
 * 移动数据
 */
data class MovementData(
    val isMoving: Boolean,
    val movementType: MovementType,
    val stepsToday: Int? = null,
    val stepsLastHour: Int? = null,
    val stationaryMinutes: Int? = null,
    val timestamp: Instant,
)

/**
 * 移动类型
 */
enum class MovementType {
    STATIONARY,
    WALKING,
    RUNNING,
    CYCLING,
    DRIVING,
    UNKNOWN;

    fun toApiString(): String = name.lowercase()
}
