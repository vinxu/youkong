import Foundation
import SwiftUI

@MainActor
class StatusAnalysisViewModel: ObservableObject {
    @Published var outputLines: [OutputLine] = []
    @Published var isAnalyzing = false
    @Published var analysisCompleted = false
    @Published var analysisResult: AnalysisData?
    @Published var errorMessage: String?

    private let deviceCollector = DeviceStatusCollector.shared
    private let screenCollector = ScreenDataCollector.shared
    private let locationCollector = LocationDataCollector.shared
    private let calendarCollector = CalendarDataCollector.shared
    private let movementCollector = MovementDataCollector.shared
    private let agentRepository = AgentRepositoryImpl()

    // MARK: - Output Line

    struct OutputLine: Identifiable {
        let id = UUID()
        let text: String
        let type: LineType

        enum LineType {
            case title      // 标题
            case phase      // 阶段
            case clue       // 线索
            case thinking   // 推理过程
            case conclusion // 结论
            case result     // 最终结果
            case error      // 错误
            case normal     // 普通文本
        }
    }

    // MARK: - Start Analysis

    func startAnalysis() async {
        guard !isAnalyzing else { return }

        isAnalyzing = true
        analysisCompleted = false
        errorMessage = nil
        outputLines = []

        appendLine("🔍 有空 Agent - 状态分析", type: .title)
        appendLine("━━━━━━━━━━━━━━━━━━━━━━━━━━", type: .title)
        appendLine("", type: .normal)

        // 收集设备数据
        appendLine("📡 正在收集设备数据...", type: .phase)
        await collectDeviceData()

        // 构建请求
        let request = buildStatusRequest()

        // 调用 API
        appendLine("", type: .normal)
        appendLine("🤖 正在分析状态...", type: .phase)

        do {
            let response = try await agentRepository.reportStatus(request: request)

            if let analysis = response.analysis {
                displayAnalysisResult(analysis)
                analysisResult = analysis
                analysisCompleted = true
            } else {
                appendLine("⚠️ 未获取到分析结果", type: .error)
            }
        } catch {
            appendLine("❌ 分析失败: \(error.localizedDescription)", type: .error)
            errorMessage = error.localizedDescription
        }

        isAnalyzing = false
    }

    // MARK: - Collect Device Data

    private func collectDeviceData() async {
        let deviceStatus = deviceCollector.currentStatus
        let screenStatus = screenCollector.currentStatus
        let locationStatus = locationCollector.currentStatus

        // 屏幕状态
        let screenText = screenStatus.isActive ? "使用中" : "空闲"
        appendLine("├─ 屏幕: \(screenText)", type: .clue)

        // 位置信息
        let placeText = locationStatus.placeName ?? locationStatus.placeType.rawValue
        appendLine("├─ 位置: \(placeText)", type: .clue)

        // 电池状态
        let batteryPercent = Int(deviceStatus.batteryLevel * 100)
        let batteryText = deviceStatus.isCharging ? "\(batteryPercent)% (充电中)" : "\(batteryPercent)%"
        appendLine("├─ 电量: \(batteryText)", type: .clue)

        // 网络状态
        appendLine("├─ 网络: \(deviceStatus.networkType.rawValue)", type: .clue)

        // 日历
        if calendarCollector.isAuthorized {
            let calendarStatus = calendarCollector.currentStatus
            if calendarStatus.hasCurrentEvent {
                appendLine("├─ 日历: 有会议进行中", type: .clue)
            } else if let nextInMinutes = calendarStatus.nextEventInMinutes, nextInMinutes > 0 {
                appendLine("├─ 日历: \(nextInMinutes)分钟后有会议", type: .clue)
            }
        }

        // 运动
        if movementCollector.isAuthorized {
            let movementStatus = movementCollector.currentStatus
            if movementStatus.isMoving {
                appendLine("└─ 运动: \(movementStatus.movementType.displayName)", type: .clue)
            } else {
                appendLine("└─ 运动: 静止", type: .clue)
            }
        } else {
            appendLine("└─ 运动: 无权限", type: .clue)
        }

        // 小延迟模拟分析过程
        try? await Task.sleep(nanoseconds: 500_000_000) // 0.5秒
    }

    // MARK: - Build Request

