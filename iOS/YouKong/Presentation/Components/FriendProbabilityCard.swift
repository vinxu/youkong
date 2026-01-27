import SwiftUI

// MARK: - Friend Probability Card

struct FriendProbabilityCard: View {
    let friend: FriendRecommendation

    private var probabilityColor: Color {
        ProbabilityColors.color(for: friend.probability)
    }

    var body: some View {
        HStack(spacing: UIConstants.Spacing.md) {
            // 头像
            AvatarView(url: friend.avatar, size: 52)

            // 名字和正在做什么
            VStack(alignment: .leading, spacing: UIConstants.Spacing.xs) {
                Text(friend.name)
                    .font(.headline)
                    .foregroundColor(.primary)

                // Emoji + 活动描述
                HStack(spacing: 4) {
                    Text(friend.displayEmoji)
                        .font(.subheadline)
                    Text(friend.displayActivity)
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .lineLimit(1)
                }
            }

            Spacer()

            // 有空程度和百分比
            VStack(alignment: .trailing, spacing: 4) {
                // 状态标签（如 "可能有空"）- 使用 reason 字段
                Text(friend.reason)
                    .font(.subheadline)
                    .fontWeight(.medium)
                    .foregroundColor(probabilityColor)

                // 百分比数值
                if friend.hasData {
                    Text(friend.probabilityText)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }

            // 箭头
            Image(systemName: "chevron.right")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(UIConstants.Spacing.md)
        .background(Color(.systemBackground))
        .cornerRadius(UIConstants.CornerRadius.md)
        .shadow(color: .black.opacity(0.05), radius: 5, y: 2)
    }
}

// MARK: - Avatar View

struct AvatarView: View {
    let url: String?
    let size: CGFloat

    var body: some View {
        Group {
            if let urlString = url, let url = URL(string: urlString) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                    case .failure:
                        placeholderView
                    case .empty:
                        ProgressView()
                    @unknown default:
                        placeholderView
                    }
                }
            } else {
                placeholderView
            }
        }
        .frame(width: size, height: size)
        .clipShape(Circle())
    }

    private var placeholderView: some View {
        ZStack {
            Circle()
                .fill(Color.gray.opacity(0.2))

            Image(systemName: "person.fill")
                .font(.system(size: size * 0.5))
                .foregroundColor(.gray)
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
