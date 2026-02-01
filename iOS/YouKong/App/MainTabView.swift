import SwiftUI

// MARK: - Main Tab View

/// 主视图 - 宫格首页
struct MainTabView: View {
    var body: some View {
        GridHomeView()
    }
}

#Preview {
    MainTabView()
        .environmentObject(AuthManager.shared)
}