    private func buildStatusRequest() -> StatusReportRequest {
        let deviceStatus = deviceCollector.currentStatus
        let screenStatus = screenCollector.currentStatus
        let locationStatus = locationCollector.currentStatus
        let calendarStatus = calendarCollector.hasPermission ? calendarCollector.currentStatus : nil
        let movementStatus = movementCollector.hasPermission ? movementCollector.currentStatus : nil

        return StatusReportRequest(
            screen: ScreenRequestData(
                isActive: screenStatus.isActive,
                activityType: screenStatus.activityType.rawValue,
                sessionDurationMinutes: screenStatus.sessionDurationMinutes,
                lastActiveMinutesAgo: screenStatus.lastActiveMinutesAgo,
                lastActiveCategory: nil
            ),
            location: LocationRequestData(
                placeType: locationStatus.placeType.rawValue,
                atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes
            ),
            extendedLocation: ExtendedLocationRequestData(
                placeType: locationStatus.placeType.rawValue,
                placeName: locationStatus.placeName,
                atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
                latitude: locationStatus.latitude,
                longitude: locationStatus.longitude
            ),
            battery: BatteryRequestData(
                batteryLevel: Int(deviceStatus.batteryLevel * 100),
                batteryState: deviceStatus.batteryState.rawValue,
                isCharging: deviceStatus.isCharging
            ),
            mode: ModeRequestData(
                isLowPowerMode: deviceStatus.isLowPowerMode,
                isFocusModeOn: deviceStatus.isFocusModeOn
            ),
            connection: ConnectionRequestData(
                isHeadphonesConnected: deviceStatus.isHeadphonesConnected,
                networkType: deviceStatus.networkType.rawValue
            ),
            display: DisplayRequestData(
                screenBrightness: deviceStatus.screenBrightness
            ),
            calendar: calendarCollector.isAuthorized ? CalendarRequestData(
                hasCurrentEvent: calendarCollector.currentStatus.hasCurrentEvent,
                currentEventTitle: calendarCollector.currentStatus.currentEventTitle,
                eventEndMinutes: calendarCollector.currentStatus.eventEndMinutes,
                nextEventInMinutes: calendarCollector.currentStatus.nextEventInMinutes,
                todayRemainingCount: calendarCollector.currentStatus.todayRemainingCount
            ) : nil,
            movement: movementCollector.isAuthorized ? MovementRequestData(
                isMoving: movementCollector.currentStatus.isMoving,
                movementType: movementCollector.currentStatus.movementType.rawValue,
                stepsToday: movementCollector.currentStatus.stepsToday,
                stepsLastHour: movementCollector.currentStatus.stepsLastHour,
                stationaryMinutes: movementCollector.currentStatus.stationaryMinutes
            ) : nil
        )
    }

    // MARK: - Display Result

    private func displayAnalysisResult(_ analysis: AnalysisData) {
        appendLine("", type: .normal)
        appendLine("✨ 分析结果", type: .phase)
        appendLine("━━━━━━━━━━━━━━━━━━━━━━━━━━", type: .phase)
        appendLine("", type: .normal)

        // 生活状态
        appendLine("\(analysis.lifeStatus.emoji) \(analysis.lifeStatus.label)", type: .result)
        if let description = analysis.lifeStatus.description {
            appendLine(description, type: .thinking)
        }

        appendLine("", type: .normal)

        // 有空概率
        let probability = analysis.availability.probability
        let status = analysis.availability.status
        appendLine("有空概率: \(probability)% (\(status))", type: .conclusion)
        appendLine("理由: \(analysis.availability.reason)", type: .thinking)
        appendLine("置信度: \(analysis.availability.confidence)", type: .thinking)

        appendLine("", type: .normal)
        appendLine("✅ 状态已更新", type: .result)
    }

    // MARK: - Helper

    private func appendLine(_ text: String, type: OutputLine.LineType) {
        outputLines.append(OutputLine(text: text, type: type))
    }
}

// MARK: - Movement Type Extension

extension MovementType {
    var displayName: String {
        switch self {
        case .stationary: return "静止"
        case .walking: return "步行中"
        case .running: return "跑步中"
        case .cycling: return "骑行中"
        case .automotive: return "驾驶中"
        case .unknown: return "未知"
        }
    }
}
