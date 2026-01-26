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
    private let lastCategoryKey = "lastActiveCategory"
    private let lastCategoryTimeKey = "lastActiveCategoryTime"

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

        let eventString = event.rawValue

        // 格式1: "screenTime_XXmin" - 总屏幕时间
        if eventString.hasPrefix("screenTime_"),
           let minuteString = eventString.components(separatedBy: "_").last?.replacingOccurrences(of: "min", with: ""),
           let minutes = Int(minuteString) {
            saveScreenTime(minutes: minutes)
        }

        // 格式2: "category_XXX_Xmin" - 分类使用时间
        // 例如: "category_games_1min", "category_socialNetworking_5min"
        if eventString.hasPrefix("category_") {
            let parts = eventString.components(separatedBy: "_")
            if parts.count >= 3 {
                let category = parts[1]
                saveCategoryActivity(category: category)

                // 同时更新总屏幕时间
                if let minuteString = parts.last?.replacingOccurrences(of: "min", with: ""),
                   let minutes = Int(minuteString) {
                    saveScreenTime(minutes: minutes)
                }
            }
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

    private func saveCategoryActivity(category: String) {
        guard let defaults = UserDefaults(suiteName: appGroupId) else { return }

        // 转换分类名称为中文显示
        let displayName = categoryDisplayName(category)

        defaults.set(displayName, forKey: lastCategoryKey)
        defaults.set(Date(), forKey: lastCategoryTimeKey)
        defaults.synchronize()
    }

    private func categoryDisplayName(_ category: String) -> String {
        switch category {
        case "games": return "游戏"
        case "socialNetworking": return "社交"
        case "entertainment": return "娱乐"
        case "productivity": return "效率"
        case "education": return "教育"
        case "shopping": return "购物"
        case "healthAndFitness": return "健康"
        case "newsAndMagazines": return "新闻"
        case "travel": return "旅行"
        case "finance": return "财务"
        case "photoAndVideo": return "照片和视频"
        case "music": return "音乐"
        case "books": return "阅读"
        case "utilities": return "工具"
        case "weather": return "天气"
        case "navigation": return "导航"
        default: return category
        }
    }

    private func resetScreenTime() {
        guard let defaults = UserDefaults(suiteName: appGroupId) else { return }

        defaults.set(0, forKey: screenTimeKey)
        defaults.set(Date(), forKey: lastUpdateKey)
        defaults.synchronize()
    }
}
