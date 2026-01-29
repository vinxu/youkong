import SwiftUI

/// CLI 风格标题组件
struct CLIHeaderView: View {
    let title: String
    let subtitle: String?

    init(title: String, subtitle: String? = nil) {
        self.title = title
        self.subtitle = subtitle
    }

    var body: some View {
        VStack(spacing: 4) {
            CLISeparatorView()

            Text(title)
                .font(.system(size: 16, design: .monospaced))
                .fontWeight(.semibold)

            if let subtitle = subtitle {
                Text(subtitle)
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundColor(.gray)
            }

            CLISeparatorView()
        }
        .padding(.vertical, 8)
    }
}

#Preview {
    VStack(spacing: 20) {
        CLIHeaderView(title: "有空的朋友")
        CLIHeaderView(title: "有空的朋友", subtitle: "6 friends online")
    }
}
