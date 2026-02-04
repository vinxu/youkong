import SwiftUI

/// 语音状态时刻表 - 对话式界面
struct VoiceScheduleSheet: View {
    @StateObject private var viewModel = VoiceScheduleViewModel()
    @Environment(\.dismiss) private var dismiss

    let onCompleted: () -> Void

    @State private var showPermissionAlert = false

    var body: some View {
        VStack(spacing: 0) {
            // 标题栏
            headerView

            Rectangle()
                .fill(CLIColors.border)
                .frame(height: 1)

            // 对话内容区域
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 16) {
                        // 引导提示（首条）
                        if viewModel.messages.isEmpty && viewModel.state == .idle {
                            guideView
                        }

                        // 消息列表
                        ForEach(viewModel.messages) { message in
                            MessageBubble(message: message)
                                .id(message.id)
                        }

                        // 处理中状态 - 显示进度反馈
                        if viewModel.state == .processing || !viewModel.progressItems.isEmpty {
                            ProgressFeedbackView(
                                progressItems: viewModel.progressItems,
                                isProcessing: viewModel.state == .processing,
                                processingStatus: viewModel.processingStatus
                            )
                        }

                        // 可见性选择
                        if viewModel.showVisibilitySelection {
                            VisibilitySelectionView(
                                selectedVisibility: $viewModel.selectedVisibility,
                                selectedCircleIDs: $viewModel.selectedCircleIDs,
                                availableCircles: viewModel.availableCircles,
                                onConfirm: {
                                    Task {
                                        await viewModel.confirmSchedule()
                                    }
                                }
                            )
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.vertical, 16)
                }
                .onChange(of: viewModel.messages.count) { _ in
                    // 新消息时滚动到底部
                    if let lastMessage = viewModel.messages.last {
                        withAnimation {
                            proxy.scrollTo(lastMessage.id, anchor: .bottom)
                        }
                    }
                }
            }

            Rectangle()
                .fill(CLIColors.border)
                .frame(height: 1)

