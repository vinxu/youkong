import SwiftUI

// MARK: - Probability Colors

/// 有空概率颜色系统
enum ProbabilityColors {
    // 主色（用于圆点、进度条等）
    static let veryLikely = Color(hex: "#22C55E")   // 80-100% 深绿
    static let likely = Color(hex: "#86EFAC")       // 60-79%  浅绿
    static let maybe = Color(hex: "#FACC15")        // 40-59%  黄色
    static let unlikely = Color(hex: "#FB923C")     // 20-39%  橙色
    static let busy = Color(hex: "#EF4444")         // 0-19%   红色
    static let unknown = Color(hex: "#9CA3AF")      // 无数据  灰色

    // MARK: - Get Color

    /// 根据概率获取颜色
    /// - Parameter probability: 0-100 的概率值，-1 表示无数据
    /// - Returns: 对应的颜色
    static func color(for probability: Int?) -> Color {
        guard let p = probability, p >= 0 else {
            return unknown
        }

        switch p {
        case 80...100:
            return veryLikely
        case 60..<80:
            return likely
        case 40..<60:
            return maybe
        case 20..<40:
            return unlikely
        case 0..<20:
            return busy
        default:
            return unknown
        }
    }

    /// 根据概率获取背景色（更淡的版本）
    static func backgroundColor(for probability: Int?) -> Color {
        color(for: probability).opacity(0.1)
    }

    /// 根据颜色代码字符串获取颜色
    static func color(from hex: String) -> Color {
        Color(hex: hex)
    }

    // MARK: - Display Text

    /// 获取概率的文字描述
    static func description(for probability: Int?) -> String {
        guard let p = probability, p >= 0 else {
            return "无数据"
        }

        switch p {
        case 80...100:
            return "很可能有空"
        case 60..<80:
            return "可能有空"
        case 40..<60:
            return "不确定"
        case 20..<40:
            return "可能没空"
        case 0..<20:
            return "很可能忙"
        default:
            return "无数据"
        }
    }
}
