import CommonCrypto
import CryptoKit
import SwiftUI

// MARK: - AI Status Inference View

/// AI 推断当下状态的视图
/// 用户点击后调用 AI 推断，然后可以确认或修改结果
struct AIStatusInferenceView: View {
    @StateObject private var viewModel = AIStatusInferenceViewModel()
    @Environment(\.dismiss) private var dismiss

    /// 状态确认后的回调
    var onStatusConfirmed: ((String, String, Bool) -> Void)?

    init(onStatusConfirmed: ((String, String, Bool) -> Void)? = nil) {
        self.onStatusConfirmed = onStatusConfirmed
    }

    var body: some View {
        ZStack(alignment: .bottom) {
            VStack(spacing: 0) {
                // CLI Header
                headerView

                Rectangle()
                    .fill(CLIColors.border)
                    .frame(height: 1)

                // Content
                if viewModel.isInferring || viewModel.isAskingUser {
                    inferringView
                } else if let inference = viewModel.inference {
                    resultView(inference: inference)
                } else if let error = viewModel.error {
                    errorView(error: error)
                } else {
                    // 首次进入直接显示 loading（onAppear 会立即触发推断）
                    inferringView
                }
            }

            // 底部浮框：用户确认选择题
            if viewModel.isAskingUser {
                askUserBottomPanel
            }
        }
        .background(CLIColors.background)
        .onAppear {
            Task {
                await viewModel.startInference()
            }
        }
    }

    // MARK: - Header

