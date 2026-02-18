import CommonCrypto
import CryptoKit
import SwiftUI

// MARK: - Inference Phase

enum InferencePhase {
    case loading    // 正在推断
    case options    // 展示 4 选 1 网格
    case editing    // 编辑发布（选中选项后）
}

// MARK: - AI Status Inference View

/// AI 推断当下状态的视图
/// 用户点击后调用 AI 推断 4 个选项，选择后进入编辑发布页
struct AIStatusInferenceView: View {
    @StateObject private var viewModel = AIStatusInferenceViewModel()
    @Environment(\.dismiss) private var dismiss

    /// 状态确认后的回调
    var onStatusConfirmed: ((String, String, Bool) -> Void)?

    init(onStatusConfirmed: ((String, String, Bool) -> Void)? = nil) {
        self.onStatusConfirmed = onStatusConfirmed
    }

    var body: some View {
        VStack(spacing: 0) {
            // CLI Header
            headerView

            Rectangle()
                .fill(CLIColors.border)
                .frame(height: 1)

            // Content
            switch viewModel.currentPhase {
            case .loading:
                inferringView
            case .options:
                optionsGridView
            case .editing:
                if let inference = viewModel.inference {
                    resultView(inference: inference)
                } else {
                    inferringView
                }
            }

            // Error overlay
            if let error = viewModel.error, viewModel.currentPhase == .loading {
                errorView(error: error)
            }
        }
        .background(CLIColors.background)
        .onAppear {
            Task {
                await viewModel.startOptionsInference()
            }
        }
    }

    // MARK: - Header

