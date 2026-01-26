import SwiftUI

#if DEBUG
/// 网络请求状态悬浮条
struct NetworkOverlayView: View {
    @ObservedObject var debugTool = DebugTool.shared

    var latestLog: NetworkLog? {
        debugTool.networkLogs.first
    }

    var pendingCount: Int {
        debugTool.networkLogs.filter { $0.status == .pending }.count
    }

    var body: some View {
        if debugTool.showNetworkOverlay, let log = latestLog {
            VStack(spacing: 0) {
                HStack(spacing: 8) {
                    // Status indicator
                    Circle()
                        .fill(log.status.color)
                        .frame(width: 8, height: 8)

                    // Method
                    Text(log.method)
                        .font(.system(.caption2, design: .monospaced))
                        .fontWeight(.bold)

                    // Path
                    Text(urlPath(log.url))
                        .font(.system(.caption2, design: .monospaced))
                        .lineLimit(1)

                    Spacer()

                    // Status code
                    if let statusCode = log.statusCode {
                        Text("\(statusCode)")
                            .font(.system(.caption2, design: .monospaced))
                            .foregroundColor(statusCode < 400 ? .green : .red)
                    }

                    // Duration
                    Text(log.durationString)
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundColor(.secondary)

                    // Pending count
                    if pendingCount > 0 {
                        Text("(\(pendingCount))")
                            .font(.system(.caption2, design: .monospaced))
                            .foregroundColor(.orange)
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
                .background(.ultraThinMaterial)
            }
            .transition(.move(edge: .top).combined(with: .opacity))
        }
    }

    private func urlPath(_ url: String) -> String {
        guard let urlObj = URL(string: url) else { return url }
        let path = urlObj.path
        if path.count > 30 {
            return "..." + String(path.suffix(27))
        }
        return path
    }
}

/// 添加网络状态覆盖层的修饰器
struct NetworkOverlayModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .safeAreaInset(edge: .top) {
                NetworkOverlayView()
            }
    }
}

extension View {
    func withNetworkOverlay() -> some View {
        modifier(NetworkOverlayModifier())
    }
}
#endif
