import SwiftUI

// MARK: - Root View

struct RootView: View {
    @EnvironmentObject private var authManager: AuthManager
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

        do {
            let repository = AgentRepositoryImpl()
            try await repository.reportStatus(screen: screenStatus, location: locationStatus)
        } catch {
            print("Failed to report status: \(error)")
        }
    }
}

#Preview {
    RootView()
        .environmentObject(AuthManager.shared)
}
