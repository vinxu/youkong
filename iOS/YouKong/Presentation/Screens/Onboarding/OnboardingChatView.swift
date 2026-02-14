import SwiftUI

// MARK: - Phase 2: V4 Text Chat + Schedule Creation

struct OnboardingChatView: View {
    let inferredEmoji: String
    let inferredActivity: String
    let onComplete: () -> Void
    let onSkip: () -> Void

    @StateObject private var viewModel = VoiceScheduleViewModel()
    @State private var inputText = ""
    @State private var showChips = true
    @State private var isFirstMessage = true
    @FocusState private var isInputFocused: Bool

    var body: some View {
        VStack(spacing: 0) {
            // Header
            headerView

            Rectangle()
                .fill(CLIColors.border)
                .frame(height: 1)

            // Message list
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 16) {
                        // AI opening message
                        aiOpeningMessage

                        // Messages
                        ForEach(viewModel.messages) { message in
                            MessageBubble(
                                message: message,
                                onConfirm: {
                                    Task {
                                        await viewModel.confirmSchedule()
                                    }
                                },
                                onCancel: {
                                    Task {
                                        await viewModel.cancelSession()
                                    }
                                }
                            )
                            .id(message.id)
                        }

                        // Processing indicator
                        if viewModel.state == .processing || !viewModel.progressItems.isEmpty {
                            SimpleProcessingView(isProcessing: viewModel.state == .processing)
                                .id("processing")
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.vertical, 16)
                }
                .onChange(of: viewModel.messages.count) { _ in
                    withAnimation {
                        if viewModel.state == .processing {
                            proxy.scrollTo("processing", anchor: .bottom)
                        } else if let lastMessage = viewModel.messages.last {
                            proxy.scrollTo(lastMessage.id, anchor: .bottom)
                        }
                    }
                }
            }

            Rectangle()
                .fill(CLIColors.border)
                .frame(height: 1)

