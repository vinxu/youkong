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

    /// 是否应该跳转到首页（互动通知点击）
    @Published var shouldNavigateToHome = false

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
        print("🔐🔐🔐 [Notification] requestPermissionIfNeeded 被调用")
        print("🔐 已请求过权限: \(hasRequestedPermission)")

        guard !hasRequestedPermission else {
            print("🔐 ⏭️ 已经请求过，跳过")
            return
        }

        print("🔐 ⚠️ 正在请求通知权限...")

        do {
            let granted = try await UNUserNotificationCenter.current().requestAuthorization(
                options: [.alert, .sound, .badge]
            )
            isAuthorized = granted
            hasRequestedPermission = true
            userDefaults.set(true, forKey: hasRequestedPermissionKey)

            if granted {
                print("🔐 ✅ 用户授予了通知权限！")
                // 注册远程推送
                await registerForRemoteNotifications()
            } else {
                print("🔐 ❌ 用户拒绝了通知权限！")
            }

            print("🔐 最终权限状态: \(granted)")
        } catch {
            print("🔐 ❌ 权限请求错误: \(error)")
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

    /// 发送本地通知（收到 WebSocket 消息时）
    func scheduleLocalNotification(for message: Message, from sender: UserProfile, conversationId: String) {
        print("📢📢📢 [Notification] scheduleLocalNotification 被调用")
        print("📢 发送者: \(sender.nickname) (\(sender.id))")
        print("📢 当前会话ID: \(currentConversationId ?? "nil")")
        print("📢 消息会话ID: \(conversationId)")

        // ✅ 过滤自己发送的消息
        let currentUserId = AuthManager.shared.currentUser?.id ?? ""
        if sender.id == currentUserId {
            print("📢 ❌ 跳过：这是自己发送的消息")
            return
        }

        // 如果当前正在这个聊天页面，不发送通知
        if currentConversationId == conversationId {
            print("📢 ❌ 跳过：正在此聊天页面")
            return
        }

        print("📢 ✅ 开始发送通知流程...")

        // ✅ 移除 App 状态检查，让 AppDelegate 的 willPresent 来决定是否显示
        // 前台时会显示 banner 通知，后台时会显示系统通知

        Task {
            print("📢 [Task] 开始异步任务...")

            // 请求权限（如果未请求过）
            await requestPermissionIfNeeded()

            // 再次检查权限状态
            await checkAuthorizationStatus()

            print("📢 当前授权状态: \(isAuthorized)")

            guard isAuthorized else {
                print("📢 ❌ 未授权，无法发送通知")
                return
            }

            print("📢 ✅ 已授权，构建通知内容...")

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

            print("📢 通知标题: \(content.title)")
            print("📢 通知内容: \(content.body)")

            // 立即发送
            let request = UNNotificationRequest(
                identifier: "message_\(message.id)",
                content: content,
                trigger: nil
            )

            print("📢 正在添加通知到通知中心...")

            UNUserNotificationCenter.current().add(request) { error in
                if let error = error {
                    print("📢 ❌❌❌ 通知添加失败: \(error)")
                } else {
                    print("📢 ✅✅✅ 通知添加成功！ID: \(message.id)")
                }
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
        case .scheduleInvite:
            return "向你发起了日程邀约"
        }
    }

    // MARK: - Notification Tap Handling

    /// 处理通知点击
    func handleNotificationTap(userInfo: [AnyHashable: Any]) {
        let type = userInfo["type"] as? String

        if type == "interaction" {
            // 互动通知：跳转首页 + 清除 badge
            shouldNavigateToHome = true
            clearBadge()
            print("[Notification] Navigate to home (interaction tap)")
            return
        }

        // 消息通知：跳转聊天（原有逻辑）
        guard let conversationId = userInfo["conversationId"] as? String ??
                                   userInfo["conversation_id"] as? String else {
            return
        }

        pendingConversationId = conversationId
        shouldNavigateToChat = true
        clearBadge()

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
