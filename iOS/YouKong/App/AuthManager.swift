import Foundation
import Combine

@MainActor
final class AuthManager: ObservableObject {
    static let shared = AuthManager()

    @Published private(set) var currentUser: User?
    @Published private(set) var isAuthenticated = false
    @Published private(set) var isLoading = false

    private let keychainManager = KeychainManager.shared
    private let apiClient = APIClient.shared

    private init() {
        checkAuthState()
    }

    private func checkAuthState() {
        if let token = keychainManager.getAccessToken() {
            isAuthenticated = true
            Task {
                await fetchCurrentUser()
            }
        }
    }

    func login(with result: LoginResult) {
        keychainManager.saveAccessToken(result.token)
        currentUser = result.user
        isAuthenticated = true
    }

    func logout() {
        // 注销设备 Token
        Task {
            await NotificationManager.shared.unregisterDeviceToken()
        }

        keychainManager.clearTokens()
        currentUser = nil
        isAuthenticated = false
    }

    func fetchCurrentUser() async {
        isLoading = true
        defer { isLoading = false }

        do {
            let user: User = try await apiClient.request(.getMe)
            currentUser = user
        } catch {
            if case APIError.unauthorized = error {
                logout()
            }
        }
    }

    func updateUser(_ user: User) {
        currentUser = user
    }
}
