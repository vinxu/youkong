import SwiftUI

struct RootView: View {
    @EnvironmentObject private var authManager: AuthManager

    var body: some View {
        Group {
            if authManager.isAuthenticated {
                MainTabView()
            } else {
                LoginView()
            }
        }
        .animation(.easeInOut, value: authManager.isAuthenticated)
        #if DEBUG
        .withNetworkOverlay()
        .withDebugButton()
        .shakeToDebug()
        #endif
    }
}

#Preview {
    RootView()
        .environmentObject(AuthManager.shared)
}
