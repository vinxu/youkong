import SwiftUI
#if canImport(FamilyControls)
import FamilyControls
#endif

// MARK: - Friends List View

struct FriendsListView: View {
    @StateObject private var viewModel = FriendsListViewModel()
    @State private var selectedFriend: FriendRecommendation?
    @State private var showAgentData = false
    @State private var showFriendsHub = false

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
                    HStack(spacing: 16) {
                        // 添加好友入口
                        Button {
                            showFriendsHub = true
                        } label: {
                            Image(systemName: "plus")
                                .font(.title3)
                                .foregroundColor(.primary)
                        }

                        // 个人中心
                        NavigationLink {
                            ProfileView()
                        } label: {
                            Image(systemName: "person.circle")
                                .font(.title3)
                                .foregroundColor(.primary)
                        }
                    }
                }
            }
            .sheet(isPresented: $showAgentData) {
                MyAgentDataSheet()
            }
            .sheet(isPresented: $showFriendsHub) {
                FriendsHubView()
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

    // LLM 预测结果
    @State private var analysisResult: AnalysisData?
    @State private var isUploading = false
    @State private var uploadError: String?

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {

                    // LLM 预测结果（顶部突出显示）
                    if let analysis = analysisResult {
                        GroupBox {
                            VStack(spacing: 16) {
                                // 生活状态 emoji
                                Text(analysis.lifeStatus.emoji)
                                    .font(.system(size: 60))

                                // 生活状态标签
                                Text(analysis.lifeStatus.label)
                                    .font(.title3)
                                    .fontWeight(.medium)

                                // 有空概率
                                HStack(spacing: 8) {
                                    Text("\(analysis.availability.probability)%")
                                        .font(.system(size: 32, weight: .bold))
                                        .foregroundColor(probabilityColor(analysis.availability.probability))

                                    Text(analysis.availability.status)
                                        .font(.headline)
                                        .padding(.horizontal, 12)
                                        .padding(.vertical, 4)
                                        .background(probabilityColor(analysis.availability.probability).opacity(0.2))
                                        .cornerRadius(12)
                                }

                                // 理由
                                Text(analysis.availability.reason)
                                    .font(.body)
                                    .foregroundColor(.secondary)
                                    .multilineTextAlignment(.center)

                                // 置信度
                                HStack {
                                    Text("置信度:")
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                    Text(analysis.availability.confidence)
                                        .font(.caption)
                                        .fontWeight(.medium)
                                }
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                        } label: {
                            Label("LLM 预测结果", systemImage: "brain.head.profile")
                                .font(.headline)
                        }
                    }

                    // 上报按钮
                    Button {
                        Task {
                            await uploadAndPredict()
                        }
                    } label: {
                        HStack {
                            if isUploading {
                                ProgressView()
                                    .tint(.white)
                            } else {
                                Image(systemName: "arrow.up.circle.fill")
                            }
                            Text(isUploading ? "上报中..." : "上报数据并获取预测")
                        }
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(isUploading ? Color.gray : Color.primaryGreen)
                        .foregroundColor(.white)
                        .cornerRadius(12)
                    }
                    .disabled(isUploading)

                    if let error = uploadError {
                        Text(error)
                            .font(.caption)
                            .foregroundColor(.red)
                    }

                    // 屏幕使用数据
                    GroupBox {
                        VStack(alignment: .leading, spacing: 12) {
                            row("是否活跃", value: screenStatus.isActive ? "是" : "否")
                            row("活动类型", value: activityTypeText(screenStatus.activityType))
                            row("本次使用时长", value: "\(screenStatus.sessionDurationMinutes) 分钟")
                            row("上次活跃", value: screenStatus.lastActiveMinutesAgo == 0 ? "刚刚" : "\(screenStatus.lastActiveMinutesAgo) 分钟前")
                            if let category = screenStatus.lastActiveCategory, !category.isEmpty {
                                row("最近使用分类", value: category)
                                    .foregroundColor(.blue)
                            }
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

                    // 屏幕时间监控设置
                    GroupBox {
                        VStack(alignment: .leading, spacing: 12) {
                            row("已选应用", value: "\(selectedAppCount) 个")
                            row("已选分类", value: "\(selectedCategoryCount) 个")
                            row("屏幕时间", value: "\(extensionMinutes) 分钟")
                            row("更新时间", value: extensionLastUpdate?.formatted(.dateTime.hour().minute().second()) ?? "无")

                            Divider()

                            // 选择要监控的应用按钮
                            Button {
                                showAppPicker = true
                            } label: {
                                HStack {
                                    Image(systemName: "apps.iphone")
                                    Text("选择要监控的应用")
                                    Spacer()
                                    Image(systemName: "chevron.right")
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                            }

                            if selectedAppCount == 0 && selectedCategoryCount == 0 {
                                Text("⚠️ 请选择要监控的应用或分类，否则无法获取屏幕使用时间")
                                    .font(.caption)
                                    .foregroundColor(.orange)
                            }

                            Button("清除测试数据") {
                                clearTestData()
                            }
                            .font(.caption)
                            .foregroundColor(.red)
                        }
                    } label: {
                        Label("屏幕时间监控", systemImage: "app.badge")
                            .font(.headline)
                    }

                    // 刷新本地数据按钮
                    Button {
                        loadData()
                    } label: {
                        HStack {
                            Image(systemName: "arrow.clockwise")
                            Text("刷新本地数据")
                        }
                        .font(.caption)
                        .foregroundColor(.secondary)
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
            .task {
                await uploadAndPredict()
            }
            .sheet(isPresented: $showAppPicker) {
                appPickerSheet
            }
        }
    }

    // MARK: - App Picker Sheet

    @ViewBuilder
    private var appPickerSheet: some View {
        #if canImport(FamilyControls)
        if #available(iOS 16.0, *) {
            NavigationStack {
                FamilyActivityPickerView(
                    selection: Binding(
                        get: { ScreenDataCollector.shared.activitySelection },
                        set: { ScreenDataCollector.shared.activitySelection = $0 }
                    )
                ) {
                    // 保存选择并重新启动监控
                    ScreenDataCollector.shared.saveSelection()
                    ScreenDataCollector.shared.startMonitoring()
                    loadData()
                    showAppPicker = false
                }
            }
        } else {
            Text("需要 iOS 16 或更高版本")
        }
        #else
        Text("当前设备不支持屏幕时间监控")
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

    private func uploadAndPredict() async {
        loadData()  // 先刷新本地数据

        isUploading = true
        uploadError = nil

        print("📤 [Agent] 开始上报数据...")

        // 构建请求数据（按 API 规范分组）
        let screenData = ScreenRequestData(
            isActive: screenStatus.isActive,
            activityType: screenStatus.activityType.rawValue,
            sessionDurationMinutes: screenStatus.sessionDurationMinutes,
            lastActiveMinutesAgo: screenStatus.lastActiveMinutesAgo,
            lastActiveCategory: screenStatus.lastActiveCategory
        )

        let locationData = LocationRequestData(
            placeType: locationStatus.placeType.rawValue,
            atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes
        )

        let batteryData = BatteryRequestData(
            batteryLevel: Int(deviceStatus.batteryLevel * 100),
            batteryState: deviceStatus.batteryState.rawValue,
            isCharging: deviceStatus.isCharging
        )

        let modeData = ModeRequestData(
            isLowPowerMode: deviceStatus.isLowPowerMode,
            isFocusModeOn: deviceStatus.isFocusModeOn
        )

        let connectionData = ConnectionRequestData(
            isHeadphonesConnected: deviceStatus.isHeadphonesConnected,
            networkType: deviceStatus.networkType.rawValue
        )

        let displayData = DisplayRequestData(
            screenBrightness: deviceStatus.screenBrightness
        )

        let request = StatusReportRequest(
            screen: screenData,
            location: locationData,
            battery: batteryData,
            mode: modeData,
            connection: connectionData,
            display: displayData
        )

        do {
            let repo = AgentRepositoryImpl()
            print("📤 [Agent] 发送请求到服务器...")
            let response = try await repo.reportStatus(request: request)
            print("✅ [Agent] 收到响应: success=\(response.success)")
            if let analysis = response.analysis {
                print("✅ [Agent] 预测结果: \(analysis.lifeStatus.emoji) \(analysis.lifeStatus.label)")
                print("✅ [Agent] 有空概率: \(analysis.availability.probability)% - \(analysis.availability.reason)")
            } else {
                print("⚠️ [Agent] 响应中没有 analysis 数据")
            }
            analysisResult = response.analysis
        } catch {
            uploadError = "上报失败: \(error.localizedDescription)"
            print("❌ [Agent] 上报失败: \(error)")
        }

        isUploading = false
    }

    private func probabilityColor(_ probability: Int) -> Color {
        switch probability {
        case 80...100: return Color(hex: "#22C55E") ?? .green
        case 60..<80: return Color(hex: "#86EFAC") ?? .green.opacity(0.7)
        case 40..<60: return Color(hex: "#FACC15") ?? .yellow
        case 20..<40: return Color(hex: "#FB923C") ?? .orange
        case 0..<20: return Color(hex: "#EF4444") ?? .red
        default: return .gray
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

    private func clearTestData() {
        if let defaults = UserDefaults(suiteName: "group.com.youkong.app") {
            defaults.removeObject(forKey: "screenTimeMinutes")
            defaults.removeObject(forKey: "screenTimeLastUpdate")
            defaults.synchronize()
            print("🗑️ [Agent] 测试数据已清除")
            loadData()
        }
    }
}

// MARK: - Family Activity Picker View

#if canImport(FamilyControls)
import FamilyControls

@available(iOS 16.0, *)
struct FamilyActivityPickerView: View {
    @Binding var selection: FamilyActivitySelection
    let onSave: () -> Void

    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            // 说明文字
            VStack(spacing: 8) {
                Text("选择要监控的应用")
                    .font(.headline)
                Text("选择后，我们可以了解你的屏幕使用情况来判断你是否有空")
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
            }
            .padding()
            .background(Color(.systemGroupedBackground))

            // FamilyActivityPicker
            FamilyActivityPicker(selection: $selection)
        }
        .navigationTitle("选择应用")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button("取消") {
                    dismiss()
                }
            }
            ToolbarItem(placement: .confirmationAction) {
                Button("保存") {
                    onSave()
                }
                .fontWeight(.semibold)
            }
        }
    }
}
#endif

#Preview {
    FriendsListView()
        .environmentObject(AuthManager.shared)
}
