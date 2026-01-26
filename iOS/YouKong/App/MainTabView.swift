import SwiftUI

// MARK: - Main Tab View

/// 简化后的主视图 - 只有朋友列表和聊天
struct MainTabView: View {
    var body: some View {
        FriendsListView()
    }
}

#Preview {
    MainTabView()
        .environmentObject(AuthManager.shared)
}
