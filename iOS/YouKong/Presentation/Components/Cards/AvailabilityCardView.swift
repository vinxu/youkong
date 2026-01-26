import SwiftUI

struct AvailabilityCardView: View {
    let availability: Availability
    @State private var showChat = false

    var body: some View {
        VStack(alignment: .leading, spacing: UIConstants.Spacing.md) {
            HStack(spacing: UIConstants.Spacing.md) {
                AvatarView(url: availability.user.avatar, size: 44)

                VStack(alignment: .leading, spacing: 2) {
                    Text(availability.user.nickname)
                        .font(.headline)

                    Text("有空")
                        .font(.subheadline)
                        .foregroundColor(.primaryGreen)
                }

                Spacer()

                Text(availability.createdAt.relativeDescription)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            VStack(alignment: .leading, spacing: UIConstants.Spacing.sm) {
                HStack(spacing: UIConstants.Spacing.sm) {
                    Image(systemName: "clock")
                        .foregroundColor(.primaryGreen)
                    Text(availability.timeRangeText)
                        .font(.subheadline)
                }

                HStack(spacing: UIConstants.Spacing.sm) {
                    Image(systemName: "mappin.circle")
                        .foregroundColor(.primaryGreen)
                    Text(availability.locationText)
                        .font(.subheadline)
                }
            }

            Button {
                showChat = true
            } label: {
                Text("发消息")
                    .font(.subheadline)
                    .fontWeight(.medium)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, UIConstants.Spacing.sm)
                    .background(Color.primaryGreen.opacity(0.1))
                    .foregroundColor(.primaryGreen)
                    .cornerRadius(UIConstants.CornerRadius.sm)
            }
        }
        .padding(UIConstants.Spacing.lg)
        .background(Color(.systemBackground))
        .cornerRadius(UIConstants.CornerRadius.lg)
        .shadow(color: .black.opacity(0.05), radius: 10, y: 4)
        .sheet(isPresented: $showChat) {
            NavigationStack {
                ChatView(partner: availability.user)
            }
        }
    }
}

struct AvatarView: View {
    let url: String?
    let size: CGFloat

    var body: some View {
        Group {
            if let urlString = url, let url = URL(string: urlString) {
                AsyncImage(url: url) { image in
                    image
                        .resizable()
                        .scaledToFill()
                } placeholder: {
                    defaultAvatar
                }
            } else {
                defaultAvatar
            }
        }
        .frame(width: size, height: size)
        .clipShape(SwiftUI.Circle())
    }

    private var defaultAvatar: some View {
        SwiftUI.Circle()
            .fill(Color.primaryGreen.opacity(0.2))
            .overlay(
                Image(systemName: "person.fill")
                    .foregroundColor(.primaryGreen)
            )
    }
}

#Preview {
    AvailabilityCardView(
        availability: Availability(
            id: "1",
            user: UserProfile(id: "1", nickname: "小明", avatar: nil),
            startTime: Date(),
            endTime: Date().addingTimeInterval(3600 * 4),
            location: Location(type: .preset, name: "三里屯", latitude: nil, longitude: nil),
            status: .active,
            circleIds: ["1"],
            createdAt: Date()
        )
    )
    .padding()
    .background(Color(.systemGroupedBackground))
}
