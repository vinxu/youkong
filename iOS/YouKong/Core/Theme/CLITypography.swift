import SwiftUI

// MARK: - CLI Terminal Typography
// 使用系统等宽字体 SF Mono

extension Font {
    // 标题字体
    static let cliTitle = Font.system(size: 18, weight: .bold, design: .monospaced)
    static let cliHeadline = Font.system(size: 16, weight: .regular, design: .monospaced)

    // 正文字体
    static let cliBody = Font.system(size: 14, design: .monospaced)
    static let cliBodySmall = Font.system(size: 13, design: .monospaced)

    // 小字体
    static let cliCaption = Font.system(size: 12, design: .monospaced)
    static let cliCaptionSmall = Font.system(size: 11, design: .monospaced)

    // 特殊用途
    static let cliCode = Font.system(size: 14, weight: .regular, design: .monospaced)
    static let cliButton = Font.system(size: 14, weight: .medium, design: .monospaced)
}

// MARK: - Text Style Modifiers
extension Text {
    /// 应用 CLI 主文本样式
    func cliPrimaryText() -> some View {
        self
            .font(.cliBody)
            .foregroundColor(CLIColors.textPrimary)
    }

    /// 应用 CLI 次文本样式
    func cliSecondaryText() -> some View {
        self
            .font(.cliBodySmall)
            .foregroundColor(CLIColors.textSecondary)
    }

    /// 应用 CLI 弱文本样式
    func cliWeakText() -> some View {
        self
            .font(.cliCaption)
            .foregroundColor(CLIColors.textWeak)
    }

    /// 应用 CLI 标题样式
    func cliHeadlineText() -> some View {
        self
            .font(.cliHeadline)
            .foregroundColor(CLIColors.textPrimary)
    }

    /// 应用 CLI 成功样式（绿色）
    func cliSuccessText() -> some View {
        self
            .font(.cliBody)
            .foregroundColor(CLIColors.green)
    }

    /// 应用 CLI 错误样式（红色）
    func cliErrorText() -> some View {
        self
            .font(.cliBody)
            .foregroundColor(CLIColors.red)
    }

    /// 应用 CLI 警告样式（黄色）
    func cliWarningText() -> some View {
        self
            .font(.cliBody)
            .foregroundColor(CLIColors.yellow)
    }
}
