import SwiftUI

// MARK: - Root View

// MARK: - App Version Response

private struct AppVersionInfo: Decodable {
    let latestVersion: String
    let minVersion: String
    let updateUrl: String
    let changelog: String
    let forceUpdate: Bool

    enum CodingKeys: String, CodingKey {
        case latestVersion = "latest_version"
        case minVersion = "min_version"
        case updateUrl = "update_url"
        case changelog
        case forceUpdate = "force_update"
    }
}

// MARK: - Onboarding Wow Phase

enum OnboardingWowPhase: Equatable {
    case inference
    case chat(emoji: String, activity: String)
    case preview(emoji: String, status: String)
}

// MARK: - Root View

struct RootView: View {
    @EnvironmentObject private var authManager: AuthManager
    @EnvironmentObject private var deepLinkManager: DeepLinkManager
    @EnvironmentObject private var notificationManager: NotificationManager
    @StateObject private var permissionManager = PermissionManager.shared
    @AppStorage("hasCompletedOnboarding") private var hasCompletedOnboarding = false
    @Environment(\.scenePhase) private var scenePhase

    // 版本更新状态
    @State private var showUpdateAlert = false
    @State private var showForceUpdateAlert = false
    @State private var versionInfo: AppVersionInfo?

    // Onboarding Wow 链路（会话级，不持久化）
    @State private var onboardingWowPhase: OnboardingWowPhase? = nil

