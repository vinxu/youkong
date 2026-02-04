import Foundation
import Combine

/// 语音状态时刻表流程状态
enum VoiceScheduleState: Equatable {
    case idle                           // 空闲（可以录音）
    case recording                      // 录音中
    case processing                     // 处理中（识别+分析）
    case discussing                     // 讨论中（多阶段对话）
    case awaitingApproval               // 等待审批
    case confirming                     // 确认中
    case completed                      // 完成
    case error(String)                  // 错误

    static func == (lhs: VoiceScheduleState, rhs: VoiceScheduleState) -> Bool {
        switch (lhs, rhs) {
        case (.idle, .idle),
             (.recording, .recording),
             (.processing, .processing),
             (.discussing, .discussing),
             (.awaitingApproval, .awaitingApproval),
             (.confirming, .confirming),
             (.completed, .completed):
            return true
        case (.error(let l), .error(let r)):
            return l == r
        default:
            return false
        }
    }
}

/// 语音状态时刻表 ViewModel
@MainActor
class VoiceScheduleViewModel: ObservableObject {
    // MARK: - Published Properties

    @Published var state: VoiceScheduleState = .idle
    @Published var isRecording = false
    @Published var recordingDuration: TimeInterval = 0
    @Published var isCancelling = false

    // 对话消息
    @Published var messages: [ChatMessage] = []

    // 处理进度
    @Published var processingStatus: String = ""
    @Published var transcript: String = ""

    // 结果数据
    @Published var schedule: [ScheduleItem] = []
    @Published var clarifyQuestions: [ClarifyQuestion] = []
    @Published var reasoning: [String] = []        // 推理依据

    // 过程反馈
    @Published var progressItems: [ProgressItem] = []

    // 可见性
    @Published var showVisibilitySelection = false
    @Published var selectedVisibility: ScheduleVisibility = .allFriends
    @Published var selectedCircleIDs: Set<String> = []
    @Published var availableCircles: [CircleInfoCompact] = []

    // ========== 多阶段对话状态机（新增）==========
    @Published var currentPhase: ConversationPhase = .understanding
    @Published var intentSummary: IntentSummary?
    @Published var draftPlan: DraftPlan?
    @Published var clarifications: [ClarificationItem] = []
    @Published var canApprove: Bool = false

    /// 过程反馈条目
    struct ProgressItem: Identifiable {
        let id = UUID()
        let action: String
        let message: String
        var detail: String?
        var isCompleted: Bool = false
    }

    // MARK: - Computed Properties

    var hasSchedule: Bool {
        !schedule.isEmpty
    }

    // MARK: - Private Properties

    private let recorder = VoiceRecordingManager()
    private let sseClient = SSEClient()
    private var sessionId: String?
    private var cancellables = Set<AnyCancellable>()

    // MARK: - Init

    init() {
        // 观察录制器状态
        recorder.$isRecording
            .receive(on: DispatchQueue.main)
            .sink { [weak self] isRecording in
                self?.isRecording = isRecording
            }
            .store(in: &cancellables)

        recorder.$recordingDuration
            .receive(on: DispatchQueue.main)
            .sink { [weak self] duration in
                self?.recordingDuration = duration
            }
            .store(in: &cancellables)

        recorder.$isCancelled
            .receive(on: DispatchQueue.main)
            .sink { [weak self] isCancelled in
                self?.isCancelling = isCancelled
            }
            .store(in: &cancellables)
    }

    // MARK: - Permission

    var hasPermission: Bool {
        recorder.hasPermission
    }

    var permissionDenied: Bool {
        recorder.permissionDenied
    }

    func requestPermission() async -> Bool {
        await recorder.requestPermission()
    }

    /// 预热音频会话（在界面显示时调用）
    func prepareAudioSession() {
        recorder.prepareAudioSession()
    }

    // MARK: - Recording

    func startRecording() {
        // 允许在非 processing/confirming 状态下录音
        // 新增：在 awaitingApproval 状态下也允许录音（用户可以说"确定"或"修改"）
        guard state != .processing && state != .confirming else {
            return
        }

        do {
            _ = try recorder.startRecording()
            state = .recording
        } catch {
            addMessage(.system, content: "录音失败: \(error.localizedDescription)")
        }
    }

