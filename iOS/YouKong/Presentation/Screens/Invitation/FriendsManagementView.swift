import SwiftUI

// MARK: - Friends Management View

struct FriendsManagementView: View {
    @StateObject private var viewModel = FriendsManagementViewModel()
    @State private var selectedTab = 0
    @State private var showAddFriend = false
    @State private var showContactFriends = false
    @State private var showFriendRequests = false

    var body: some View {
        VStack(spacing: 0) {
            // Tab 选择器
            Picker("", selection: $selectedTab) {
                Text("全部好友").tag(0)
                Text("我邀请的").tag(1)
                Text("邀请我的").tag(2)
            }
            .pickerStyle(.segmented)
            .padding()

            // 内容
            TabView(selection: $selectedTab) {
                allFriendsTab
                    .tag(0)

                invitedByMeTab
                    .tag(1)

                invitedMeTab
                    .tag(2)
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
        }
        .navigationTitle("好友管理")
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Button {
                    showFriendRequests = true
                } label: {
                    ZStack(alignment: .topTrailing) {
                        Image(systemName: "bell")

                        if viewModel.pendingRequestCount > 0 {
                            Text("\(viewModel.pendingRequestCount)")
                                .font(.system(size: 10, weight: .bold))
                                .foregroundColor(.white)
                                .frame(minWidth: 16, minHeight: 16)
                                .background(Color.red)
                                .clipShape(Circle())
                                .offset(x: 8, y: -8)
                        }
                    }
                }
            }

            ToolbarItem(placement: .topBarTrailing) {
                Menu {
                    Button {
                        showAddFriend = true
                    } label: {
                        Label("通过手机号添加", systemImage: "phone")
                    }

                    Button {
                        showContactFriends = true
                    } label: {
                        Label("从通讯录添加", systemImage: "person.crop.rectangle.stack")
                    }
                } label: {
                    Image(systemName: "person.badge.plus")
                }
            }
        }
        .sheet(isPresented: $showAddFriend) {
            AddFriendSheetView()
        }
        .sheet(isPresented: $showContactFriends) {
            NavigationStack {
                ContactFriendsView()
                    .toolbar {
                        ToolbarItem(placement: .topBarLeading) {
                            Button("完成") {
                                showContactFriends = false
                            }
                        }
                    }
            }
        }
        .sheet(isPresented: $showFriendRequests) {
            NavigationStack {
                FriendRequestsView()
                    .toolbar {
                        ToolbarItem(placement: .topBarLeading) {
                            Button("完成") {
                                showFriendRequests = false
                            }
                        }
                    }
            }
        }
        .task {
            await viewModel.loadFriends()
            await viewModel.loadInvitedByMe()
            await viewModel.loadInvitedMe()
            await viewModel.loadPendingRequestCount()
        }
    }

    // MARK: - All Friends Tab

    private var allFriendsTab: some View {
        Group {
            if viewModel.isLoading && viewModel.friends.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if viewModel.friends.isEmpty {
                emptyView(
                    icon: "person.2",
                    title: "还没有好友",
                    subtitle: "邀请朋友加入吧"
                )
            } else {
                List {
                    ForEach(viewModel.friends) { friend in
                        FriendRow(friend: friend)
                            .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                                Button(role: .destructive) {
                                    Task {
                                        await viewModel.deleteFriend(friend)
                                    }
                                } label: {
                                    Label("删除", systemImage: "trash")
                                }
                            }
                    }
                }
                .listStyle(.plain)
            }
        }
    }

    // MARK: - Invited By Me Tab

    private var invitedByMeTab: some View {
        Group {
            if viewModel.invitedByMe.isEmpty {
                emptyView(
                    icon: "person.badge.plus",
                    title: "还没有邀请过好友",
                    subtitle: "创建邀请链接分享给朋友"
                )
            } else {
                List {
                    ForEach(viewModel.invitedByMe) { friend in
                        FriendWithInvitationRow(friend: friend, showCircle: true)
                    }
                }
                .listStyle(.plain)
            }
        }
    }

    // MARK: - Invited Me Tab

    private var invitedMeTab: some View {
        Group {
            if viewModel.invitedMe.isEmpty {
                emptyView(
                    icon: "person.wave.2",
                    title: "还没有人邀请你",
                    subtitle: "让朋友分享邀请链接给你"
                )
            } else {
                List {
                    ForEach(viewModel.invitedMe) { friend in
                        FriendWithInvitationRow(friend: friend, showCircle: true)
                    }
                }
                .listStyle(.plain)
            }
        }
    }

    private func emptyView(icon: String, title: String, subtitle: String) -> some View {
        VStack(spacing: 16) {
            Image(systemName: icon)
                .font(.system(size: 48))
                .foregroundColor(.secondary)
            Text(title)
                .font(.headline)
                .foregroundColor(.secondary)
            Text(subtitle)
                .font(.subheadline)
                .foregroundColor(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

// MARK: - Friend Row

struct FriendRow: View {
    let friend: FriendInfo

    var body: some View {
        HStack(spacing: 12) {
            AvatarView(url: friend.user.avatar, size: 44)

            VStack(alignment: .leading, spacing: 4) {
                Text(friend.user.nickname)
                    .fontWeight(.medium)

                HStack(spacing: 4) {
                    Image(systemName: sourceIcon)
                        .font(.caption2)
                    Text(sourceText)
                        .font(.caption)
                }
                .foregroundColor(.secondary)
            }

            Spacer()
        }
        .padding(.vertical, 4)
    }

    private var sourceIcon: String {
        switch friend.source {
        case .invitation:
            return "link"
        case .search:
            return "magnifyingglass"
        case .manual:
            return "hand.raised"
        }
    }

    private var sourceText: String {
        switch friend.source {
        case .invitation:
            return "通过邀请"
        case .search:
            return "通过搜索"
        case .manual:
            return "手动添加"
        }
    }
}

// MARK: - Friend With Invitation Row

struct FriendWithInvitationRow: View {
    let friend: FriendWithInvitation
    let showCircle: Bool

    var body: some View {
        HStack(spacing: 12) {
            AvatarView(url: friend.user.avatar, size: 44)

            VStack(alignment: .leading, spacing: 4) {
                Text(friend.user.nickname)
                    .fontWeight(.medium)

                if showCircle, let circleName = friend.circleName {
                    HStack(spacing: 4) {
                        Image(systemName: "circle.grid.2x2")
                            .font(.caption2)
                        Text(circleName)
                            .font(.caption)
                    }
                    .foregroundColor(.secondary)
                }
            }

            Spacer()

            Text(friend.createdAt, style: .date)
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 4)
    }
}

#Preview {
    NavigationStack {
        FriendsManagementView()
    }
}
