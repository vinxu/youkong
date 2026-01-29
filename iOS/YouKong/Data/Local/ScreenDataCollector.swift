import Foundation
import Combine
import UIKit

// 条件导入 FamilyControls（iOS 16+）
#if canImport(FamilyControls)
import FamilyControls
import DeviceActivity
import ManagedSettings
#endif

// MARK: - Screen Data Collector

/// ⚠️ 屏幕使用时间数据收集器（已禁用 - 方案 C）
/// 保留接口但不进行任何实际数据收集，始终返回空数据
class ScreenDataCollector: ObservableObject {
    static let shared = ScreenDataCollector()

    @Published private(set) var currentStatus: ScreenStatus = .idle
    @Published private(set) var isMonitoring = false
    @Published private(set) var isAuthorized = false

    #if canImport(FamilyControls)
    @available(iOS 16.0, *)
    @Published var activitySelection = FamilyActivitySelection()
    #endif

    private var sessionStartTime: Date?
    private var lastActiveTime: Date?
    private var screenOnMinutes: Int = 0  // 通过回调累计的屏幕使用时间

    // App Group 用于与 Extension 共享数据
    private let appGroupId = "group.com.youkong.app"
    private let screenTimeKey = "screenTimeMinutes"
    private let lastUpdateKey = "screenTimeLastUpdate"
    private let selectionKey = "familyActivitySelection"
    private let lastCategoryKey = "lastActiveCategory"
    private let lastCategoryTimeKey = "lastActiveCategoryTime"

    // 最近活跃的应用分类
    @Published private(set) var lastActiveCategory: String = ""
    @Published private(set) var lastActiveCategoryTime: Date?

    private init() {
        setupNotifications()
        loadSharedData()
        loadSelection()
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
    }

    // MARK: - Authorization (已禁用)

    /// ⚠️ 始终返回 false（方案 C）
    @MainActor
    func requestAuthorization() async -> Bool {
        isAuthorized = false
        return false
    }

    /// ⚠️ 始终返回未授权（方案 C）
    @MainActor
    func checkAuthorization() {
        isAuthorized = false
    }

    // MARK: - Selection Management

    #if canImport(FamilyControls)
    @available(iOS 16.0, *)
    func saveSelection() {
        guard let defaults = UserDefaults(suiteName: appGroupId) else { return }

        do {
            let data = try JSONEncoder().encode(activitySelection)
            defaults.set(data, forKey: selectionKey)
            defaults.synchronize()
            print("Activity selection saved")

            // 重新启动监控以应用新选择
            if isMonitoring {
                stopMonitoring()
                startMonitoring()
            }
        } catch {
            print("Failed to save activity selection: \(error)")
        }
    }

    @available(iOS 16.0, *)
    var hasSelection: Bool {
        !activitySelection.applicationTokens.isEmpty || !activitySelection.categoryTokens.isEmpty
    }
    #endif

