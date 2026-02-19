import UIKit
import UserNotifications
import CoreLocation

class AppDelegate: NSObject, UIApplicationDelegate, UNUserNotificationCenterDelegate {

    func application(_ application: UIApplication,
                     didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
        // 设置通知中心代理
        UNUserNotificationCenter.current().delegate = self

        // 注册后台状态上报任务
        StatusReportManager.registerBackgroundTask()

        // 检查是否因位置变化被唤醒
        if launchOptions?[.location] != nil {
            print("[AppDelegate] App launched by significant location change")
            LocationDataCollector.shared.resumeSignificantLocationMonitoringIfNeeded()
            Task {
                await StatusReportManager.shared.reportStatus()
            }
        }

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

        let userInfo = notification.request.content.userInfo
        let type = userInfo["type"] as? String

        if type == "interaction" {
            // 互动通知：前台也显示横幅和声音
            print("🎯 ✅ 互动通知，显示横幅+声音")
            completionHandler([.banner, .sound, .badge])
        } else {
            // 消息通知：前台只更新 badge（UI 内部红点处理）
            print("🎯 ✅ App 在前台，不显示横幅，只更新 badge")
            completionHandler([.badge])
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
