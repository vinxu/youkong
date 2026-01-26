import SwiftUI

struct MyAvailabilitiesView: View {
    @StateObject private var viewModel = MyAvailabilitiesViewModel()

    var body: some View {
        ScrollView {
            LazyVStack(spacing: UIConstants.Spacing.lg) {
                if viewModel.availabilities.isEmpty && !viewModel.isLoading {
                    EmptyStateView(
                        icon: "calendar.badge.clock",
                        title: "暂无有空记录",
                        subtitle: "发布一个有空状态吧"
                    )
                    .padding(.top, 100)
                } else {
                    ForEach(viewModel.availabilities) { availability in
                        MyAvailabilityCard(
                            availability: availability,
                            onCancel: {
                                Task {
                                    await viewModel.cancel(id: availability.id)
                                }
                            }
                        )
                    }
                }
            }
            .padding(UIConstants.Spacing.lg)
        }
        .refreshable {
            await viewModel.refresh()
        }
        .navigationTitle("我的有空")
        .task {
            await viewModel.loadAvailabilities()
        }
    }
}

struct MyAvailabilityCard: View {
    let availability: Availability
    let onCancel: () -> Void
    @State private var showCancelConfirm = false

    var body: some View {
        VStack(alignment: .leading, spacing: UIConstants.Spacing.md) {
            HStack {
                StatusBadge(status: availability.status)
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

            if availability.status == .active {
                Button {
                    showCancelConfirm = true
                } label: {
                    Text("取消")
                        .font(.subheadline)
                        .fontWeight(.medium)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, UIConstants.Spacing.sm)
                        .background(Color.red.opacity(0.1))
                        .foregroundColor(.red)
                        .cornerRadius(UIConstants.CornerRadius.sm)
                }
            }
        }
        .padding(UIConstants.Spacing.lg)
        .background(Color(.systemBackground))
        .cornerRadius(UIConstants.CornerRadius.lg)
        .shadow(color: .black.opacity(0.05), radius: 10, y: 4)
        .alert("取消有空", isPresented: $showCancelConfirm) {
            Button("取消", role: .cancel) {}
            Button("确定", role: .destructive) {
                onCancel()
            }
        } message: {
            Text("确定要取消这个有空状态吗？")
        }
    }
}

struct StatusBadge: View {
    let status: AvailabilityStatus

    var body: some View {
        Text(status.displayName)
            .font(.caption)
            .fontWeight(.medium)
            .padding(.horizontal, UIConstants.Spacing.sm)
            .padding(.vertical, UIConstants.Spacing.xs)
            .background(backgroundColor)
            .foregroundColor(foregroundColor)
            .cornerRadius(UIConstants.CornerRadius.sm)
    }

    private var backgroundColor: Color {
        switch status {
        case .active: return Color.primaryGreen.opacity(0.2)
        case .expired: return Color.gray.opacity(0.2)
        case .cancelled: return Color.red.opacity(0.2)
        case .fulfilled: return Color.blue.opacity(0.2)
        }
    }

    private var foregroundColor: Color {
        switch status {
        case .active: return .primaryGreen
        case .expired: return .gray
        case .cancelled: return .red
        case .fulfilled: return .blue
        }
    }
}

#Preview {
    NavigationStack {
        MyAvailabilitiesView()
    }
}