    private func loadSelection() {
        #if canImport(FamilyControls)
        if #available(iOS 16.0, *) {
            loadSelectionImpl()
        }
        #endif
    }

    #if canImport(FamilyControls)
    @available(iOS 16.0, *)
    private func loadSelectionImpl() {
        guard let defaults = UserDefaults(suiteName: appGroupId),
              let data = defaults.data(forKey: selectionKey) else { return }

        do {
            activitySelection = try JSONDecoder().decode(FamilyActivitySelection.self, from: data)
            print("Activity selection loaded: \(activitySelection.applicationTokens.count) apps, \(activitySelection.categoryTokens.count) categories")
        } catch {
            print("Failed to load activity selection: \(error)")
        }
    }
    #endif

    // MARK: - Monitoring (已禁用)

    /// ⚠️ 空实现（方案 C）
    func startMonitoring() {
        isMonitoring = false
        print("⚠️ [ScreenData] 屏幕数据收集已禁用（方案 C）")
    }

    /// ⚠️ 空实现（方案 C）
    func stopMonitoring() {
        isMonitoring = false
    }

    /// ⚠️ 始终返回空闲状态（方案 C）
    func getCurrentStatus() -> ScreenStatus {
        return .idle
    }

    // MARK: - Device Activity Monitoring (iOS 16+)

    #if canImport(FamilyControls)
    @available(iOS 16.0, *)
    private func startDeviceActivityMonitoring() {
        let center = DeviceActivityCenter()

        // 先停止之前的监控
        center.stopMonitoring()

        // 设置每日监控计划
        let schedule = DeviceActivitySchedule(
            intervalStart: DateComponents(hour: 0, minute: 0),
            intervalEnd: DateComponents(hour: 23, minute: 59),
            repeats: true
        )

        var events: [DeviceActivityEvent.Name: DeviceActivityEvent] = [:]

        // 方案1: 如果用户选择了应用/分类，同时监控用户选择 + 预定义分类
        if hasSelection {
            // 监控用户选择的应用和分类（用于总屏幕时间）
            for minutes in [1, 2, 3, 5, 10, 15, 20, 30, 45, 60, 90, 120] {
                let eventName = DeviceActivityEvent.Name("screenTime_\(minutes)min")
                events[eventName] = DeviceActivityEvent(
                    applications: activitySelection.applicationTokens,
                    categories: activitySelection.categoryTokens,
                    threshold: DateComponents(minute: minutes)
                )
            }
            print("Monitoring user selection: \(activitySelection.applicationTokens.count) apps, \(activitySelection.categoryTokens.count) categories")
        }

        // 注意：预定义分类监控需要用户通过 FamilyActivityPicker 选择
        // 选择后，分类信息会包含在 activitySelection.categoryTokens 中
        if !activitySelection.categoryTokens.isEmpty {
            print("Monitoring \(activitySelection.categoryTokens.count) selected categories")
        }

        do {
            try center.startMonitoring(
                .daily,
                during: schedule,
                events: events
            )
            print("Device Activity monitoring started with \(events.count) events")
        } catch {
            print("Failed to start Device Activity monitoring: \(error)")
        }
    }

    @available(iOS 16.0, *)
    private func stopDeviceActivityMonitoring() {
        let center = DeviceActivityCenter()
        center.stopMonitoring()
    }
    #endif

    // MARK: - Shared Data (App Group)

    /// 从 App Group 加载共享数据（由 Extension 写入）
    private func loadSharedData() {
        guard let defaults = UserDefaults(suiteName: appGroupId) else { return }

        let minutes = defaults.integer(forKey: screenTimeKey)
        let lastUpdate = defaults.object(forKey: lastUpdateKey) as? Date

        // 检查数据是否是今天的
        if let lastUpdate = lastUpdate, Calendar.current.isDateInToday(lastUpdate) {
            screenOnMinutes = minutes
        } else {
            screenOnMinutes = 0
        }

        // 加载最近活跃的应用分类
        lastActiveCategory = defaults.string(forKey: lastCategoryKey) ?? ""
        lastActiveCategoryTime = defaults.object(forKey: lastCategoryTimeKey) as? Date
    }

    /// 供 Extension 调用：记录屏幕使用时间达到了某个阈值
    /// 注意：Extension 只能写入 App Group，主 App 通过定时读取获得数据
    static func recordThresholdReached(minutes: Int) {
        guard let defaults = UserDefaults(suiteName: "group.com.youkong.app") else { return }

        defaults.set(minutes, forKey: "screenTimeMinutes")
        defaults.set(Date(), forKey: "screenTimeLastUpdate")
        defaults.synchronize()
    }

    /// 供 Extension 调用：记录触发阈值的应用分类
    static func recordCategoryActivity(category: String) {
        guard let defaults = UserDefaults(suiteName: "group.com.youkong.app") else { return }

        defaults.set(category, forKey: "lastActiveCategory")
        defaults.set(Date(), forKey: "lastActiveCategoryTime")
        defaults.synchronize()
    }

    // MARK: - App Lifecycle

    private func setupNotifications() {
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(appDidBecomeActive),
            name: UIApplication.didBecomeActiveNotification,
            object: nil
        )

        NotificationCenter.default.addObserver(
            self,
            selector: #selector(appWillResignActive),
            name: UIApplication.willResignActiveNotification,
            object: nil
        )

        NotificationCenter.default.addObserver(
            self,
            selector: #selector(appDidEnterBackground),
            name: UIApplication.didEnterBackgroundNotification,
            object: nil
        )
    }

    @objc private func appDidBecomeActive() {
        sessionStartTime = Date()
        lastActiveTime = Date()
        loadSharedData()
        updateStatus()
    }

    @objc private func appWillResignActive() {
        lastActiveTime = Date()
        updateStatus()
    }

    @objc private func appDidEnterBackground() {
        updateStatus()
    }

    // MARK: - Update Status

    private func updateStatus() {
        let isActive = UIApplication.shared.applicationState == .active

        // 优先使用从 Extension 获取的屏幕使用时间
        let sessionDuration: Int
        if screenOnMinutes > 0 {
            sessionDuration = screenOnMinutes
        } else if let startTime = sessionStartTime, isActive {
            sessionDuration = Int(Date().timeIntervalSince(startTime) / 60)
        } else {
            sessionDuration = 0
        }

        let lastActiveMinutesAgo: Int
        if let lastTime = lastActiveTime {
            lastActiveMinutesAgo = Int(Date().timeIntervalSince(lastTime) / 60)
        } else {
            lastActiveMinutesAgo = 0
        }

        // 根据应用分类推断活动类型
        let activityType: ActivityType
        if !isActive && lastActiveMinutesAgo > 5 {
            activityType = .idle
        } else if !lastActiveCategory.isEmpty {
            // 基于应用分类判断活动类型
            activityType = inferActivityType(from: lastActiveCategory)
        } else if sessionDuration > 30 {
            activityType = .entertainment  // 长时间使用，可能在刷手机
        } else if sessionDuration > 10 {
            activityType = .communication  // 中等时间，可能在聊天
        } else {
            activityType = .idle
        }

        // 获取最近的分类信息（如果5分钟内有效）
        let validCategory: String?
        if let categoryTime = lastActiveCategoryTime,
           Date().timeIntervalSince(categoryTime) < 300 {  // 5分钟内有效
            validCategory = lastActiveCategory.isEmpty ? nil : lastActiveCategory
        } else {
            validCategory = nil
        }

        currentStatus = ScreenStatus(
            isActive: isActive || screenOnMinutes > 0,
            activityType: activityType,
            sessionDurationMinutes: sessionDuration,
            lastActiveMinutesAgo: isActive ? 0 : lastActiveMinutesAgo,
            lastActiveCategory: validCategory
        )
    }

    /// 根据应用分类推断活动类型
    private func inferActivityType(from category: String) -> ActivityType {
        switch category {
        case "游戏", "娱乐", "照片和视频", "音乐":
            return .entertainment
        case "社交", "通讯":
            return .communication
        case "效率", "教育", "工具", "财务":
            return .productivity
        default:
            return .idle
        }
    }
}

// MARK: - Device Activity Name Extension

#if canImport(FamilyControls)
@available(iOS 16.0, *)
extension DeviceActivityName {
    static let daily = DeviceActivityName("daily")
}
#endif
