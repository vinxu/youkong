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

/// 屏幕使用时间数据收集器
/// 使用 FamilyControls + DeviceActivity 的阈值回调机制
class ScreenDataCollector: ObservableObject {
    static let shared = ScreenDataCollector()

    @Published private(set) var currentStatus: ScreenStatus = .idle
    @Published private(set) var isMonitoring = false
    @Published private(set) var isAuthorized = false

    private var sessionStartTime: Date?
    private var lastActiveTime: Date?
    private var screenOnMinutes: Int = 0  // 通过回调累计的屏幕使用时间

    // App Group 用于与 Extension 共享数据
    private let appGroupId = "group.com.youkong.app"
    private let screenTimeKey = "screenTimeMinutes"
    private let lastUpdateKey = "screenTimeLastUpdate"

    private init() {
        setupNotifications()
        loadSharedData()
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
    }

    // MARK: - Authorization

    /// 请求屏幕使用时间授权
    @MainActor
    func requestAuthorization() async -> Bool {
        #if canImport(FamilyControls)
        if #available(iOS 16.0, *) {
            do {
                try await AuthorizationCenter.shared.requestAuthorization(for: .individual)
                isAuthorized = true
                return true
            } catch {
                print("Screen Time authorization failed: \(error)")
                isAuthorized = false
                return false
            }
        }
        #endif

        // iOS 16 以下或模拟器，使用模拟模式
        isAuthorized = true
        return true
    }

    /// 检查授权状态
    @MainActor
    func checkAuthorization() {
        #if canImport(FamilyControls)
        if #available(iOS 16.0, *) {
            isAuthorized = AuthorizationCenter.shared.authorizationStatus == .approved
            return
        }
        #endif
        isAuthorized = true
    }

    // MARK: - Monitoring

    func startMonitoring() {
        isMonitoring = true

        #if canImport(FamilyControls)
        if #available(iOS 16.0, *) {
            startDeviceActivityMonitoring()
        }
        #endif

        // 同时监听 App 前后台状态作为补充
        if UIApplication.shared.applicationState == .active {
            sessionStartTime = Date()
            lastActiveTime = Date()
        }
        updateStatus()
    }

    func stopMonitoring() {
        isMonitoring = false

        #if canImport(FamilyControls)
        if #available(iOS 16.0, *) {
            stopDeviceActivityMonitoring()
        }
        #endif

        sessionStartTime = nil
    }

    func getCurrentStatus() -> ScreenStatus {
        loadSharedData()  // 从 App Group 加载最新数据
        updateStatus()
        return currentStatus
    }

    // MARK: - Device Activity Monitoring (iOS 16+)

    #if canImport(FamilyControls)
    @available(iOS 16.0, *)
    private func startDeviceActivityMonitoring() {
        let center = DeviceActivityCenter()

        // 设置每日监控计划
        let schedule = DeviceActivitySchedule(
            intervalStart: DateComponents(hour: 0, minute: 0),
            intervalEnd: DateComponents(hour: 23, minute: 59),
            repeats: true
        )

        // 设置多个阈值事件（5分钟、10分钟、15分钟...直到120分钟）
        // 每当用户屏幕使用时间达到阈值，Extension 会收到回调
        var events: [DeviceActivityEvent.Name: DeviceActivityEvent] = [:]

        for minutes in stride(from: 5, through: 120, by: 5) {
            let eventName = DeviceActivityEvent.Name("screenTime_\(minutes)min")
            events[eventName] = DeviceActivityEvent(
                threshold: DateComponents(minute: minutes)
            )
        }

        do {
            try center.startMonitoring(
                .daily,
                during: schedule,
                events: events
            )
            print("Device Activity monitoring started")
        } catch {
            print("Failed to start Device Activity monitoring: \(error)")
        }
    }

    @available(iOS 16.0, *)
    private func stopDeviceActivityMonitoring() {
        let center = DeviceActivityCenter()
        center.stopMonitoring([.daily])
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
    }

    /// 供 Extension 调用：记录屏幕使用时间达到了某个阈值
    /// 注意：Extension 只能写入 App Group，主 App 通过定时读取获得数据
    static func recordThresholdReached(minutes: Int) {
        guard let defaults = UserDefaults(suiteName: "group.com.youkong.app") else { return }

        defaults.set(minutes, forKey: "screenTimeMinutes")
        defaults.set(Date(), forKey: "screenTimeLastUpdate")
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

        // 根据屏幕使用时间推断活动类型
        let activityType: ActivityType
        if !isActive && lastActiveMinutesAgo > 5 {
            activityType = .idle
        } else if sessionDuration > 30 {
            activityType = .entertainment  // 长时间使用，可能在刷手机
        } else if sessionDuration > 10 {
            activityType = .communication  // 中等时间，可能在聊天
        } else {
            activityType = .idle
        }

        currentStatus = ScreenStatus(
            isActive: isActive || screenOnMinutes > 0,
            activityType: activityType,
            sessionDurationMinutes: sessionDuration,
            lastActiveMinutesAgo: isActive ? 0 : lastActiveMinutesAgo
        )
    }
}

// MARK: - Device Activity Name Extension

#if canImport(FamilyControls)
@available(iOS 16.0, *)
extension DeviceActivityName {
    static let daily = DeviceActivityName("daily")
}
#endif