            // 底部操作区域
            bottomActionBar
        }
        .background(CLIColors.background)
        .alert("需要麦克风权限", isPresented: $showPermissionAlert) {
            Button("去设置") {
                if let url = URL(string: UIApplication.openSettingsURLString) {
                    UIApplication.shared.open(url)
                }
            }
            Button("取消", role: .cancel) {
                dismiss()
            }
        } message: {
            Text("请在设置中允许访问麦克风，以便使用语音输入功能")
        }
        .onAppear {
            checkPermission()
        }
        .onChange(of: viewModel.state) { newState in
            if newState == .completed {
                onCompleted()
                // 延迟关闭，让用户看到成功状态
                DispatchQueue.main.asyncAfter(deadline: .now() + 1) {
                    dismiss()
                }
            }
        }
    }

    // MARK: - Header View

    private var headerView: some View {
        HStack {
            Text("━━ 训练AI发状态 ━━")
                .font(.cliHeadline)
                .foregroundColor(CLIColors.cyan)

            Spacer()

            Button {
                dismiss()
            } label: {
                Text("[关闭]")
                    .font(.cliBodySmall)
                    .foregroundColor(CLIColors.textSecondary)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    // MARK: - Guide View

    private var guideView: some View {
        VStack(spacing: 16) {
            Image(systemName: "wand.and.stars")
                .font(.system(size: 40))
                .foregroundColor(CLIColors.green)

            Text("告诉 AI 你当下在做什么\n或者接下来的行程安排")
                .font(.cliBody)
                .foregroundColor(CLIColors.textSecondary)
                .multilineTextAlignment(.center)

            // 示例
            VStack(alignment: .leading, spacing: 8) {
                Text("例如：")
                    .font(.cliCaption)
                    .foregroundColor(CLIColors.textWeak)

                Text("\"我下午两点开会，然后五点去吃饭\"")
                    .font(.cliBodySmall)
                    .foregroundColor(CLIColors.yellow)
                    .padding(8)
                    .background(CLIColors.backgroundSecondary)

                Text("\"我现在在家休息\"")
                    .font(.cliBodySmall)
                    .foregroundColor(CLIColors.yellow)
                    .padding(8)
                    .background(CLIColors.backgroundSecondary)
            }
        }
        .padding(.vertical, 32)
    }


    // MARK: - Bottom Action Bar

    private var bottomActionBar: some View {
        VStack(spacing: 12) {
            // 如果有时刻表，显示确认区域
            if viewModel.hasSchedule {
                scheduleConfirmBar
            }

            // 语音输入按钮（微信风格）
            WechatStyleVoiceButton(
                isRecording: viewModel.isRecording,
                isCancelling: viewModel.isCancelling,
                recordingDuration: viewModel.recordingDuration,
                onStart: {
                    viewModel.startRecording()
                },
                onEnd: {
                    await viewModel.submitRecording()
                },
                onCancel: {
                    viewModel.cancelRecording()
                }
            )
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(CLIColors.background)
    }

    // MARK: - Schedule Confirm Bar

    private var scheduleConfirmBar: some View {
        HStack(spacing: 12) {
            // 取消按钮
            Button {
                Task {
                    await viewModel.cancelSession()
                }
            } label: {
                Text("取消")
                    .font(.cliBody)
                    .foregroundColor(CLIColors.textSecondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .overlay(
                        Rectangle()
                            .stroke(CLIColors.border, lineWidth: 1)
                    )
            }
            .buttonStyle(.borderless)

            // 确认执行按钮
            Button {
                Task {
                    await viewModel.confirmSchedule()
                }
            } label: {
                Text("✓ 确认执行")
                    .font(.cliBody)
                    .foregroundColor(CLIColors.background)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .background(CLIColors.green)
            }
            .buttonStyle(.borderless)
            .disabled(viewModel.state == .confirming)
        }
    }

    // MARK: - Permission

    private func checkPermission() {
        if viewModel.permissionDenied {
            showPermissionAlert = true
        } else if !viewModel.hasPermission {
            Task {
                let granted = await viewModel.requestPermission()
                if !granted {
                    showPermissionAlert = true
                }
            }
        }
    }
}

// MARK: - Message Model

struct ChatMessage: Identifiable, Equatable {
    let id = UUID()
    let type: MessageType
    let content: String
    let timestamp: Date
    var schedule: [ScheduleItem]?
    var questions: [ClarifyQuestion]?
    var reasoning: [String]?

    enum MessageType {
        case user          // 用户语音转文字
        case aiThinking    // AI 思考中
        case aiText        // AI 文字回复
        case aiSchedule    // AI 生成的时刻表
        case aiQuestion    // AI 提问
        case system        // 系统消息
    }

    static func == (lhs: ChatMessage, rhs: ChatMessage) -> Bool {
        lhs.id == rhs.id
    }
}

// MARK: - Message Bubble

struct MessageBubble: View {
    let message: ChatMessage

    var body: some View {
        HStack {
            if message.type == .user {
                Spacer(minLength: 60)
            }

            VStack(alignment: message.type == .user ? .trailing : .leading, spacing: 4) {
                // 消息内容
                messageContent

                // 时间
                Text(formatTime(message.timestamp))
                    .font(.cliCaption)
                    .foregroundColor(CLIColors.textWeak)
            }

            if message.type != .user {
                Spacer(minLength: 60)
            }
        }
    }

    @ViewBuilder
    private var messageContent: some View {
        switch message.type {
        case .user:
            Text(message.content)
                .font(.cliBody)
                .foregroundColor(CLIColors.background)
                .padding(12)
                .background(CLIColors.green)

        case .aiText, .aiThinking:
            Text(message.content)
                .font(.cliBody)
                .foregroundColor(CLIColors.textPrimary)
                .padding(12)
                .background(CLIColors.backgroundSecondary)
                .overlay(
                    Rectangle()
                        .stroke(CLIColors.border, lineWidth: 1)
                )

        case .aiSchedule:
            if let schedule = message.schedule {
                SchedulePreview(schedule: schedule, reasoning: message.reasoning ?? [])
            } else {
                Text(message.content)
                    .font(.cliBody)
                    .foregroundColor(CLIColors.textPrimary)
            }

        case .aiQuestion:
            if let questions = message.questions {
                QuestionPreview(questions: questions, content: message.content)
            } else {
                Text(message.content)
                    .font(.cliBody)
                    .foregroundColor(CLIColors.textPrimary)
            }

        case .system:
            Text(message.content)
                .font(.cliCaption)
                .foregroundColor(CLIColors.yellow)
                .padding(8)
                .frame(maxWidth: .infinity)
                .background(CLIColors.yellow.opacity(0.1))
        }
    }

    private func formatTime(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: date)
    }
}

// MARK: - Schedule Preview

struct SchedulePreview: View {
    let schedule: [ScheduleItem]
    var reasoning: [String] = []
    @State private var showReasoning = false

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("📋 生成的时刻表：")
                .font(.cliCaption)
                .foregroundColor(CLIColors.cyan)

            ForEach(schedule) { item in
                HStack(spacing: 8) {
                    Text("\(item.startTime)-\(item.endTime)")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textWeak)
                        .frame(width: 90, alignment: .leading)

                    Text(item.emoji)

                    Text(item.status)
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textPrimary)

                    Spacer()

                    if item.executed == true {
                        Text("✓")
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.green)
                    }
                }
            }

            // 推理依据（可展开）
            if !reasoning.isEmpty {
                Rectangle()
                    .fill(CLIColors.border)
                    .frame(height: 1)
                    .padding(.vertical, 4)

                Button {
                    withAnimation { showReasoning.toggle() }
                } label: {
                    HStack(spacing: 4) {
                        Text("推理依据")
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.textWeak)
                        Image(systemName: showReasoning ? "chevron.up" : "chevron.down")
                            .font(.system(size: 10))
                            .foregroundColor(CLIColors.textWeak)
                    }
                }
                .buttonStyle(.plain)

                if showReasoning {
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(reasoning, id: \.self) { reason in
                            HStack(alignment: .top, spacing: 6) {
                                Text("•")
                                    .font(.cliCaption)
                                    .foregroundColor(CLIColors.textWeak)
                                Text(reason)
                                    .font(.cliCaption)
                                    .foregroundColor(CLIColors.textSecondary)
                            }
                        }
                    }
                    .transition(.opacity.combined(with: .move(edge: .top)))
                }
            }

            Text("可以继续说话修改，或点击\"确认执行\"")
                .font(.cliCaption)
                .foregroundColor(CLIColors.textWeak)
                .padding(.top, 4)
        }
        .padding(12)
        .background(CLIColors.backgroundSecondary)
        .overlay(
            Rectangle()
                .stroke(CLIColors.cyan.opacity(0.3), lineWidth: 1)
        )
    }
}