    func cancelRecording() {
        recorder.cancelRecording()
        if state == .recording {
            state = .idle
        }
    }

    func setCancelling(_ cancelling: Bool) {
        isCancelling = cancelling
    }

    // MARK: - Submit Recording

    func submitRecording() async {
        guard let result = recorder.stopRecording() else {
            // 录音太短
            addMessage(.system, content: "录音时间太短，请长按说话")
            state = .idle
            return
        }

        guard let audioData = try? Data(contentsOf: result.url) else {
            addMessage(.system, content: "读取录音文件失败")
            state = .idle
            return
        }

        await processAudio(audioData)
    }

    // MARK: - Process Audio

    private func processAudio(_ audioData: Data) async {
        state = .processing
        processingStatus = "上传中..."

        // 清空上一轮的进度项，避免累积
        progressItems = []

        do {
            // 传递 sessionId 以恢复多轮对话上下文（如目标日期）
            let newSessionId = try await sseClient.streamVoiceSchedule(
                audioData: audioData,
                sessionId: sessionId,  // 传递已有的 sessionId
                onEvent: { [weak self] event in
                    self?.handleEvent(event)
                }
            )

            // 保存/更新 session ID
            if let sid = newSessionId {
                sessionId = sid
            }
        } catch {
            addMessage(.system, content: "处理失败: \(error.localizedDescription)")
            state = .idle
        }
    }

    // MARK: - Handle SSE Events

