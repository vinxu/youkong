import SwiftUI

// MARK: - Root View

struct RootView: View {
    @EnvironmentObject private var authManager: AuthManager
    @EnvironmentObject private var deepLinkManager: DeepLinkManager
    @EnvironmentObject private var notificationManager: NotificationManager
    @StateObject private var permissionManager = PermissionManager.shared
    @AppStorage("hasCompletedOnboarding") private var hasCompletedOnboarding = false

    var body: some View {
        Group {
            if authManager.isAuthenticated {
                if hasCompletedOnboarding {
                    // 已完成引导，直接进入主页面
                    MainTabView()
                        .task {
                            // 检查权限状态（用于数据收集）
                            await permissionManager.checkAllPermissions()
                            // 启动数据收集
                            startDataCollection()
                            // 刷新未读消息 Badge
                            await notificationManager.refreshBadgeFromServer()
                        }
                } else if permissionManager.isChecking {
                    // 正在检查权限状态
                    ProgressView("检查权限...")
                } else if permissionManager.status.allGranted {
                    // 权限都已授权，直接进入主页面
                    MainTabView()
                        .task {
                            hasCompletedOnboarding = true
                            startDataCollection()
                        }
                } else {
                    // 显示权限请求页面
                    PermissionRequestView(isCompleted: $hasCompletedOnboarding)
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

        // 定时上报状态
        startStatusReporting()
    }

    private func startStatusReporting() {
        // 每30秒上报一次状态
        Timer.scheduledTimer(withTimeInterval: 30, repeats: true) { _ in
            Task {
                await reportStatus()
            }
        }
    }

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
            atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes
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
}

#Preview {
    RootView()
        .environmentObject(AuthManager.shared)
}
