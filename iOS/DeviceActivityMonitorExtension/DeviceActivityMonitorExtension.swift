import DeviceActivity
import Foundation

/// Device Activity Monitor Extension
/// 接收屏幕使用时间阈值回调，并将数据写入 App Group 供主 App 读取
///
/// 注意：这个 Extension 需要在 Xcode 中手动创建：
/// 1. File -> New -> Target -> Device Activity Monitor Extension
/// 2. 添加 App Group capability: group.com.youkong.app
/// 3. 将这个文件拖入新创建的 Extension target

@available(iOS 16.0, *)
class DeviceActivityMonitorExtension: DeviceActivityMonitor {

    private let appGroupId = "group.com.youkong.app"
    private let screenTimeKey = "screenTimeMinutes"
    private let lastUpdateKey = "screenTimeLastUpdate"

    // MARK: - Interval Callbacks

    override func intervalDidStart(for activity: DeviceActivityName) {
        super.intervalDidStart(for: activity)
        // 新的一天开始，重置计数
        resetScreenTime()
    }

    override func intervalDidEnd(for activity: DeviceActivityName) {
        super.intervalDidEnd(for: activity)
        // 一天结束
    }

    // MARK: - Event Callbacks (阈值回调)

    override func eventDidReachThreshold(_ event: DeviceActivityEvent.Name, activity: DeviceActivityName) {
        super.eventDidReachThreshold(event, activity: activity)

        // 从事件名称解析出分钟数
        // 事件名称格式: "screenTime_XXmin"
        let eventString = event.rawValue
        if eventString.hasPrefix("screenTime_"),
           let minuteString = eventString.components(separatedBy: "_").last?.replacingOccurrences(of: "min", with: ""),
           let minutes = Int(minuteString) {

            // 写入 App Group
            saveScreenTime(minutes: minutes)
        }
    }

    override func eventWillReachThresholdWarning(_ event: DeviceActivityEvent.Name, activity: DeviceActivityName) {
        super.eventWillReachThresholdWarning(event, activity: activity)
        // 即将达到阈值的警告（可选实现）
    }

    // MARK: - App Group Data

    private func saveScreenTime(minutes: Int) {
        guard let defaults = UserDefaults(suiteName: appGroupId) else { return }

        defaults.set(minutes, forKey: screenTimeKey)
        defaults.set(Date(), forKey: lastUpdateKey)
        defaults.synchronize()

        // 注意：Extension 中不能发网络请求
        // 数据会在主 App 下次启动或前台时读取
    }

    private func resetScreenTime() {
        guard let defaults = UserDefaults(suiteName: appGroupId) else { return }

        defaults.set(0, forKey: screenTimeKey)
        defaults.set(Date(), forKey: lastUpdateKey)
        defaults.synchronize()
    }
}
