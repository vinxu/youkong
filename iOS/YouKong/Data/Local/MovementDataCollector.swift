import Foundation
import CoreMotion
import Combine

// MARK: - Movement Status

/// 运动状态数据
struct MovementStatus: Codable, Equatable {
    let isMoving: Bool              // 是否在移动
    let movementType: MovementType  // 移动类型
    let stepsToday: Int             // 今日步数
    let stepsLastHour: Int          // 最近一小时步数
    let stationaryMinutes: Int      // 静止时长（分钟）

    enum CodingKeys: String, CodingKey {
        case isMoving = "is_moving"
        case movementType = "movement_type"
        case stepsToday = "steps_today"
        case stepsLastHour = "steps_last_hour"
        case stationaryMinutes = "stationary_minutes"
    }

    static let unknown = MovementStatus(
        isMoving: false,
        movementType: .unknown,
        stepsToday: 0,
        stepsLastHour: 0,
        stationaryMinutes: 0
    )

    static let noPermission = MovementStatus(
        isMoving: false,
        movementType: .unknown,
        stepsToday: -1,  // -1 表示无权限
        stepsLastHour: -1,
        stationaryMinutes: -1
    )
}

// MARK: - Movement Type

enum MovementType: String, Codable {
    case stationary     // 静止
    case walking        // 步行
    case running        // 跑步
    case cycling        // 骑行
    case driving        // 驾驶
    case unknown        // 未知

    var displayName: String {
        switch self {
        case .stationary: return "静止"
        case .walking: return "步行"
        case .running: return "跑步"
        case .cycling: return "骑行"
        case .driving: return "驾驶"
        case .unknown: return "未知"
        }
    }
}

// MARK: - Movement Data Collector

class MovementDataCollector: ObservableObject {
    static let shared = MovementDataCollector()

    @Published private(set) var currentStatus: MovementStatus = .unknown
    @Published private(set) var isAuthorized = false
    @Published private(set) var isMonitoring = false

    private let activityManager = CMMotionActivityManager()
    private let pedometer = CMPedometer()

    private var lastStationaryTime: Date?
    private var currentMovementType: MovementType = .unknown

    private init() {
        checkAuthorization()
    }

    // MARK: - Authorization

