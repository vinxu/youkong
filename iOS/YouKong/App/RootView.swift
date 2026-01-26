import SwiftUI

// MARK: - Root View

struct RootView: View {
    @EnvironmentObject private var authManager: AuthManager
    @EnvironmentObject private var deepLinkManager: DeepLinkManager
    @StateObject private var permissionManager = PermissionManager.shared
    @AppStorage("hasCompletedOnboarding") private var hasCompletedOnboarding = false

    var body: some View {
        Group {
            if authManager.isAuthenticated {
                if hasCompletedOnboarding || permissionManager.status.allGranted {
                    MainTabView()
                        .task {
                            // 启动数据收集
                            startDataCollection()
                        }
                } else {
                    PermissionRequestView(isCompleted: $hasCompletedOnboarding)
                }
            } else {
                LoginView()
            }
        }
        .animation(.easeInOut, value: authManager.isAuthenticated)
        .animation(.easeInOut, value: hasCompletedOnboarding)
        .sheet(isPresented: $deepLinkManager.showInvitationSheet) {
            if let code = deepLinkManager.pendingInvitationCode {
                AcceptInvitationView(code: code) {
                    deepLinkManager.clearPendingInvitation()
                }
            }
        }
        #if DEBUG
        .withNetworkOverlay()
        .withDebugButton()
        .shakeToDebug()
        #endif
    }

    private func startDataCollection() {
        // 启动位置数据收集
        if permissionManager.status.location {
            LocationDataCollector.shared.startMonitoring()
        }

        // 启动屏幕使用数据收集
        if permissionManager.status.screenTime {
            ScreenDataCollector.shared.startMonitoring()
        }

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
        let screenStatus = ScreenDataCollector.shared.getCurrentStatus()
        let locationStatus = LocationDataCollector.shared.getCurrentStatus()
        let deviceStatus = DeviceStatusCollector.shared.getCurrentStatus()

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
            let repository = AgentRepositoryImpl()
            _ = try await repository.reportStatus(request: request)
        } catch {
            print("Failed to report status: \(error)")
        }
    }
}

#Preview {
    RootView()
        .environmentObject(AuthManager.shared)
}