    var body: some View {
        Group {
            if authManager.isAuthenticated {
                if let phase = onboardingWowPhase {
                    // 3-Phase Wow 链路
                    wowPhaseView(phase: phase)
                } else if hasCompletedOnboarding {
                    // 已完成引导，直接进入主页面
                    MainTabView()
                        #if DEBUG
                        .onLongPressGesture(minimumDuration: 3) {
                            // 长按 3 秒触发 Wow 链路（调试用）
                            onboardingWowPhase = .inference
                        }
                        #endif
                        .task {
                            // 🔔 请求通知权限
                            await notificationManager.requestPermissionIfNeeded()
                            // 🔔 如果已授权，注册远程推送
                            await notificationManager.checkAuthorizationStatus()
                            if notificationManager.isAuthorized {
                                await notificationManager.registerForRemoteNotifications()
                            }
                            // 检查权限状态（用于数据收集）
                            await permissionManager.checkAllPermissions()
                            // 启动数据收集
                            startDataCollection()
                            // 刷新未读消息 Badge
                            await notificationManager.refreshBadgeFromServer()
                            // 检查版本更新
                            await checkAppVersion()
                        }
                } else if permissionManager.isChecking {
                    // 正在检查权限状态
                    ProgressView("检查权限...")
                } else if permissionManager.status.allGranted {
                    // 权限都已授权，启动数据收集并进入 Wow 链路
                    Color.clear.task {
                        hasCompletedOnboarding = true
                        startDataCollection()
                        onboardingWowPhase = .inference
                    }
                } else {
                    // 显示完整引导流程（4屏）
                    OnboardingView(isCompleted: $hasCompletedOnboarding)
                        .onChange(of: hasCompletedOnboarding) { completed in
                            if completed {
                                startDataCollection()
                                onboardingWowPhase = .inference
                            }
                        }
                }
            } else {
                LoginView()
            }
        }
        .task {
            // APP 启动时检查权限状态
            if authManager.isAuthenticated && !hasCompletedOnboarding {
                await permissionManager.checkAllPermissions()
            }
        }
        .animation(.easeInOut, value: authManager.isAuthenticated)
        .animation(.easeInOut, value: hasCompletedOnboarding)
        .onChange(of: authManager.isAuthenticated) { isAuth in
            if !isAuth {
                WebSocketManager.shared.disconnect()
            }
        }
        .onChange(of: scenePhase) { newPhase in
            guard authManager.isAuthenticated && hasCompletedOnboarding else { return }
            switch newPhase {
            case .active, .background:
                // 前台/后台切换时自动上报状态（带 60s 冷却）
                Task { await StatusReportManager.shared.reportIfNeeded() }
            default:
                break
            }
        }
        .alert("发现新版本 v\(versionInfo?.latestVersion ?? "")", isPresented: $showUpdateAlert) {
            Button("稍后再说", role: .cancel) {
                UserDefaults.standard.set(Date().timeIntervalSince1970, forKey: "lastUpdateAlertTime")
            }
            Button("去更新") {
                if let urlString = versionInfo?.updateUrl, let url = URL(string: urlString) {
                    UIApplication.shared.open(url)
                }
            }
        } message: {
            Text(versionInfo?.changelog ?? "")
        }
        .alert("需要更新", isPresented: $showForceUpdateAlert) {
            Button("去更新") {
                if let urlString = versionInfo?.updateUrl, let url = URL(string: urlString) {
                    UIApplication.shared.open(url)
                }
                // 重新弹出，不可关闭
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
                    showForceUpdateAlert = true
                }
            }
        } message: {
            Text("当前版本过低，请更新到最新版本后继续使用\n\n\(versionInfo?.changelog ?? "")")
        }
        .sheet(isPresented: $deepLinkManager.showInvitationSheet) {
            if let code = deepLinkManager.pendingInvitationCode {
                AcceptInvitationView(code: code) {
                    deepLinkManager.clearPendingInvitation()
                }
            }
        }
        // #if DEBUG
        // .withNetworkOverlay()
        // .withDebugButton()
        // .shakeToDebug()
        // #endif
    }

    // MARK: - Wow Phase Views

    @ViewBuilder
    private func wowPhaseView(phase: OnboardingWowPhase) -> some View {
        switch phase {
        case .inference:
            OnboardingInferenceView(
                onConfirm: { emoji, activity in
                    withAnimation {
                        onboardingWowPhase = .chat(emoji: emoji, activity: activity)
                    }
                },
                onSkip: {
                    // 跳过到主页，不用动画（避免 MainTabView 被反复创建导致请求取消）
                    onboardingWowPhase = nil
                }
            )
        case .chat(let emoji, let activity):
            OnboardingChatView(
                inferredEmoji: emoji,
                inferredActivity: activity,
                onComplete: {
                    withAnimation {
                        onboardingWowPhase = .preview(emoji: emoji, status: activity)
                    }
                },
                onSkip: {
                    onboardingWowPhase = nil
                }
            )
        case .preview(let emoji, let status):
            OnboardingPreviewView(
                emoji: emoji,
                status: status,
                onInvite: {
                    // Share sheet is handled inside OnboardingPreviewView
                },
                onEnter: {
                    onboardingWowPhase = nil
                }
            )
        }
    }

    private func startDataCollection() {
        // 启动位置数据收集
        if permissionManager.status.location {
            LocationDataCollector.shared.startMonitoring()
        }

        // ⚠️ 屏幕使用数据收集已禁用（方案 C）
        // if permissionManager.status.screenTime {
        //     ScreenDataCollector.shared.startMonitoring()
        // }

        // 启动运动数据收集
        if permissionManager.status.motion {
            MovementDataCollector.shared.startMonitoring()
        }

        // 启动设备状态收集（无需权限）
        DeviceStatusCollector.shared.startMonitoring()

        // 日历数据收集器在查询时自动获取，无需启动监控

        // 启动 WebSocket 连接
        WebSocketManager.shared.connect()

        // ⚠️ 禁用自动上报（调试模式）
        // startStatusReporting()
    }

    // ⚠️ 已禁用自动上报，改为手动上报（在 Holmes Agent 页面点击按钮）
    // private func startStatusReporting() {
    //     // 每30秒上报一次状态
    //     Timer.scheduledTimer(withTimeInterval: 30, repeats: true) { _ in
    //         Task {
    //             await reportStatus()
    //         }
    //     }
    // }

    private func reportStatus() async {
        print("=== [STATUS REPORT] Starting ===")
        // ⚠️ 屏幕数据已禁用（方案 C）
        // let screenStatus = ScreenDataCollector.shared.getCurrentStatus()
        let locationStatus = LocationDataCollector.shared.getCurrentStatus()
        let deviceStatus = DeviceStatusCollector.shared.getCurrentStatus()
        let calendarStatus = CalendarDataCollector.shared.getCurrentStatus()
        let movementStatus = MovementDataCollector.shared.getCurrentStatus()

        // print("[STATUS] Screen: active=\(screenStatus.isActive), type=\(screenStatus.activityType.rawValue), duration=\(screenStatus.sessionDurationMinutes)min")
        print("[STATUS] Location: \(locationStatus.placeName ?? locationStatus.placeType.rawValue), since=\(locationStatus.atPlaceSinceMinutes)min")
        print("[STATUS] Calendar: hasEvent=\(calendarStatus.hasCurrentEvent), remaining=\(calendarStatus.todayRemainingCount)")
        print("[STATUS] Movement: moving=\(movementStatus.isMoving), steps=\(movementStatus.stepsToday), type=\(movementStatus.movementType.rawValue)")
        print("[STATUS] Battery: \(Int(deviceStatus.batteryLevel * 100))%, charging=\(deviceStatus.isCharging)")

        // ⚠️ 不上报屏幕数据（方案 C）
        let screenData: ScreenRequestData? = nil

        let locationData = LocationRequestData(
            placeType: locationStatus.placeType.rawValue,
            atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
            city: nil
        )

        // 扩展位置数据（包含地点名称和坐标）
        let extendedLocationData = ExtendedLocationRequestData(
            placeType: locationStatus.placeType.rawValue,
            placeName: locationStatus.placeName,
            atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
            latitude: locationStatus.latitude,
            longitude: locationStatus.longitude
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

        // 日历数据（如果有权限）
        let calendarData: CalendarRequestData?
        if calendarStatus.todayRemainingCount >= 0 {
            calendarData = CalendarRequestData(
                hasCurrentEvent: calendarStatus.hasCurrentEvent,
                currentEventTitle: calendarStatus.currentEventTitle,
                eventEndMinutes: calendarStatus.eventEndMinutes,
                nextEventInMinutes: calendarStatus.nextEventInMinutes,
                todayRemainingCount: calendarStatus.todayRemainingCount
            )
        } else {
            calendarData = nil
        }

        // 运动数据（如果有权限）
        let movementData: MovementRequestData?
        if movementStatus.stepsToday >= 0 {
            movementData = MovementRequestData(
                isMoving: movementStatus.isMoving,
                movementType: movementStatus.movementType.rawValue,
                stepsToday: movementStatus.stepsToday,
                stepsLastHour: movementStatus.stepsLastHour,
                stationaryMinutes: movementStatus.stationaryMinutes
            )
        } else {
            movementData = nil
        }

        let request = StatusReportRequest(
            screen: screenData,
            location: locationData,
            extendedLocation: extendedLocationData,
            battery: batteryData,
            mode: modeData,
            connection: connectionData,
            display: displayData,
            calendar: calendarData,
            movement: movementData
        )

        do {
            let repository = AgentRepositoryImpl()
            let response = try await repository.reportStatus(request: request)
            print("[STATUS] ✓ Report success, nextReportIn: \(response.nextReportIn)s")
            if let analysis = response.analysis {
                print("[STATUS] Server analysis: \(analysis.availability.status) (\(analysis.availability.probability)%) - \(analysis.availability.reason)")
            }
        } catch {
            print("[STATUS] ✗ Report failed: \(error)")
        }
        print("=== [STATUS REPORT] Completed ===\n")
    }

    private func checkAppVersion() async {
        let currentVersion = Bundle.main.shortVersion
        do {
            let info: AppVersionInfo = try await APIClient.shared.request(
                .checkAppVersion(platform: "ios", currentVersion: currentVersion)
            )
            self.versionInfo = info

            if info.forceUpdate {
                showForceUpdateAlert = true
                return
            }

            // 有新版本且非强制更新
            if Bundle.compareVersions(currentVersion, info.latestVersion) == .orderedAscending {
                // 24h 节流
                let lastAlert = UserDefaults.standard.double(forKey: "lastUpdateAlertTime")
                let elapsed = Date().timeIntervalSince1970 - lastAlert
                if elapsed > 86400 {
                    showUpdateAlert = true
                }
            }
        } catch {
            print("[VERSION CHECK] Failed: \(error)")
        }
    }
}

#Preview {
    RootView()
        .environmentObject(AuthManager.shared)
}
