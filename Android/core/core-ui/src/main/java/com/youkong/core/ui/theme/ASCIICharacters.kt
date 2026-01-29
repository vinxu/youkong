package com.youkong.core.ui.theme

/**
 * ASCII 字符库 - 用于 CLI Terminal UI
 */
object ASCII {
    // 状态点
    const val BULLET = "●"           // 实心圆（在线/有数据）
    const val BULLET_EMPTY = "○"     // 空心圆（离线/无数据）
    const val BULLET_LARGE = "◉"     // 大实心圆（强调）

    // 箭头
    const val ARROW_RIGHT = "→"      // 右箭头
    const val ARROW_LEFT = "←"       // 左箭头
    const val ARROW_UP = "↑"         // 上箭头
    const val ARROW_DOWN = "↓"       // 下箭头
    const val ARROW_RIGHT_DOUBLE = "⇒" // 双右箭头

    // 进度条字符
    const val PROGRESS_FULL = "▇"    // 实心进度块
    const val PROGRESS_EMPTY = "░"   // 空心进度块
    const val PROGRESS_LIGHT = "▁"   // 轻度进度
    const val PROGRESS_MEDIUM = "▄"  // 中度进度
    const val PROGRESS_HEAVY = "█"   // 重度进度

    // 分隔线字符
    const val SEPARATOR = "─"        // 水平分隔线
    const val SEPARATOR_HEAVY = "═"  // 粗水平分隔线
    const val SEPARATOR_DOTTED = "┄" // 点状分隔线

    // 边框字符
    const val BOX_TOP_LEFT = "┌"     // 左上角
    const val BOX_TOP_RIGHT = "┐"    // 右上角
    const val BOX_BOTTOM_LEFT = "└"  // 左下角
    const val BOX_BOTTOM_RIGHT = "┘" // 右下角
    const val BOX_VERTICAL = "│"     // 垂直线
    const val BOX_HORIZONTAL = "─"   // 水平线
    const val BOX_CROSS = "┼"        // 十字交叉
    const val BOX_T_LEFT = "├"       // T形左
    const val BOX_T_RIGHT = "┤"      // T形右

    // 特殊符号
    const val CHECKMARK = "✓"        // 勾选
    const val CROSS = "✗"            // 叉号
    const val WARNING = "⚠"          // 警告
    const val INFO = "ℹ"             // 信息
    const val CURSOR = "▌"           // 光标
    const val PROMPT = "›"           // 提示符
    const val ELLIPSIS = "…"         // 省略号

    // MARK: - Helper Functions

    /**
     * 生成水平分隔线
     * @param length 长度
     * @param style 分隔线样式（默认为普通分隔线）
     * @return 分隔线字符串
     */
    fun horizontalLine(length: Int, style: String = SEPARATOR): String {
        return style.repeat(length)
    }

    /**
     * 生成进度条字符串
     * @param percentage 百分比 (0-100)
     * @param totalBlocks 总块数（默认 10）
     * @return 进度条字符串，如 "▇▇▇▇▇▇▇░░░"
     */
    fun progressBar(percentage: Int, totalBlocks: Int = 10): String {
        val clampedPercentage = percentage.coerceIn(0, 100)
        val fullBlocks = (clampedPercentage * totalBlocks) / 100
        val emptyBlocks = totalBlocks - fullBlocks

        return PROGRESS_FULL.repeat(fullBlocks) + PROGRESS_EMPTY.repeat(emptyBlocks)
    }

    /**
     * 生成带边框的文本块
     * @param text 文本内容（支持多行）
     * @param width 边框宽度（不包括边框字符）
     * @return 带边框的文本字符串
     */
    fun boxedText(text: String, width: Int? = null): String {
        val lines = text.split("\n")
        val maxLength = width ?: (lines.maxOfOrNull { it.length } ?: 0)

        val result = StringBuilder()

        // 顶部边框
        result.append(BOX_TOP_LEFT)
        result.append(BOX_HORIZONTAL.repeat(maxLength + 2))
        result.append(BOX_TOP_RIGHT)
        result.append("\n")

        // 内容行
        for (line in lines) {
            val padding = " ".repeat(maxLength - line.length)
            result.append("$BOX_VERTICAL $line$padding $BOX_VERTICAL\n")
        }

        // 底部边框
        result.append(BOX_BOTTOM_LEFT)
        result.append(BOX_HORIZONTAL.repeat(maxLength + 2))
        result.append(BOX_BOTTOM_RIGHT)

        return result.toString()
    }

    /**
     * 生成标题分隔线（带标题文本）
     * @param title 标题文本
     * @param totalLength 总长度
     * @param style 分隔线样式
     * @return 格式化的标题行，如 "─── 标题 ───────────"
     */
    fun titleLine(title: String, totalLength: Int = 40, style: String = SEPARATOR): String {
        val prefix = style.repeat(3)
        val titleWithSpaces = " $title "
        val remainingLength = (totalLength - prefix.length - titleWithSpaces.length).coerceAtLeast(0)
        val suffix = style.repeat(remainingLength)

        return "$prefix$titleWithSpaces$suffix"
    }

    /**
     * 生成居中文本
     * @param text 文本
     * @param totalLength 总长度
     * @return 居中的文本字符串
     */
    fun centeredText(text: String, totalLength: Int): String {
        val padding = ((totalLength - text.length) / 2).coerceAtLeast(0)
        val leftPadding = " ".repeat(padding)
        val rightPadding = " ".repeat(totalLength - text.length - padding)

        return "$leftPadding$text$rightPadding"
    }
}