    private var headerView: some View {
        HStack {
            Button {
                if viewModel.currentPhase == .editing {
                    // 从编辑页返回选项页
                    viewModel.backToOptions()
                } else {
                    dismiss()
                }
            } label: {
                Text(viewModel.currentPhase == .editing ? "[返回]" : "[X]")
                    .font(.cliBodySmall)
                    .foregroundColor(CLIColors.textSecondary)
            }

            Spacer()

            Text(viewModel.currentPhase == .editing ? "━━ 编辑发布 ━━" : "━━ AI 推断当下状态 ━━")
                .font(.cliHeadline)
                .foregroundColor(CLIColors.cyan)

            Spacer()

            // Placeholder for balance
            Text("[返回]")
                .font(.cliBodySmall)
                .foregroundColor(.clear)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(CLIColors.background)
    }

    // MARK: - Inferring View (Loading)

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

    // MARK: - Options Grid View (4选1)

    private var optionsGridView: some View {
        ScrollView {
            VStack(spacing: 16) {
                Text("AI 为你推断了以下可能的状态")
                    .font(.cliBody)
                    .foregroundColor(CLIColors.textSecondary)
                    .padding(.top, 16)

                // 2×2 网格
                let columns = [GridItem(.flexible(), spacing: 12), GridItem(.flexible(), spacing: 12)]
                LazyVGrid(columns: columns, spacing: 12) {
                    ForEach(viewModel.statusOptions) { option in
                        StatusOptionCard(
                            option: option,
                            isSelected: viewModel.selectedIndex == option.index,
                            isGifLoading: viewModel.gifLoadingIndices.contains(option.index)
                        )
                        .onTapGesture {
                            withAnimation(.easeInOut(duration: 0.2)) {
                                viewModel.selectOption(option.index)
                            }
                        }
                    }
                }
                .padding(.horizontal, 16)

                // 底部按钮
                VStack(spacing: 12) {
                    // 确认选择
                    Button {
                        viewModel.confirmSelection()
                    } label: {
                        HStack {
                            Text("✓")
                            Text("确认选择")
                        }
                        .font(.cliBody)
                        .foregroundColor(viewModel.selectedIndex != nil ? CLIColors.background : CLIColors.textWeak)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .background(viewModel.selectedIndex != nil ? CLIColors.green : CLIColors.backgroundSecondary)
                        .cornerRadius(4)
                    }
                    .disabled(viewModel.selectedIndex == nil)

                    // 换一批
                    Button {
                        Task {
                            await viewModel.refreshOptions()
                        }
                    } label: {
                        HStack(spacing: 6) {
                            if viewModel.isRefreshing {
                                ProgressView()
                                    .tint(CLIColors.cyan)
                                    .scaleEffect(0.7)
                            }
                            Text("换一批")
                        }
                        .font(.cliBody)
                        .foregroundColor(CLIColors.cyan)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .overlay(
                            RoundedRectangle(cornerRadius: 4)
                                .stroke(CLIColors.cyan, lineWidth: 1)
                        )
                    }
                    .disabled(viewModel.isRefreshing)
                }
                .padding(.horizontal, 16)
                .padding(.top, 8)

                Spacer()
                    .frame(height: 32)
            }
        }
    }

    // MARK: - Result View (编辑发布)

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
        .scrollDismissesKeyboard(.immediately)
        .dismissKeyboardOnTouchDown()
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
                    await viewModel.startOptionsInference()
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

// MARK: - Status Option Card

struct StatusOptionCard: View {
    let option: StatusCardOption
    let isSelected: Bool
    var isGifLoading: Bool = false

    var body: some View {
        GeometryReader { geo in
            ZStack {
                // GIF 背景（填满卡片，和安卓一致）
                if let gifUrl = option.gifUrl, !gifUrl.isEmpty, let url = URL(string: gifUrl) {
                    GifImageView(url: url, contentMode: .scaleAspectFill) {}
                        .frame(width: geo.size.width, height: geo.size.height)
                        .clipped()
                } else {
                    // 无 GIF 时用渐变背景
                    LinearGradient(
                        colors: [CLIColors.backgroundSecondary, CLIColors.background],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                }

                // 暗化遮罩（始终有）
                Color.black.opacity(0.35)

                // 内容
                VStack(spacing: 6) {
                    // 动画 Emoji（从 COS 加载 APNG）
                    EmojiStateView(emoji: option.emoji)
                        .frame(width: 56, height: 56)
                        .shadow(color: .black.opacity(0.5), radius: 2, x: 0, y: 1)
                    Text(option.activity)
                        .font(.system(size: 14, weight: .bold))
                        .foregroundColor(.white)
                        .shadow(color: .black.opacity(0.7), radius: 2, x: 0, y: 1)
                        .multilineTextAlignment(.center)
                        .lineLimit(2)
                }
                .padding(8)

                // GIF 加载中指示器（右下角）
                if isGifLoading && (option.gifUrl == nil || option.gifUrl?.isEmpty == true) {
                    VStack {
                        Spacer()
                        HStack {
                            Spacer()
                            HStack(spacing: 3) {
                                ProgressView()
                                    .tint(.white)
                                    .scaleEffect(0.5)
                                Text("GIF")
                                    .font(.system(size: 9, design: .monospaced))
                                    .foregroundColor(.white.opacity(0.8))
                            }
                            .padding(.horizontal, 6)
                            .padding(.vertical, 3)
                            .background(Color.black.opacity(0.5))
                            .cornerRadius(8)
                            .padding(6)
                        }
                    }
                }
            }
        }
        .aspectRatio(1, contentMode: .fit)
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .overlay(
            RoundedRectangle(cornerRadius: 16)
                .stroke(isSelected ? CLIColors.green : Color.clear, lineWidth: 3)
        )
        .shadow(color: isSelected ? CLIColors.green.opacity(0.3) : .clear, radius: 8)
    }
}

// MARK: - AI Status Inference View Model

@MainActor
class AIStatusInferenceViewModel: ObservableObject {
    // 阶段
    @Published var currentPhase: InferencePhase = .loading

    // 4选1 选项
    @Published var statusOptions: [StatusCardOption] = []
    @Published var selectedIndex: Int? = nil
    @Published var inferenceSessionId = ""
    @Published var isRefreshing = false

    // 编辑状态
    @Published var inference: CurrentStatusInference?
    @Published var error: String?
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

    // GIF 加载状态
    @Published var gifLoadingIndices: Set<Int> = []

    // 流式状态
    @Published var streamingPhase = ""
    @Published var streamingLogs: [String] = []

    private let agentRepository = AgentRepositoryImpl()
    private let deviceCollector = DeviceStatusCollector.shared
    private let locationCollector = LocationDataCollector.shared
    private let calendarCollector = CalendarDataCollector.shared
    private let movementCollector = MovementDataCollector.shared

    // 已展示过的活动（用于换一批排除）
    private var shownActivities: [String] = []

    /// 是否有修改
    var hasChanges: Bool {
        guard let inference = inference else { return false }
        return editingEmoji != inference.emoji ||
               editingActivity != inference.activity ||
               editingIsAvailable != inference.isAvailable ||
               editingPlace != (inference.place ?? "") ||
               useGif // GIF 模式切换也算修改
    }

    // MARK: - Start Options Inference (4选1)

    func startOptionsInference() async {
        currentPhase = .loading
        error = nil
        statusOptions = []
        selectedIndex = nil
        shownActivities = []
        streamingPhase = ""
        streamingLogs = []

        // 1. 确保位置数据已启动
        print("📍 [AIInference] 获取位置数据...")
        locationCollector.startMonitoring()
        try? await Task.sleep(nanoseconds: 500_000_000)

        // 2. 构建传感器数据
        let sensorData = await buildInferOptionsRequest(excludeActivities: nil, sessionId: nil)

        // 3. 展示收集到的传感器线索
        streamingPhase = "正在收集数据..."
        streamingLogs.append("> 正在收集设备数据...")
        appendSensorClues(sensorData)
        await Task.yield()

        // 4. 调用 4选1 推断接口
        do {
            streamingPhase = "正在推断状态..."
            streamingLogs.append("▸ 正在发送数据到 AI...")
            await Task.yield()

            streamingLogs.append("▸ AI 正在生成 4 个选项...")

            let response = try await agentRepository.inferOptions(request: sensorData)

            inferenceSessionId = response.sessionId
            statusOptions = response.options

            // 记录已展示的活动
            for opt in response.options {
                shownActivities.append(opt.activity)
            }

            streamingLogs.append("  ✓ 已生成 \(response.options.count) 个选项")
            print("✅ [AIInference] 4选1 推断完成: \(response.options.count) 个选项, sessionId=\(response.sessionId)")

            // 切到选项页
            print("🔀 [AIInference] 切换到 options 阶段, statusOptions.count=\(statusOptions.count)")
            currentPhase = .options

            // 异步获取 GIF（不阻塞 UI 切换）
            Task { await fetchGifsForOptions() }
        } catch {
            self.error = error.localizedDescription
            print("❌ [AIInference] 4选1 推断失败: \(error)")
        }
    }

    // MARK: - Refresh Options (换一批)

    func refreshOptions() async {
        guard !isRefreshing else { return }
        isRefreshing = true
        selectedIndex = nil

        do {
            let request = await buildInferOptionsRequest(
                excludeActivities: shownActivities,
                sessionId: inferenceSessionId
            )

            let response = try await agentRepository.inferOptions(request: request)

            inferenceSessionId = response.sessionId
            statusOptions = response.options

            // 追加已展示的活动
            for opt in response.options {
                shownActivities.append(opt.activity)
            }

            print("✅ [AIInference] 换一批完成: \(response.options.count) 个选项")

            // 异步获取 GIF
            Task { await fetchGifsForOptions() }
        } catch {
            print("❌ [AIInference] 换一批失败: \(error)")
        }

        isRefreshing = false
    }

    // MARK: - Select Option

    func selectOption(_ index: Int) {
        if selectedIndex == index {
            selectedIndex = nil
        } else {
            selectedIndex = index
        }
    }

    // MARK: - Confirm Selection → 进入编辑页

    func confirmSelection() {
        guard let idx = selectedIndex,
              let option = statusOptions.first(where: { $0.index == idx }) else { return }

        // 用选中的选项构建 CurrentStatusInference
        inference = CurrentStatusInference(
            emoji: option.emoji,
            activity: option.activity,
            place: option.place,
            isAvailable: option.isAvailable,
            confidence: option.confidence,
            gifUrl: option.gifUrl,
            giphyQuery: option.giphyQuery
        )

        // 设置编辑初始值
        editingEmoji = String(option.emoji.prefix(2))
        editingActivity = option.activity
        editingPlace = option.place ?? ""
        editingIsAvailable = option.isAvailable

        // 如果有 GIF URL，默认使用 GIF 模式
        if option.gifUrl != nil && !(option.gifUrl?.isEmpty ?? true) {
            useGif = true
        }

        currentPhase = .editing
    }

    // MARK: - Back to Options

    func backToOptions() {
        currentPhase = .options
    }

    // MARK: - Confirm Status (发布)

    func confirmStatus() async {
        guard !isConfirming else { return }
        guard inference != nil else {
            self.error = "没有可发布的状态"
            return
        }

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
                useGif: useGif,
                inferenceSessionId: inferenceSessionId.isEmpty ? nil : inferenceSessionId,
                selectedOptionIdx: selectedIndex
            )
            try await agentRepository.submitStatusFeedback(request: request)
            isConfirmed = true
        } catch {
            self.error = error.localizedDescription
        }

        isConfirming = false
        confirmingMessage = ""
    }

    // MARK: - Toggle GIF Mode

    func toggleGifMode() {
        useGif = true
        // 切换到 GIF 模式时，如果还没搜过 GIF，触发搜索
        if inference?.gifUrl == nil {
            if let query = inference?.giphyQuery, !query.isEmpty {
                Task { await searchGiphy(query: query) }
            } else {
                let query = inference?.activity ?? editingActivity
                if !query.isEmpty {
                    Task { await searchGiphy(query: query) }
                }
            }
        }
    }

    // MARK: - GIF Fetch via gif.playa.cn Proxy (和首页一致)

    /// 为 4 个选项串行获取 GIF（和安卓/首页一致的 gif.playa.cn 代理方式）
    private func fetchGifsForOptions() async {
        let optionsSnapshot = statusOptions
        let needGif = optionsSnapshot.filter { ($0.gifUrl ?? "").isEmpty && !$0.giphyQuery.isEmpty }
        gifLoadingIndices = Set(needGif.map { $0.index })

        for option in needGif {
            let query = option.giphyQuery.isEmpty ? option.activity : option.giphyQuery
            let seed = "opt_\(option.index)_\(query)_\(Int(Date().timeIntervalSince1970))"
            print("🎬 [GIF] 搜索: index=\(option.index) query=\(query)")
            let gifUrl = await fetchGifViaProxy(query: query, seed: seed)
            print("🎬 [GIF] 结果: index=\(option.index) url=\(gifUrl?.prefix(60) ?? "nil")")

            if let gifUrl = gifUrl {
                if let idx = statusOptions.firstIndex(where: { $0.index == option.index }) {
                    statusOptions[idx].gifUrl = gifUrl
                }
            }
            gifLoadingIndices.remove(option.index)
        }
    }

    /// 通过 gif.playa.cn 代理获取 GIF cos_url（和首页 GridHomeViewModel 一致）
    /// 失败自动重试 2 次，间隔递增
    private func fetchGifViaProxy(query: String, seed: String) async -> String? {
        let encoded = query.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? query
        let seedHash = md5String("\(seed)\(query)")

        for attempt in 0..<3 {
            do {
                guard let url = URL(string: "https://gif.playa.cn/api/giphy?q=\(encoded)&seed=\(seedHash)") else { return nil }
                var request = URLRequest(url: url)
                request.timeoutInterval = 15
                let (data, _) = try await URLSession.shared.data(for: request)
                let body = String(data: data, encoding: .utf8) ?? ""

                // 检查超时错误
                if body.contains("FUNCTION_INVOCATION_TIMEOUT") || body.contains("\"error\"") {
                    print("⚠️ [GIF] 代理超时 attempt=\(attempt) query=\(query)")
                    if attempt < 2 {
                        let delay = UInt64((attempt + 1) * 1_500_000_000) // 1.5s, 3s
                        try? await Task.sleep(nanoseconds: delay)
                        continue
                    }
                    return nil
                }

                if let json = try JSONSerialization.jsonObject(with: data) as? [String: Any],
                   let result = json["result"] as? [String: Any],
                   let cosUrl = result["cos_url"] as? String, !cosUrl.isEmpty {
                    return cosUrl
                }
                // JSON 解析成功但没有 cos_url，重试无意义
                print("⚠️ [GIF] 代理返回无 cos_url attempt=\(attempt)")
                return nil
            } catch {
                print("⚠️ [GIF] 代理请求失败 attempt=\(attempt): \(error.localizedDescription)")
                if attempt < 2 {
                    let delay = UInt64((attempt + 1) * 1_500_000_000)
                    try? await Task.sleep(nanoseconds: delay)
                }
            }
        }
        return nil
    }

    /// 编辑页搜索 GIF（也走 gif.playa.cn 代理）
    private func searchGiphy(query: String) async {
        isSearchingGif = true
        let seed = "edit_\(Int(Date().timeIntervalSince1970))"
        if let gifUrl = await fetchGifViaProxy(query: query, seed: seed) {
            inference?.gifUrl = gifUrl
        }
        isSearchingGif = false
    }

    private func md5String(_ input: String) -> String {
        let data = input.data(using: .utf8)!
        var digest = [UInt8](repeating: 0, count: Int(CC_MD5_DIGEST_LENGTH))
        data.withUnsafeBytes { bytes in
            _ = CC_MD5(bytes.baseAddress, CC_LONG(data.count), &digest)
        }
        return digest.map { String(format: "%02x", $0) }.joined()
    }

    // MARK: - Sensor Clues

    private func appendSensorClues(_ data: InferOptionsRequest) {
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

    // MARK: - Upload GIF to COS (STS 直传)

    private func uploadGifToCOS(gifUrl: String) async -> String? {
        guard let url = URL(string: gifUrl) else { return nil }

        do {
            let (gifData, _) = try await URLSession.shared.data(from: url)
            let stsResult = try await agentRepository.getSTSCredentials()
            let cred = stsResult.sts

            let md5Hex = UUID().uuidString.replacingOccurrences(of: "-", with: "").lowercased()
            let objectKey = "\(cred.prefix)\(md5Hex).gif"

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

    // MARK: - Build Infer Options Request

    private func buildInferOptionsRequest(excludeActivities: [String]?, sessionId: String?) async -> InferOptionsRequest {
        let deviceStatus = await deviceCollector.getCurrentStatusAsync()
        let locationStatus = locationCollector.currentStatus

        return InferOptionsRequest(
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
                networkType: deviceStatus.networkType.rawValue,
                wifiSSID: deviceStatus.wifiSSID,
                bluetoothDeviceType: deviceStatus.bluetoothDeviceType
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
            ) : nil,
            excludeActivities: excludeActivities,
            sessionId: sessionId
        )
    }
}

// MARK: - Preview

#Preview {
    AIStatusInferenceView()
}