    private var headerView: some View {
        HStack {
            Button {
                dismiss()
            } label: {
                Text("[X]")
                    .font(.cliBodySmall)
                    .foregroundColor(CLIColors.textSecondary)
            }

            Spacer()

            Text("━━ AI 推断当下状态 ━━")
                .font(.cliHeadline)
                .foregroundColor(CLIColors.cyan)

            Spacer()

            // Placeholder for balance
            Text("[X]")
                .font(.cliBodySmall)
                .foregroundColor(.clear)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(CLIColors.background)
    }

    // MARK: - Inferring View

    private var inferringView: some View {
        VStack(alignment: .leading, spacing: 0) {
            // 顶部状态
            HStack(spacing: 8) {
                ProgressView()
                    .tint(CLIColors.yellow)
                Text(viewModel.streamingPhase.isEmpty ? "正在连接..." : viewModel.streamingPhase)
                    .font(.cliBody)
                    .foregroundColor(CLIColors.yellow)
            }
            .padding(16)

            // 流式日志区域
            ScrollViewReader { proxy in
                ScrollView {
                    VStack(alignment: .leading, spacing: 4) {
                        if viewModel.streamingLogs.isEmpty {
                            Text("> 正在收集设备数据...")
                                .font(.system(size: 12, design: .monospaced))
                                .foregroundColor(CLIColors.textWeak)
                        }
                        ForEach(Array(viewModel.streamingLogs.enumerated()), id: \.offset) { index, log in
                            let isThinking = log.hasPrefix("  │")
                            Text(log)
                                .font(.system(size: 11, design: .monospaced))
                                .foregroundColor(colorForLog(log))
                                .lineLimit(isThinking ? nil : 1)
                                .truncationMode(.tail)
                                .id(index)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(12)
                }
                .background(CLIColors.backgroundSecondary)
                .onChange(of: viewModel.streamingLogs.count) { count in
                    if count > 0 {
                        withAnimation {
                            proxy.scrollTo(count - 1, anchor: .bottom)
                        }
                    }
                }
            }
        }
    }

    private func colorForLog(_ log: String) -> Color {
        if log.hasPrefix("▸") { return CLIColors.cyan }
        if log.contains("⚙") { return CLIColors.yellow }
        if log.contains("✓") { return CLIColors.green }
        if log.contains("✕") { return CLIColors.red }
        if log.contains("?") { return CLIColors.yellow }
        if log.contains("→") { return CLIColors.green }
        return CLIColors.textSecondary
    }

    // MARK: - Ask User Bottom Panel

    private var askUserBottomPanel: some View {
        VStack(spacing: 12) {
            Text(viewModel.askUserQuestion)
                .font(.cliBody)
                .foregroundColor(CLIColors.textPrimary)
                .multilineTextAlignment(.center)

            ForEach(viewModel.askUserOptions, id: \.self) { option in
                Button {
                    Task {
                        await viewModel.respondToAskUser(answer: option)
                    }
                } label: {
                    Text(option)
                        .font(.cliBody)
                        .foregroundColor(CLIColors.green)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .overlay(
                            RoundedRectangle(cornerRadius: 4)
                                .stroke(CLIColors.green, lineWidth: 1)
                        )
                }
            }
        }
        .padding(16)
        .background(CLIColors.backgroundSecondary)
        .overlay(
            Rectangle()
                .frame(height: 1)
                .foregroundColor(CLIColors.border),
            alignment: .top
        )
        .padding(.horizontal, 16)
        .padding(.bottom, 16)
    }

    // startView removed - inferringView is shown by default

    // MARK: - Result View

    private func resultView(inference: CurrentStatusInference) -> some View {
        ScrollView {
            VStack(spacing: 20) {
                // 推断结果展示
                VStack(spacing: 12) {
                    // GIF 或 Emoji（根据 useGif 切换）+ [编辑]/[收起] 按钮
                    ZStack(alignment: .topTrailing) {
                        if viewModel.useGif, let gifUrl = inference.gifUrl, let url = URL(string: gifUrl) {
                            GifImageView(url: url) {
                                // 加载失败时不做任何事
                            }
                            .frame(maxHeight: 160)
                            .cornerRadius(12)
                        } else if viewModel.useGif && viewModel.isSearchingGif {
                            VStack(spacing: 8) {
                                ProgressView()
                                    .tint(CLIColors.cyan)
                                Text("搜索 GIF 中...")
                                    .font(.cliCaptionSmall)
                                    .foregroundColor(CLIColors.textSecondary)
                            }
                            .frame(height: 80)
                        } else {
                            Text(viewModel.editingEmoji)
                                .font(.system(size: emojiSize(for: viewModel.editingEmoji.count)))
                                .frame(height: 80)
                        }

                        Button {
                            viewModel.showEmojiPicker.toggle()
                        } label: {
                            Text(viewModel.showEmojiPicker ? "[收起]" : "[编辑]")
                                .font(.system(size: 11, design: .monospaced))
                                .foregroundColor(CLIColors.textSecondary)
                        }
                    }

                    // Emoji Picker 展开区域
                    if viewModel.showEmojiPicker {
                        VStack(alignment: .leading, spacing: 8) {
                            // 已选 emoji 行
                            HStack(spacing: 8) {
                                Text("> Emoji:")
                                    .font(.system(size: 12, design: .monospaced))
                                    .foregroundColor(CLIColors.textSecondary)

                                let emojis = splitEmojis(viewModel.editingEmoji)
                                if emojis.isEmpty {
                                    Text("点击下方选择")
                                        .font(.system(size: 10, design: .monospaced))
                                        .foregroundColor(CLIColors.textWeak)
                                } else {
                                    ForEach(emojis, id: \.self) { emoji in
                                        Button {
                                            viewModel.editingEmoji = viewModel.editingEmoji.replacingOccurrences(of: emoji, with: "")
                                        } label: {
                                            HStack(spacing: 0) {
                                                Text(emoji).font(.system(size: 24))
                                                Text("×")
                                                    .font(.system(size: 10, design: .monospaced))
                                                    .foregroundColor(CLIColors.textWeak)
                                            }
                                        }
                                    }
                                }
                            }

                            // 分类 emoji 网格
                            EmojiGridPicker(selected: $viewModel.editingEmoji)
                        }
                        .padding(12)
                        .background(
                            RoundedRectangle(cornerRadius: 4)
                                .stroke(CLIColors.border, lineWidth: 1)
                                .background(
                                    RoundedRectangle(cornerRadius: 4)
                                        .fill(CLIColors.backgroundSecondary)
                                )
                        )
                    }

                    // Emoji ↔ GIF 切换按钮
                    emojiGifToggle

                    // 活动描述 [编辑]/[完成] 切换
                    HStack(spacing: 8) {
                        if viewModel.isEditingActivity {
                            TextField("活动描述", text: $viewModel.editingActivity)
                                .font(.cliHeadline)
                                .foregroundColor(CLIColors.textPrimary)
                                .multilineTextAlignment(.center)
                                .textFieldStyle(.plain)
                                .padding(.horizontal, 16)
                                .padding(.vertical, 8)
                                .background(
                                    RoundedRectangle(cornerRadius: 4)
                                        .stroke(CLIColors.border, lineWidth: 1)
                                )
                        } else {
                            Text(viewModel.editingActivity)
                                .font(.cliHeadline)
                                .foregroundColor(CLIColors.textPrimary)
                        }

                        Button {
                            viewModel.isEditingActivity.toggle()
                        } label: {
                            Text(viewModel.isEditingActivity ? "[完成]" : "[编辑]")
                                .font(.system(size: 11, design: .monospaced))
                                .foregroundColor(CLIColors.textSecondary)
                        }
                    }

                    // 场所（如果有）
                    if let place = inference.place, !place.isEmpty {
                        HStack(spacing: 4) {
                            Text("📍")
                            Text(place)
                                .font(.cliBody)
                                .foregroundColor(CLIColors.textSecondary)
                        }
                    }

                    // 预计持续时长
                    if let durationHint = inference.durationHint, !durationHint.isEmpty {
                        HStack(spacing: 4) {
                            Text("⏱️")
                            Text(durationHint)
                                .font(.cliBodySmall)
                                .foregroundColor(CLIColors.textWeak)
                        }
                    }
                }
                .padding(.top, 24)

                // 有空状态开关
                availabilityToggle

                // 置信度
                confidenceView(confidence: inference.confidence)

                // 推理依据（如果有）
                if let reasoning = inference.reasoning, !reasoning.isEmpty {
                    reasoningView(reasoning: reasoning)
                }

                // 操作按钮
                actionButtons

                Spacer()
                    .frame(height: 32)
            }
            .padding(.horizontal, 16)
        }
    }

    // MARK: - Availability Toggle

    private var availabilityToggle: some View {
        Button {
            viewModel.editingIsAvailable.toggle()
        } label: {
            HStack(spacing: 12) {
                Text(viewModel.editingIsAvailable ? "👑" : "💤")
                    .font(.system(size: 24))

                VStack(alignment: .leading, spacing: 2) {
                    Text(viewModel.editingIsAvailable ? "有空" : "忙碌")
                        .font(.cliBody)
                        .foregroundColor(viewModel.editingIsAvailable ? CLIColors.yellow : CLIColors.textSecondary)

                    Text(viewModel.editingIsAvailable ? "好友可以看到你有空" : "暂时不方便被打扰")
                        .font(.cliCaptionSmall)
                        .foregroundColor(CLIColors.textWeak)
                }

                Spacer()

                Text(viewModel.editingIsAvailable ? "[ON]" : "[OFF]")
                    .font(.cliBodySmall)
                    .foregroundColor(viewModel.editingIsAvailable ? CLIColors.green : CLIColors.textWeak)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(
                RoundedRectangle(cornerRadius: 4)
                    .stroke(viewModel.editingIsAvailable ? CLIColors.yellow : CLIColors.border, lineWidth: 1)
                    .background(
                        RoundedRectangle(cornerRadius: 4)
                            .fill(viewModel.editingIsAvailable ? CLIColors.yellow.opacity(0.1) : CLIColors.backgroundSecondary)
                    )
            )
        }
    }

    // MARK: - Confidence View

    private func confidenceView(confidence: String) -> some View {
        HStack(spacing: 8) {
            Text("置信度:")
                .font(.cliCaptionSmall)
                .foregroundColor(CLIColors.textWeak)

            Text(confidenceText(confidence))
                .font(.cliCaptionSmall)
                .foregroundColor(confidenceColor(confidence))
        }
    }

    private func confidenceText(_ confidence: String) -> String {
        switch confidence {
        case "high": return "高"
        case "medium": return "中"
        case "low": return "低"
        default: return confidence
        }
    }

    private func confidenceColor(_ confidence: String) -> Color {
        switch confidence {
        case "high": return CLIColors.green
        case "medium": return CLIColors.yellow
        case "low": return CLIColors.textWeak
        default: return CLIColors.textSecondary
        }
    }

    // 拆分 emoji 字符串为单个 emoji 数组
    private func splitEmojis(_ str: String) -> [String] {
        str.map { String($0) }
    }

    // 根据 emoji 数量计算字体大小
    private func emojiSize(for count: Int) -> CGFloat {
        switch count {
        case 0, 1: return 72
        default: return 56 // 2个
        }
    }

    // MARK: - Reasoning View

    private func reasoningView(reasoning: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("推理依据:")
                .font(.cliCaptionSmall)
                .foregroundColor(CLIColors.textWeak)

            Text(reasoning)
                .font(.cliCaptionSmall)
                .foregroundColor(CLIColors.textSecondary)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(
                    RoundedRectangle(cornerRadius: 4)
                        .fill(CLIColors.backgroundSecondary)
                )
        }
    }

    // MARK: - Emoji/GIF Toggle

    private var emojiGifToggle: some View {
        HStack(spacing: 0) {
            // Emoji 按钮
            Button {
                viewModel.useGif = false
            } label: {
                Text("😀")
                    .font(.system(size: 20))
                    .frame(width: 56, height: 36)
                    .background(
                        RoundedRectangle(cornerRadius: 4)
                            .stroke(!viewModel.useGif ? CLIColors.green : CLIColors.border, lineWidth: !viewModel.useGif ? 2 : 1)
                            .background(
                                RoundedRectangle(cornerRadius: 4)
                                    .fill(!viewModel.useGif ? CLIColors.green.opacity(0.1) : Color.clear)
                            )
                    )
            }

            Spacer().frame(width: 8)

            // GIF 按钮
            Button {
                viewModel.toggleGifMode()
            } label: {
                HStack(spacing: 4) {
                    if viewModel.isSearchingGif {
                        ProgressView()
                            .tint(CLIColors.green)
                            .scaleEffect(0.6)
                    }
                    Text("GIF")
                        .font(.cliBodySmall)
                        .fontWeight(.bold)
                        .foregroundColor(viewModel.useGif ? CLIColors.green : CLIColors.textSecondary)
                }
                .frame(width: 56, height: 36)
                .background(
                    RoundedRectangle(cornerRadius: 4)
                        .stroke(viewModel.useGif ? CLIColors.green : CLIColors.border, lineWidth: viewModel.useGif ? 2 : 1)
                        .background(
                            RoundedRectangle(cornerRadius: 4)
                                .fill(viewModel.useGif ? CLIColors.green.opacity(0.1) : Color.clear)
                        )
                )
            }
        }
    }

    // MARK: - Action Buttons

    private var actionButtons: some View {
        VStack(spacing: 12) {
            // 确认发布按钮
            Button {
                Task {
                    await viewModel.confirmStatus()
                    if viewModel.isConfirmed {
                        onStatusConfirmed?(viewModel.editingEmoji, viewModel.editingActivity, viewModel.editingIsAvailable)
                        dismiss()
                    }
                }
            } label: {
                HStack {
                    if viewModel.isConfirming {
                        ProgressView()
                            .tint(CLIColors.background)
                            .scaleEffect(0.8)
                    } else {
                        Text("✓")
                    }
                    Text(viewModel.isConfirming && !viewModel.confirmingMessage.isEmpty
                         ? viewModel.confirmingMessage
                         : (viewModel.hasChanges ? "确认修改并发布" : "确认发布"))
                }
                .font(.cliBody)
                .foregroundColor(CLIColors.background)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 12)
                .background(CLIColors.green)
                .cornerRadius(4)
            }
            .disabled(viewModel.isConfirming)
        }
        .padding(.top, 16)
    }

    // MARK: - Error View

    private func errorView(error: String) -> some View {
        VStack(spacing: 20) {
            Spacer()

            Text("❌")
                .font(.system(size: 64))

            Text("推断失败")
                .font(.cliBody)
                .foregroundColor(CLIColors.red)

            Text(error)
                .font(.cliCaptionSmall)
                .foregroundColor(CLIColors.textSecondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 32)

            Button {
                Task {
                    await viewModel.startInference()
                }
            } label: {
                Text("[重试]")
                    .font(.cliBody)
                    .foregroundColor(CLIColors.cyan)
                    .padding(.horizontal, 24)
                    .padding(.vertical, 12)
                    .overlay(
                        RoundedRectangle(cornerRadius: 4)
                            .stroke(CLIColors.cyan, lineWidth: 1)
                    )
            }

            Spacer()
        }
    }
}

// MARK: - AI Status Inference View Model

@MainActor
class AIStatusInferenceViewModel: ObservableObject {
    @Published var isInferring = false
    @Published var inference: CurrentStatusInference?
    @Published var error: String?

    // 编辑状态
    @Published var editingEmoji = ""
    @Published var editingActivity = ""
    @Published var editingPlace = ""
    @Published var editingIsAvailable = false
    @Published var showEmojiPicker = false
    @Published var isEditingActivity = false

    // 确认状态
    @Published var isConfirming = false
    @Published var confirmingMessage = ""
    @Published var isConfirmed = false

    // GIF 模式切换
    @Published var useGif = false
    @Published var isSearchingGif = false

    // V2 流式状态
    @Published var streamingPhase = ""
    @Published var streamingLogs: [String] = []

    // V2 用户确认状态
    @Published var isAskingUser = false
    @Published var askUserQuestion = ""
    @Published var askUserOptions: [String] = []
    @Published var askUserContext: String?
    private let agentRepository = AgentRepositoryImpl()
    private let deviceCollector = DeviceStatusCollector.shared
    private let locationCollector = LocationDataCollector.shared
    private let calendarCollector = CalendarDataCollector.shared
    private let movementCollector = MovementDataCollector.shared

    /// 是否有修改
    var hasChanges: Bool {
        guard let inference = inference else { return false }
        return editingEmoji != inference.emoji ||
               editingActivity != inference.activity ||
               editingIsAvailable != inference.isAvailable ||
               editingPlace != (inference.place ?? "") ||
               useGif // GIF 模式切换也算修改
    }

    // MARK: - V3 推断相关
    private var v3SessionId: String?

    // MARK: - Start Inference (V3)

    func startInference() async {
        guard !isInferring else { return }

        isInferring = true
        error = nil
        inference = nil
        streamingPhase = ""
        streamingLogs = []
        isAskingUser = false
        v3SessionId = nil

        // 1. 确保位置数据已启动
        print("📍 [AIInference] 获取位置数据...")
        locationCollector.startMonitoring()
        try? await Task.sleep(nanoseconds: 500_000_000)

        // 2. 构建传感器数据
        let sensorData = buildSensorData()

        // 3. 展示收集到的传感器线索
        streamingPhase = "正在收集数据..."
        streamingLogs.append("> 正在收集设备数据...")
        appendSensorClues(sensorData)
        await Task.yield() // 让 UI 渲染传感器线索

        // 4. 调用 V3 同步推断接口
        do {
            streamingPhase = "正在推断状态..."
            streamingLogs.append("▸ 正在发送数据到 AI...")
            await Task.yield()

            let endpoint = APIEndpoint.inferStatusV2(request: sensorData)
            let urlRequest = try buildAPIRequest(for: endpoint)

            streamingLogs.append("▸ AI 正在分析...")

            let (data, response) = try await URLSession.shared.data(for: urlRequest)

            guard let httpResponse = response as? HTTPURLResponse,
                  httpResponse.statusCode == 200 else {
                throw SSEError.httpError(statusCode: (response as? HTTPURLResponse)?.statusCode ?? 0)
            }

            // 解析 V3 响应
            let decoder = JSONDecoder()
            let apiResponse = try decoder.decode(APIResponse<InferenceResponse>.self, from: data)

            guard let inferResp = apiResponse.data else {
                throw SSEError.httpError(statusCode: -1)
            }

            if inferResp.phase == "completed", let result = inferResp.result {
                // 逐条展示推理过程
                streamingLogs.append("▸ 推理完成")
                await Task.yield()

                if let reasoning = result.reasoning, !reasoning.isEmpty {
                    streamingLogs.append("  │ \(reasoning)")
                    await Task.yield()
                }
                let confLabel = result.confidence == "high" ? "高" : result.confidence == "medium" ? "中" : "低"
                streamingLogs.append("  │ 置信度: \(confLabel)")
                await Task.yield()

                streamingLogs.append("  ✓ \(result.emoji) \(result.activity)")
                print("✅ [AIInference] V3 推断完成: \(result.emoji) \(result.activity)")

                // 延迟 1.5s 让用户看到推理过程，再切到结果页
                try? await Task.sleep(nanoseconds: 1_500_000_000)
                applyInferenceResult(result)
                isConfirmed = true
            } else if inferResp.phase == "awaiting_choice",
                      let options = inferResp.options, !options.isEmpty {
                // 信号矛盾，需要用户选择
                v3SessionId = inferResp.sessionId
                askUserQuestion = inferResp.question ?? "AI 不太确定，请选择："
                askUserOptions = options.map { "\($0.emoji) \($0.activity)" }
                isAskingUser = true
                streamingLogs.append("  ? 信号矛盾，需要你来选择")
                for opt in options {
                    let reason = opt.reason.map { " - \($0)" } ?? ""
                    streamingLogs.append("    \(opt.emoji) \(opt.activity)\(reason)")
                }
                print("🤔 [AIInference] V3 等待用户选择: \(options.count) 个选项")
            } else {
                self.error = "推断返回了未知格式"
            }
        } catch {
            self.error = error.localizedDescription
            print("❌ [AIInference] V3 推断失败: \(error)")
        }

        isInferring = false
    }

    // MARK: - Respond to Ask User (V3 选项选择)

    func respondToAskUser(answer: String) async {
        guard let sessionId = v3SessionId else { return }

        // 从 answer 文本找到选项 index
        let selectedIndex = askUserOptions.firstIndex(of: answer) ?? 0

        isAskingUser = false
        isInferring = true
        streamingPhase = "正在确认选择..."
        streamingLogs.append("▸ 已选择: \(answer)")

        do {
            let endpoint = APIEndpoint.inferStatusV3Respond(sessionId: sessionId, selectedIndex: selectedIndex)
            let urlRequest = try buildAPIRequest(for: endpoint)
            let (data, _) = try await URLSession.shared.data(for: urlRequest)

            let decoder = JSONDecoder()
            let apiResponse = try decoder.decode(APIResponse<InferenceResponse>.self, from: data)

            if let result = apiResponse.data?.result {
                applyInferenceResult(result)
                isConfirmed = true
                streamingLogs.append("  ✓ \(result.emoji) \(result.activity)")
            }
        } catch {
            self.error = error.localizedDescription
            print("❌ [AIInference] V3 respond 失败: \(error)")
        }

        isInferring = false
    }

    // MARK: - Sensor Clues

    private func appendSensorClues(_ data: StatusReportRequest) {
        if let loc = data.extendedLocation {
            let placeLabel: String
            switch loc.placeType {
            case "home": placeLabel = "家中"
            case "work": placeLabel = "公司"
            case "leisure": placeLabel = "休闲场所"
            case "transit": placeLabel = "移动中"
            default: placeLabel = loc.placeName ?? "未知"
            }
            streamingLogs.append("  > 位置: \(placeLabel)")
        }
        if let battery = data.battery {
            let state = battery.isCharging ? "充电中" : "未充电"
            streamingLogs.append("  > 电量: \(battery.batteryLevel)% (\(state))")
        }
        if let movement = data.movement {
            let movLabel = movement.isMoving ? "运动中" : "静止"
            streamingLogs.append("  > 运动: \(movLabel), 今日 \(movement.stepsToday) 步")
        }
        if let conn = data.connection {
            let net = conn.networkType
            let headphone = conn.isHeadphonesConnected ? ", 耳机已连接" : ""
            streamingLogs.append("  > 网络: \(net)\(headphone)")
        }
        if let cal = data.calendar, cal.hasCurrentEvent {
            let title = cal.currentEventTitle ?? "日程"
            streamingLogs.append("  > 日历: 正在进行「\(title)」")
        }
        if let mode = data.mode {
            if mode.isFocusModeOn { streamingLogs.append("  > 专注模式: 开启") }
            if mode.isLowPowerMode { streamingLogs.append("  > 低电量模式: 开启") }
        }
    }

    // MARK: - Toggle GIF Mode

    func toggleGifMode() {
        useGif = true
        // 切换到 GIF 模式时，如果还没搜过 GIF，触发搜索
        if inference?.gifUrl == nil {
            if let query = inference?.giphyQuery, !query.isEmpty {
                Task { await searchGiphy(query: query) }
            } else {
                // 用活动描述作为搜索词
                let query = inference?.activity ?? editingActivity
                if !query.isEmpty {
                    Task { await searchGiphy(query: query) }
                }
            }
        }
    }

    // MARK: - Giphy Client Search

    private func searchGiphy(query: String) async {
        isSearchingGif = true
        let apiKey = "Ned8bTKUTSlYen6oZXmacc1sLmlLn9U8"
        let encoded = query.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? query
        let urlStr = "https://api.giphy.com/v1/gifs/search?api_key=\(apiKey)&q=\(encoded)&limit=1&rating=g"

        guard let url = URL(string: urlStr) else {
            isSearchingGif = false
            return
        }

        do {
            let (data, _) = try await URLSession.shared.data(from: url)
            if let json = try JSONSerialization.jsonObject(with: data) as? [String: Any],
               let dataArray = json["data"] as? [[String: Any]],
               let first = dataArray.first,
               let images = first["images"] as? [String: Any],
               let fixedHeight = images["fixed_height"] as? [String: Any],
               let gifUrl = fixedHeight["url"] as? String {
                print("✅ [Giphy] GIF URL: \(gifUrl)")
                inference?.gifUrl = gifUrl
            }
        } catch {
            print("⚠️ [Giphy] 搜索失败: \(error.localizedDescription)")
        }
        isSearchingGif = false
    }

    private func applyInferenceResult(_ result: CurrentStatusInference) {
        inference = result
        editingEmoji = String(result.emoji.prefix(2))
        editingActivity = result.activity
        editingPlace = result.place ?? ""
        editingIsAvailable = result.isAvailable
    }

    // MARK: - Confirm Status

    func confirmStatus() async {
        guard !isConfirming else { return }
        guard inference != nil else {
            self.error = "没有可发布的状态"
            return
        }

        // 推断结果不再自动保存，必须通过 feedback 接口发布
        isConfirming = true
        confirmingMessage = "发布中..."

        do {
            // 如果使用 GIF 模式且 URL 是 Giphy 的，先上传到 COS
            var finalGifUrl = inference?.gifUrl
            if useGif, let gifUrl = finalGifUrl, gifUrl.contains("giphy.com") {
                confirmingMessage = "正在上传 GIF..."
                if let cosUrl = await uploadGifToCOS(gifUrl: gifUrl) {
                    finalGifUrl = cosUrl
                    inference?.gifUrl = cosUrl
                }
            }

            let request = StatusFeedbackRequest(
                originalEmoji: inference?.emoji,
                originalActivity: inference?.activity,
                correctedEmoji: editingEmoji,
                correctedActivity: editingActivity,
                correctedPlace: editingPlace.isEmpty ? nil : editingPlace,
                correctedIsAvailable: editingIsAvailable,
                gifUrl: finalGifUrl,
                giphyQuery: inference?.giphyQuery,
                useGif: useGif
            )
            try await agentRepository.submitStatusFeedback(request: request)
            isConfirmed = true
        } catch {
            self.error = error.localizedDescription
        }

        isConfirming = false
        confirmingMessage = ""
    }

    // MARK: - Upload GIF to COS (STS 直传)

    /// 下载 Giphy GIF → 获取 STS 临时凭证 → 直传 COS，返回 COS URL
    private func uploadGifToCOS(gifUrl: String) async -> String? {
        guard let url = URL(string: gifUrl) else { return nil }

        do {
            // 1. 下载 GIF 数据
            let (gifData, _) = try await URLSession.shared.data(from: url)

            // 2. 获取 STS 临时凭证
            let stsResult = try await agentRepository.getSTSCredentials()
            let cred = stsResult.sts

            // 3. 生成 COS key (UUID 去重)
            let md5Hex = UUID().uuidString.replacingOccurrences(of: "-", with: "").lowercased()
            let objectKey = "\(cred.prefix)\(md5Hex).gif"

            // 4. 构建 COS PUT 请求
            let host = "\(cred.bucket).cos.\(cred.region).myqcloud.com"
            let cosURL = "https://\(host)/\(objectKey)"
            guard let uploadURL = URL(string: cosURL) else { return nil }

            let timestamp = Int(Date().timeIntervalSince1970)
            let signature = generateCOSSignature(
                secretId: cred.accessKeyId,
                secretKey: cred.secretAccessKey,
                method: "PUT",
                path: "/\(objectKey)",
                host: host,
                timestamp: timestamp
            )

            var request = URLRequest(url: uploadURL)
            request.httpMethod = "PUT"
            request.setValue(host, forHTTPHeaderField: "Host")
            request.setValue("image/gif", forHTTPHeaderField: "Content-Type")
            request.setValue("public-read", forHTTPHeaderField: "x-cos-acl")
            request.setValue(cred.sessionToken, forHTTPHeaderField: "x-cos-security-token")
            request.setValue(signature, forHTTPHeaderField: "Authorization")
            request.httpBody = gifData

            // 5. 上传
            let (_, response) = try await URLSession.shared.data(for: request)
            let statusCode = (response as? HTTPURLResponse)?.statusCode ?? 0
            if (200...299).contains(statusCode) {
                return cosURL
            } else {
                print("⚠️ [UploadGif] COS 返回 HTTP \(statusCode)")
            }
        } catch {
            print("⚠️ [UploadGif] 失败: \(error.localizedDescription)")
        }
        return nil
    }

    /// COS 签名算法（参考 CastReader）
    private func generateCOSSignature(secretId: String, secretKey: String, method: String, path: String, host: String, timestamp: Int) -> String {
        let keyTime = "\(timestamp);\(timestamp + 3600)"
        let signKey = hmacSHA1(key: secretKey, data: keyTime)
        let httpString = "\(method.lowercased())\n\(path)\n\nhost=\(host)\n"
        let sha1HttpString = sha1Hex(httpString)
        let stringToSign = "sha1\n\(keyTime)\n\(sha1HttpString)\n"
        let signature = hmacSHA1(key: signKey, data: stringToSign)
        return "q-sign-algorithm=sha1&q-ak=\(secretId)&q-sign-time=\(keyTime)&q-key-time=\(keyTime)&q-header-list=host&q-url-param-list=&q-signature=\(signature)"
    }

    private func hmacSHA1(key: String, data: String) -> String {
        let keyData = key.data(using: .utf8)!
        let dataData = data.data(using: .utf8)!
        var result = [UInt8](repeating: 0, count: Int(CC_SHA1_DIGEST_LENGTH))
        keyData.withUnsafeBytes { keyBytes in
            dataData.withUnsafeBytes { dataBytes in
                CCHmac(CCHmacAlgorithm(kCCHmacAlgSHA1),
                        keyBytes.baseAddress, keyData.count,
                        dataBytes.baseAddress, dataData.count,
                        &result)
            }
        }
        return result.map { String(format: "%02x", $0) }.joined()
    }

    private func sha1Hex(_ input: String) -> String {
        let data = input.data(using: .utf8)!
        var digest = [UInt8](repeating: 0, count: Int(CC_SHA1_DIGEST_LENGTH))
        data.withUnsafeBytes { bytes in
            _ = CC_SHA1(bytes.baseAddress, CC_LONG(data.count), &digest)
        }
        return digest.map { String(format: "%02x", $0) }.joined()
    }

    // MARK: - API Helpers

    private func buildAPIRequest(for endpoint: APIEndpoint) throws -> URLRequest {
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

    // MARK: - Build Sensor Data

    private func buildSensorData() -> StatusReportRequest {
        let deviceStatus = deviceCollector.currentStatus
        let locationStatus = locationCollector.currentStatus

        return StatusReportRequest(
            screen: nil,
            location: LocationRequestData(
                placeType: locationStatus.placeType.rawValue,
                atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
                city: nil
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
}

// MARK: - Preview

#Preview {
    AIStatusInferenceView()
}