// MARK: - Question Preview

struct QuestionPreview: View {
    let questions: [ClarifyQuestion]
    let content: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(content)
                .font(.cliBody)
                .foregroundColor(CLIColors.textPrimary)

            ForEach(questions) { question in
                VStack(alignment: .leading, spacing: 4) {
                    Text("• \(question.question)")
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.yellow)

                    // 选项
                    FlowLayout(spacing: 8) {
                        ForEach(question.options, id: \.self) { option in
                            Text(option)
                                .font(.cliCaption)
                                .foregroundColor(CLIColors.textSecondary)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(CLIColors.background)
                                .overlay(
                                    Rectangle()
                                        .stroke(CLIColors.border, lineWidth: 1)
                                )
                        }
                    }
                }
            }

            Text("请语音回答以上问题")
                .font(.cliCaption)
                .foregroundColor(CLIColors.textWeak)
                .padding(.top, 4)
        }
        .padding(12)
        .background(CLIColors.backgroundSecondary)
        .overlay(
            Rectangle()
                .stroke(CLIColors.yellow.opacity(0.3), lineWidth: 1)
        )
    }
}

// MARK: - Wechat Style Voice Button

struct WechatStyleVoiceButton: View {
    let isRecording: Bool
    let isCancelling: Bool
    let recordingDuration: TimeInterval
    let onStart: () -> Void
    let onEnd: () async -> Void
    let onCancel: () -> Void

    @State private var dragOffset: CGFloat = 0
    @State private var isInCancelZone = false
    private let cancelThreshold: CGFloat = -60

