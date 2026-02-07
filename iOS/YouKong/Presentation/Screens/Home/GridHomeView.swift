import SwiftUI

struct GridHomeView: View {
    @StateObject private var viewModel = GridHomeViewModel()
    @StateObject private var voiceVM = VoiceScheduleViewModel()
    @StateObject private var scheduleVM = MyScheduleTimelineViewModel()
    @State private var showStatusAnalysis = false
    @State private var showAIInference = false
    @State private var selectedTab = 0

    // 按住说话相关状态
    @State private var isPressingVoiceButton = false
    @State private var dragOffset: CGFloat = 0
    @State private var showCancelZone = false
    @State private var showPermissionAlert = false

    // 引导气泡（关闭后持久记住）
    @State private var showGuideBubble = !UserDefaults.standard.bool(forKey: "voice_guide_dismissed")

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
                            // 立即刷新首页
                            Task {
                                await viewModel.loadGrid()
                            }
                            // 刷新我的状态表
                            Task {
                                await scheduleVM.refresh()
                            }
                            // 延迟关闭覆盖层
                            DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) {
                                voiceVM.showOverlay = false
                                voiceVM.reset()
                            }
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
            // 切回好友 tab 时自动刷新数据
            if newTab == 0 {
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
                    gifUrl: friend.gifUrl,
                    giphyQuery: friend.giphyQuery,
                    useGif: friend.useGif
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

            // Tab 内容区域 - 支持左右滑动
            TabView(selection: $selectedTab) {
                // Tab 0: 好友宫格
                friendsTabContent
                    .tag(0)

                // Tab 1: 我的状态表
                MyScheduleTimelineContent(viewModel: scheduleVM)
                    .tag(1)
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
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
                            .foregroundColor(CLIColors.yellow)
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
                // 宫格内容 - 使用 ScrollView
                ScrollView {
                    FriendGrid(
                        friends: viewModel.friends,
                        getUnreadCount: { friendId in
                            viewModel.getUnreadCount(for: friendId)
                        }
                    )
                    .padding(.horizontal, 16)
                    .padding(.top, 16)
                    .padding(.bottom, 16)
                }
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

            GeometryReader { geometry in
                HStack(spacing: 8) {
                    // 一键生成按钮（左边 1/3）
                    Button {
                        showAIInference = true
                    } label: {
                        HStack(spacing: 4) {
                            Text("🤖")
                            Text("一键生成")
                        }
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.cyan)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .overlay(
                            Rectangle()
                                .stroke(CLIColors.cyan, lineWidth: 1)
                        )
                    }
                    .buttonStyle(.borderless)
                    .frame(width: (geometry.size.width - 24) / 3)

                    // 按住说话生成按钮（右边 2/3）
                    voiceButtonContent
                        .frame(width: (geometry.size.width - 24) * 2 / 3)
                        .gesture(
                            DragGesture(minimumDistance: 0)
                                .onChanged { value in
                                    // 如果不能录音（AI 正在处理），忽略触摸
                                    guard voiceVM.canRecord else { return }

                                    if !isPressingVoiceButton {
                                        // 检查权限
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
                                        // 开始按住
                                        isPressingVoiceButton = true
                                        voiceVM.startRecording()
                                    }
                                    // 更新拖动偏移
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
                                        // 取消录音
                                        voiceVM.cancelRecording()
                                    } else {
                                        // 提交录音（覆盖层由 onChange 控制）
                                        Task {
                                            await voiceVM.submitRecording()
                                        }
                                    }
                                    dragOffset = 0
                                    showCancelZone = false
                                }
                        )
                }
                .padding(.horizontal, 16)
            }
            .frame(height: 44)
            .padding(.vertical, 12)
            .background(CLIColors.background)
        }
    }

    // MARK: - Guide Bubble View

    private var guideBubbleView: some View {
        HStack(alignment: .top, spacing: 8) {
            Text("💡")
                .font(.system(size: 14))

            Text("按住说话，告诉我你在做什么或接下来的安排\n例：\"我在吃饭\" 或 \"明天上午开会，下午健身\"")
                .font(.system(size: 11, design: .monospaced))
                .foregroundColor(CLIColors.textSecondary)
                .lineSpacing(4)
                .frame(maxWidth: .infinity, alignment: .leading)

            Button {
                showGuideBubble = false
                UserDefaults.standard.set(true, forKey: "voice_guide_dismissed")
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
                Text("按住说话生成")
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

    private var gridSize: Int {
        let count = friends.count
        if count <= 1 { return 1 }
        if count <= 4 { return 2 }
        if count <= 9 { return 3 }
        return 4
    }

    private var columns: [GridItem] {
        Array(repeating: GridItem(.flexible(), spacing: 8), count: gridSize)
    }

    var body: some View {
        LazyVGrid(columns: columns, spacing: 8) {
            ForEach(friends) { friend in
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

// MARK: - Friend Card (CLI 风格)

struct FriendCard: View {
    let friend: FriendStatus
    var unreadCount: Int = 0

    // 优先级：有空 > 有未读消息 > 普通
    private var borderColor: Color {
        if friend.isAvailable {
            return CLIColors.yellow  // 金色皇冠边框
        }
        return unreadCount > 0 ? CLIColors.green : CLIColors.border
    }

    // 有空时使用皇冠边框样式
    private var topBorder: String {
        friend.isAvailable ? "╔═[有空]═╗" : "┌──────────┐"
    }

    private var bottomBorder: String {
        friend.isAvailable ? "╚════════╝" : "└──────────┘"
    }

    var body: some View {
        VStack(spacing: 0) {
            // 顶部边框（带未读标记）
            ZStack {
                Text(topBorder)
                    .font(.cliCaptionSmall)
                    .foregroundColor(borderColor)

                // 未读消息角标（有空状态下也显示）
                if unreadCount > 0 {
                    HStack {
                        Spacer()
                        Text("[\(unreadCount)]")
                            .font(.cliCaptionSmall)
                            .foregroundColor(CLIColors.green)
                    }
                    .padding(.trailing, 4)
                }
            }

            // 内容区域
            VStack(spacing: 4) {
                if friend.useGif, let gifUrlStr = friend.gifUrl, let gifUrl = URL(string: gifUrlStr) {
                    GifImageView(url: gifUrl)
                        .frame(width: 40, height: 40)
                        .cornerRadius(6)
                } else {
                    Text(friend.emoji)
                        .font(.system(size: 32))
                }

                Text(friend.nickname)
                    .font(.cliBodySmall)
                    .fontWeight(.bold)
                    .foregroundColor(CLIColors.textPrimary)
                    .lineLimit(1)

                Text(friend.status)
                    .font(.cliCaptionSmall)
                    .foregroundColor(CLIColors.textSecondary)
                    .lineLimit(1)

                // 显示城市（如果有），否则不显示
                if let city = friend.city, !city.isEmpty {
                    Text(city)
                        .font(.cliCaptionSmall)
                        .foregroundColor(CLIColors.textWeak)
                }
            }
            .padding(.vertical, 8)

            // 底部边框
            Text(bottomBorder)
                .font(.cliCaptionSmall)
                .foregroundColor(borderColor)
        }
        .frame(maxWidth: .infinity)
        .background(
            friend.isAvailable
                ? CLIColors.backgroundHighlight  // 有空时背景微亮
                : CLIColors.backgroundSecondary
        )
        // 有空时添加光晕效果
        .shadow(
            color: friend.isAvailable ? CLIColors.yellow.opacity(0.4) : .clear,
            radius: friend.isAvailable ? 8 : 0,
            x: 0,
            y: 0
        )
    }
}

#Preview {
    GridHomeView()
}
