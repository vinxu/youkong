import SwiftUI

/// CLI 风格常量定义
struct CLIConstants {
    // MARK: - Unicode 字符
    static let separator = "━"
    static let bullet = "●"
    static let hollowBullet = "○"
    static let dash = "-"
    static let arrow = "▸"
    static let rightArrow = "→"

    // MARK: - Box Drawing 字符
    static let boxTopLeft = "┌"
    static let boxTopRight = "┐"
    static let boxBottomLeft = "└"
    static let boxBottomRight = "┘"
    static let boxHorizontal = "─"
    static let boxVertical = "│"

    // MARK: - 状态符号（纯字符）
    /// 根据概率返回对应的状态符号
    static func statusSymbol(for probability: Int) -> String {
        if probability == -1 { return "-" }
        switch probability {
        case 80...100: return "●"      // 实心圆 = 很有空
        case 60..<80: return "◉"       // 双圈实心 = 有空
        case 40..<60: return "◐"       // 半实心 = 可能
        case 20..<40: return "◎"       // 双圈空心 = 少空
        case 0..<20: return "○"        // 空心圆 = 忙碌
        default: return "-"            // 横线 = 未知
        }
    }

    // MARK: - 颜色映射
    /// 根据概率返回对应的颜色
    static func color(for probability: Int) -> Color {
        if probability == -1 { return .gray }
        switch probability {
        case 80...100: return .green
        case 60..<80: return Color(hex: "#86EFAC")  // 浅绿
        case 40..<60: return .yellow
        case 20..<40: return .orange
        case 0..<20: return .red
        default: return .gray
        }
    }

    // MARK: - ASCII Art 气泡字符
    static let bubbleTop = "__________________"
    static let bubbleSlash = "/"
    static let bubbleBackslash = "\\"
    static let bubbleUnderscore = "_"
    static let bubbleVertical = "|"
}
