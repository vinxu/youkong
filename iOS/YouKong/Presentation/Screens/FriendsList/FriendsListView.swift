import SwiftUI

// MARK: - Friends List View

struct FriendsListView: View {
    @StateObject private var viewModel = FriendsListViewModel()
    @State private var selectedFriend: FriendRecommendation?

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.isLoading && viewModel.friends.isEmpty {
                    loadingView
                } else if viewModel.isEmpty {
                    emptyView
                } else {
                    friendsList
                }
            }
            .navigationTitle("谁有空")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    NavigationLink {
                        ProfileView()
                    } label: {
                        Image(systemName: "person.circle")
                            .font(.title3)
                            .foregroundColor(.primary)
                    }
                }
            }
            .navigationDestination(for: FriendRecommendation.self) { friend in
                ChatView(partnerId: friend.friendId, partnerName: friend.name, partnerAvatar: friend.avatar)
            }
        }
        .task {
            await viewModel.loadFriends()
        }
    }

    // MARK: - Friends List

    private var friendsList: some View {
        ScrollView {
            LazyVStack(spacing: UIConstants.Spacing.md) {
                // 更新时间提示
                if let lastUpdatedText = viewModel.lastUpdatedText {
                    HStack {
                        Spacer()
                        Text("更新于 \(lastUpdatedText)")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    .padding(.horizontal, UIConstants.Spacing.lg)
                    .padding(.top, UIConstants.Spacing.sm)
                }

                // 朋友列表
                ForEach(viewModel.friends) { friend in
                    NavigationLink(value: friend) {
                        FriendProbabilityCard(friend: friend)
                    }
                    .buttonStyle(PlainButtonStyle())
                }
            }
            .padding(.horizontal, UIConstants.Spacing.lg)
            .padding(.bottom, UIConstants.Spacing.xxxl)
        }
        .refreshable {
            await viewModel.refresh()
        }
    }

    // MARK: - Loading View

    private var loadingView: some View {
        VStack(spacing: UIConstants.Spacing.lg) {
            ProgressView()
                .scaleEffect(1.2)
            Text("正在分析朋友状态...")
                .font(.subheadline)
                .foregroundColor(.secondary)
        }
    }

    // MARK: - Empty View

    private var emptyView: some View {
        VStack(spacing: UIConstants.Spacing.xl) {
            Image(systemName: "person.2.slash")
                .font(.system(size: 60))
                .foregroundColor(.secondary)

            Text("还没有朋友")
                .font(.title3)
                .fontWeight(.medium)

            Text("授权通讯录权限，找到你的朋友")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)

            Button {
                Task {
                    await viewModel.refresh()
                }
            } label: {
                Text("刷新")
                    .font(.headline)
                    .foregroundColor(.white)
                    .padding(.horizontal, UIConstants.Spacing.xxl)
                    .padding(.vertical, UIConstants.Spacing.md)
                    .background(Color.primaryGreen)
                    .cornerRadius(UIConstants.CornerRadius.md)
            }
        }
        .padding()
    }
}

#Preview {
    FriendsListView()
        .environmentObject(AuthManager.shared)
}
