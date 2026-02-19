import Foundation
import Combine
import EventKit

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

    // ========== V4 Agent 状态 ==========
    @Published var v4Phase: String = ""              // 当前处理阶段
    @Published var currentToolName: String = ""      // 当前执行的工具名
    @Published var friendsResult: [V4FriendInfo] = [] // 好友查询结果
    @Published var pendingMessage: V4PendingMessage?  // 待确认消息
    @Published var pendingInvite: V4PendingInvite?    // 待确认邀请
    @Published var pendingDeletion: V4PendingDeletion? // 待确认删除
    @Published var streamingText: String = ""          // 流式文本累积

    // ========== Legacy 多阶段对话状态机 ==========
    @Published var currentPhase: ConversationPhase = .understanding
    @Published var intentSummary: IntentSummary?
    @Published var draftPlan: DraftPlan?
    @Published var clarifications: [ClarificationItem] = []
    @Published var canApprove: Bool = false

    // 覆盖层和录音控制
    @Published var showOverlay: Bool = false

    /// 是否可以录音（非处理/确认状态时可录音）
    var canRecord: Bool {
        state != .processing && state != .confirming
    }

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

    /// SSE 流专用 Session：15 秒无数据超时，30 秒总限
    private lazy var sseSession: URLSession = {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 15
        config.timeoutIntervalForResource = 30
        return URLSession(configuration: config)
    }()

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

    // MARK: - Text Input

    /// 提交文本输入（走 V4 Agent 流程，跳过语音转文字）
    func submitText(_ text: String) async {
        let trimmed = text.trimmingCharacters(in: .whitespaces)
        guard !trimmed.isEmpty else { return }
        guard state != .processing && state != .confirming else { return }

        state = .processing
        processingStatus = "处理中..."
        progressItems = []

        do {
            let endpoint = APIEndpoint.voiceScheduleText(sessionId: sessionId, text: trimmed)

            var urlRequest = try buildSSERequest(for: endpoint)
            urlRequest.setValue("text/event-stream", forHTTPHeaderField: "Accept")

            let (bytes, response) = try await sseSession.bytes(for: urlRequest)

            guard let httpResponse = response as? HTTPURLResponse,
                  httpResponse.statusCode == 200 else {
                throw SSEError.httpError(statusCode: (response as? HTTPURLResponse)?.statusCode ?? 0)
            }

            for try await line in bytes.lines {
                guard !line.isEmpty else { continue }
                if line == "data: [DONE]" { break }

                if let event = parseVoiceScheduleEvent(line) {
                    handleEvent(event)

                    if let sid = event.sessionId, sessionId == nil {
                        sessionId = sid
                    }

                    if event.type == .error || event.type == .statusUpdated ||
                       event.type == .scheduleSaved || event.type == .confirmed ||
                       event.type == .messageSent || event.type == .inviteSent ||
                       event.type == .deletePreview || event.type == .scheduleDeleted ||
                       event.type == .friendRemoved || event.type == .inviteResponded {
                        break
                    }
                }
            }
        } catch {
            let code = (error as NSError).code
            if code == NSURLErrorTimedOut {
                addMessage(.system, content: "处理超时，请重试")
            } else if code != NSURLErrorCancelled {
                addMessage(.system, content: "处理失败，请重试")
            }
            state = .idle
        }
    }

    // MARK: - Process Audio

    private func processAudio(_ audioData: Data) async {
        state = .processing
        processingStatus = "上传中..."

        // 清空上一轮的进度项，避免累积
        progressItems = []

        do {
            let endpoint = APIEndpoint.voiceScheduleStream

            // 构建 multipart 请求
            let boundary = UUID().uuidString
            var urlRequest = try buildSSERequest(for: endpoint)
            urlRequest.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
            urlRequest.setValue("text/event-stream", forHTTPHeaderField: "Accept")

            var body = Data()
            // 音频文件部分
            body.append("--\(boundary)\r\n".data(using: .utf8)!)
            body.append("Content-Disposition: form-data; name=\"audio\"; filename=\"voice.wav\"\r\n".data(using: .utf8)!)
            body.append("Content-Type: audio/wav\r\n\r\n".data(using: .utf8)!)
            body.append(audioData)
            body.append("\r\n".data(using: .utf8)!)

            // session_id 部分（如果有）
            if let sid = sessionId {
                body.append("--\(boundary)\r\n".data(using: .utf8)!)
                body.append("Content-Disposition: form-data; name=\"session_id\"\r\n\r\n".data(using: .utf8)!)
                body.append(sid.data(using: .utf8)!)
                body.append("\r\n".data(using: .utf8)!)
            }

            body.append("--\(boundary)--\r\n".data(using: .utf8)!)
            urlRequest.httpBody = body

            let (bytes, response) = try await sseSession.bytes(for: urlRequest)

            guard let httpResponse = response as? HTTPURLResponse,
                  httpResponse.statusCode == 200 else {
                throw SSEError.httpError(statusCode: (response as? HTTPURLResponse)?.statusCode ?? 0)
            }

            for try await line in bytes.lines {
                guard !line.isEmpty else { continue }
                if line == "data: [DONE]" { break }

                if let event = parseVoiceScheduleEvent(line) {
                    handleEvent(event)

                    // 保存 session ID
                    if let sid = event.sessionId, sessionId == nil {
                        sessionId = sid
                    }

                    // 终态事件后退出 stream（防止 [DONE] 未收到时卡住）
                    if event.type == .error || event.type == .statusUpdated ||
                       event.type == .scheduleSaved || event.type == .confirmed ||
                       event.type == .messageSent || event.type == .inviteSent ||
                       event.type == .deletePreview || event.type == .scheduleDeleted ||
                       event.type == .friendRemoved || event.type == .inviteResponded {
                        break
                    }
                }
            }
        } catch {
            let code = (error as NSError).code
            if code == NSURLErrorTimedOut {
                addMessage(.system, content: "处理超时，请重试")
            } else if code != NSURLErrorCancelled {
                addMessage(.system, content: "处理失败，请重试")
            }
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

            transcript = event.message ?? ""
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
            // 不覆盖终态（.completed），避免 chat 事件把已完成的状态冲掉
            if state != .completed {
                state = .idle
            }

        case .confirmed:
            addMessage(.system, content: "✓ 已保存！状态将按时刻表自动更新")
            state = .completed

        case .error:
            let msg = event.message ?? "未知错误"
            addMessage(.system, content: "错误: \(msg)")
            state = .idle
            progressItems = []

        // ========== V4 Agent 架构事件处理 ==========
        case .phase:
            // V4 处理阶段（understanding, thinking, continuing, complete）
            v4Phase = event.v4Phase ?? ""
            processingStatus = event.message ?? "处理中..."
            print("[VoiceSchedule-V4] phase=\(v4Phase) loop=\(event.loop ?? 0)")

        case .toolStart:
            // V4 工具开始执行
            currentToolName = event.toolName ?? ""
            let toolDesc = event.message ?? "执行 \(currentToolName)..."
            let item = ProgressItem(action: "tool", message: toolDesc, detail: nil)
            progressItems.append(item)
            processingStatus = toolDesc

        case .toolEnd:
            // V4 工具执行完成
            if !progressItems.isEmpty {
                progressItems[progressItems.count - 1].isCompleted = true
            }
            currentToolName = ""

        case .schedulePreview:
            // V4 时刻表预览（替代旧 schedule 事件）
            if let items = event.items {
                schedule = items
                let dateStr = event.date ?? ""
                let msg = event.message ?? "已生成时刻表预览"
                addMessage(.aiSchedule, content: msg, schedule: items, reasoning: nil, isQuery: false)
            }
            progressItems = []
            state = .awaitingApproval
            canApprove = true

        case .scheduleSaved:
            // V4 时刻表已保存（替代旧 confirmed 事件）
            if let items = event.items {
                schedule = items
                // 为有提醒的 items 创建系统提醒事项
                let dateStr = event.date ?? todayDateString()
                scheduleSystemReminders(items: items, date: dateStr)
            }
            addMessage(.system, content: "✓ 时刻表已保存！状态将按时刻表自动更新")
            state = .completed
            // 通知首页刷新 Grid
            NotificationCenter.default.post(name: .gridNeedsRefresh, object: nil)

        case .statusUpdated:
            // V4 当前状态已更新（替代旧 current_status 事件）
            let emoji = event.emoji ?? "🤔"
            let status = event.status ?? "状态已更新"
            let msg = event.message ?? "\(emoji) \(status)"
            addMessage(.aiText, content: msg)
            progressItems = []
            state = .completed
            // 通知首页刷新 Grid（更新 GIF 等）
            NotificationCenter.default.post(name: .gridNeedsRefresh, object: nil)

        case .preferenceUpdated:
            // V4 偏好设置已更新
            let msg = event.message ?? "偏好设置已更新"
            addMessage(.system, content: "✓ \(msg)")
            progressItems = []
            state = .idle

        case .chatStream:
            // V4 流式文本片段
            if let text = event.message {
                streamingText += text
                // 替换最后一条 AI 消息或创建新的
                if let lastIndex = messages.lastIndex(where: { $0.type == .aiText }) {
                    var msg = messages[lastIndex]
                    messages[lastIndex] = ChatMessage(
                        type: .aiText,
                        content: streamingText,
                        timestamp: msg.timestamp
                    )
                } else {
                    addMessage(.aiText, content: streamingText)
                }
            }

        case .chatStreamEnd:
            // V4 流式输出结束
            streamingText = ""
            progressItems = []
            state = .idle

        case .friendsResult:
            // V4 好友查询结果
            friendsResult = event.friends ?? []
            let msg = event.message ?? "查询完成"
            addMessage(.aiText, content: msg)
            progressItems = []
            state = .idle

        case .messagePreview:
            // V4 消息预览（待确认）
            pendingMessage = event.pendingMessage
            let msg = event.message ?? "消息预览"
            addMessage(.aiText, content: msg, awaitingAction: true)
            progressItems = []
            state = .awaitingApproval
            canApprove = true

        case .messageSent:
            // V4 消息已发送
            pendingMessage = nil
            let sentTo = event.sentTo ?? ""
            let msg = event.message ?? "消息已发送给 \(sentTo)"
            addMessage(.system, content: "✓ \(msg)")
            state = .idle

        case .invitePreview:
            // V4 邀请预览（待确认）
            pendingInvite = event.pendingInvite
            let msg = event.message ?? "邀请预览"
            addMessage(.aiText, content: msg, awaitingAction: true)
            progressItems = []
            state = .awaitingApproval
            canApprove = true

        case .inviteSent:
            // V4 邀请已发送
            pendingInvite = nil
            let sentTo = event.sentTo ?? ""
            let msg = event.message ?? "邀请已发送给 \(sentTo)"
            addMessage(.system, content: "✓ \(msg)")
            state = .idle

        case .deletePreview:
            // V4 删除预览（待确认）
            pendingDeletion = event.pendingDeletion
            let msg = event.message ?? "将删除安排，等待确认"
            addMessage(.aiText, content: msg, awaitingAction: true)
            progressItems = []
            state = .awaitingApproval
            canApprove = true

        case .scheduleDeleted:
            // V4 日程已删除
            pendingDeletion = nil
            if let items = event.items {
                schedule = items
            } else {
                schedule = []
            }
            let msg = event.message ?? "已删除"
            addMessage(.system, content: "✓ \(msg)")
            state = .completed

        case .friendRemoved:
            // V4 好友已删除
            pendingDeletion = nil
            let msg = event.message ?? "好友已删除"
            addMessage(.system, content: "✓ \(msg)")
            state = .completed

        case .inviteResponded:
            // V4 收到的邀请已响应
            let msg = event.message ?? "已响应邀请"
            addMessage(.system, content: "✓ \(msg)")
            state = .completed

        // ========== Legacy 多阶段对话状态机事件 ==========
        case .phaseChange:
            if let phase = event.legacyPhase {
                currentPhase = phase
                processingStatus = event.message ?? "阶段: \(phase.rawValue)"
            }

        case .intentSummary:
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
            let message = event.message ?? "需要确认一些信息"
            if let clarifications = event.clarifications {
                self.clarifications = clarifications
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
            let message = event.message ?? "确认后将保存时刻表"
            canApprove = event.canApprove ?? true
            if let plan = event.draftPlan, let scheduleItems = plan.schedule {
                schedule = scheduleItems
                addMessage(.aiSchedule, content: message, schedule: scheduleItems, reasoning: nil, isQuery: false)
            } else if !schedule.isEmpty {
                addMessage(.system, content: message)
            }
            state = .awaitingApproval

        case .unknown, .none:
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

    /// 带可见性设置的确认（通过 V4 text 端点发送"确认"）
    func confirmScheduleWithVisibility(visibility: ScheduleVisibility, circleIDs: [String]) async {
        guard let sid = sessionId else {
            // 没有 session，直接确认当前状态
            if !schedule.isEmpty {
                state = .confirming
                try? await Task.sleep(nanoseconds: 500_000_000)
                addMessage(.system, content: "✓ 已保存！")
                state = .completed
            }
            return
        }

        state = .confirming

        do {
            // 通过 V4 text 端点发送"确认"，让 LLM 调用 confirm 工具
            let endpoint = APIEndpoint.voiceScheduleText(sessionId: sid, text: "确认")

            var urlRequest = try buildSSERequest(for: endpoint)
            urlRequest.setValue("text/event-stream", forHTTPHeaderField: "Accept")

            let (bytes, response) = try await sseSession.bytes(for: urlRequest)

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

                    if event.type == .error || event.type == .confirmed || event.type == .scheduleSaved
                        || event.type == .messageSent || event.type == .inviteSent
                        || event.type == .scheduleDeleted || event.type == .friendRemoved
                        || event.type == .inviteResponded {
                        break
                    }
                }
            }
        } catch {
            let code = (error as NSError).code
            if code == NSURLErrorTimedOut {
                addMessage(.system, content: "处理超时，请重试")
            } else if code != NSURLErrorCancelled {
                addMessage(.system, content: "处理失败，请重试")
            }
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

    // MARK: - Cancel Pending

    /// 放弃当前待确认操作（预览/草稿），保留对话继续修改
    func cancelPending() {
        schedule = []
        pendingMessage = nil
        pendingInvite = nil
        pendingDeletion = nil
        canApprove = false
        showVisibilitySelection = false
        state = .idle
        addMessage(.system, content: "已放弃，你可以继续说")
    }

    // MARK: - Cancel

    func cancelSession() async {
        if let sid = sessionId {
            do {
                let requestBody = VoiceScheduleInteractionRequest(
                    sessionId: sid,
                    action: VoiceScheduleAction.cancel.rawValue,
                    data: nil
                )
                let endpoint = APIEndpoint.voiceScheduleInteraction(request: requestBody)
                var urlRequest = try buildSSERequest(for: endpoint)
                urlRequest.setValue("text/event-stream", forHTTPHeaderField: "Accept")

                let (bytes, _) = try await sseSession.bytes(for: urlRequest)
                for try await line in bytes.lines {
                    guard !line.isEmpty else { continue }
                    if line == "data: [DONE]" { break }
                }
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

        // V4 Agent 状态重置
        v4Phase = ""
        currentToolName = ""
        friendsResult = []
        pendingMessage = nil
        pendingInvite = nil
        pendingDeletion = nil
        streamingText = ""

        // Legacy 多阶段对话状态机重置
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
        isQuery: Bool = false,
        awaitingAction: Bool = false
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
        message.awaitingAction = awaitingAction
        messages.append(message)
    }

    private func todayDateString() -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: Date())
    }

    // MARK: - System Reminders (EKReminder)

    func scheduleSystemReminders(items: [ScheduleItem], date: String) {
        Self.scheduleReminders(items: items, date: date)
    }

    static func scheduleReminders(items: [ScheduleItem], date: String) {
        let reminderItems = items.filter { ($0.remindBefore ?? 0) > 0 }
        guard !reminderItems.isEmpty else { return }

        let store = EKEventStore()

        let requestAccess: (@escaping (Bool, Error?) -> Void) -> Void = { completion in
            if #available(iOS 17.0, *) {
                store.requestFullAccessToReminders(completion: completion)
            } else {
                store.requestAccess(to: .reminder, completion: completion)
            }
        }

        requestAccess { granted, error in
            guard granted else {
                print("[Reminder] 权限未授予: \(error?.localizedDescription ?? "unknown")")
                return
            }

            for item in reminderItems {
                createReminder(store: store, item: item, date: date)
            }
        }
    }

    private static func createReminder(store: EKEventStore, item: ScheduleItem, date: String) {
        guard let dueDate = parseDateTime(date: date, time: item.startTime) else { return }
        let remindMinutes = item.remindBefore ?? 15

        let reminder = EKReminder(eventStore: store)
        reminder.title = "\(item.emoji) \(item.status)"
        reminder.calendar = store.defaultCalendarForNewReminders()
        reminder.dueDateComponents = Calendar.current.dateComponents(
            [.year, .month, .day, .hour, .minute], from: dueDate
        )
        reminder.addAlarm(EKAlarm(relativeOffset: TimeInterval(-remindMinutes * 60)))

        do {
            try store.save(reminder, commit: true)
            let key = "reminder_\(date)_\(item.startTime)"
            UserDefaults.standard.set(reminder.calendarItemIdentifier, forKey: key)
            print("[Reminder] 已创建: \(item.emoji) \(item.status), 提前\(remindMinutes)分钟")
        } catch {
            print("[Reminder] 创建失败: \(error)")
        }
    }

    static func removeSystemReminder(date: String, startTime: String) {
        let key = "reminder_\(date)_\(startTime)"
        guard let identifier = UserDefaults.standard.string(forKey: key) else { return }

        let store = EKEventStore()
        if let calendarItem = store.calendarItem(withIdentifier: identifier) as? EKReminder {
            do {
                try store.remove(calendarItem, commit: true)
                print("[Reminder] 已删除: \(date) \(startTime)")
            } catch {
                print("[Reminder] 删除失败: \(error)")
            }
        }
        UserDefaults.standard.removeObject(forKey: key)
    }

    private static func parseDateTime(date: String, time: String) -> Date? {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd HH:mm"
        return formatter.date(from: "\(date) \(time)")
    }
}
