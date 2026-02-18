import SwiftUI

struct GridHomeView: View {
    @StateObject private var viewModel = GridHomeViewModel()
    @StateObject private var voiceVM = VoiceScheduleViewModel()
    @StateObject private var scheduleVM = MyScheduleTimelineViewModel()
    @State private var showStatusAnalysis = false
    @State private var showAIInference = false
    @State private var selectedTab = 0
    @Environment(\.scenePhase) private var scenePhase

    // 按住说话相关状态
    @State private var isPressingVoiceButton = false
    @State private var dragOffset: CGFloat = 0
    @State private var showCancelZone = false
    @State private var showPermissionAlert = false

    // 输入模式：voice / text
    @State private var inputMode: String = "voice"
    @State private var textInput: String = ""
    @State private var isTextFieldFocused: Bool = false

    // 引导气泡（关闭后持久记住）
    @State private var showGuideBubble = UserDefaults.standard.integer(forKey: "voice_guide_dismiss_count") < 3

    // 取消区域阈值
    private let cancelThreshold: CGFloat = -80

    var body: some View {
        VStack(spacing: 0) {
            // 上方区域（内容 + 覆盖层叠加）
            ZStack {
                // 主内容区域
                contentArea

                // 覆盖层只覆盖内容区域，不覆盖底部按钮
                if voiceVM.showOverlay {
                    VoiceOverlayView(
                        viewModel: voiceVM,
                        onDismiss: {
                            // 关闭时刷新首页数据（防止 onCompleted 未触发时数据过期）
                            Task { await viewModel.loadGrid() }
                            Task { await scheduleVM.refresh() }
                            voiceVM.showOverlay = false
                            voiceVM.reset()
                        },
                        onCompleted: {
                            // 操作完成时立即刷新首页数据，不自动关闭覆盖层
                            Task { await viewModel.loadGrid() }
                            Task { await scheduleVM.refresh() }
                        }
                    )
                    .transition(.opacity)
                }

                // 取消区域提示（录音时显示）
                if isPressingVoiceButton {
                    VStack {
                        Spacer()
                        cancelZoneView
                            .opacity(showCancelZone ? 1 : 0.5)
                        Spacer()
                            .frame(height: 60)
                    }
                    .transition(.opacity)
                }
            }
            .animation(.easeInOut(duration: 0.25), value: voiceVM.showOverlay)
            .animation(.easeInOut(duration: 0.15), value: isPressingVoiceButton)

            // 底部按钮 - 始终可见（不被覆盖层遮盖）
            bottomButtonsBar
        }
        .background(CLIColors.background)
        .task {
            await viewModel.loadGrid()
        }
        .task {
            await scheduleVM.loadInitialData()
        }
        .onAppear {
            voiceVM.prepareAudioSession()
        }
        .onChange(of: voiceVM.state) { newState in
            // 当开始处理时显示覆盖层
            if newState == .processing {
                voiceVM.showOverlay = true
            }
        }
        .onChange(of: selectedTab) { newTab in
            if newTab == 0 {
                // 切回好友 tab 时自动刷新数据
                Task { await viewModel.loadGrid() }
            } else if newTab == 1 {
                // 切到"我的"tab 时刷新时刻表数据并滚动到当前时间
                Task { await scheduleVM.refresh() }
                scheduleVM.triggerScrollToNow()
            }
        }
        .onChange(of: scheduleVM.changeCount) { _ in
            // 时刻表编辑/删除/切换有空后立即刷新首页
            Task { await viewModel.loadGrid() }
        }
        .onChange(of: scenePhase) { newPhase in
            if newPhase == .active {
                // 回前台时刷新 AI 就绪状态（权限可能在系统设置中变更）
                Task { await scheduleVM.loadSettings() }
                // 刷新时刻表数据（后台可能有 AI 自动推测更新）
                Task { await scheduleVM.refresh() }
                Task { await viewModel.loadGrid() }
            }
        }
        .alert("需要麦克风权限", isPresented: $showPermissionAlert) {
            Button("去设置") {
                if let url = URL(string: UIApplication.openSettingsURLString) {
                    UIApplication.shared.open(url)
                }
            }
            Button("取消", role: .cancel) {}
        } message: {
            Text("请在设置中允许访问麦克风，以便使用语音输入功能")
        }
        .alert("错误", isPresented: .constant(viewModel.error != nil)) {
            Button("确定") {
                viewModel.error = nil
            }
        } message: {
            if let error = viewModel.error {
                Text(error.localizedDescription)
            }
        }
        .sheet(isPresented: $showStatusAnalysis) {
            StatusAnalysisView {
                Task {
                    await viewModel.loadGrid()
                }
            }
        }
        .sheet(isPresented: $viewModel.showPosterSheet) {
            PosterShareView(friends: viewModel.friends.map { friend in
                FriendGridItem(
                    userId: friend.id,
                    nickname: friend.nickname,
                    avatar: friend.avatar,
                    emoji: friend.emoji,
                    status: friend.status,
                    updatedAt: ISO8601DateFormatter().string(from: friend.updatedAt),
                    relativeTime: friend.relativeTime,
                    city: friend.city,
                    isAvailable: friend.isAvailable,
                    isVisiting: friend.isVisiting,
                    gifUrl: friend.gifUrl,
                    giphyQuery: friend.giphyQuery,
                    useGif: friend.useGif,
                    needsSchedule: friend.needsSchedule,
                    scenePose: friend.sceneConfig?.pose,
                    sceneArms: friend.sceneConfig?.arms,
                    sceneExpression: friend.sceneConfig?.expression,
                    sceneProp: friend.sceneConfig?.prop,
                    sceneSurface: friend.sceneConfig?.surface,
                    interactions: friend.interactions.isEmpty ? nil : friend.interactions,
                    interactionCount: friend.interactionCount
                )
            })
        }
        .sheet(isPresented: $showAIInference) {
            AIStatusInferenceView { emoji, activity, isAvailable in
                // 状态确认后延迟刷新首页（等后端保存完成）
                Task {
                    try? await Task.sleep(nanoseconds: 500_000_000) // 0.5s
                    await viewModel.loadGrid()
                }
                Task {
                    try? await Task.sleep(nanoseconds: 500_000_000)
                    await scheduleVM.refresh()
                }
            }
        }
        .navigationDestination(for: FriendStatus.self) { friend in
            ChatView(
                partnerId: friend.id,
                partnerName: friend.nickname,
                partnerAvatar: friend.avatar
            )
        }
    }

