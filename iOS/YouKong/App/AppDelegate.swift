import UIKit
import UserNotifications

class AppDelegate: NSObject, UIApplicationDelegate, UNUserNotificationCenterDelegate {

    func application(_ application: UIApplication,
                     didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
        // 设置通知中心代理
        UNUserNotificationCenter.current().delegate = self

        // 监听 App 状态变化
        NotificationCenter.default.addObserver(
            forName: UIApplication.didEnterBackgroundNotification,
            object: nil,
            queue: .main
        ) { _ in
            print("🔄🔄🔄 App 进入后台")
        }

        NotificationCenter.default.addObserver(
            forName: UIApplication.willEnterForegroundNotification,
            object: nil,
            queue: .main
        ) { _ in
            print("🔄🔄🔄 App 即将进入前台")
        }

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
        print("🎯🎯🎯 [AppDelegate] willPresent 被调用！")
        print("🎯 标题: \(notification.request.content.title)")
        print("🎯 内容: \(notification.request.content.body)")

        // ✅ App 在前台时不显示横幅，只更新 badge
        // UI 提示由 App 内部的未读红点系统处理
        print("🎯 ✅ App 在前台，不显示横幅，只更新 badge")
        completionHandler([.badge])
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
