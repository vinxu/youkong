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
    private let locationCollector = LocationDataCollector.shared
    private let calendarCollector = CalendarDataCollector.shared
    private let movementCollector = MovementDataCollector.shared
    private let sseClient = SSEClient()
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

        appendLine("🔍 福尔摩斯 Agent - 状态推理", type: .title)
        appendLine("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", type: .title)
        appendLine("", type: .normal)

        // 收集设备数据
        appendLine("📡 阶段一：数据收集", type: .phase)
        appendLine("", type: .normal)
        await collectDeviceData()

        // 构建请求
        let request = buildStatusRequest()

        // 调用流式 API
        appendLine("", type: .normal)
        appendLine("🧠 阶段二：Holmes 推理", type: .phase)
        appendLine("", type: .normal)

        do {
            try await sseClient.streamHolmesAnalysis(request: request) { [weak self] event in
                self?.handleSSEEvent(event)
            }
        } catch SSEError.noEventsReceived {
            // SSE 失败，使用同步 API 作为 fallback
            appendLine("⚠️ 流式推理不可用，使用标准分析...", type: .thinking)
            await fallbackToSyncAnalysis(request: request)
        } catch {
            appendLine("❌ 推理失败: \(error.localizedDescription)", type: .error)
            errorMessage = error.localizedDescription
        }

        isAnalyzing = false
    }

    // MARK: - Handle SSE Event

    private func handleSSEEvent(_ event: HolmesStreamEvent) {
        switch event.type {
        case .phase:
            if let phase = event.phase {
                appendLine("", type: .normal)
                appendLine("📌 \(phase)", type: .phase)
            }

        case .clue:
            if let content = event.content {
                appendLine("  🔹 \(content)", type: .clue)
            }

        case .feature:
            if let content = event.content {
                appendLine("  ✓ \(content)", type: .clue)
            }

        case .thinking:
            if let content = event.content {
                appendLine("  💭 \(content)", type: .thinking)
            }

        case .conclusion:
            if let content = event.content {
                appendLine("", type: .normal)
                appendLine("📋 \(content)", type: .conclusion)
            }

        case .done:
            displayFinalResult(event.result)
            analysisCompleted = true

        case .error:
            if let content = event.content {
                appendLine("❌ \(content)", type: .error)
                errorMessage = content
            }

        // Holmes 2.0 新增事件类型
        case .context:
            if let content = event.content {
                appendLine("  🎯 \(content)", type: .clue)
            }

        case .anomaly:
            if let content = event.content {
                appendLine("  ⚠️ \(content)", type: .thinking)
            }

        case .narrative:
            if let content = event.content {
                appendLine("  📝 \(content)", type: .thinking)
            }

        case .none:
            // 可能是最终结果
            if event.isFinalResult {
                displayFinalResult(event.result)
                analysisCompleted = true
            }
        }
    }

    // MARK: - Display Final Result

    private func displayFinalResult(_ fullResult: SSEFullResult?) {
        guard let result = fullResult else { return }

        appendLine("", type: .normal)
        appendLine("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", type: .title)
        appendLine("✨ 推理结论", type: .phase)
        appendLine("", type: .normal)

        // Holmes 2.0: 优先显示创意叙事结果
        if let creative = result.creative {
            // 显示叙事推理过程
            if let narrative = creative.narrative, !narrative.isEmpty {
                appendLine("【叙事推理】", type: .phase)
                for line in narrative.components(separatedBy: "\n") where !line.isEmpty {
                    appendLine("  \(line)", type: .thinking)
                }
                appendLine("", type: .normal)
            }

            // 显示场景描述
            if let scene = creative.scene, !scene.isEmpty {
                let emoji = creative.emoji ?? "🤔"
                appendLine("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", type: .title)
                appendLine("", type: .normal)
                appendLine("\(emoji) \(scene)", type: .result)
                appendLine("", type: .normal)
            }

            // 显示心情维度
            if let mood = creative.mood {
                let moodDesc = describeMood(mood)
                appendLine("心情: \(moodDesc)", type: .conclusion)
            }

            // 显示推断依据
            if let basis = creative.basis, !basis.isEmpty {
                appendLine("依据: \(basis.joined(separator: "、"))", type: .thinking)
            }

            // 显示置信度
            if let confidence = creative.confidence {
                appendLine("置信度: \(translateConfidence(confidence))", type: .thinking)
            }
        } else {
            // 兼容旧版 Holmes 1.0 格式
            // 显示推理过程
            if let reasoning = result.reasoning {
                if let thinking = reasoning.thinking, !thinking.isEmpty {
                    appendLine("【推理过程】", type: .phase)
                    for line in thinking.components(separatedBy: "\n") where !line.isEmpty {
                        appendLine("  \(line)", type: .thinking)
                    }
                    appendLine("", type: .normal)
                }

                if let conclusion = reasoning.conclusion, !conclusion.isEmpty {
                    appendLine("【结论】", type: .phase)
                    appendLine("  \(conclusion)", type: .conclusion)
                    appendLine("", type: .normal)
                }
            }

            // 显示最终状态结果
            if let finalResult = result.result {
                appendLine("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", type: .title)
                appendLine("", type: .normal)

                // 大号 emoji + 状态
                let emoji = finalResult.emoji ?? "🤔"
                let summary = finalResult.summary ?? "分析中..."
                appendLine("\(emoji) \(summary)", type: .result)
                appendLine("", type: .normal)

                // 原因
                if let reason = finalResult.reason, !reason.isEmpty {
                    appendLine("理由: \(reason)", type: .thinking)
                }

                // 置信度
                if let confidence = finalResult.confidence {
                    appendLine("置信度: \(translateConfidence(confidence))", type: .thinking)
                }
            }
        }

        appendLine("", type: .normal)
        appendLine("✅ 状态已更新", type: .result)
    }

    // MARK: - Mood Description

    private func describeMood(_ mood: SSEMoodVector) -> String {
        // 效价描述
        let valenceDesc: String
        if let valence = mood.valence {
            if valence > 0.5 {
                valenceDesc = "积极"
            } else if valence > 0 {
                valenceDesc = "偏积极"
            } else if valence > -0.5 {
                valenceDesc = "偏消极"
            } else {
                valenceDesc = "消极"
            }
        } else {
            valenceDesc = "未知"
        }

        // 唤醒度描述
        let arousalDesc: String
        if let arousal = mood.arousal {
            if arousal > 0.7 {
                arousalDesc = "激动"
            } else if arousal > 0.4 {
                arousalDesc = "平稳"
            } else {
                arousalDesc = "平静"
            }
        } else {
            arousalDesc = "未知"
        }

        // 社交开放度描述
        let opennessDesc: String
        if let openness = mood.openness {
            if openness > 0.7 {
                opennessDesc = "开放"
            } else if openness > 0.4 {
                opennessDesc = "中性"
            } else {
                opennessDesc = "封闭"
            }
        } else {
            opennessDesc = "未知"
        }

        return "\(valenceDesc)·\(arousalDesc)·\(opennessDesc)"
    }

    private func translateConfidence(_ confidence: String) -> String {
        switch confidence {
        case "high":
            return "高"
        case "medium":
            return "中"
        case "low":
            return "低"
        default:
            return confidence
        }
    }

    // MARK: - Fallback to Sync Analysis

    private func fallbackToSyncAnalysis(request: StatusReportRequest) async {
        do {
            let response = try await agentRepository.reportStatus(request: request)

            if let analysis = response.analysis {
                displaySyncAnalysisResult(analysis)
                analysisResult = analysis
                analysisCompleted = true
            } else {
                appendLine("⚠️ 未获取到分析结果", type: .error)
            }
        } catch {
            appendLine("❌ 分析失败: \(error.localizedDescription)", type: .error)
            errorMessage = error.localizedDescription
        }
    }

    // MARK: - Display Sync Analysis Result

    private func displaySyncAnalysisResult(_ analysis: AnalysisData) {
        appendLine("", type: .normal)
        appendLine("✨ 分析结论", type: .phase)
        appendLine("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", type: .phase)
        appendLine("", type: .normal)

        // 生活状态 - 重点突出
        appendLine("\(analysis.lifeStatus.emoji) \(analysis.lifeStatus.label)", type: .result)
        if let description = analysis.lifeStatus.description {
            appendLine(description, type: .thinking)
        }

        appendLine("", type: .normal)

        // 有空分析 - 简化显示
        appendLine("判断: \(analysis.availability.status)", type: .conclusion)
        appendLine("理由: \(analysis.availability.reason)", type: .thinking)

        appendLine("", type: .normal)
        appendLine("✅ 状态已更新", type: .result)
    }

    // MARK: - Collect Device Data

    private func collectDeviceData() async {
        let deviceStatus = deviceCollector.currentStatus
        let locationStatus = locationCollector.currentStatus

        // 位置信息
        let placeText = locationStatus.placeName ?? locationStatus.placeType.rawValue
        appendLine("  ├─ 📍 位置: \(placeText)", type: .clue)
        if locationStatus.atPlaceSinceMinutes > 0 {
            appendLine("  │    已停留 \(locationStatus.atPlaceSinceMinutes) 分钟", type: .thinking)
        }

        // 电池状态
        let batteryPercent = Int(deviceStatus.batteryLevel * 100)
        let batteryText = deviceStatus.isCharging ? "\(batteryPercent)% (充电中)" : "\(batteryPercent)%"
        appendLine("  ├─ 🔋 电量: \(batteryText)", type: .clue)

        // 网络状态
        appendLine("  ├─ 📶 网络: \(deviceStatus.networkType.rawValue)", type: .clue)

        // 专注模式
        if deviceStatus.isFocusModeOn {
            appendLine("  ├─ 🔕 专注模式: 已开启", type: .clue)
        }

        // 耳机
        if deviceStatus.isHeadphonesConnected {
            appendLine("  ├─ 🎧 耳机: 已连接", type: .clue)
        }

        // 日历
        if calendarCollector.isAuthorized {
            let calendarStatus = calendarCollector.currentStatus
            if calendarStatus.hasCurrentEvent {
                let title = calendarStatus.currentEventTitle ?? "会议"
                appendLine("  ├─ 📅 日历: 正在进行「\(title)」", type: .clue)
            } else if let nextInMinutes = calendarStatus.nextEventInMinutes, nextInMinutes > 0 {
                appendLine("  ├─ 📅 日历: \(nextInMinutes)分钟后有安排", type: .clue)
            } else {
                appendLine("  ├─ 📅 日历: 暂无安排", type: .clue)
            }
        }

        // 运动
        if movementCollector.isAuthorized {
            let movementStatus = movementCollector.currentStatus
            if movementStatus.isMoving {
                appendLine("  └─ 🏃 运动: \(movementStatus.movementType.displayName)", type: .clue)
            } else {
                let minutes = movementStatus.stationaryMinutes
                if minutes > 0 {
                    appendLine("  └─ 🧘 状态: 静止 \(minutes) 分钟", type: .clue)
                } else {
                    appendLine("  └─ 🧘 状态: 静止", type: .clue)
                }
            }
        } else {
            appendLine("  └─ 🏃 运动: 无权限", type: .thinking)
        }

        // 小延迟让用户能看清数据
        try? await Task.sleep(nanoseconds: 300_000_000) // 0.3秒
    }

    // MARK: - Build Request

    private func buildStatusRequest() -> StatusReportRequest {
        let deviceStatus = deviceCollector.currentStatus
        let locationStatus = locationCollector.currentStatus
        let calendarStatus = calendarCollector.isAuthorized ? calendarCollector.currentStatus : nil
        let movementStatus = movementCollector.isAuthorized ? movementCollector.currentStatus : nil

        return StatusReportRequest(
            screen: nil, // 屏幕数据已移除
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

    // MARK: - Helper

    private func appendLine(_ text: String, type: OutputLine.LineType) {
        outputLines.append(OutputLine(text: text, type: type))
    }
}