    // MARK: - Content Area (不包含底部按钮)

    private var contentArea: some View {
        VStack(spacing: 0) {
            // 顶部导航栏：tabs 居左 + icons 居右
            HStack(spacing: 0) {
                // 好友 Tab
                Button {
                    withAnimation(.easeInOut(duration: 0.2)) {
                        selectedTab = 0
                    }
                } label: {
                    VStack(spacing: 4) {
                        Text("好友")
                            .font(.system(size: 16, weight: selectedTab == 0 ? .bold : .regular, design: .monospaced))
                            .foregroundColor(selectedTab == 0 ? CLIColors.green : CLIColors.textSecondary)
                        Rectangle()
                            .fill(selectedTab == 0 ? CLIColors.green : Color.clear)
                            .frame(width: 24, height: 2)
                    }
                }
                .buttonStyle(.plain)
                .padding(.trailing, 16)

                // 我的 Tab
                Button {
                    withAnimation(.easeInOut(duration: 0.2)) {
                        selectedTab = 1
                    }
                } label: {
                    VStack(spacing: 4) {
                        Text("我的")
                            .font(.system(size: 16, weight: selectedTab == 1 ? .bold : .regular, design: .monospaced))
                            .foregroundColor(selectedTab == 1 ? CLIColors.green : CLIColors.textSecondary)
                        Rectangle()
                            .fill(selectedTab == 1 ? CLIColors.green : Color.clear)
                            .frame(width: 24, height: 2)
                    }
                }
                .buttonStyle(.plain)

                Spacer()

                // 加好友 icon
                NavigationLink(destination: FriendsManagementView()) {
                    Text("➕")
                        .font(.system(size: 18))
                }
                .padding(.trailing, 8)

                // 设置 icon
                NavigationLink(destination: SettingsView()) {
                    Text("⚙️")
                        .font(.system(size: 18))
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
            .background(CLIColors.background)

            Rectangle()
                .fill(CLIColors.border)
                .frame(height: 1)

            // Tab 内容区域（支持左右滑动切换）
            TabView(selection: $selectedTab) {
                friendsTabContent
                    .tag(0)

                VStack(spacing: 0) {
                    AutoPredictToggleBar(viewModel: scheduleVM)
                    MyScheduleTimelineContent(viewModel: scheduleVM)
                }
                .tag(1)
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
            .dismissKeyboardOnTouchDown()
        }
        .background(CLIColors.background)
    }

    // tabSwitcher merged into top nav bar

    // MARK: - Friends Tab Content

    private var friendsTabContent: some View {
        Group {
            if viewModel.isLoading && viewModel.friends.isEmpty {
                // CLI 加载中
                VStack {
                    Spacer()
                    HStack(spacing: 8) {
                        Text("⏳")
                        Text("加载中...")
                            .foregroundColor(CLIColors.green)
                    }
                    .font(.cliBody)
                    Spacer()
                }
            } else if viewModel.friends.isEmpty {
                // CLI 空状态
                VStack(spacing: 16) {
                    Spacer()

                    Text("""
                    ┌─────────────────┐
                    │                 │
                    │      👥        │
                    │                 │
                    └─────────────────┘
                    """)
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.border)
                        .multilineTextAlignment(.center)

                    Text("> 还没有好友")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)

                    Text("  去邀请好友加入吧")
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textWeak)

                    Spacer()
                }
            } else {
                // 宫格内容 - 使用 List 以支持 TabView(.page) 内的下拉刷新
                List {
                    FriendGrid(
                        friends: viewModel.friends,
                        getUnreadCount: { friendId in
                            viewModel.getUnreadCount(for: friendId)
                        },
                        onNeedsScheduleTap: {
                            showAIInference = true
                        },
                        onInteraction: { friendId, interaction in
                            Task { await viewModel.sendInteraction(to: friendId, interaction: interaction) }
                        }
                    )
                    .padding(.horizontal, 16)
                    .padding(.top, 16)
                    .padding(.bottom, 16)
                    .listRowInsets(EdgeInsets())
                    .listRowSeparator(.hidden)
                    .listRowBackground(Color.clear)
                }
                .listStyle(.plain)
                .scrollContentBackground(.hidden)
                .background(CLIColors.background)
                .refreshable {
                    await viewModel.refresh()
                }
                .onAppear {
                    UIRefreshControl.appearance().tintColor = UIColor(CLIColors.green)
                }
            }
        }
    }

