import SwiftUI
#if canImport(FamilyControls)
import FamilyControls
#endif

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

    @State private var screenStatus: ScreenStatus = .idle
    @State private var locationStatus: LocationStatus = .unknown
    @State private var deviceStatus: DeviceStatus = .unknown
    @State private var currentCoordinate: (lat: Double, lng: Double)?
    @State private var homeCoordinate: (lat: Double, lng: Double)?
    @State private var workCoordinate: (lat: Double, lng: Double)?
    @State private var extensionMinutes: Int = 0
    @State private var extensionLastUpdate: Date?
    @State private var showAppPicker = false
    @State private var selectedAppCount = 0
    @State private var selectedCategoryCount = 0

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {

                    // 应用选择（关键！）
                    GroupBox {
                        VStack(alignment: .leading, spacing: 12) {
                            Text("需要选择要监控的应用才能获取屏幕时间数据")
                                .font(.caption)
                                .foregroundColor(.secondary)

                            row("已选应用", value: "\(selectedAppCount) 个")
                            row("已选分类", value: "\(selectedCategoryCount) 个")

                            Button {
                                showAppPicker = true
                            } label: {
                                HStack {
                                    Image(systemName: "plus.app")
                                    Text("选择要监控的应用")
                                }
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 10)
                                .background(Color.blue)
                                .foregroundColor(.white)
                                .cornerRadius(8)
                            }
                        }
                    } label: {
                        Label("应用选择", systemImage: "apps.iphone")
                            .font(.headline)
                    }

                    // 屏幕使用数据
                    GroupBox {
                        VStack(alignment: .leading, spacing: 12) {
                            row("是否活跃", value: screenStatus.isActive ? "是" : "否")
                            row("活动类型", value: activityTypeText(screenStatus.activityType))
                            row("本次使用时长", value: "\(screenStatus.sessionDurationMinutes) 分钟")
                            row("上次活跃", value: screenStatus.lastActiveMinutesAgo == 0 ? "刚刚" : "\(screenStatus.lastActiveMinutesAgo) 分钟前")
                        }
                    } label: {
                        Label("屏幕使用", systemImage: "iphone")
                            .font(.headline)
                    }

                    // 位置数据
                    GroupBox {
                        VStack(alignment: .leading, spacing: 12) {
                            row("位置类型", value: placeTypeText(locationStatus.placeType))
                            row("在此位置", value: "\(locationStatus.atPlaceSinceMinutes) 分钟")

                            if let coord = currentCoordinate {
                                row("坐标", value: String(format: "%.4f, %.4f", coord.lat, coord.lng))
                            }

                            Divider()

                            if let home = homeCoordinate {
                                row("🏠 家", value: String(format: "%.4f, %.4f", home.lat, home.lng))
                            } else {
                                row("🏠 家", value: "未学习")
                            }

                            if let work = workCoordinate {
                                row("🏢 公司", value: String(format: "%.4f, %.4f", work.lat, work.lng))
                            } else {
                                row("🏢 公司", value: "未学习")
                            }
                        }
                    } label: {
                        Label("地理位置", systemImage: "location")
                            .font(.headline)
                    }

                    // 设备状态
                    GroupBox {
                        VStack(alignment: .leading, spacing: 12) {
                            row("电池电量", value: "\(Int(deviceStatus.batteryLevel * 100))%")
                            row("充电状态", value: batteryStateText(deviceStatus.batteryState))
                            row("低电量模式", value: deviceStatus.isLowPowerMode ? "开启" : "关闭")
                            row("专注模式", value: deviceStatus.isFocusModeOn ? "开启" : "关闭")
                            row("耳机连接", value: deviceStatus.isHeadphonesConnected ? "已连接" : "未连接")
                            row("网络类型", value: networkTypeText(deviceStatus.networkType))
                            row("屏幕亮度", value: "\(Int(deviceStatus.screenBrightness * 100))%")
                        }
                    } label: {
                        Label("设备状态", systemImage: "battery.100.bolt")
                            .font(.headline)
                    }

                    // App Group 数据
                    GroupBox {
                        VStack(alignment: .leading, spacing: 12) {
                            row("屏幕时间", value: "\(extensionMinutes) 分钟")
                            row("更新时间", value: extensionLastUpdate?.formatted(.dateTime.hour().minute().second()) ?? "无")

                            Divider()

                            // 测试按钮：手动写入数据测试 App Group
                            Button("测试写入 (模拟 Extension)") {
                                testWriteToAppGroup()
                            }
                            .font(.caption)
                            .foregroundColor(.orange)
                        }
                    } label: {
                        Label("Extension 数据", systemImage: "app.badge")
                            .font(.headline)
                    }

                    // 刷新按钮
                    Button {
                        loadData()
                    } label: {
                        HStack {
                            Image(systemName: "arrow.clockwise")
                            Text("刷新数据")
                        }
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color.primaryGreen)
                        .foregroundColor(.white)
                        .cornerRadius(12)
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
            .onAppear {
                loadData()
            }
            .familyActivityPicker(isPresented: $showAppPicker, selection: pickerSelection)
            .onChange(of: showAppPicker) { newValue in
                if !newValue {
                    // Picker 关闭后保存选择并刷新
                    saveSelectionAndReload()
                }
            }
        }
    }

    #if canImport(FamilyControls)
    @available(iOS 16.0, *)
    private var pickerSelection: Binding<FamilyActivitySelection> {
        Binding(
            get: { ScreenDataCollector.shared.activitySelection },
            set: { ScreenDataCollector.shared.activitySelection = $0 }
        )
    }
    #endif

    private func saveSelectionAndReload() {
        #if canImport(FamilyControls)
        if #available(iOS 16.0, *) {
            ScreenDataCollector.shared.saveSelection()
            loadData()
        }
        #endif
    }

    private func loadData() {
        // 屏幕数据
        let screen = ScreenDataCollector.shared
        screenStatus = screen.currentStatus

        // 选择的应用数量
        #if canImport(FamilyControls)
        if #available(iOS 16.0, *) {
            selectedAppCount = screen.activitySelection.applicationTokens.count
            selectedCategoryCount = screen.activitySelection.categoryTokens.count
        }
        #endif

        // 位置数据
        let location = LocationDataCollector.shared
        locationStatus = location.currentStatus

        if let loc = location.currentLocation {
            currentCoordinate = (loc.coordinate.latitude, loc.coordinate.longitude)
        }
        if let home = location.homeLocation {
            homeCoordinate = (home.coordinate.latitude, home.coordinate.longitude)
        }
        if let work = location.workLocation {
            workCoordinate = (work.coordinate.latitude, work.coordinate.longitude)
        }

        // 设备状态数据
        let device = DeviceStatusCollector.shared
        deviceStatus = device.getCurrentStatus()

        // Extension 数据
        if let defaults = UserDefaults(suiteName: "group.com.youkong.app") {
            extensionMinutes = defaults.integer(forKey: "screenTimeMinutes")
            extensionLastUpdate = defaults.object(forKey: "screenTimeLastUpdate") as? Date
        }
    }

    private func testWriteToAppGroup() {
        // 模拟 Extension 写入数据，测试 App Group 是否正常
        if let defaults = UserDefaults(suiteName: "group.com.youkong.app") {
            defaults.set(99, forKey: "screenTimeMinutes")
            defaults.set(Date(), forKey: "screenTimeLastUpdate")
            defaults.synchronize()
            print("Test write to App Group completed")
            // 重新加载
            loadData()
        } else {
            print("Failed to access App Group")
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

    private func batteryStateText(_ state: BatteryState) -> String {
        switch state {
        case .unknown: return "未知"
        case .unplugged: return "未充电"
        case .charging: return "充电中"
        case .full: return "已充满"
        }
    }

    private func networkTypeText(_ type: NetworkType) -> String {
        switch type {
        case .wifi: return "WiFi"
        case .cellular: return "蜂窝网络"
        case .none: return "无网络"
        case .unknown: return "未知"
        }
    }
}

#Preview {
    FriendsListView()
        .environmentObject(AuthManager.shared)
}
