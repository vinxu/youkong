import Foundation
import UserNotifications
import UIKit

@MainActor
final class NotificationManager: ObservableObject {
    static let shared = NotificationManager()

    // MARK: - Published Properties

    /// 是否已请求过通知权限
    @Published private(set) var hasRequestedPermission = false

    /// 通知权限状态
    @Published private(set) var isAuthorized = false

    /// 待跳转的会话 ID（用于通知点击跳转）
    @Published var pendingConversationId: String?

    /// 是否应该跳转到聊天页面
    @Published var shouldNavigateToChat = false

    /// 当前正在查看的会话 ID（用于前台时不显示通知）
    var currentConversationId: String?

    // MARK: - Private Properties

    private let userDefaults = UserDefaults.standard
    private let hasRequestedPermissionKey = "hasRequestedNotificationPermission"
    private var deviceToken: String?

    // MARK: - Initialization

    private init() {
        hasRequestedPermission = userDefaults.bool(forKey: hasRequestedPermissionKey)
        Task {
            await checkAuthorizationStatus()
        }
    }

    // MARK: - Permission Management

    /// 检查通知权限状态
    func checkAuthorizationStatus() async {
        let settings = await UNUserNotificationCenter.current().notificationSettings()
        isAuthorized = settings.authorizationStatus == .authorized
    }

    /// 首次收到消息时请求通知权限
    func requestPermissionIfNeeded() async {
        guard !hasRequestedPermission else { return }

        do {
            let granted = try await UNUserNotificationCenter.current().requestAuthorization(
                options: [.alert, .sound, .badge]
            )
            isAuthorized = granted
            hasRequestedPermission = true
            userDefaults.set(true, forKey: hasRequestedPermissionKey)

            if granted {
                // 注册远程推送
                await registerForRemoteNotifications()
            }

            print("[Notification] Permission granted: \(granted)")
        } catch {
            print("[Notification] Permission request error: \(error)")
        }
    }

    /// 注册远程推送（APNs）
    func registerForRemoteNotifications() async {
        await MainActor.run {
            UIApplication.shared.registerForRemoteNotifications()
        }
    }

    // MARK: - Device Token Management

    /// 处理 APNs Token 注册成功
    func handleDeviceTokenRegistration(_ token: String) async {
        self.deviceToken = token

        // 上报 Token 到服务器
        await uploadDeviceToken(token)
    }

    /// 上报设备 Token 到服务器
    private func uploadDeviceToken(_ token: String) async {
        do {
            let _: EmptyResponse = try await APIClient.shared.request(
                .registerDeviceToken(token: token, platform: "ios")
            )
            print("[Notification] Device token uploaded successfully")
        } catch {
            print("[Notification] Failed to upload device token: \(error)")
        }
    }

    /// 注销设备 Token（退出登录时调用）
    func unregisterDeviceToken() async {
        guard let token = deviceToken else { return }

        do {
            let _: EmptyResponse = try await APIClient.shared.request(
                .unregisterDeviceToken(token: token)
            )
            print("[Notification] Device token unregistered successfully")
        } catch {
            print("[Notification] Failed to unregister device token: \(error)")
        }
    }

    // MARK: - Local Notification

    /// 发送本地通知（App 后台时收到 WebSocket 消息）
    func scheduleLocalNotification(for message: Message, from sender: UserProfile, conversationId: String) {
        // 如果当前正在这个聊天页面，不发送通知
        if currentConversationId == conversationId {
            return
        }

        // 检查 App 状态
        let state = UIApplication.shared.applicationState
        guard state == .background || state == .inactive else {
            // App 在前台，不需要本地通知（会通过 AppDelegate 的 willPresent 处理）
            return
        }

        // 请求权限（如果未请求过）
        Task {
            await requestPermissionIfNeeded()
        }

        // 构建通知内容
        let content = UNMutableNotificationContent()
        content.title = sender.nickname
        content.body = formatMessageContent(message)
        content.sound = .default
        content.userInfo = [
            "conversationId": conversationId,
            "messageId": message.id,
            "senderId": sender.id
        ]

        // 立即发送
        let request = UNNotificationRequest(
            identifier: "message_\(message.id)",
            content: content,
            trigger: nil
        )

        UNUserNotificationCenter.current().add(request) { error in
            if let error = error {
                print("[Notification] Failed to schedule: \(error)")
            } else {
                print("[Notification] Scheduled for message: \(message.id)")
            }
        }
    }

    /// 格式化消息内容为通知文本
    private func formatMessageContent(_ message: Message) -> String {
        switch message.type {
        case .text:
            return message.content ?? "发来一条消息"
        case .availabilityCard:
            return "分享了一个有空状态"
        case .confirmRequest:
            return "向你发起了确认请求"
        case .confirmResponse:
            return "回复了你的确认请求"
        }
    }

    // MARK: - Notification Tap Handling

    /// 处理通知点击
    func handleNotificationTap(userInfo: [AnyHashable: Any]) {
        guard let conversationId = userInfo["conversationId"] as? String else {
            return
        }

        pendingConversationId = conversationId
        shouldNavigateToChat = true

        print("[Notification] Navigate to conversation: \(conversationId)")
    }

    /// 清除待跳转状态
    func clearPendingNavigation() {
        pendingConversationId = nil
        shouldNavigateToChat = false
    }

    // MARK: - Badge Management

    /// 更新 Badge 数量
    func updateBadge(count: Int) {
        Task { @MainActor in
            if #available(iOS 16.0, *) {
                try? await UNUserNotificationCenter.current().setBadgeCount(count)
            } else {
                UIApplication.shared.applicationIconBadgeNumber = count
            }
        }
    }

    /// 从服务器刷新 Badge 数量
    func refreshBadgeFromServer() async {
        do {
            let response: BadgeCountResponse = try await APIClient.shared.request(.getBadgeCount)
            updateBadge(count: response.count)
        } catch {
            print("[Notification] Failed to refresh badge: \(error)")
        }
    }

    /// 清除 Badge
    func clearBadge() {
        updateBadge(count: 0)
    }
}

// MARK: - Response Types

struct BadgeCountResponse: Codable {
    let count: Int
}