    var body: some View {
        ZStack {
            // 取消区域（录音时显示在上方）
            if isRecording {
                VStack {
                    // 取消区域
                    cancelZone
                        .frame(height: 80)
                        .padding(.bottom, 20)

                    Spacer()
                }
                .transition(.opacity)
            }

            VStack(spacing: 8) {
                Spacer()

                // 录音状态提示
                if isRecording && !isInCancelZone {
                    HStack(spacing: 8) {
                        // 录音波形动画
                        RecordingWaveform()

                        Text(formatDuration(recordingDuration))
                            .font(.cliBodySmall)
                            .foregroundColor(CLIColors.green)

                        Text("↑ 上滑取消")
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.textWeak)
                    }
                    .frame(height: 24)
                    .transition(.opacity)
                }

                // 按钮
                voiceButton
            }
        }
        .frame(height: isRecording ? 180 : 50)
        .animation(.easeInOut(duration: 0.2), value: isRecording)
        .animation(.easeInOut(duration: 0.15), value: isInCancelZone)
    }

    // MARK: - Cancel Zone

    private var cancelZone: some View {
        VStack(spacing: 8) {
            Image(systemName: isInCancelZone ? "xmark.circle.fill" : "xmark.circle")
                .font(.system(size: 40))
                .foregroundColor(isInCancelZone ? CLIColors.red : CLIColors.textWeak)
                .scaleEffect(isInCancelZone ? 1.2 : 1.0)

            Text(isInCancelZone ? "松开取消" : "上滑到此取消")
                .font(.cliBodySmall)
                .foregroundColor(isInCancelZone ? CLIColors.red : CLIColors.textWeak)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 12)
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(isInCancelZone ? CLIColors.red.opacity(0.15) : CLIColors.backgroundSecondary)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(isInCancelZone ? CLIColors.red : CLIColors.border, lineWidth: isInCancelZone ? 2 : 1)
        )
    }

    // MARK: - Voice Button

    private var voiceButton: some View {
        Text(isRecording ? (isInCancelZone ? "松开取消" : "松开发送") : "按住说话")
            .font(.cliBody)
            .foregroundColor(
                isRecording
                    ? (isInCancelZone ? CLIColors.red : CLIColors.green)
                    : CLIColors.textPrimary
            )
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .background(
                isRecording
                    ? (isInCancelZone ? CLIColors.red.opacity(0.1) : CLIColors.green.opacity(0.1))
                    : CLIColors.backgroundSecondary
            )
            .overlay(
                Rectangle()
                    .stroke(
                        isRecording
                            ? (isInCancelZone ? CLIColors.red : CLIColors.green)
                            : CLIColors.border,
                        lineWidth: 1
                    )
            )
            .gesture(
                DragGesture(minimumDistance: 0)
                    .onChanged { value in
                        if !isRecording {
                            onStart()
                        }
                        dragOffset = value.translation.height
                        isInCancelZone = dragOffset < cancelThreshold
                    }
                    .onEnded { _ in
                        if isRecording {
                            if isInCancelZone {
                                onCancel()
                            } else {
                                Task {
                                    await onEnd()
                                }
                            }
                        }
                        dragOffset = 0
                        isInCancelZone = false
                    }
                )
        }
    }

    private func formatDuration(_ duration: TimeInterval) -> String {
        let seconds = Int(duration)
        return String(format: "%d\"", seconds)
    }


// MARK: - Recording Waveform

struct RecordingWaveform: View {
    @State private var animationPhase: CGFloat = 0

    var body: some View {
        HStack(spacing: 3) {
            ForEach(0..<5, id: \.self) { i in
                RoundedRectangle(cornerRadius: 2)
                    .fill(CLIColors.green)
                    .frame(width: 4, height: waveHeight(for: i))
            }
        }
        .onAppear {
            withAnimation(.easeInOut(duration: 0.5).repeatForever(autoreverses: true)) {
                animationPhase = 1
            }
        }
    }

    private func waveHeight(for index: Int) -> CGFloat {
        let baseHeights: [CGFloat] = [8, 16, 20, 14, 10]
        let variation: CGFloat = animationPhase * 8
        let offset = CGFloat(index) * 0.3
        return baseHeights[index] + sin(animationPhase * .pi + offset) * variation
    }
}

