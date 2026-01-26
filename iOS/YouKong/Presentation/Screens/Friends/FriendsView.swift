//
//  FriendsView.swift
//  YouKong
//
//  好友列表页面
//

import SwiftUI

struct FriendsView: View {
    @StateObject private var viewModel = FriendsViewModel()
    @State private var showDeleteConfirm = false
    @State private var friendToDelete: String?

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Tab 选择器
                Picker("", selection: $viewModel.selectedTab) {
                    ForEach(FriendTab.allCases, id: \.self) { tab in
                        Text(tab.rawValue).tag(tab)
                    }
                }
                .pickerStyle(.segmented)
                .padding()

                // 列表内容
                ScrollView {
                    LazyVStack(spacing: UIConstants.Spacing.md) {
                        switch viewModel.selectedTab {
                        case .all:
                            if viewModel.friends.isEmpty && !viewModel.isLoading {
                                FriendsEmptyView(message: "还没有好友")
                            } else {
                                ForEach(viewModel.friends) { friend in
                                    FriendRowView(
                                        profile: friend.user,
                                        subtitle: friend.source.displayText,
                                        onDelete: {
                                            friendToDelete = friend.user.id
                                            showDeleteConfirm = true
                                        }
                                    )
                                }
                            }

                        case .invitedByMe:
                            if viewModel.invitedByMe.isEmpty && !viewModel.isLoading {
                                FriendsEmptyView(message: "还没有邀请过好友")
                            } else {
                                ForEach(viewModel.invitedByMe) { friend in
                                    FriendRowView(
                                        profile: friend.user,
                                        subtitle: friend.circleName ?? "直接邀请",
                                        onDelete: {
                                            friendToDelete = friend.user.id
                                            showDeleteConfirm = true
                                        }
                                    )
                                }
                            }

                        case .invitedMe:
                            if viewModel.invitedMe.isEmpty && !viewModel.isLoading {
                                FriendsEmptyView(message: "还没有被邀请过")
                            } else {
                                ForEach(viewModel.invitedMe) { friend in
                                    FriendRowView(
                                        profile: friend.user,
                                        subtitle: friend.circleName ?? "直接邀请",
                                        onDelete: {
                                            friendToDelete = friend.user.id
                                            showDeleteConfirm = true
                                        }
                                    )
                                }
                            }
                        }
                    }
                    .padding(UIConstants.Spacing.lg)
                }
                .refreshable {
                    await viewModel.refresh()
                }
            }
            .navigationTitle("好友")
            .alert("删除好友", isPresented: $showDeleteConfirm) {
                Button("取消", role: .cancel) {
                    friendToDelete = nil
                }
                Button("删除", role: .destructive) {
                    if let userId = friendToDelete {
                        Task {
                            await viewModel.deleteFriend(userId: userId)
                        }
                    }
                    friendToDelete = nil
                }
            } message: {
                Text("确定要删除这位好友吗？")
            }
        }
        .task {
            await viewModel.loadFriends()
        }
    }
}

struct FriendRowView: View {
    let profile: UserProfile
    let subtitle: String
    let onDelete: () -> Void

    var body: some View {
        HStack(spacing: UIConstants.Spacing.md) {
            AvatarView(url: profile.avatar, size: 50)

            VStack(alignment: .leading, spacing: 2) {
                Text(profile.nickname)
                    .font(.headline)

                Text(subtitle)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            Spacer()
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(UIConstants.CornerRadius.md)
        .shadow(color: .black.opacity(0.03), radius: 4, y: 2)
        .contextMenu {
            Button(role: .destructive) {
                onDelete()
            } label: {
                Label("删除好友", systemImage: "person.badge.minus")
            }
        }
        .swipeActions(edge: .trailing) {
            Button(role: .destructive) {
                onDelete()
            } label: {
                Label("删除", systemImage: "trash")
            }
        }
    }
}

struct FriendsEmptyView: View {
    let message: String

    var body: some View {
        VStack(spacing: UIConstants.Spacing.lg) {
            Image(systemName: "person.2")
                .font(.system(size: 60))
                .foregroundColor(.gray.opacity(0.5))

            Text(message)
                .font(.headline)
                .foregroundColor(.secondary)
        }
        .padding(.top, 100)
    }
}

extension FriendSource {
    var displayText: String {
        switch self {
        case .invitation:
            return "通过邀请添加"
        case .search:
            return "通过搜索添加"
        case .manual:
            return "手动添加"
        }
    }
}

#Preview {
    FriendsView()
}
