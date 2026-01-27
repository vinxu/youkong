import SwiftUI

@main
struct YouKongApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
    @StateObject private var authManager = AuthManager.shared
    @StateObject private var deepLinkManager = DeepLinkManager.shared
    @StateObject private var notificationManager = NotificationManager.shared

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(authManager)
                .environmentObject(deepLinkManager)
                .environmentObject(notificationManager)
                .onOpenURL { url in
                    deepLinkManager.handleURL(url)
                }
        }
    }
}

// MARK: - Deep Link Manager

@MainActor
class DeepLinkManager: ObservableObject {
    static let shared = DeepLinkManager()

    @Published var pendingInvitationCode: String?
    @Published var showInvitationSheet = false

    private init() {}

    func handleURL(_ url: URL) {
        // 支持的 URL 格式:
        // 1. youkong://invite/ABC123
        // 2. http://49.232.13.41:8080/i/ABC123
        // 3. https://youkong.app/i/ABC123

        if url.scheme == "youkong" {
            // Custom URL Scheme
            if url.host == "invite", let code = url.pathComponents.last, !code.isEmpty {
                handleInvitationCode(code)
            }
        } else if url.host == "49.232.13.41" || url.host == "youkong.app" {
            // Universal Link
            let path = url.path
            if path.hasPrefix("/i/") {
                let code = String(path.dropFirst(3))
                if !code.isEmpty {
                    handleInvitationCode(code)
                }
            }
        }
    }

    private func handleInvitationCode(_ code: String) {
        pendingInvitationCode = code
        showInvitationSheet = true
    }

    func clearPendingInvitation() {
        pendingInvitationCode = nil
        showInvitationSheet = false
    }
}
