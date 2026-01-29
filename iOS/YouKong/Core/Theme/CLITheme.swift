import SwiftUI

// MARK: - CLI Terminal Color Scheme
struct CLIColors {
    // 背景色
    static let background = Color(hex: "#0D1117")          // 主背景（GitHub Dark）
    static let backgroundSecondary = Color(hex: "#161B22") // 次背景（卡片/提升）
    static let border = Color(hex: "#30363D")              // 边框/分隔线

    // 文本色
    static let textPrimary = Color(hex: "#E6EDF3")         // 主文本（亮白）
    static let textSecondary = Color(hex: "#8B949E")       // 次文本（灰色）
    static let textWeak = Color(hex: "#484F58")            // 弱文本（暗灰）

    // 语义色
    static let green = Color(hex: "#3FB950")               // 成功/有空
    static let greenLight = Color(hex: "#56D364")          // 浅绿（可能有空）
    static let yellow = Color(hex: "#D29922")              // 警告/可能
    static let orange = Color(hex: "#F0883E")              // 橙色（可能没空）
    static let red = Color(hex: "#F85149")                 // 错误/忙碌
    static let blue = Color(hex: "#58A6FF")                // 信息/链接
    static let purple = Color(hex: "#BC8CFF")              // 特殊
    static let cyan = Color(hex: "#39C5CF")                // 强调

    // 概率颜色映射（0-100%）
    static func probabilityColor(for probability: Int) -> Color {
        switch probability {
        case 80...100:
            return green           // 很可能有空
        case 60..<80:
            return greenLight      // 可能有空
        case 40..<60:
            return yellow          // 不确定
        case 20..<40:
            return orange          // 可能没空
        case 0..<20:
            return red             // 很可能忙
        default:
            return textSecondary   // 无数据（灰色）
        }
    }
}
