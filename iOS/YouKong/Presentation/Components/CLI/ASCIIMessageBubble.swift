import SwiftUI

/// ASCII Art 消息气泡组件
struct ASCIIMessageBubble: View {
    let message: String
    let isSent: Bool  // true = 自己发送，false = 对方发送

    var body: some View {
        VStack(alignment: isSent ? .trailing : .leading, spacing: 2) {
            // 气泡顶部
            Text(bubbleTop)
                .font(.system(size: 10, design: .monospaced))
                .foregroundColor(bubbleColor)

            // 气泡内容
            HStack(spacing: 0) {
                Text(CLIConstants.bubbleSlash)
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundColor(bubbleColor)

                Text(" \(message) ")
                    .font(.system(size: 14, design: .monospaced))
                    .foregroundColor(.primary)
                    .lineLimit(nil)
                    .fixedSize(horizontal: false, vertical: true)

                Text(CLIConstants.bubbleBackslash)
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundColor(bubbleColor)
            }

            // 气泡底部
            Text(bubbleBottom)
                .font(.system(size: 10, design: .monospaced))
                .foregroundColor(bubbleColor)
        }
        .frame(maxWidth: .infinity, alignment: isSent ? .trailing : .leading)
        .padding(.horizontal, isSent ? 60 : 16)
    }

    private var bubbleColor: Color {
        isSent ? .green.opacity(0.7) : .gray.opacity(0.7)
    }

    private var bubbleTop: String {
        let width = max(message.count, 10)
        let line = String(repeating: "_", count: width + 2)
        return " " + line
    }

    private var bubbleBottom: String {
        let width = max(message.count, 10)
        let line = String(repeating: "_", count: width + 2)
        return CLIConstants.bubbleBackslash + line + CLIConstants.bubbleSlash
    }
}

#Preview {
    VStack(spacing: 20) {
        ASCIIMessageBubble(message: "嗨，晚上有空吗？", isSent: false)
        ASCIIMessageBubble(message: "晚上8点可以吗？", isSent: true)
        ASCIIMessageBubble(message: "好的", isSent: false)
    }
    .padding()
}