    // MARK: - Bottom Buttons Bar (始终可见)

    private var bottomButtonsBar: some View {
        VStack(spacing: 0) {
            Rectangle()
                .fill(CLIColors.border)
                .frame(height: 1)

            // 引导气泡
            if showGuideBubble {
                guideBubbleView
            }

            HStack(spacing: 8) {
                // 🤖 一键生成按钮（icon only）
                Button {
                    showAIInference = true
                } label: {
                    Text("🤖")
                        .font(.system(size: 18))
                        .frame(width: 44, height: 44)
                        .overlay(
                            Rectangle()
                                .stroke(CLIColors.border, lineWidth: 1)
                        )
                }
                .buttonStyle(.borderless)

                // 中间区域：语音 / 文本
                if inputMode == "voice" {
                    voiceButtonContent
                        .gesture(
                            DragGesture(minimumDistance: 0)
                                .onChanged { value in
                                    guard voiceVM.canRecord else { return }

                                    if !isPressingVoiceButton {
                                        if voiceVM.permissionDenied {
                                            showPermissionAlert = true
                                            return
                                        }
                                        if !voiceVM.hasPermission {
                                            Task {
                                                let granted = await voiceVM.requestPermission()
                                                if granted {
                                                    isPressingVoiceButton = true
                                                    voiceVM.startRecording()
                                                } else {
                                                    showPermissionAlert = true
                                                }
                                            }
                                            return
                                        }
                                        isPressingVoiceButton = true
                                        voiceVM.startRecording()
                                    }
                                    dragOffset = value.translation.height
                                    showCancelZone = dragOffset < cancelThreshold / 2
                                }
                                .onEnded { value in
                                    guard isPressingVoiceButton else {
                                        dragOffset = 0
                                        showCancelZone = false
                                        return
                                    }
                                    isPressingVoiceButton = false
                                    let shouldCancel = value.translation.height < cancelThreshold

                                    if shouldCancel {
                                        voiceVM.cancelRecording()
                                    } else {
                                        Task {
                                            await voiceVM.submitRecording()
                                        }
                                    }
                                    dragOffset = 0
                                    showCancelZone = false
                                }
                        )
                } else {
                    // 文本输入框（UIKit wrapper，发送时键盘不收起）
                    NonDismissingTextField(
                        text: $textInput,
                        placeholder: "输入大致的行程安排",
                        placeholderColor: UIColor(CLIColors.textSecondary),
                        textColor: UIColor(CLIColors.textPrimary),
                        font: .monospacedSystemFont(ofSize: 13, weight: .regular),
                        isFocused: $isTextFieldFocused,
                        onSend: { text in
                            textInput = ""
                            voiceVM.showOverlay = true
                            Task {
                                await voiceVM.submitText(text)
                            }
                        }
                    )
                    .padding(.horizontal, 12)
                    .frame(height: 44)
                    .background(CLIColors.backgroundSecondary)
                    .overlay(
                        RoundedRectangle(cornerRadius: 4)
                            .stroke(isTextFieldFocused ? CLIColors.green : CLIColors.border, lineWidth: 1)
                    )
                }

                // ⌨️/🎤 切换按钮
                Button {
                    if inputMode == "voice" {
                        inputMode = "text"
                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                            isTextFieldFocused = true
                        }
                    } else {
                        isTextFieldFocused = false
                        inputMode = "voice"
                    }
                } label: {
                    Text(inputMode == "voice" ? "⌨️" : "🎤")
                        .font(.system(size: 18))
                        .frame(width: 44, height: 44)
                        .overlay(
                            Rectangle()
                                .stroke(CLIColors.border, lineWidth: 1)
                        )
                }
                .buttonStyle(.borderless)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 12)
            .background(CLIColors.background)
        }
    }

    // MARK: - Guide Bubble View

    private var guideBubbleView: some View {
        HStack(alignment: .top, spacing: 8) {
            Text("💡")
                .font(.system(size: 14))

            Text(inputMode == "voice"
                ? "按住说话，告诉我你在做什么或接下来的安排\n例：\"我在吃饭\" 或 \"明天上午开会，下午健身\""
                : "输入你的行程安排，我来帮你生成时间表\n例：\"明天上午开会，下午健身\" 或 \"周末想约朋友吃饭\"")
                .font(.system(size: 11, design: .monospaced))
                .foregroundColor(CLIColors.textSecondary)
                .lineSpacing(4)
                .frame(maxWidth: .infinity, alignment: .leading)

            Button {
                showGuideBubble = false
                let count = UserDefaults.standard.integer(forKey: "voice_guide_dismiss_count") + 1
                UserDefaults.standard.set(count, forKey: "voice_guide_dismiss_count")
            } label: {
                Text("✕")
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundColor(CLIColors.textWeak)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(
            RoundedRectangle(cornerRadius: 8)
                .fill(CLIColors.green.opacity(0.1))
                .overlay(
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(CLIColors.green.opacity(0.3), lineWidth: 1)
                )
        )
        .padding(.horizontal, 16)
        .padding(.vertical, 6)
    }

    // MARK: - Voice Button Content

    private var voiceButtonContent: some View {
        let canRecord = voiceVM.canRecord

        return HStack(spacing: 8) {
            Image(systemName: "mic.fill")
            if voiceVM.isRecording {
                // 录音中：显示时长
                Text(formatDuration(voiceVM.recordingDuration))
                    .monospacedDigit()
            } else if !canRecord {
                // 禁用状态：显示处理中
                Text("处理中...")
            } else {
                Text("和AI说说你的行程")
            }
        }
        .font(.cliBody)
        .foregroundColor(
            !canRecord ? CLIColors.background.opacity(0.7) :
            (isPressingVoiceButton ? CLIColors.green : CLIColors.background)
        )
        .frame(maxWidth: .infinity)
        .padding(.vertical, 12)
        .background(
            !canRecord ? CLIColors.green.opacity(0.5) :
            (isPressingVoiceButton ? CLIColors.background : CLIColors.green)
        )
        .overlay(
            Rectangle()
                .stroke(CLIColors.green, lineWidth: isPressingVoiceButton ? 2 : 0)
        )
        .scaleEffect(isPressingVoiceButton ? 0.96 : 1.0)
        .animation(.easeInOut(duration: 0.1), value: isPressingVoiceButton)
    }

    // MARK: - Cancel Zone View

    private var cancelZoneView: some View {
        HStack(spacing: 8) {
            Image(systemName: showCancelZone ? "xmark.circle.fill" : "hand.point.up.fill")
                .font(.system(size: 20))
            Text(showCancelZone ? "松开取消" : "上移取消")
                .font(.cliBody)
        }
        .foregroundColor(showCancelZone ? CLIColors.red : CLIColors.textSecondary)
        .padding(.horizontal, 20)
        .padding(.vertical, 10)
        .background(
            RoundedRectangle(cornerRadius: 20)
                .fill(showCancelZone ? CLIColors.red.opacity(0.15) : CLIColors.backgroundSecondary)
        )
        .offset(y: -60)
    }

    // MARK: - Helper

    private func formatDuration(_ duration: TimeInterval) -> String {
        let seconds = Int(duration)
        let minutes = seconds / 60
        let remainingSeconds = seconds % 60
        return String(format: "%d:%02d", minutes, remainingSeconds)
    }
}

// MARK: - Friend Grid

struct FriendGrid: View {
    let friends: [FriendStatus]
    let getUnreadCount: (String) -> Int
    var onNeedsScheduleTap: (() -> Void)? = nil
    var onInteraction: ((String, InteractionOptionItem) -> Void)? = nil

    private var columns: [GridItem] {
        Array(repeating: GridItem(.flexible(), spacing: 8), count: 2)
    }

    var body: some View {
        LazyVGrid(columns: columns, spacing: 8) {
            ForEach(friends) { friend in
                if friend.needsSchedule {
                    FriendCard(
                        friend: friend,
                        unreadCount: getUnreadCount(friend.id)
                    )
                    .onTapGesture {
                        onNeedsScheduleTap?()
                    }
                } else {
                    NavigationLink(value: friend) {
                        FriendCard(
                            friend: friend,
                            unreadCount: getUnreadCount(friend.id)
                        )
                    }
                    .buttonStyle(PlainButtonStyle())
                }
            }
        }
    }
}

// MARK: - Friend Card (与 Android 对齐)

struct FriendCard: View {
    let friend: FriendStatus
    var unreadCount: Int = 0

    private var borderColor: Color {
        if friend.isAvailable { return CLIColors.green }
        return unreadCount > 0 ? CLIColors.green : CLIColors.border
    }

    var body: some View {
        ZStack {
            // 底色
            (friend.isAvailable ? CLIColors.backgroundHighlight : CLIColors.backgroundSecondary)

            // GIF 氛围背景（填满整个卡片）
            if let gifUrlStr = friend.gifUrl, !gifUrlStr.isEmpty, let gifUrl = URL(string: gifUrlStr) {
                GifImageView(url: gifUrl, contentMode: .scaleAspectFill)
                    .opacity(0.35)
            }

            // 内容
            VStack(spacing: 4) {
                // 有空标签
                Text(friend.isAvailable ? "[有空]" : " ")
                    .font(.system(size: 10, design: .monospaced))
                    .foregroundColor(friend.isAvailable ? CLIColors.green : .clear)

                Spacer(minLength: 4)

                // 动画 emoji
                if friend.needsSchedule {
                    Text("➕")
                        .font(.system(size: 36))
                        .frame(width: 72, height: 72)
                } else {
                    EmojiStateView(emoji: friend.emoji)
                        .frame(width: 72, height: 72)
                }

                Spacer(minLength: 4)

                // 昵称
                Text(friend.nickname)
                    .font(.system(size: 12, weight: .bold, design: .monospaced))
                    .foregroundColor(CLIColors.textPrimary)
                    .lineLimit(1)

                // 状态
                if friend.needsSchedule {
                    Text("+状态")
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundColor(CLIColors.green)
                        .lineLimit(1)
                } else {
                    Text(friend.status)
                        .font(.system(size: 10, design: .monospaced))
                        .foregroundColor(CLIColors.textSecondary)
                        .lineLimit(1)
                }

                // 城市
                if let city = friend.city, !city.isEmpty {
                    Text(friend.isVisiting ? "📍来访·\(city)" : city)
                        .font(.system(size: 10, design: .monospaced))
                        .foregroundColor(friend.isVisiting ? CLIColors.yellow : CLIColors.textSecondary)
                        .lineLimit(1)
                } else {
                    Text(" ")
                        .font(.system(size: 10, design: .monospaced))
                }
            }
            .padding(.horizontal, 4)
            .padding(.vertical, 8)

            // 未读角标（右上角）
            if unreadCount > 0 {
                VStack {
                    HStack {
                        Spacer()
                        Text("[\(unreadCount)]")
                            .font(.system(size: 10, design: .monospaced))
                            .foregroundColor(CLIColors.green)
                            .padding(.trailing, 4)
                            .padding(.top, 2)
                    }
                    Spacer()
                }
            }
        }
        .frame(maxWidth: .infinity)
        .aspectRatio(1, contentMode: .fit)
        .clipShape(RoundedRectangle(cornerRadius: 4))
        .overlay(
            RoundedRectangle(cornerRadius: 4)
                .stroke(borderColor, lineWidth: 1)
        )
    }
}

#Preview {
    GridHomeView()
}