    private func handleEvent(_ event: VoiceScheduleEvent) {
        switch event.type {
        case .sessionStart:
            if let sid = event.sessionId {
                sessionId = sid
            }
            processingStatus = "已连接"

        case .recognizing:
            processingStatus = event.status ?? "识别中..."

        case .progress:
            // 处理过程反馈
            handleProgressEvent(event)

        case .transcript:
            // 标记所有进度项为已完成
            for i in progressItems.indices {
                progressItems[i].isCompleted = true
            }

            transcript = event.text ?? ""
            processingStatus = "分析中..."
            // 添加用户消息
            if !transcript.isEmpty {
                addMessage(.user, content: transcript)
            }

        case .thinking:
            processingStatus = event.status ?? "AI 思考中..."

        case .clarify:
            // 需要澄清
            if let questions = event.questions {
                clarifyQuestions = questions
                addMessage(.aiQuestion, content: "需要确认一些信息：", questions: questions)
            }
            state = .idle  // 回到可录音状态

        case .schedule:
            // 收到时刻表
            if let items = event.items {
                schedule = items
                reasoning = event.reasoning ?? []
                let isQueryMode = event.isQuery ?? false
                addMessage(.aiSchedule, content: isQueryMode ? "当前时刻表" : "已生成时刻表", schedule: items, reasoning: event.reasoning, isQuery: isQueryMode)
            }
            progressItems = []  // 清除进度
            state = .idle  // 回到可录音状态，用户可以继续修改

        case .currentStatus:
            // 无行程，猜测当前状态
            let emoji = event.emoji ?? "🤔"
            let status = event.statusText ?? "未知状态"
            let reason = event.reason ?? ""

            // 创建单条状态作为时刻表
            let now = Date()
            let formatter = DateFormatter()
            formatter.dateFormat = "HH:mm"
            let currentTime = formatter.string(from: now)

            let item = ScheduleItem(
                startTime: currentTime,
                endTime: "--:--",
                emoji: emoji,
                status: status,
                executed: false
            )
            schedule = [item]
            reasoning = event.reasoning ?? []

            var content = "\(emoji) \(status)"
            if !reason.isEmpty {
                content += "\n\(reason)"
            }
            addMessage(.aiText, content: content)
            progressItems = []  // 清除进度
            state = .idle

        case .visibilityPrompt:
            // 提示选择可见性
            showVisibilitySelection = true

        case .circleList:
            // 更新可用圈子列表
            if let circles = event.circles {
                availableCircles = circles
            }

        case .chat:
            // 聊天回复（非时刻表操作）
            let message = event.message ?? "我在这里，有什么可以帮你的？"
            addMessage(.aiText, content: message)
            progressItems = []  // 清除进度
            state = .idle  // 保持可对话状态

        case .confirmed:
            addMessage(.system, content: "✓ 已保存！状态将按时刻表自动更新")
            state = .completed

        case .error:
            let msg = event.message ?? "未知错误"
            addMessage(.system, content: "错误: \(msg)")
            state = .idle
            progressItems = []

        // ========== 多阶段对话状态机事件处理（新增）==========
        case .phaseChange:
            // 阶段转换
            if let phase = event.phase {
                currentPhase = phase
                processingStatus = event.message ?? "阶段: \(phase.rawValue)"
                print("[VoiceSchedule] 阶段变更: \(event.previousPhase?.rawValue ?? "nil") -> \(phase.rawValue)")
            }

        case .intentSummary:
            // 意图理解结果
            if let summary = event.intentSummary {
                intentSummary = summary
                let activities = summary.activities?.joined(separator: "、") ?? ""
                let message = event.message ?? "理解到：\(activities)"
                addMessage(.aiText, content: message)
                if let reasoning = summary.reasoning, !reasoning.isEmpty {
                    self.reasoning = reasoning
                }
            }

        case .discussion:
            // 讨论消息
            let message = event.message ?? "需要确认一些信息"
            if let clarifications = event.clarifications {
                self.clarifications = clarifications
                // 转换为旧格式的问题
                let questions = clarifications.filter { !($0.answered ?? false) }.map { item in
                    ClarifyQuestion(id: item.id, question: item.question, options: [], allowVoice: true)
                }
                if !questions.isEmpty {
                    addMessage(.aiQuestion, content: message, questions: questions)
                } else {
                    addMessage(.aiText, content: message)
                }
            } else {
                addMessage(.aiText, content: message)
            }
            state = .discussing

        case .draftPlan:
            // 计划草案（待审批）
            if let plan = event.draftPlan {
                draftPlan = plan
                if let scheduleItems = plan.schedule {
                    schedule = scheduleItems
                }
                if let planReasoning = plan.reasoning {
                    reasoning = planReasoning
                }
                let summary = plan.summary ?? "已生成时刻表"
                addMessage(.aiSchedule, content: summary, schedule: plan.schedule, reasoning: plan.reasoning, isQuery: false)

                // 显示变更列表
                if let changes = plan.changes, !changes.isEmpty {
                    var changesText = "变更：\n"
                    for change in changes {
                        let typeEmoji = change.type == "add" ? "➕" : (change.type == "delete" ? "➖" : "✏️")
                        changesText += "\(typeEmoji) \(change.description) (\(change.timeRange))\n"
                    }
                    addMessage(.system, content: changesText)
                }
            }
            canApprove = event.canApprove ?? true
            state = .awaitingApproval

        case .approvalPrompt:
            // 审批提示
            let message = event.message ?? "确认后将保存时刻表"
            canApprove = event.canApprove ?? true
            if let plan = event.draftPlan, let scheduleItems = plan.schedule {
                schedule = scheduleItems
                addMessage(.aiSchedule, content: message, schedule: scheduleItems, reasoning: nil, isQuery: false)
            } else if !schedule.isEmpty {
                addMessage(.system, content: message)
            }
            state = .awaitingApproval

        case .none:
            break
        }
    }

    /// 处理过程反馈事件
    private func handleProgressEvent(_ event: VoiceScheduleEvent) {
        if let action = event.action, let message = event.message {
            // 新的进度项
            let item = ProgressItem(action: action, message: message, detail: event.detail)
            progressItems.append(item)
            processingStatus = message
        } else if let detail = event.detail, !progressItems.isEmpty {
            // 更新最后一个进度项的详情
            var lastIndex = progressItems.count - 1
            progressItems[lastIndex].detail = detail
        }
    }

    // MARK: - Confirm

    func confirmSchedule() async {
        await confirmScheduleWithVisibility(visibility: selectedVisibility, circleIDs: Array(selectedCircleIDs))
    }

