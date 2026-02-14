package com.youkong.core.ui.theme

import androidx.compose.ui.graphics.Color

/**
 * CLI Terminal 配色方案
 * 基于 GitHub Dark 终端风格
 */
object CLIColors {
    // 背景色
    val Background = Color(0xFF0D1117)          // 主背景（GitHub Dark）
    val BackgroundSecondary = Color(0xFF161B22) // 次背景（卡片/提升）
    val BackgroundHighlight = Color(0xFF1C2333) // 微亮背景（有空高亮）
    val Border = Color(0xFF30363D)              // 边框/分隔线

    // 文本色
    val TextPrimary = Color(0xFFE6EDF3)         // 主文本（亮白）
    val TextSecondary = Color(0xFF8B949E)       // 次文本（灰色）
    val TextWeak = Color(0xFF484F58)            // 弱文本（暗灰）

    // 语义色
    val Green = Color(0xFF3FB950)               // 成功/有空
    val GreenLight = Color(0xFF56D364)          // 浅绿（可能有空）
    val Yellow = Color(0xFFD29922)              // 警告/可能
    val Orange = Color(0xFFF0883E)              // 橙色（可能没空）
    val Red = Color(0xFFF85149)                 // 错误/忙碌
    val Blue = Color(0xFF58A6FF)                // 信息/链接
    val Purple = Color(0xFFBC8CFF)              // 特殊
    val Cyan = Color(0xFF39C5CF)                // 强调

    /**
     * 根据概率值返回对应的颜色
     * @param probability 概率值 (0-100)，-1 表示无数据
     * @return 对应的颜色
     */
    fun probabilityColor(probability: Int): Color {
        return when {
            probability < 0 -> TextSecondary    // 无数据（灰色）
            probability >= 80 -> Green          // 很可能有空
            probability >= 60 -> GreenLight     // 可能有空
            probability >= 40 -> Yellow         // 不确定
            probability >= 20 -> Orange         // 可能没空
            else -> Red                         // 很可能忙
        }
    }
}

/**
 * 概率文本描述
 */
object ProbabilityText {
    fun description(probability: Int): String {
        return when {
            probability < 0 -> "无数据"
            probability >= 80 -> "很可能有空"
            probability >= 60 -> "可能有空"
            probability >= 40 -> "不确定"
            probability >= 20 -> "可能没空"
            else -> "很可能忙"
        }
    }
}
