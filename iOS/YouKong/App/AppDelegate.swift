import UIKit
import UserNotifications

class AppDelegate: NSObject, UIApplicationDelegate, UNUserNotificationCenterDelegate {

    func application(_ application: UIApplication,
                     didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
        // 设置通知中心代理
        UNUserNotificationCenter.current().delegate = self
        return true
    }

    // MARK: - APNs Token 注册回调

    func application(_ application: UIApplication,
                     didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
        let tokenString = deviceToken.map { String(format: "%02.2hhx", $0) }.joined()
        print("[APNs] Device token: \(tokenString)")

        Task { @MainActor in
            await NotificationManager.shared.handleDeviceTokenRegistration(tokenString)
        }
    }

    func application(_ application: UIApplication,
                     didFailToRegisterForRemoteNotificationsWithError error: Error) {
        print("[APNs] Failed to register: \(error)")
    }

    // MARK: - UNUserNotificationCenterDelegate

    /// 前台收到通知时调用
    func userNotificationCenter(_ center: UNUserNotificationCenter,
                                willPresent notification: UNNotification,
                                withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void) {
        let userInfo = notification.request.content.userInfo

        // 检查是否是当前聊天的消息，如果是则不显示通知
        if let conversationId = userInfo["conversationId"] as? String {
            Task { @MainActor in
                if NotificationManager.shared.currentConversationId == conversationId {
                    // 当前正在这个聊天页面，不显示通知
                    completionHandler([])
                    return
                }
                completionHandler([.banner, .sound, .badge])
            }
        } else {
            completionHandler([.banner, .sound, .badge])
        }
    }

    /// 点击通知时调用
    func userNotificationCenter(_ center: UNUserNotificationCenter,
                                didReceive response: UNNotificationResponse,
                                withCompletionHandler completionHandler: @escaping () -> Void) {
        let userInfo = response.notification.request.content.userInfo
        print("[Notification] Tapped with userInfo: \(userInfo)")

        Task { @MainActor in
            NotificationManager.shared.handleNotificationTap(userInfo: userInfo)
        }
        completionHandler()
    }
}
