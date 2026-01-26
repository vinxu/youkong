import SwiftUI

// MARK: - Friends List View

struct FriendsListView: View {
    @StateObject private var viewModel = FriendsListViewModel()
    @State private var selectedFriend: FriendRecommendation?
    @State private var showAgentData = false

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
                ToolbarItem(placement: .topBarLeading) {
                    Button {
                        showAgentData = true
                    } label: {
                        Image(systemName: "chart.bar.doc.horizontal")
                            .font(.title3)
                            .foregroundColor(.primary)
                    }
                }
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
            .sheet(isPresented: $showAgentData) {
                MyAgentDataSheet()
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

// MARK: - My Agent Data Sheet

struct MyAgentDataSheet: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var screenCollector = ScreenDataCollector.shared
    @StateObject private var locationCollector = LocationDataCollector.shared

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {

                    // 屏幕使用数据
                    GroupBox {
                        let status = screenCollector.getCurrentStatus()
                        VStack(alignment: .leading, spacing: 12) {
                            row("是否活跃", value: status.isActive ? "是" : "否")
                            row("活动类型", value: activityTypeText(status.activityType))
                            row("本次使用时长", value: "\(status.sessionDurationMinutes) 分钟")
                            row("上次活跃", value: status.lastActiveMinutesAgo == 0 ? "刚刚" : "\(status.lastActiveMinutesAgo) 分钟前")
                        }
                    } label: {
                        Label("屏幕使用", systemImage: "iphone")
                            .font(.headline)
                    }

                    // 位置数据
                    GroupBox {
                        let status = locationCollector.getCurrentStatus()
                        VStack(alignment: .leading, spacing: 12) {
                            row("位置类型", value: placeTypeText(status.placeType))
                            row("在此位置", value: "\(status.atPlaceSinceMinutes) 分钟")

                            if let loc = locationCollector.currentLocation {
                                row("坐标", value: String(format: "%.4f, %.4f", loc.coordinate.latitude, loc.coordinate.longitude))
                            }

                            Divider()

                            if let home = locationCollector.homeLocation {
                                row("🏠 家", value: String(format: "%.4f, %.4f", home.coordinate.latitude, home.coordinate.longitude))
                            } else {
                                row("🏠 家", value: "未学习")
                            }

                            if let work = locationCollector.workLocation {
                                row("🏢 公司", value: String(format: "%.4f, %.4f", work.coordinate.latitude, work.coordinate.longitude))
                            } else {
                                row("🏢 公司", value: "未学习")
                            }
                        }
                    } label: {
                        Label("地理位置", systemImage: "location")
                            .font(.headline)
                    }

                    // App Group 数据
                    GroupBox {
                        if let defaults = UserDefaults(suiteName: "group.com.youkong.app") {
                            let minutes = defaults.integer(forKey: "screenTimeMinutes")
                            let lastUpdate = defaults.object(forKey: "screenTimeLastUpdate") as? Date

                            VStack(alignment: .leading, spacing: 12) {
                                row("屏幕时间", value: "\(minutes) 分钟")
                                row("更新时间", value: lastUpdate?.formatted(.dateTime.hour().minute().second()) ?? "无")
                            }
                        }
                    } label: {
                        Label("Extension 数据", systemImage: "app.badge")
                            .font(.headline)
                    }
                }
                .padding()
            }
            .navigationTitle("我的 Agent 数据")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("关闭") {
                        dismiss()
                    }
                }
            }
        }
    }

    private func row(_ label: String, value: String) -> some View {
        HStack {
            Text(label)
                .foregroundColor(.secondary)
            Spacer()
            Text(value)
                .fontWeight(.medium)
        }
    }

    private func activityTypeText(_ type: ActivityType) -> String {
        switch type {
        case .entertainment: return "娱乐"
        case .productivity: return "工作"
        case .communication: return "聊天"
        case .idle: return "空闲"
        }
    }

    private func placeTypeText(_ type: PlaceType) -> String {
        switch type {
        case .home: return "在家"
        case .work: return "在公司"
        case .leisure: return "休闲场所"
        case .transit: return "在路上"
        case .unknown: return "未知"
        }
    }
}

#Preview {
    FriendsListView()
        .environmentObject(AuthManager.shared)
}