    /// 检查运动权限状态
    func checkAuthorization() {
        // 检查运动活动权限
        let activityAvailable = CMMotionActivityManager.isActivityAvailable()
        // 检查计步器权限
        let pedometerAvailable = CMPedometer.isStepCountingAvailable()

        // iOS 11+ 需要检查授权状态
        if #available(iOS 11.0, *) {
            let status = CMMotionActivityManager.authorizationStatus()
            isAuthorized = activityAvailable && pedometerAvailable && (status == .authorized)
        } else {
            isAuthorized = activityAvailable && pedometerAvailable
        }
    }

    /// 请求运动权限
    /// 注意：CoreMotion 没有显式的请求权限 API，权限会在首次访问数据时触发
    func requestAuthorization() async -> Bool {
        guard CMMotionActivityManager.isActivityAvailable() else {
            return false
        }

        // 通过查询活动来触发权限请求
        return await withCheckedContinuation { continuation in
            let now = Date()
            let oneHourAgo = now.addingTimeInterval(-3600)

            activityManager.queryActivityStarting(from: oneHourAgo, to: now, to: .main) { [weak self] _, error in
                if let error = error as NSError?, error.code == Int(CMErrorMotionActivityNotAuthorized.rawValue) {
                    self?.isAuthorized = false
                    continuation.resume(returning: false)
                } else {
                    self?.isAuthorized = true
                    continuation.resume(returning: true)
                }
            }
        }
    }

    // MARK: - Monitoring

    /// 开始监控运动状态
    func startMonitoring() {
        guard CMMotionActivityManager.isActivityAvailable() else {
            return
        }

        isMonitoring = true
        lastStationaryTime = Date()

        // 监控运动活动
        activityManager.startActivityUpdates(to: .main) { [weak self] activity in
            guard let self = self, let activity = activity else { return }

            let previousType = self.currentMovementType
            self.currentMovementType = self.mapActivityToMovementType(activity)

            // 如果从移动变为静止，记录静止开始时间
            if self.currentMovementType == .stationary && previousType != .stationary {
                self.lastStationaryTime = Date()
            }

            // 如果从静止变为移动，重置静止时间
            if self.currentMovementType != .stationary && previousType == .stationary {
                self.lastStationaryTime = nil
            }

            self.updateStatus()
        }

        // 初始状态更新
        updateStatus()
    }

    /// 停止监控
    func stopMonitoring() {
        isMonitoring = false
        activityManager.stopActivityUpdates()
    }

    // MARK: - Get Current Status

    /// 获取当前运动状态（同步版本，返回缓存的状态）
    func getCurrentStatus() -> MovementStatus {
        guard isAuthorized else {
            print("[Movement] Not authorized, returning noPermission")
            return .noPermission
        }

        // 如果没有在监控，尝试同步获取一次
        if !isMonitoring {
            print("[Movement] Not monitoring, returning cached status: \(currentStatus)")
        }

        return currentStatus
    }

    /// 获取当前运动状态（异步版本，实时获取）
    func fetchCurrentStatus() async -> MovementStatus {
        guard isAuthorized else {
            print("[Movement] Not authorized")
            return .noPermission
        }

        let stepsToday = await getTodaySteps()
        let stepsLastHour = await getLastHourSteps()

        let stationaryMinutes: Int
        if currentMovementType == .stationary, let startTime = lastStationaryTime {
            stationaryMinutes = Int(Date().timeIntervalSince(startTime) / 60)
        } else {
            stationaryMinutes = 0
        }

        let isMoving = currentMovementType != .stationary && currentMovementType != .unknown

        let status = MovementStatus(
            isMoving: isMoving,
            movementType: currentMovementType,
            stepsToday: stepsToday,
            stepsLastHour: stepsLastHour,
            stationaryMinutes: stationaryMinutes
        )

        await MainActor.run {
            self.currentStatus = status
        }

        print("[Movement] Fetched status: steps=\(stepsToday), moving=\(isMoving)")
        return status
    }

    // MARK: - Private Methods

    private func updateStatus() {
        Task {
            let stepsToday = await getTodaySteps()
            let stepsLastHour = await getLastHourSteps()

            let stationaryMinutes: Int
            if currentMovementType == .stationary, let startTime = lastStationaryTime {
                stationaryMinutes = Int(Date().timeIntervalSince(startTime) / 60)
            } else {
                stationaryMinutes = 0
            }

            let isMoving = currentMovementType != .stationary && currentMovementType != .unknown

            await MainActor.run {
                self.currentStatus = MovementStatus(
                    isMoving: isMoving,
                    movementType: self.currentMovementType,
                    stepsToday: stepsToday,
                    stepsLastHour: stepsLastHour,
                    stationaryMinutes: stationaryMinutes
                )
            }
        }
    }

    /// 获取今日步数
    private func getTodaySteps() async -> Int {
        guard CMPedometer.isStepCountingAvailable() else {
            return 0
        }

        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: Date())

        return await withCheckedContinuation { continuation in
            pedometer.queryPedometerData(from: startOfDay, to: Date()) { data, error in
                if let data = data {
                    continuation.resume(returning: data.numberOfSteps.intValue)
                } else {
                    continuation.resume(returning: 0)
                }
            }
        }
    }

    /// 获取最近一小时步数
    private func getLastHourSteps() async -> Int {
        guard CMPedometer.isStepCountingAvailable() else {
            return 0
        }

        let oneHourAgo = Date().addingTimeInterval(-3600)

        return await withCheckedContinuation { continuation in
            pedometer.queryPedometerData(from: oneHourAgo, to: Date()) { data, error in
                if let data = data {
                    continuation.resume(returning: data.numberOfSteps.intValue)
                } else {
                    continuation.resume(returning: 0)
                }
            }
        }
    }

    /// 将 CMMotionActivity 映射为 MovementType
    private func mapActivityToMovementType(_ activity: CMMotionActivity) -> MovementType {
        if activity.stationary {
            return .stationary
        } else if activity.running {
            return .running
        } else if activity.cycling {
            return .cycling
        } else if activity.automotive {
            return .driving
        } else if activity.walking {
            return .walking
        } else {
            return .unknown
        }
    }

    // MARK: - Request Data Format

    /// 转换为可上报的请求数据格式
    func toRequestData() -> MovementRequestData? {
        guard isAuthorized else { return nil }

        let status = getCurrentStatus()
        return MovementRequestData(
            isMoving: status.isMoving,
            movementType: status.movementType.rawValue,
            stepsToday: status.stepsToday,
            stepsLastHour: status.stepsLastHour,
            stationaryMinutes: status.stationaryMinutes
        )
    }
}
