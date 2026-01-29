import SwiftUI

// MARK: - Friend Probability Card (CLI Style)

struct FriendProbabilityCard: View {
    let friend: FriendRecommendation

    private var probabilityColor: Color {
        CLIConstants.color(for: friend.probability)
    }

    var body: some View {
        HStack(spacing: 0) {
            // 状态指示符（纯字符符号）
            Text(CLIConstants.statusSymbol(for: friend.probability))
                .font(.system(size: 18, design: .monospaced))
                .foregroundColor(probabilityColor)
                .frame(width: 30)

            // 朋友名字
            Text(friend.name)
                .font(.system(size: 16, design: .monospaced))
                .fontWeight(.medium)
                .foregroundColor(.primary)
                .lineLimit(1)

            Spacer()

            // 概率百分比
            Text(probabilityText)
                .font(.system(size: 14, design: .monospaced))
                .foregroundColor(probabilityColor)
                .frame(width: 50, alignment: .trailing)

            // 状态文字
            Text(statusText)
                .font(.system(size: 14, design: .monospaced))
                .foregroundColor(probabilityColor)
                .frame(width: 60, alignment: .trailing)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(Color(.systemBackground))
    }

    private var probabilityText: String {
        friend.probability == -1 ? "--" : "\(friend.probability)%"
    }

    private var statusText: String {
        if friend.probability == -1 { return "未知" }
        switch friend.probability {
        case 80...100: return "有空"
        case 60..<80: return "有空"
        case 40..<60: return "可能"
        case 20..<40: return "少空"
        case 0..<20: return "忙碌"
        default: return "未知"
        }
    }
}


#Preview {
    VStack(spacing: 12) {
        FriendProbabilityCard(friend: FriendRecommendation(
            friendId: "1",
            name: "张三",
            avatar: nil,
            probability: 95,
            confidence: .high,
            reason: "很可能有空",
            color: "#22C55E",
            emoji: "🎮",
            activity: "在玩游戏",
            updatedAt: Int(Date().timeIntervalSince1970 * 1000)
        ))

        FriendProbabilityCard(friend: FriendRecommendation(
            friendId: "2",
            name: "李四",
            avatar: nil,
            probability: 65,
            confidence: .medium,
            reason: "可能有空",
            color: "#86EFAC",
            emoji: "📺",
            activity: "在刷视频",
            updatedAt: Int(Date().timeIntervalSince1970 * 1000)
        ))

        FriendProbabilityCard(friend: FriendRecommendation(
            friendId: "3",
            name: "王五",
            avatar: nil,
            probability: 40,
            confidence: .medium,
            reason: "不太确定",
            color: "#FACC15",
            emoji: "💼",
            activity: "在公司加班",
            updatedAt: Int(Date().timeIntervalSince1970 * 1000)
        ))

        FriendProbabilityCard(friend: FriendRecommendation(
            friendId: "4",
            name: "赵六",
            avatar: nil,
            probability: 15,
            confidence: .high,
            reason: "可能在忙",
            color: "#EF4444",
            emoji: "🚗",
            activity: "在路上",
            updatedAt: Int(Date().timeIntervalSince1970 * 1000)
        ))

        FriendProbabilityCard(friend: FriendRecommendation(
            friendId: "5",
            name: "钱七",
            avatar: nil,
            probability: -1,
            confidence: .low,
            reason: "数据不足",
            color: "#9CA3AF",
            emoji: nil,
            activity: nil,
            updatedAt: Int(Date().timeIntervalSince1970 * 1000)
        ))
    }
    .padding()
    .background(Color(.systemGroupedBackground))
}
