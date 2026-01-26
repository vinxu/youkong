package com.youkong.core.ui.theme

import androidx.compose.ui.graphics.Color

// Primary Colors
val Primary = Color(0xFF10B981)      // Emerald-500
val PrimaryDark = Color(0xFF059669)  // Emerald-600
val PrimaryLight = Color(0xFF34D399) // Emerald-400

val Secondary = Color(0xFF14B8A6)    // Teal-500
val SecondaryDark = Color(0xFF0D9488) // Teal-600
val SecondaryLight = Color(0xFF2DD4BF) // Teal-400

// Circle Colors - 圈子颜色
object CircleColors {
    val Pink = Color(0xFFEC4899)
    val Orange = Color(0xFFF97316)
    val Blue = Color(0xFF3B82F6)
    val Green = Color(0xFF22C55E)
    val Purple = Color(0xFF8B5CF6)
    val Yellow = Color(0xFFEAB308)

    val all = listOf(Pink, Orange, Blue, Green, Purple, Yellow)

    fun fromHex(hex: String): Color {
        return try {
            Color(android.graphics.Color.parseColor(hex))
        } catch (e: Exception) {
            Pink
        }
    }
}

// Probability Colors - 有空概率颜色
object ProbabilityColors {
    val VeryHigh = Color(0xFF22C55E)    // 绿色 80-100%
    val High = Color(0xFF86EFAC)        // 浅绿 60-79%
    val Medium = Color(0xFFFACC15)      // 黄色 40-59%
    val Low = Color(0xFFFB923C)         // 橙色 20-39%
    val VeryLow = Color(0xFFEF4444)     // 红色 0-19%
    val NoData = Color(0xFF9CA3AF)      // 灰色 无数据

    /**
     * 根据概率值获取对应颜色
     * @param probability 概率值 (0-100)，-1 或 null 表示无数据
     */
    fun fromProbability(probability: Int?): Color = when {
        probability == null || probability < 0 -> NoData
        probability >= 80 -> VeryHigh
        probability >= 60 -> High
        probability >= 40 -> Medium
        probability >= 20 -> Low
        else -> VeryLow
    }

    /**
     * 根据十六进制颜色字符串获取颜色
     */
    fun fromHex(hex: String): Color {
        return try {
            Color(android.graphics.Color.parseColor(hex))
        } catch (e: Exception) {
            NoData
        }
    }
}

// Neutral Colors
val White = Color(0xFFFFFFFF)
val Black = Color(0xFF000000)

val Gray50 = Color(0xFFF9FAFB)
val Gray100 = Color(0xFFF3F4F6)
val Gray200 = Color(0xFFE5E7EB)
val Gray300 = Color(0xFFD1D5DB)
val Gray400 = Color(0xFF9CA3AF)
val Gray500 = Color(0xFF6B7280)
val Gray600 = Color(0xFF4B5563)
val Gray700 = Color(0xFF374151)
val Gray800 = Color(0xFF1F2937)
val Gray900 = Color(0xFF111827)

// Semantic Colors
val Success = Color(0xFF22C55E)
val Warning = Color(0xFFF59E0B)
val Error = Color(0xFFEF4444)
val Info = Color(0xFF3B82F6)

// Background Colors
val Background = Color(0xFFF9FAFB)
val Surface = Color(0xFFFFFFFF)
val SurfaceVariant = Color(0xFFF3F4F6)

// Text Colors
val TextPrimary = Color(0xFF111827)
val TextSecondary = Color(0xFF6B7280)
val TextTertiary = Color(0xFF9CA3AF)
val TextOnPrimary = Color(0xFFFFFFFF)

// Border Colors
val Border = Color(0xFFE5E7EB)
val BorderLight = Color(0xFFF3F4F6)
