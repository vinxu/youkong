import SwiftUI

struct GridHomeView: View {
    @StateObject private var viewModel = GridHomeViewModel()
    @State private var showSettings = false

    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                // Header
                HeaderView(showSettings: $showSettings)

                // Grid
                if viewModel.isLoading && viewModel.friends.isEmpty {
                    ProgressView("加载中...")
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let errorMessage = viewModel.errorMessage {
                    ErrorView(message: errorMessage) {
                        Task {
                            await viewModel.loadGrid()
                        }
                    }
                } else if viewModel.friends.isEmpty {
                    EmptyStateView()
                } else {
                    ScrollView {
                        FriendGrid(friends: viewModel.friends, gridSize: viewModel.gridSize)
                            .padding()
                    }
                }

                Spacer()

                // Bottom Buttons
                BottomButtons(viewModel: viewModel)
            }
            .background(Color(.systemGroupedBackground))
            .navigationBarHidden(true)
        }
        .navigationViewStyle(.stack)
        .task {
            await viewModel.loadGrid()
        }
        .refreshable {
            await viewModel.loadGrid()
        }
        .sheet(isPresented: $viewModel.showPosterSheet) {
            PosterShareView(friends: viewModel.friends)
        }
        .sheet(isPresented: $showSettings) {
            ProfileView()
        }
    }
}

// MARK: - Header

struct HeaderView: View {
    @Binding var showSettings: Bool

    var body: some View {
        HStack {
            Text("有空")
                .font(.largeTitle)
                .fontWeight(.bold)

            Spacer()

            Button {
                showSettings = true
            } label: {
                Image(systemName: "gearshape")
                    .font(.title3)
                    .foregroundColor(.primary)
            }
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
        .background(Color(.systemBackground))
    }
}

// MARK: - Friend Grid

struct FriendGrid: View {
    let friends: [FriendGridItem]
    let gridSize: Int

    var columns: [GridItem] {
        Array(repeating: GridItem(.flexible(), spacing: 12), count: gridSize)
    }

    var body: some View {
        LazyVGrid(columns: columns, spacing: 12) {
            ForEach(friends) { friend in
                FriendCard(friend: friend)
            }
        }
    }
}

// MARK: - Friend Card

struct FriendCard: View {
    let friend: FriendGridItem

    var body: some View {
        VStack(spacing: 8) {
            // Emoji
            Text(friend.emoji)
                .font(.system(size: 40))

            // Nickname
            Text(friend.nickname)
                .font(.caption)
                .fontWeight(.bold)
                .lineLimit(1)

            // Status
            Text(friend.status)
                .font(.caption2)
                .foregroundColor(.secondary)
                .lineLimit(1)

            // Relative Time
            Text(friend.relativeTime)
                .font(.caption2)
                .foregroundColor(.gray)
        }
        .frame(maxWidth: .infinity)
        .aspectRatio(1.0, contentMode: .fill)
        .padding(12)
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .shadow(color: Color.black.opacity(0.05), radius: 4, x: 0, y: 2)
    }
}

// MARK: - Bottom Buttons

struct BottomButtons: View {
    @ObservedObject var viewModel: GridHomeViewModel

    var body: some View {
        HStack(spacing: 16) {
            // 更新状态按钮
            Button {
                Task {
                    await viewModel.updateStatus()
                }
            } label: {
                HStack {
                    Image(systemName: "arrow.clockwise")
                    Text("更新状态")
                }
                .font(.headline)
                .foregroundColor(.white)
                .frame(maxWidth: .infinity)
                .frame(height: 50)
                .background(Color.accentColor)
                .cornerRadius(12)
            }
            .disabled(viewModel.isLoading)

            // 分享按钮
            Button {
                viewModel.generatePoster()
            } label: {
                HStack {
                    Image(systemName: "square.and.arrow.up")
                    Text("分享")
                }
                .font(.headline)
                .foregroundColor(.accentColor)
                .frame(maxWidth: .infinity)
                .frame(height: 50)
                .background(Color(.systemBackground))
                .cornerRadius(12)
                .overlay(
                    RoundedRectangle(cornerRadius: 12)
                        .stroke(Color.accentColor, lineWidth: 2)
                )
            }
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 16)
        .background(Color(.systemGroupedBackground))
    }
}

// MARK: - Empty State

struct EmptyStateView: View {
    var body: some View {
        VStack(spacing: 20) {
            Image(systemName: "person.2")
                .font(.system(size: 60))
                .foregroundColor(.gray)

            Text("暂无好友")
                .font(.title3)
                .fontWeight(.semibold)
                .foregroundColor(.primary)

            Text("邀请朋友加入，看看他们此刻在做什么")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 40)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

// MARK: - Error View

struct ErrorView: View {
    let message: String
    let retry: () -> Void

    var body: some View {
        VStack(spacing: 20) {
            Image(systemName: "exclamationmark.triangle")
                .font(.system(size: 60))
                .foregroundColor(.orange)

            Text("加载失败")
                .font(.title3)
                .fontWeight(.semibold)

            Text(message)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 40)

            Button("重试", action: retry)
                .buttonStyle(.borderedProminent)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

// MARK: - Preview

#Preview {
    GridHomeView()
        .environmentObject(AuthManager.shared)
}