// MARK: - Flow Layout (for question options)

struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let result = FlowResult(in: proposal.width ?? 0, subviews: subviews, spacing: spacing)
        return result.size
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let result = FlowResult(in: bounds.width, subviews: subviews, spacing: spacing)
        for (index, subview) in subviews.enumerated() {
            subview.place(at: CGPoint(x: bounds.minX + result.positions[index].x,
                                      y: bounds.minY + result.positions[index].y),
                         proposal: .unspecified)
        }
    }

    struct FlowResult {
        var size: CGSize = .zero
        var positions: [CGPoint] = []

        init(in maxWidth: CGFloat, subviews: Subviews, spacing: CGFloat) {
            var x: CGFloat = 0
            var y: CGFloat = 0
            var lineHeight: CGFloat = 0

            for subview in subviews {
                let size = subview.sizeThatFits(.unspecified)

                if x + size.width > maxWidth && x > 0 {
                    x = 0
                    y += lineHeight + spacing
                    lineHeight = 0
                }

                positions.append(CGPoint(x: x, y: y))
                lineHeight = max(lineHeight, size.height)
                x += size.width + spacing

                self.size.width = max(self.size.width, x)
            }

            self.size.height = y + lineHeight
        }
    }
}

// MARK: - Progress Feedback View

struct ProgressFeedbackView: View {
    let progressItems: [VoiceScheduleViewModel.ProgressItem]
    let isProcessing: Bool
    let processingStatus: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // 已完成的进度项
            ForEach(Array(progressItems.enumerated()), id: \.element.id) { index, item in
                let isLatest = index == progressItems.count - 1 && isProcessing

                HStack(alignment: .top, spacing: 8) {
                    // 状态图标
                    if isLatest && !item.isCompleted {
                        ProgressView()
                            .progressViewStyle(CircularProgressViewStyle(tint: CLIColors.cyan))
                            .scaleEffect(0.6)
                            .frame(width: 16, height: 16)
                    } else {
                        Text("✓")
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.green)
                            .frame(width: 16, height: 16)
                    }

                    VStack(alignment: .leading, spacing: 2) {
                        Text(item.message)
                            .font(.cliCaption)
                            .foregroundColor(isLatest ? CLIColors.textPrimary : CLIColors.textSecondary)

                        // 详细信息
                        if let detail = item.detail {
                            Text("→ \(detail)")
                                .font(.cliCaption)
                                .foregroundColor(CLIColors.cyan)
                        }
                    }
                }
                .opacity(isLatest ? 1.0 : 0.7)
            }

            // 当前处理状态（如果没有进度项）
            if progressItems.isEmpty && isProcessing {
                HStack(spacing: 8) {
                    ProgressView()
                        .progressViewStyle(CircularProgressViewStyle(tint: CLIColors.cyan))
                        .scaleEffect(0.6)

                    Text(processingStatus)
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.cyan)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .background(CLIColors.backgroundSecondary)
        .overlay(
            Rectangle()
                .stroke(CLIColors.cyan.opacity(0.3), lineWidth: 1)
        )
    }
}

// MARK: - Visibility Selection View

struct VisibilitySelectionView: View {
    @Binding var selectedVisibility: ScheduleVisibility
    @Binding var selectedCircleIDs: Set<String>
    let availableCircles: [CircleInfoCompact]
    let onConfirm: () -> Void

    @State private var showCircleSelection = false

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("这些状态谁可以看到？")
                .font(.cliBody)
                .foregroundColor(CLIColors.textPrimary)

            // 可见性选项
            VStack(spacing: 8) {
                ForEach(ScheduleVisibility.allCases, id: \.self) { visibility in
                    VisibilityOptionRow(
                        visibility: visibility,
                        isSelected: selectedVisibility == visibility,
                        onTap: {
                            selectedVisibility = visibility
                            if visibility == .circles {
                                showCircleSelection = true
                            }
                        }
                    )
                }
            }