    /// 带可见性设置的确认
    func confirmScheduleWithVisibility(visibility: ScheduleVisibility, circleIDs: [String]) async {
        guard let sid = sessionId else {
            // 没有 session，直接确认当前状态
            if !schedule.isEmpty {
                state = .confirming
                // 模拟确认
                try? await Task.sleep(nanoseconds: 500_000_000)
                addMessage(.system, content: "✓ 已保存！")
                state = .completed
            }
            return
        }

        state = .confirming

        do {
            // 构建带可见性的交互数据
            let interactionData = VoiceScheduleInteractionData(
                visibility: visibility.rawValue,
                circleIds: visibility == .circles ? circleIDs : nil
            )

            let requestBody = VoiceScheduleInteractionRequest(
                sessionId: sid,
                action: VoiceScheduleAction.confirm.rawValue,
                data: interactionData
            )

            let endpoint = APIEndpoint.voiceScheduleInteraction(request: requestBody)

            var urlRequest = try buildSSERequest(for: endpoint)
            urlRequest.setValue("text/event-stream", forHTTPHeaderField: "Accept")

            let (bytes, response) = try await URLSession.shared.bytes(for: urlRequest)

            guard let httpResponse = response as? HTTPURLResponse,
                  httpResponse.statusCode == 200 else {
                throw SSEError.httpError(statusCode: (response as? HTTPURLResponse)?.statusCode ?? 0)
            }

            for try await line in bytes.lines {
                guard !line.isEmpty else { continue }

                if line == "data: [DONE]" {
                    break
                }

                if let event = parseVoiceScheduleEvent(line) {
                    handleEvent(event)

                    if event.type == .error || event.type == .confirmed {
                        break
                    }
                }
            }
        } catch {
            addMessage(.system, content: "确认失败: \(error.localizedDescription)")
            state = .idle
        }
    }

    // MARK: - SSE Helpers

    private func buildSSERequest(for endpoint: APIEndpoint) throws -> URLRequest {
        #if DEBUG
        let custom = UserDefaults.standard.string(forKey: "debug_baseURL") ?? ""
        let baseURL = custom.isEmpty ? "http://49.232.13.41:8080" : custom
        #else
        let baseURL = "http://49.232.13.41:8080"
        #endif

        guard let url = URL(string: baseURL + endpoint.path) else {
            throw SSEError.invalidURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = endpoint.method.rawValue
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if endpoint.requiresAuth {
            if let token = KeychainManager.shared.getAccessToken() {
                request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            }
        }

        if let body = endpoint.body {
            request.httpBody = try JSONEncoder().encode(AnyEncodable(body))
        }

        return request
    }

    private func parseVoiceScheduleEvent(_ text: String) -> VoiceScheduleEvent? {
        guard text.hasPrefix("data: ") else { return nil }

        let jsonString = String(text.dropFirst(6))

        guard let data = jsonString.data(using: .utf8) else { return nil }

        do {
            let event = try JSONDecoder().decode(VoiceScheduleEvent.self, from: data)
            return event
        } catch {
            print("[SSE] Parse voice schedule error: \(error)")
            return nil
        }
    }

    // MARK: - Cancel

    func cancelSession() async {
        if let sid = sessionId {
            do {
                try await sseClient.streamVoiceScheduleInteraction(
                    sessionId: sid,
                    action: .cancel,
                    onEvent: { _ in }
                )
            } catch {
                // 忽略取消错误
            }
        }

        reset()
    }

    // MARK: - Reset

    func reset() {
        state = .idle
        sessionId = nil
        processingStatus = ""
        transcript = ""
        schedule = []
        clarifyQuestions = []
        reasoning = []
        messages = []
        progressItems = []
        showVisibilitySelection = false
        selectedVisibility = .allFriends
        selectedCircleIDs = []

        // 多阶段对话状态机重置
        currentPhase = .understanding
        intentSummary = nil
        draftPlan = nil
        clarifications = []
        canApprove = false
    }

    // MARK: - Helper

    private func addMessage(
        _ type: ChatMessage.MessageType,
        content: String,
        schedule: [ScheduleItem]? = nil,
        questions: [ClarifyQuestion]? = nil,
        reasoning: [String]? = nil,
        isQuery: Bool = false
    ) {
        var message = ChatMessage(
            type: type,
            content: content,
            timestamp: Date(),
            schedule: schedule,
            questions: questions,
            reasoning: reasoning
        )
        message.isQuery = isQuery
        messages.append(message)
    }
}
