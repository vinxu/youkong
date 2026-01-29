import SwiftUI

/// CLI 风格分隔线组件
struct CLISeparatorView: View {
    let style: Style

    enum Style {
        case full    // 全宽分隔线
        case short   // 短分隔线
    }

    init(style: Style = .full) {
        self.style = style
    }

    var body: some View {
        Text(String(repeating: CLIConstants.separator, count: style == .full ? 42 : 20))
            .font(.system(size: 12, design: .monospaced))
            .foregroundColor(.gray.opacity(0.5))
            .frame(maxWidth: .infinity)
    }
}

#Preview {
    VStack(spacing: 16) {
        CLISeparatorView(style: .full)
        CLISeparatorView(style: .short)
    }
    .padding()
}