            // 已选择的圈子
            if selectedVisibility == .circles && !selectedCircleIDs.isEmpty {
                VStack(alignment: .leading, spacing: 8) {
                    Text("已选择的圈子：")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textWeak)

                    FlowLayout(spacing: 8) {
                        ForEach(availableCircles.filter { selectedCircleIDs.contains($0.id) }) { circle in
                            HStack(spacing: 4) {
                                Text(circle.emoji)
                                Text(circle.name)
                            }
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.textPrimary)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(CLIColors.cyan.opacity(0.2))
                        }
                    }

                    Button {
                        showCircleSelection = true
                    } label: {
                        Text("[修改]")
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.cyan)
                    }
                    .buttonStyle(.plain)
                }
            }

            // 确认按钮
            Button {
                onConfirm()
            } label: {
                Text("✓ 确认并执行")
                    .font(.cliBody)
                    .foregroundColor(CLIColors.background)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .background(
                        (selectedVisibility == .circles && selectedCircleIDs.isEmpty)
                            ? CLIColors.textWeak
                            : CLIColors.green
                    )
            }
            .buttonStyle(.plain)
            .disabled(selectedVisibility == .circles && selectedCircleIDs.isEmpty)
        }
        .padding(12)
        .background(CLIColors.backgroundSecondary)
        .overlay(
            Rectangle()
                .stroke(CLIColors.yellow.opacity(0.3), lineWidth: 1)
        )
        .sheet(isPresented: $showCircleSelection) {
            CircleSelectionSheet(
                circles: availableCircles,
                selectedIDs: $selectedCircleIDs
            )
        }
    }
}

// MARK: - Visibility Option Row

struct VisibilityOptionRow: View {
    let visibility: ScheduleVisibility
    let isSelected: Bool
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            HStack(spacing: 12) {
                Image(systemName: visibility.icon)
                    .foregroundColor(isSelected ? CLIColors.cyan : CLIColors.textWeak)
                    .frame(width: 20)

                VStack(alignment: .leading, spacing: 2) {
                    Text(visibility.label)
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textPrimary)

                    Text(visibility.description)
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textWeak)
                }

                Spacer()

                if isSelected {
                    Text("✓")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.cyan)
                }
            }
            .padding(10)
            .background(CLIColors.background)
            .overlay(
                Rectangle()
                    .stroke(isSelected ? CLIColors.cyan : CLIColors.border, lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
    }
}

// MARK: - Circle Selection Sheet

struct CircleSelectionSheet: View {
    let circles: [CircleInfoCompact]
    @Binding var selectedIDs: Set<String>
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                // 标题
                HStack {
                    Text("━━ 选择可见的圈子 ━━")
                        .font(.cliHeadline)
                        .foregroundColor(CLIColors.cyan)
                    Spacer()
                }
                .padding()
                .background(CLIColors.background)

                Rectangle()
                    .fill(CLIColors.border)
                    .frame(height: 1)

                // 圈子列表
                ScrollView {
                    VStack(spacing: 0) {
                        ForEach(circles) { circle in
                            Button {
                                if selectedIDs.contains(circle.id) {
                                    selectedIDs.remove(circle.id)
                                } else {
                                    selectedIDs.insert(circle.id)
                                }
                            } label: {
                                HStack {
                                    Text(circle.emoji)
                                        .font(.title2)

                                    Text(circle.name)
                                        .font(.cliBody)
                                        .foregroundColor(CLIColors.textPrimary)

                                    Spacer()

                                    Text("\(circle.memberCount)人")
                                        .font(.cliCaption)
                                        .foregroundColor(CLIColors.textWeak)

                                    if selectedIDs.contains(circle.id) {
                                        Text("✓")
                                            .font(.cliBody)
                                            .foregroundColor(CLIColors.green)
                                    }
                                }
                                .padding()
                            }
                            .buttonStyle(.plain)

                            Rectangle()
                                .fill(CLIColors.border)
                                .frame(height: 1)
                        }
                    }
                }
                .background(CLIColors.background)

                // 完成按钮
                Button {
                    dismiss()
                } label: {
                    Text("完成")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.background)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .background(CLIColors.green)
                }
                .buttonStyle(.plain)
                .padding()
            }
            .background(CLIColors.background)
            .navigationBarHidden(true)
        }
    }
}

#Preview {
    VoiceScheduleSheet(onCompleted: {})
}
