import SwiftUI

/// CLI 风格底部状态栏组件
struct CLIFooterView: View {
    let lastUpdate: Date?
    let totalCount: Int

    var body: some View {
        VStack(spacing: 0) {
            CLISeparatorView()

            HStack(spacing: 8) {
                Text("\(totalCount) friends")
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundColor(.gray)

                if let lastUpdate = lastUpdate {
                    Text("·")
                        .foregroundColor(.gray.opacity(0.5))

                    Text("最后更新 \(formatTime(lastUpdate))")
                        .font(.system(size: 12, design: .monospaced))
                        .foregroundColor(.gray)
                }
            }
            .padding(.vertical, 8)
        }
        .background(Color(.systemGroupedBackground))
    }

    private func formatTime(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: date)
    }
}

#Preview {
    VStack {
        Spacer()
        CLIFooterView(lastUpdate: Date(), totalCount: 6)
    }
}
