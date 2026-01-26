import SwiftUI

struct HomeView: View {
    @StateObject private var viewModel = HomeViewModel()

    var body: some View {
        NavigationStack {
            ScrollView {
                LazyVStack(spacing: UIConstants.Spacing.lg) {
                    if viewModel.availabilities.isEmpty && !viewModel.isLoading {
                        EmptyStateView(
                            icon: "calendar.badge.clock",
                            title: "暂无朋友有空",
                            subtitle: "去邀请朋友加入圈子吧"
                        )
                        .padding(.top, 100)
                    } else {
                        ForEach(viewModel.availabilities) { availability in
                            AvailabilityCardView(availability: availability)
                                .padding(.horizontal, UIConstants.Spacing.lg)
                        }
                    }
                }
                .padding(.vertical, UIConstants.Spacing.lg)
            }
            .refreshable {
                await viewModel.refresh()
            }
            .navigationTitle("有空")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        viewModel.showMyAvailabilities.toggle()
                    } label: {
                        Image(systemName: viewModel.showMyAvailabilities ? "person.fill" : "person.2.fill")
                    }
                }
            }
        }
        .task {
            await viewModel.loadAvailabilities()
        }
    }
}

struct EmptyStateView: View {
    let icon: String
    let title: String
    let subtitle: String

    var body: some View {
        VStack(spacing: UIConstants.Spacing.lg) {
            Image(systemName: icon)
                .font(.system(size: 60))
                .foregroundColor(.gray.opacity(0.5))

            VStack(spacing: UIConstants.Spacing.sm) {
                Text(title)
                    .font(.headline)
                    .foregroundColor(.secondary)

                Text(subtitle)
                    .font(.subheadline)
                    .foregroundColor(.secondary.opacity(0.8))
            }
        }
    }
}

#Preview {
    HomeView()
        .environmentObject(AuthManager.shared)
}
