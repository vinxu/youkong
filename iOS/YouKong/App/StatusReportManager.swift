import Foundation
import Factory
import BackgroundTasks

@MainActor
class StatusReportManager: ObservableObject {
    static let shared = StatusReportManager()

    static let bgTaskIdentifier = "com.youkong.app.statusReport"

    @Published var isReporting = false
    @Published var lastReportTime: Date?
    @Published var lastReportError: String?

    @Injected(\.agentRepository) private var repository

    /// 最小上报间隔（秒），防止前后台快速切换时频繁上报
    private let minReportInterval: TimeInterval = 60

    /// 定时上报间隔（秒），与 Android 15 分钟对齐
    private let periodicInterval: TimeInterval = 15 * 60

    private var periodicTimer: Timer?

    private init() {}

    // MARK: - 定时上报（App 活跃期间）

    /// 启动 15 分钟定时上报（App 前台时运行）
    func startPeriodicReporting() {
        stopPeriodicReporting()
        periodicTimer = Timer.scheduledTimer(withTimeInterval: periodicInterval, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                await self?.reportIfNeeded()
            }
        }
        print("[STATUS REPORT] Periodic timer started (15min)")
    }

    /// 停止定时上报
    func stopPeriodicReporting() {
        periodicTimer?.invalidate()
        periodicTimer = nil
    }

    // MARK: - BGAppRefreshTask（App 后台时）

    /// 注册后台任务（在 AppDelegate didFinishLaunchingWithOptions 中调用）
    static func registerBackgroundTask() {
        BGTaskScheduler.shared.register(forTaskWithIdentifier: bgTaskIdentifier, using: nil) { task in
            guard let refreshTask = task as? BGAppRefreshTask else { return }
            Task { @MainActor in
                await StatusReportManager.shared.handleBackgroundRefresh(refreshTask)
            }
        }
        print("[STATUS REPORT] Background task registered")
    }

    /// 调度下一次后台刷新
    func scheduleBackgroundRefresh() {
        let request = BGAppRefreshTaskRequest(identifier: Self.bgTaskIdentifier)
        request.earliestBeginDate = Date(timeIntervalSinceNow: periodicInterval)
        do {
            try BGTaskScheduler.shared.submit(request)
            print("[STATUS REPORT] Background refresh scheduled in \(Int(periodicInterval))s")
        } catch {
            print("[STATUS REPORT] Failed to schedule background refresh: \(error)")
        }
    }

    /// 处理后台刷新任务
    private func handleBackgroundRefresh(_ task: BGAppRefreshTask) async {
        // 调度下一次
        scheduleBackgroundRefresh()

        task.expirationHandler = {
            task.setTaskCompleted(success: false)
        }

        await reportStatus()
        task.setTaskCompleted(success: true)
    }

    /// 场景切换触发的自动上报（带冷却）
    func reportIfNeeded() async {
        if let last = lastReportTime, Date().timeIntervalSince(last) < minReportInterval {
            print("[STATUS REPORT] Skipped (cooldown \(Int(minReportInterval - Date().timeIntervalSince(last)))s)")
            return
        }
        await reportStatus()
    }

    /// 手动上报状态
    func reportStatus() async {
        print("=== [STATUS REPORT] Starting ===")
        isReporting = true
        lastReportError = nil
        defer { isReporting = false }

        // 收集数据
        let locationStatus = LocationDataCollector.shared.getCurrentStatus()
        let deviceStatus = DeviceStatusCollector.shared.getCurrentStatus()
        let calendarStatus = CalendarDataCollector.shared.getCurrentStatus()
        let movementStatus = MovementDataCollector.shared.getCurrentStatus()

        // 打印日志
        print("[STATUS] Location: \(locationStatus.placeName ?? locationStatus.placeType.rawValue), since=\(locationStatus.atPlaceSinceMinutes)min")
        print("[STATUS] Calendar: hasEvent=\(calendarStatus.hasCurrentEvent), remaining=\(calendarStatus.todayRemainingCount)")
        print("[STATUS] Movement: moving=\(movementStatus.isMoving), steps=\(movementStatus.stepsToday), type=\(movementStatus.movementType.rawValue)")
        print("[STATUS] Battery: \(Int(deviceStatus.batteryLevel * 100))%, charging=\(deviceStatus.isCharging)")

        // 构建请求
        let screenData: ScreenRequestData? = nil  // 不上报屏幕数据

        let hasLocationPermission = LocationDataCollector.shared.isLocationAuthorized

        let locationData: LocationRequestData? = hasLocationPermission ?
            LocationRequestData(
                placeType: locationStatus.placeType.rawValue,
                atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
                city: locationStatus.city
            ) : nil

        let extendedLocationData: ExtendedLocationRequestData? = hasLocationPermission ?
            ExtendedLocationRequestData(
                placeType: locationStatus.placeType.rawValue,
                placeName: locationStatus.placeName,
                atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
                latitude: locationStatus.latitude,
                longitude: locationStatus.longitude
            ) : nil

        let batteryData = BatteryRequestData(
            batteryLevel: Int(deviceStatus.batteryLevel * 100),
            batteryState: deviceStatus.batteryState.rawValue,
            isCharging: deviceStatus.isCharging
        )

        let modeData = ModeRequestData(
            isLowPowerMode: deviceStatus.isLowPowerMode,
            isFocusModeOn: deviceStatus.isFocusModeOn
        )

        let connectionData = ConnectionRequestData(
            isHeadphonesConnected: deviceStatus.isHeadphonesConnected,
            networkType: deviceStatus.networkType.rawValue
        )

        let displayData = DisplayRequestData(
            screenBrightness: deviceStatus.screenBrightness
        )

        let calendarData: CalendarRequestData? = calendarStatus.todayRemainingCount >= 0 ?
            CalendarRequestData(
                hasCurrentEvent: calendarStatus.hasCurrentEvent,
                currentEventTitle: calendarStatus.currentEventTitle,
                eventEndMinutes: calendarStatus.eventEndMinutes,
                nextEventInMinutes: calendarStatus.nextEventInMinutes,
                todayRemainingCount: calendarStatus.todayRemainingCount
            ) : nil

        let movementData: MovementRequestData? = movementStatus.stepsToday >= 0 ?
            MovementRequestData(
                isMoving: movementStatus.isMoving,
                movementType: movementStatus.movementType.rawValue,
                stepsToday: movementStatus.stepsToday,
                stepsLastHour: movementStatus.stepsLastHour,
                stationaryMinutes: movementStatus.stationaryMinutes
            ) : nil

        let request = StatusReportRequest(
            screen: screenData,
            location: locationData,
            extendedLocation: extendedLocationData,
            battery: batteryData,
            mode: modeData,
            connection: connectionData,
            display: displayData,
            calendar: calendarData,
            movement: movementData
        )

        // 发送请求
        do {
            let response = try await repository.reportStatus(request: request)
            lastReportTime = Date()
            print("=== [STATUS REPORT] Success ===")
            print("[RESPONSE] \(response.message ?? "分析已触发")")
            if let analysis = response.analysis {
                print("[ANALYSIS] Available: \(analysis.availability.status)")
                print("[ANALYSIS] Probability: \(analysis.availability.probability)%")
                print("[ANALYSIS] Reason: \(analysis.availability.reason)")
                print("[LIFE STATUS] \(analysis.lifeStatus.emoji) \(analysis.lifeStatus.label)")
            }
        } catch {
            lastReportError = error.localizedDescription
            print("=== [STATUS REPORT] Error ===")
            print("[ERROR] \(error)")
        }
    }
}
