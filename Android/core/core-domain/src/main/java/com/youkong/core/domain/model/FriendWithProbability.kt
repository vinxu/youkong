package com.youkong.core.domain.model

/**
 * 带有空概率的好友
 */
data class FriendWithProbability(
    val user: UserProfile,
    val probability: Int?, // 0-100, null 表示无数据
    val confidence: Confidence,
    val reason: String,
    val color: String,
    val updatedAt: Long, // 毫秒时间戳
)

/**
 * 概率置信度
 */
enum class Confidence {
    HIGH,
    MEDIUM,
    LOW,
    ;

    companion object {
        fun fromString(value: String): Confidence {
            return when (value.lowercase()) {
                "high" -> HIGH
                "medium" -> MEDIUM
                "low" -> LOW
                else -> LOW
            }
        }
    }
}

/**
 * 概率等级
 */
enum class ProbabilityLevel {
    VERY_HIGH,  // 80-100% 绿
    HIGH,       // 60-79% 浅绿
    MEDIUM,     // 40-59% 黄
    LOW,        // 20-39% 橙
    VERY_LOW,   // 0-19% 红
    NO_DATA,    // 无数据 灰
    ;

    companion object {
        fun fromProbability(probability: Int?): ProbabilityLevel {
            return when {
                probability == null || probability < 0 -> NO_DATA
                probability >= 80 -> VERY_HIGH
                probability >= 60 -> HIGH
                probability >= 40 -> MEDIUM
                probability >= 20 -> LOW
                else -> VERY_LOW
            }
        }
    }
}