            // Suggestion chips + approval buttons + text input
            bottomSection
        }
        .background(CLIColors.background.ignoresSafeArea())
        .onChange(of: viewModel.state) { newState in
            if newState == .completed {
                onComplete()
            }
        }
    }

    // MARK: - Header

    private var headerView: some View {
        HStack {
            Text("━━ AI 助手 ━━")
                .font(.cliHeadline)
                .foregroundColor(CLIColors.cyan)

            Spacer()

            Button {
                onSkip()
            } label: {
                Text("[跳过]")
                    .font(.cliBodySmall)
                    .foregroundColor(CLIColors.textSecondary)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    // MARK: - AI Opening Message

    private var aiOpeningMessage: some View {
        let greeting = buildGreeting()
        return HStack {
            VStack(alignment: .leading, spacing: 4) {
                Text(greeting)
                    .font(.cliBody)
                    .foregroundColor(CLIColors.textPrimary)
                    .padding(12)
                    .background(CLIColors.backgroundSecondary)
                    .overlay(
                        Rectangle()
                            .stroke(CLIColors.border, lineWidth: 1)
                    )
            }
            Spacer(minLength: 60)
        }
    }

    // MARK: - Bottom Section

    private var bottomSection: some View {
        VStack(spacing: 12) {
            // Error state: show retry button
            if case .error(let msg) = viewModel.state {
                errorRetryBar(message: msg)
            }

            // Suggestion chips (first interaction only)
            if showChips && viewModel.messages.isEmpty {
                suggestionChips
            }

            // Text input bar (hide when error)
            if case .error = viewModel.state {
                // hidden
            } else {
                textInputBar
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(CLIColors.background)
    }

    // MARK: - Suggestion Chips

    private var suggestionChips: some View {
        let chips = buildSuggestionChips()
        return VStack(spacing: 8) {
            ForEach(chips, id: \.self) { chip in
                Button {
                    showChips = false
                    let textToSend = withContextPrefix(chip)
                    Task {
                        await viewModel.submitText(textToSend)
                    }
                } label: {
                    Text(chip)
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.cyan)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 10)
                        .background(CLIColors.backgroundSecondary)
                        .overlay(
                            RoundedRectangle(cornerRadius: 4)
                                .stroke(CLIColors.cyan.opacity(0.3), lineWidth: 1)
                        )
                }
            }
        }
    }

    // MARK: - Text Input Bar

    private var textInputBar: some View {
        HStack(spacing: 8) {
            ZStack(alignment: .leading) {
                if inputText.isEmpty {
                    Text("告诉我你的安排...")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                        .padding(.horizontal, 12)
                }
                TextField("", text: $inputText)
                    .font(.cliBody)
                    .foregroundColor(CLIColors.textPrimary)
                    .padding(.horizontal, 12)
            }
            .padding(.vertical, 10)
            .background(CLIColors.backgroundSecondary)
            .overlay(
                RoundedRectangle(cornerRadius: 4)
                    .stroke(CLIColors.border, lineWidth: 1)
            )
            .focused($isInputFocused)
                .onSubmit {
                    sendMessage()
                }
                .disabled(viewModel.state == .processing || viewModel.state == .confirming)

            Button {
                sendMessage()
            } label: {
                Text("发送")
                    .font(.cliBody)
                    .foregroundColor(inputText.trimmingCharacters(in: .whitespaces).isEmpty
                                     ? CLIColors.textWeak
                                     : CLIColors.background)
                    .padding(.horizontal, 16)
                    .padding(.vertical, 10)
                    .background(inputText.trimmingCharacters(in: .whitespaces).isEmpty
                                ? CLIColors.backgroundSecondary
                                : CLIColors.green)
                    .cornerRadius(4)
            }
            .disabled(inputText.trimmingCharacters(in: .whitespaces).isEmpty
                      || viewModel.state == .processing
                      || viewModel.state == .confirming)
        }
    }

    // MARK: - Approval Buttons

    private var approvalButtons: some View {
        HStack(spacing: 12) {
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

            Button {
                Task {
                    await viewModel.confirmSchedule()
                }
            } label: {
                Text("✓ 确认保存")
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

    // MARK: - Error Retry

    private func errorRetryBar(message: String) -> some View {
        VStack(spacing: 8) {
            Text(message)
                .font(.cliCaption)
                .foregroundColor(CLIColors.red)

            HStack(spacing: 12) {
                Button {
                    viewModel.reset()
                    showChips = true
                } label: {
                    Text("重试")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.background)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .background(CLIColors.cyan)
                        .cornerRadius(4)
                }

                Button {
                    onSkip()
                } label: {
                    Text("跳过")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .overlay(
                            RoundedRectangle(cornerRadius: 4)
                                .stroke(CLIColors.border, lineWidth: 1)
                        )
                }
            }
        }
    }

    // MARK: - Helpers

    private func sendMessage() {
        let text = inputText.trimmingCharacters(in: .whitespaces)
        guard !text.isEmpty else { return }

        let textToSend = withContextPrefix(text)
        inputText = ""
        showChips = false
        isInputFocused = false

        Task {
            await viewModel.submitText(textToSend)
        }
    }

    /// Onboarding 不注入当前状态前缀，避免 LLM 把历史推断状态塞进时间表
    private func withContextPrefix(_ text: String) -> String {
        isFirstMessage = false
        return text
    }

    private func buildGreeting() -> String {
        let hour = Calendar.current.component(.hour, from: Date())
        let timeGreeting: String
        switch hour {
        case 0..<6: timeGreeting = "这么晚还没睡呀"
        case 6..<9: timeGreeting = "早上好"
        case 9..<12: timeGreeting = "上午好"
        case 12..<14: timeGreeting = "中午好"
        case 14..<17: timeGreeting = "下午好"
        case 17..<19: timeGreeting = "傍晚好"
        case 19..<22: timeGreeting = "晚上好"
        default: timeGreeting = "夜深了"
        }

        return "\(timeGreeting)！我是你的 AI 助手 👋\n告诉我接下来的安排，我帮你生成今天的时间表"
    }

    private func buildSuggestionChips() -> [String] {
        let hour = Calendar.current.component(.hour, from: Date())

        if hour < 12 {
            return [
                "帮我安排今天的时间表",
                "下午两点到四点想去健身"
            ]
        } else if hour < 18 {
            return [
                "帮我安排今天剩下的时间",
                "晚上七点到九点想去运动"
            ]
        } else {
            return [
                "帮我安排明天的时间表",
                "明天上午十点有个会议"
            ]
        }
    }
}
