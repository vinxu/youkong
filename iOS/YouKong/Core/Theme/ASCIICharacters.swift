import Foundation

// MARK: - ASCII Characters Library for CLI Terminal UI

struct ASCII {
    // 状态点
    static let bullet = "●"           // 实心圆（在线/有数据）
    static let bulletEmpty = "○"      // 空心圆（离线/无数据）
    static let bulletLarge = "◉"      // 大实心圆（强调）

    // 箭头
    static let arrowRight = "→"       // 右箭头
    static let arrowLeft = "←"        // 左箭头
    static let arrowUp = "↑"          // 上箭头
    static let arrowDown = "↓"        // 下箭头
    static let arrowRightDouble = "⇒" // 双右箭头

    // 进度条字符
    static let progressFull = "▇"     // 实心进度块
    static let progressEmpty = "░"    // 空心进度块
    static let progressLight = "▁"    // 轻度进度
    static let progressMedium = "▄"   // 中度进度
    static let progressHeavy = "█"    // 重度进度

    // 分隔线字符
    static let separator = "─"        // 水平分隔线
    static let separatorHeavy = "═"   // 粗水平分隔线
    static let separatorDotted = "┄"  // 点状分隔线

    // 边框字符
    static let boxTopLeft = "┌"       // 左上角
    static let boxTopRight = "┐"      // 右上角
    static let boxBottomLeft = "└"    // 左下角
    static let boxBottomRight = "┘"   // 右下角
    static let boxVertical = "│"      // 垂直线
    static let boxHorizontal = "─"    // 水平线
    static let boxCross = "┼"         // 十字交叉
    static let boxTLeft = "├"         // T形左
    static let boxTRight = "┤"        // T形右

    // 特殊符号
    static let checkmark = "✓"        // 勾选
    static let cross = "✗"            // 叉号
    static let warning = "⚠"          // 警告
    static let info = "ℹ"             // 信息
    static let cursor = "▌"           // 光标
    static let prompt = "›"           // 提示符
    static let ellipsis = "…"         // 省略号

    // 特殊空格（用于对齐）
    static let space = " "            // 普通空格
    static let nbsp = "\u{00A0}"      // 不换行空格

    // MARK: - Helper Functions

    /// 生成水平分隔线（指定长度）
    static func horizontalLine(length: Int, style: String = separator) -> String {
        String(repeating: style, count: length)
    }

    /// 生成进度条字符串
    /// - Parameters:
    ///   - percentage: 百分比 (0-100)
    ///   - totalBlocks: 总块数（默认 10）
    /// - Returns: 进度条字符串，如 "▇▇▇▇▇▇▇░░░"
    static func progressBar(percentage: Int, totalBlocks: Int = 10) -> String {
        let percentage = max(0, min(100, percentage)) // 限制在 0-100
        let fullBlocks = (percentage * totalBlocks) / 100
        let emptyBlocks = totalBlocks - fullBlocks

        let fullPart = String(repeating: progressFull, count: fullBlocks)
        let emptyPart = String(repeating: progressEmpty, count: emptyBlocks)

        return fullPart + emptyPart
    }

    /// 生成带边框的文本块
    /// - Parameters:
    ///   - text: 文本内容（支持多行）
    ///   - width: 边框宽度（不包括边框字符）
    /// - Returns: 带边框的文本字符串
    static func boxedText(_ text: String, width: Int? = nil) -> String {
        let lines = text.components(separatedBy: "\n")
        let maxLength = width ?? (lines.map { $0.count }.max() ?? 0)

        var result = ""

        // 顶部边框
        result += boxTopLeft + String(repeating: boxHorizontal, count: maxLength + 2) + boxTopRight + "\n"

        // 内容行
        for line in lines {
            let padding = String(repeating: " ", count: maxLength - line.count)
            result += boxVertical + " " + line + padding + " " + boxVertical + "\n"
        }

        // 底部边框
        result += boxBottomLeft + String(repeating: boxHorizontal, count: maxLength + 2) + boxBottomRight

        return result
    }
}

// MARK: - ASCII Art Helpers
extension ASCII {
    /// 生成标题分隔线（带标题文本）
    /// - Parameters:
    ///   - title: 标题文本
    ///   - totalLength: 总长度
    ///   - style: 分隔线样式
    /// - Returns: 格式化的标题行，如 "─── 标题 ───────────"
    static func titleLine(_ title: String, totalLength: Int = 40, style: String = separator) -> String {
        let prefix = String(repeating: style, count: 3)
        let titleWithSpaces = " \(title) "
        let remainingLength = max(0, totalLength - prefix.count - titleWithSpaces.count)
        let suffix = String(repeating: style, count: remainingLength)

        return prefix + titleWithSpaces + suffix
    }

    /// 生成居中文本
    /// - Parameters:
    ///   - text: 文本
    ///   - totalLength: 总长度
    /// - Returns: 居中的文本字符串
    static func centeredText(_ text: String, totalLength: Int) -> String {
        let padding = max(0, (totalLength - text.count) / 2)
        let leftPadding = String(repeating: " ", count: padding)
        let rightPadding = String(repeating: " ", count: totalLength - text.count - padding)

        return leftPadding + text + rightPadding
    }
}
