import SwiftUI

// MARK: - Friend Probability Card

struct FriendProbabilityCard: View {
    let friend: FriendRecommendation

    private var probabilityColor: Color {
        ProbabilityColors.color(for: friend.probability)
    }

    var body: some View {
        HStack(spacing: UIConstants.Spacing.md) {
            // 概率颜色指示点
            Circle()
                .fill(probabilityColor)
                .frame(width: 12, height: 12)

            // 头像
            AvatarView(url: friend.avatar, size: 48)

            // 名字和原因
            VStack(alignment: .leading, spacing: UIConstants.Spacing.xs) {
                Text(friend.name)
                    .font(.headline)
                    .foregroundColor(.primary)

                Text(friend.reason)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .lineLimit(1)
            }

            Spacer()

            // 概率百分比
            VStack(alignment: .trailing, spacing: 2) {
                Text(friend.probabilityText)
                    .font(.title3)
                    .fontWeight(.semibold)
                    .foregroundColor(probabilityColor)

                if friend.hasData {
                    Text(ProbabilityColors.description(for: friend.probability))
                        .font(.caption2)
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
            probability: 85,
            confidence: .high,
            reason: "刷了40分钟手机，在家",
            color: "#22C55E",
            updatedAt: Int(Date().timeIntervalSince1970 * 1000)
        ))

        FriendProbabilityCard(friend: FriendRecommendation(
            friendId: "2",
            name: "李四",
            avatar: nil,
            probability: 45,
            confidence: .medium,
            reason: "在公司，快下班了",
            color: "#FACC15",
            updatedAt: Int(Date().timeIntervalSince1970 * 1000)
        ))

        FriendProbabilityCard(friend: FriendRecommendation(
            friendId: "3",
            name: "王五",
            avatar: nil,
            probability: -1,
            confidence: .low,
            reason: "数据不足",
            color: "#9CA3AF",
            updatedAt: Int(Date().timeIntervalSince1970 * 1000)
        ))
    }
    .padding()
    .background(Color(.systemGroupedBackground))
}
