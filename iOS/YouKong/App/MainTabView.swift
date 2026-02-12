import SwiftUI

// MARK: - Main Tab View

/// 主视图 - 宫格首页
struct MainTabView: View {
    var body: some View {
        NavigationStack {
            GridHomeView()
        }
        .task {
            // 进入主页后立即上报一次状态
            print("📡 [MainTab] 触发开屏状态上报")
            await StatusReportManager.shared.reportIfNeeded()
        }
    }
}

#Preview {
    MainTabView()
        .environmentObject(AuthManager.shared)
}
