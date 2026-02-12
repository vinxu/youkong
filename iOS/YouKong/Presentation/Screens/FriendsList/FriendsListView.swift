import SwiftUI

// MARK: - Friends List View

struct FriendsListView: View {
    @StateObject private var viewModel = FriendsListViewModel()
    @StateObject private var notificationManager = NotificationManager.shared
    @State private var selectedFriend: FriendRecommendation?
    @State private var showProfile = false
    @State private var showFriendsHub = false
    @State private var navigationPath = NavigationPath()

    var body: some View {
        NavigationStack(path: $navigationPath) {
            ZStack {
                // 终端背景
                Color.black.ignoresSafeArea()

                VStack(spacing: 0) {
                    // 终端头部
                    terminalHeader

                    // 内容区域
                    if viewModel.isLoading && viewModel.friends.isEmpty {
                        loadingView
                    } else if viewModel.isEmpty {
                        emptyView
                    } else {
                        friendsList
                    }
                }
            }
            .navigationBarHidden(true)
            .sheet(isPresented: $showProfile) {
                ProfileView()
            }
            .sheet(isPresented: $showFriendsHub) {
                FriendsHubView()
            }
            .navigationDestination(for: FriendRecommendation.self) { friend in
                ChatView(partnerId: friend.friendId, partnerName: friend.name, partnerAvatar: friend.avatar)
            }
            .navigationDestination(for: NotificationDestination.self) { destination in
                ChatView(conversationId: destination.conversationId)
            }
        }
        .task {
            await viewModel.loadFriends()
        }
        .onChange(of: notificationManager.shouldNavigateToChat) { shouldNavigate in
            if shouldNavigate, let conversationId = notificationManager.pendingConversationId {
                handleNotificationNavigation(conversationId: conversationId)
            }
        }
    }

    private func handleNotificationNavigation(conversationId: String) {
        // 使用 NotificationDestination 进行导航
        navigationPath.append(NotificationDestination(conversationId: conversationId))
        notificationManager.clearPendingNavigation()
    }

    // MARK: - Terminal Header

    private var terminalHeader: some View {
        VStack(spacing: 0) {
            HStack {
                // 窗口按钮
                HStack(spacing: 8) {
                    Circle().fill(Color.red).frame(width: 12, height: 12)
                    Circle().fill(Color.yellow).frame(width: 12, height: 12)
                    Circle().fill(Color.green).frame(width: 12, height: 12)
                }

                Spacer()

                Text("YouKong — zsh")
                    .font(.system(.caption, design: .monospaced))
                    .foregroundColor(.gray)

                Spacer()

                Button {
                    showProfile = true
                } label: {
                    Image(systemName: "gearshape")
                        .font(.caption)
                        .foregroundColor(.gray)
                }

                Button {
                    showFriendsHub = true
                } label: {
                    Image(systemName: "plus")
                        .font(.caption)
                        .foregroundColor(.gray)
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(Color(white: 0.15))

            // 分隔线
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .frame(height: 1)
        }
    }

    // MARK: - Friends List (CLI Style)

    private var friendsList: some View {
        VStack(spacing: 0) {
            // 朋友列表
            ScrollView {
                LazyVStack(spacing: 0) {
                    // 自己的状态卡片
                    myStatusCard
                        .padding(.bottom, 8)

                    Rectangle()
                        .fill(Color.gray.opacity(0.3))
                        .frame(height: 1)
                        .padding(.bottom, 8)

                    ForEach(viewModel.friends) { friend in
                        Button {
                            navigationPath.append(friend)
                        } label: {
                            terminalFriendRow(friend: friend)
                        }
                        .buttonStyle(PlainButtonStyle())

                        Rectangle()
                            .fill(Color.gray.opacity(0.2))
                            .frame(height: 1)
                            .padding(.leading, 46)
                    }
                }
                .padding(.vertical, 8)
            }
            .refreshable {
                await viewModel.refresh()
            }

            // 底部状态栏
            terminalFooter
        }
    }

    // MARK: - My Status Card

    private var myStatusCard: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 0) {
                // 状态指示符 + Emoji
                HStack(spacing: 4) {
                    if let analysis = viewModel.myAnalysis {
                        Text(CLIConstants.statusSymbol(for: analysis.availability.probability))
                            .font(.system(size: 16, design: .monospaced))
                            .foregroundColor(CLIConstants.color(for: analysis.availability.probability))

                        Text(analysis.lifeStatus.emoji)
                            .font(.system(size: 14))
                    } else {
                        Text("💤")
                            .font(.system(size: 14))
                    }
                }
                .frame(width: 40)

                // 标题
                Text("ME")
                    .font(.system(size: 15, design: .monospaced))
                    .fontWeight(.bold)
                    .foregroundColor(.cyan)

                Spacer()

                // 概率
                if let analysis = viewModel.myAnalysis {
                    Text("\(analysis.availability.probability)%")
                        .font(.system(size: 13, design: .monospaced))
                        .foregroundColor(CLIConstants.color(for: analysis.availability.probability))
                } else {
                    Text("NO DATA")
                        .font(.system(size: 13, design: .monospaced))
                        .foregroundColor(.gray)
                }
            }

            // 状态描述
            if let analysis = viewModel.myAnalysis {
                Text(analysis.lifeStatus.label)
                    .font(.system(size: 13, design: .monospaced))
                    .foregroundColor(.gray)
                    .padding(.leading, 40)

                // 更新时间
                if let updatedAt = analysis.updatedAt {
                    Text(updatedAt.relativeTimeText())
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundColor(.gray.opacity(0.6))
                        .padding(.leading, 40)
                }
            }
        }
        .padding(.vertical, 10)
        .padding(.horizontal, 16)
        .background(Color(white: 0.08))
    }

    private func terminalFriendRow(friend: FriendRecommendation) -> some View {
        let unreadCount = viewModel.getUnreadCount(for: friend.friendId)

        return VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 0) {
                // 状态指示符 + Emoji
                HStack(spacing: 4) {
                    Text(CLIConstants.statusSymbol(for: friend.probability))
                        .font(.system(size: 16, design: .monospaced))
                        .foregroundColor(CLIConstants.color(for: friend.probability))

                    if let emoji = friend.emoji {
                        Text(emoji)
                            .font(.system(size: 14))
                    }
                }
                .frame(width: 40)

                // 朋友名字
                Text(friend.name)
                    .font(.system(size: 15, design: .monospaced))
                    .foregroundColor(Color(white: 0.9))

                Spacer()

                // 未读红点（如果有未读消息）
                if unreadCount > 0 {
                    ZStack {
                        Circle()
                            .fill(Color.red)
                            .frame(width: 18, height: 18)

                        Text("\(unreadCount)")
                            .font(.system(size: 10, design: .monospaced))
                            .foregroundColor(.white)
                            .fontWeight(.semibold)
                    }
                }

                // 概率百分比
                Text(friend.probability == -1 ? "--" : "\(friend.probability)%")
                    .font(.system(size: 13, design: .monospaced))
                    .foregroundColor(CLIConstants.color(for: friend.probability))
                    .frame(width: 50, alignment: .trailing)
            }

            // 自然语言描述
            if friend.hasData {
                Text("> \(friend.displayActivity)")
                    .font(.system(size: 13, design: .monospaced))
                    .foregroundColor(.gray)
                    .lineLimit(2)  // 支持多行
                    .padding(.leading, 44)  // 对齐状态符号
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }


    private var terminalFooter: some View {
        VStack(spacing: 0) {
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .frame(height: 1)

            HStack(spacing: 8) {
                Circle()
                    .fill(Color.green)
                    .frame(width: 6, height: 6)

                Text("\(viewModel.friends.count) friends")
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundColor(.gray)

                if let lastUpdate = viewModel.lastUpdated {
                    Text("·")
                        .foregroundColor(.gray.opacity(0.5))

                    Text("更新于 \(formatTime(lastUpdate))")
                        .font(.system(size: 12, design: .monospaced))
                        .foregroundColor(.gray)
                }

                Spacer()

                Text("ready")
                    .font(.system(size: 11, design: .monospaced))
                    .foregroundColor(.gray.opacity(0.7))
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(Color(white: 0.1))
        }
    }

    private func formatTime(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: date)
    }

    // MARK: - Loading View (CLI Style)

    private var loadingView: some View {
        VStack(spacing: 0) {
            Spacer()

            VStack(spacing: 16) {
                ProgressView()
                    .scaleEffect(1.2)
                    .progressViewStyle(CircularProgressViewStyle(tint: .green))

                Text("> 正在分析朋友状态...")
                    .font(.system(size: 14, design: .monospaced))
                    .foregroundColor(.green)
            }

            Spacer()
        }
    }

    // MARK: - Empty View (CLI Style)

    private var emptyView: some View {
        VStack(spacing: 0) {
            Spacer()

            VStack(spacing: 20) {
                // ASCII Art 表情
                Text("( •_•)")
                    .font(.system(size: 30, design: .monospaced))
                    .foregroundColor(.gray)

                Text("/>💌<\\")
                    .font(.system(size: 24, design: .monospaced))
                    .foregroundColor(.gray)

                Text("还没有朋友哦")
                    .font(.system(size: 16, design: .monospaced))
                    .foregroundColor(Color(white: 0.7))
                    .padding(.top, 8)

                Button {
                    showFriendsHub = true
                } label: {
                    Text("[+] 邀请朋友加入有空")
                        .font(.system(size: 15, design: .monospaced))
                        .foregroundColor(.green)
                        .padding(.vertical, 12)
                        .padding(.horizontal, 20)
                        .overlay(
                            RoundedRectangle(cornerRadius: 4)
                                .stroke(Color.green.opacity(0.5), lineWidth: 1)
                        )
                }
                .padding(.top, 12)
            }

            Spacer()
        }
    }
}

// MARK: - Notification Destination

struct NotificationDestination: Hashable {
    let conversationId: String
}

// MARK: - My Agent Data Sheet (Terminal Style with SSE)

struct MyAgentDataSheet: View {
    @Environment(\.dismiss) private var dismiss

    // 终端输出行
    @State private var terminalLines: [TerminalLine] = [
        TerminalLine(text: "", type: .output),
        TerminalLine(text: "┌─────────────────────────────────────────┐", type: .separator),
        TerminalLine(text: "│     🔍 Holmes Agent v2.0 (Debug)        │", type: .header),
        TerminalLine(text: "│     福尔摩斯推理框架 - 调试模式        │", type: .header),
        TerminalLine(text: "└─────────────────────────────────────────┘", type: .separator),
        TerminalLine(text: "", type: .output),
        TerminalLine(text: "💡 调试模式说明:", type: .info),
        TerminalLine(text: "   • 点击 [上报] 按钮手动上报状态", type: .output),
        TerminalLine(text: "   • 点击 [运行] 按钮开始福尔摩斯分析", type: .output),
        TerminalLine(text: "   • 点击 [清屏] 按钮清空终端输出", type: .output),
        TerminalLine(text: "", type: .output),
        TerminalLine(text: "⏳ 等待操作...", type: .info),
        TerminalLine(text: "", type: .output)
    ]
    @State private var isRunning = false
    @State private var showCursor = true
    @State private var currentThinkingLine: String = ""

    // 数据状态
    // screenStatus 已移除
    @State private var locationStatus: LocationStatus = .unknown
    @State private var deviceStatus: DeviceStatus = .unknown

    // 状态上报管理器
    @StateObject private var statusReportManager = StatusReportManager.shared

    // SSE 客户端
    private let sseClient = SSEClient()

    // 光标闪烁定时器
    let cursorTimer = Timer.publish(every: 0.5, on: .main, in: .common).autoconnect()

    var body: some View {
        NavigationStack {
            ZStack {
                // 终端背景
                Color.black.ignoresSafeArea()

                VStack(spacing: 0) {
                    // 终端头部
                    terminalHeader

                    // 终端输出区域
                    ScrollViewReader { proxy in
                        ScrollView {
                            LazyVStack(alignment: .leading, spacing: 1) {
                                ForEach(terminalLines) { line in
                                    TerminalLineView(line: line)
                                        .id(line.id)
                                }

                                // 正在输入的思考内容
                                if !currentThinkingLine.isEmpty {
                                    Text("   " + currentThinkingLine)
                                        .font(.system(size: 12, design: .monospaced))
                                        .foregroundColor(.cyan.opacity(0.8))
                                        .id("thinking")
                                }

                                // 光标行（运行时显示）
                                if isRunning {
                                    HStack(spacing: 0) {
                                        Text("› ")
                                            .foregroundColor(.green)
                                        if showCursor {
                                            Text("▌")
                                                .foregroundColor(.green)
                                        }
                                    }
                                    .font(.system(.caption, design: .monospaced))
                                }

                                // 底部锚点（始终存在）
                                Color.clear
                                    .frame(height: 1)
                                    .id("bottom")
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 8)
                        }
                        .onChange(of: terminalLines.count) { _ in
                            scrollToBottom(proxy: proxy)
                        }
                        .onChange(of: currentThinkingLine) { _ in
                            scrollToBottom(proxy: proxy)
                        }
                        .onChange(of: isRunning) { running in
                            if !running {
                                // 分析结束时确保滚动到底部
                                DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                                    scrollToBottom(proxy: proxy)
                                }
                            }
                        }
                    }

                    // 底部操作栏
                    bottomBar
                }
            }
            .navigationBarHidden(true)
            // ⚠️ 禁用自动分析（调试模式）- 需要手动点击"运行"按钮
            // .task {
            //     await runStreamAnalysis()
            // }
            .onReceive(cursorTimer) { _ in
                if isRunning {
                    showCursor.toggle()
                }
            }
        }
        .preferredColorScheme(.dark)
    }

    // MARK: - Terminal Header

    private var terminalHeader: some View {
        HStack {
            // 窗口按钮
            HStack(spacing: 8) {
                Circle().fill(Color.red).frame(width: 12, height: 12)
                Circle().fill(Color.yellow).frame(width: 12, height: 12)
                Circle().fill(Color.green).frame(width: 12, height: 12)
            }

            Spacer()

            Text("Holmes Agent — zsh")
                .font(.system(.caption, design: .monospaced))
                .foregroundColor(.gray)

            Spacer()

            Button {
                dismiss()
            } label: {
                Image(systemName: "xmark")
                    .font(.caption)
                    .foregroundColor(.gray)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(Color(white: 0.15))
    }

    // MARK: - Bottom Bar

    private var bottomBar: some View {
        HStack(spacing: 12) {
            Button {
                Task {
                    await runStreamAnalysis()
                }
            } label: {
                HStack(spacing: 6) {
                    Image(systemName: isRunning ? "stop.fill" : "play.fill")
                        .font(.caption)
                    Text(isRunning ? "分析中..." : "运行")
                }
                .font(.system(.caption, design: .monospaced))
                .foregroundColor(isRunning ? .gray : .black)
                .padding(.horizontal, 14)
                .padding(.vertical, 8)
                .background(isRunning ? Color.gray.opacity(0.3) : Color.green)
                .cornerRadius(4)
            }
            .disabled(isRunning)

            Button {
                terminalLines.removeAll()
                currentThinkingLine = ""
            } label: {
                HStack(spacing: 6) {
                    Image(systemName: "trash")
                        .font(.caption)
                    Text("清屏")
                }
                .font(.system(.caption, design: .monospaced))
                .foregroundColor(.green)
                .padding(.horizontal, 14)
                .padding(.vertical, 8)
                .overlay(
                    RoundedRectangle(cornerRadius: 4)
                        .stroke(Color.green.opacity(0.5), lineWidth: 1)
                )
            }
            .disabled(isRunning)

            // 上报状态按钮
            Button {
                Task {
                    await statusReportManager.reportStatus()
                }
            } label: {
                HStack(spacing: 6) {
                    Image(systemName: statusReportManager.isReporting ? "arrow.up.circle.fill" : "arrow.up.circle")
                        .font(.caption)
                    Text(statusReportManager.isReporting ? "上报中..." : "上报")
                }
                .font(.system(.caption, design: .monospaced))
                .foregroundColor(statusReportManager.isReporting ? .gray : .cyan)
                .padding(.horizontal, 14)
                .padding(.vertical, 8)
                .overlay(
                    RoundedRectangle(cornerRadius: 4)
                        .stroke((statusReportManager.isReporting ? Color.gray : Color.cyan).opacity(0.5), lineWidth: 1)
                )
            }
            .disabled(isRunning || statusReportManager.isReporting)

            Spacer()

            // 状态指示
            VStack(alignment: .trailing, spacing: 2) {
                HStack(spacing: 4) {
                    Circle()
                        .fill(isRunning ? Color.yellow : Color.green)
                        .frame(width: 6, height: 6)
                    Text(isRunning ? "streaming" : "ready")
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundColor(.gray)
                }

                // 上次上报时间
                if let lastReportTime = statusReportManager.lastReportTime {
                    Text("上报于 \(formatTime(lastReportTime))")
                        .font(.system(size: 9, design: .monospaced))
                        .foregroundColor(.gray.opacity(0.7))
                } else if statusReportManager.lastReportError != nil {
                    Text("上报失败")
                        .font(.system(size: 9, design: .monospaced))
                        .foregroundColor(.red.opacity(0.7))
                }
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(Color(white: 0.1))
    }

    // MARK: - Terminal Operations

    private func appendLine(_ text: String, type: TerminalLine.LineType = .output) {
        terminalLines.append(TerminalLine(text: text, type: type))
    }

    private func appendCommand(_ command: String) {
        terminalLines.append(TerminalLine(text: command, type: .command))
    }

    private func scrollToBottom(proxy: ScrollViewProxy) {
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.05) {
            withAnimation(.easeOut(duration: 0.15)) {
                proxy.scrollTo("bottom", anchor: .bottom)
            }
        }
    }

    // MARK: - Stream Analysis

    private func runStreamAnalysis() async {
        guard !isRunning else { return }

        isRunning = true
        terminalLines.removeAll()
        currentThinkingLine = ""

        // 显示启动信息
        appendLine("", type: .output)
        appendLine("┌─────────────────────────────────────────┐", type: .separator)
        appendLine("│     🔍 Holmes Agent v2.0 (Stream)       │", type: .header)
        appendLine("│     福尔摩斯推理框架                    │", type: .header)
        appendLine("└─────────────────────────────────────────┘", type: .separator)
        appendLine("", type: .output)

        // 采集本地数据
        appendCommand("$ holmes collect --local")
        try? await Task.sleep(nanoseconds: 200_000_000)

        loadData()
        let calendarStatus = CalendarDataCollector.shared.getCurrentStatus()
        let movementStatus = MovementDataCollector.shared.getCurrentStatus()

        // 显示本地采集的数据摘要
        appendLine("", type: .output)
        appendLine("📱 本地数据采集完成:", type: .section)
        appendLine("├─ 位置: \(locationStatus.placeName ?? locationStatus.placeType.rawValue) (停留\(locationStatus.atPlaceSinceMinutes)分钟)", type: .output)
        appendLine("├─ 日历: \(calendarStatus.hasCurrentEvent ? "有日程" : "无日程") (剩余\(calendarStatus.todayRemainingCount)个)", type: .output)
        appendLine("├─ 运动: \(movementStatus.movementType.displayName) (今日\(movementStatus.stepsToday)步)", type: .output)
        appendLine("└─ 电量: \(Int(deviceStatus.batteryLevel * 100))% (\(deviceStatus.isCharging ? "充电中" : "未充电"))", type: .output)
        appendLine("", type: .output)

        // 添加数据时间戳
        let collectionTime = Date().formatted(.dateTime.hour().minute().second())
        appendLine("⏰ 采集时间: \(collectionTime)", type: .info)
        appendLine("", type: .output)

        try? await Task.sleep(nanoseconds: 300_000_000)

        // 发起流式请求
        appendCommand("$ holmes analyze --stream --model qwen3-max")
        appendLine("", type: .output)

        let request = buildRequest(calendarStatus: calendarStatus, movementStatus: movementStatus)

        do {
            try await sseClient.streamHolmesAnalysis(request: request) { [self] event in
                self.handleStreamEvent(event)
            }
        } catch {
            // 流式 API 失败，回退到普通 API
            print("[Holmes] SSE failed: \(error.localizedDescription), falling back to normal API")
            appendLine("", type: .output)
            appendLine("[WARN] 流式连接失败: \(error.localizedDescription)", type: .warning)
            appendLine("[INFO] 切换到普通模式...", type: .info)
            appendLine("", type: .output)

            await fallbackToNonStreamingAPI(request: request)
        }

        // 完成
        appendLine("", type: .output)
        appendLine("─────────────────────────────────────────", type: .separator)
        let timestamp = Date().formatted(.dateTime.hour().minute().second())
        appendLine("分析完成 @ \(timestamp)", type: .info)
        appendLine("", type: .output)

        isRunning = false
    }

    // MARK: - Diagnostics

    private func runDiagnostics() async {
        guard !isRunning else { return }

        isRunning = true
        terminalLines.removeAll()

        appendLine("", type: .output)
        appendLine("┌─────────────────────────────────────────┐", type: .separator)
        appendLine("│        🔍 数据采集诊断工具              │", type: .header)
        appendLine("└─────────────────────────────────────────┘", type: .separator)
        appendLine("", type: .output)

        appendCommand("$ holmes diagnose --all")
        appendLine("", type: .output)

        // 1. 权限检查
        appendLine("═══════════════════════════════════════", type: .separator)
        appendLine("1️⃣  权限状态检查", type: .section)
        appendLine("═══════════════════════════════════════", type: .separator)
        appendLine("", type: .output)

        let permissionManager = PermissionManager.shared
        appendLine("📍 位置权限: \(permissionManager.status.location ? "✅ 已授权" : "❌ 未授权")", type: .output)
        appendLine("📅 日历权限: \(permissionManager.status.calendar ? "✅ 已授权" : "❌ 未授权")", type: .output)
        appendLine("🏃 运动权限: \(permissionManager.status.motion ? "✅ 已授权" : "❌ 未授权")", type: .output)
        appendLine("", type: .output)

        // 2. 数据采集器状态
        appendLine("═══════════════════════════════════════", type: .separator)
        appendLine("2️⃣  数据采集器状态", type: .section)
        appendLine("═══════════════════════════════════════", type: .separator)
        appendLine("", type: .output)

        // 位置数据
        let locationCollector = LocationDataCollector.shared
        appendLine("📍 位置数据采集器:", type: .section)
        appendLine("   - 监控状态: \(locationCollector.isMonitoring ? "✅ 运行中" : "❌ 未启动")", type: .output)
        let locationStatus = locationCollector.currentStatus
        appendLine("   - 地点类型: \(locationStatus.placeType.rawValue)", type: .output)
        appendLine("   - 地点名称: \(locationStatus.placeName ?? "未知")", type: .output)
        if let lat = locationStatus.latitude, let lng = locationStatus.longitude {
            appendLine("   - 坐标: (\(String(format: "%.4f", lat)), \(String(format: "%.4f", lng)))", type: .output)
        }
        appendLine("", type: .output)

        // 运动数据
        let movementCollector = MovementDataCollector.shared
        appendLine("🏃 运动数据采集器:", type: .section)
        appendLine("   - 权限状态: \(movementCollector.isAuthorized ? "✅ 已授权" : "❌ 未授权")", type: .output)
        appendLine("   - 监控状态: \(movementCollector.isMonitoring ? "✅ 运行中" : "❌ 未启动")", type: .output)

        if movementCollector.isAuthorized {
            appendLine("   - 正在获取实时数据...", type: .info)
            let movementStatus = await movementCollector.fetchCurrentStatus()
            appendLine("   - 移动状态: \(movementStatus.isMoving ? "移动中" : "静止")", type: .output)
            appendLine("   - 移动类型: \(movementStatus.movementType.rawValue)", type: .output)
            appendLine("   - 今日步数: \(movementStatus.stepsToday)", type: .output)
            appendLine("   - 最近一小时步数: \(movementStatus.stepsLastHour)", type: .output)
        } else {
            appendLine("   - ⚠️  需要授权运动与健身权限", type: .warning)
        }
        appendLine("", type: .output)

        // 日历数据
        let calendarCollector = CalendarDataCollector.shared
        appendLine("📅 日历数据采集器:", type: .section)
        appendLine("   - 权限状态: \(calendarCollector.isAuthorized ? "✅ 已授权" : "❌ 未授权")", type: .output)

        if calendarCollector.isAuthorized {
            let calendarStatus = calendarCollector.getCurrentStatus()
            appendLine("   - 当前日程: \(calendarStatus.hasCurrentEvent ? "有" : "无")", type: .output)
            if let title = calendarStatus.currentEventTitle {
                appendLine("   - 日程标题: \(title)", type: .output)
            }
            appendLine("   - 今日剩余: \(calendarStatus.todayRemainingCount) 个", type: .output)
        } else {
            appendLine("   - ⚠️  需要授权日历权限", type: .warning)
        }
        appendLine("", type: .output)

        // 3. 设备状态
        appendLine("═══════════════════════════════════════", type: .separator)
        appendLine("3️⃣  设备状态", type: .section)
        appendLine("═══════════════════════════════════════", type: .separator)
        appendLine("", type: .output)

        let deviceStatus = DeviceStatusCollector.shared.getCurrentStatus()
        appendLine("🔋 电量: \(Int(deviceStatus.batteryLevel * 100))%", type: .output)
        appendLine("⚡ 充电: \(deviceStatus.isCharging ? "是" : "否")", type: .output)
        appendLine("🔅 亮度: \(Int(deviceStatus.screenBrightness * 100))%", type: .output)
        appendLine("📶 网络: \(deviceStatus.networkType.rawValue)", type: .output)
        appendLine("🎧 耳机: \(deviceStatus.isHeadphonesConnected ? "已连接" : "未连接")", type: .output)
        appendLine("🔕 专注: \(deviceStatus.isFocusModeOn ? "开启" : "关闭")", type: .output)
        appendLine("🪫 低电量: \(deviceStatus.isLowPowerMode ? "开启" : "关闭")", type: .output)
        appendLine("", type: .output)

        // 4. 建议
        appendLine("═══════════════════════════════════════", type: .separator)
        appendLine("4️⃣  问题诊断与建议", type: .section)
        appendLine("═══════════════════════════════════════", type: .separator)
        appendLine("", type: .output)

        var issues: [String] = []

        if !movementCollector.isAuthorized {
            issues.append("⚠️  运动数据未授权 → 无法获取步数和运动状态")
        } else if !movementCollector.isMonitoring {
            issues.append("⚠️  运动数据采集器未启动 → 重启 App 后应自动启动")
        }

        if !calendarCollector.isAuthorized {
            issues.append("⚠️  日历未授权 → 无法判断日程安排")
        }

        if !locationCollector.isMonitoring {
            issues.append("⚠️  位置采集器未启动 → 重启 App 后应自动启动")
        }

        if issues.isEmpty {
            appendLine("✅ 所有数据采集器工作正常！", type: .success)
        } else {
            for issue in issues {
                appendLine(issue, type: .warning)
                appendLine("", type: .output)
            }

            appendLine("💡 建议操作:", type: .info)
            appendLine("   1. 前往「设置」→「重新授权权限」", type: .output)
            appendLine("   2. 重启 App 以启动所有数据采集器", type: .output)
        }

        appendLine("", type: .output)
        appendLine("─────────────────────────────────────────", type: .separator)
        appendLine("诊断完成", type: .info)
        appendLine("", type: .output)

        isRunning = false
    }

    // MARK: - Fallback to Non-Streaming API

    private func fallbackToNonStreamingAPI(request: StatusReportRequest) async {
        print("[Holmes] Starting fallback to non-streaming API...")
        appendCommand("$ holmes analyze --no-stream")
        appendLine("", type: .output)

        do {
            let repo = AgentRepositoryImpl()
            print("[Holmes] Calling reportStatus...")
            let response = try await repo.reportStatus(request: request)
            print("[Holmes] Got response, analysis: \(response.analysis != nil)")

            if let analysis = response.analysis {
                print("[Holmes] Showing analysis results...")
                // 显示分析结果
                appendLine("📡 收集线索...", type: .phase)
                appendLine("├─ 时间: \(formatCurrentTime())", type: .clue)
                appendLine("└─ 位置: \(locationStatus.placeName ?? locationStatus.placeType.rawValue)", type: .clue)
                appendLine("", type: .output)

                appendLine("🧠 分析特征...", type: .phase)
                appendLine("├─ 状态类型: \(analysis.lifeStatus.label)", type: .feature)
                appendLine("└─ 置信度: \(analysis.availability.confidence)", type: .feature)
                appendLine("", type: .output)

                appendLine("💭 推理完成", type: .phase)
                appendLine("", type: .output)

                // 显示结论
                appendLine("═══════════════════════════════════════", type: .separator)
                appendLine("           📊 最终判断结果", type: .header)
                appendLine("═══════════════════════════════════════", type: .separator)
                appendLine("", type: .output)

                let statusEmoji = analysis.availability.probability >= 50 ? "🟢" : "🔴"
                let statusText = analysis.availability.probability >= 50 ? "有空" : "忙碌"
                appendLine("   \(statusEmoji) 状态: \(statusText) \(analysis.lifeStatus.emoji)", type: .result)
                appendLine("   📈 概率: \(analysis.availability.probability)%", type: .result)
                appendLine("   🎯 置信度: \(analysis.availability.confidence)", type: .result)
                appendLine("   💬 \(analysis.lifeStatus.label)", type: .result)
                appendLine("", type: .output)
                appendLine("   \(analysis.availability.reason)", type: .info)
                appendLine("", type: .output)
                appendLine("═══════════════════════════════════════", type: .separator)
            } else {
                appendLine("[WARN] 服务器未返回分析结果", type: .warning)
            }
        } catch {
            appendLine("[ERROR] \(error.localizedDescription)", type: .error)
        }
    }

    private func formatCurrentTime() -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "EEEE HH:mm"
        formatter.locale = Locale(identifier: "zh_CN")
        return formatter.string(from: Date())
    }

    private func formatTime(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm:ss"
        return formatter.string(from: date)
    }

    // MARK: - Handle Stream Events

    private func handleStreamEvent(_ event: HolmesStreamEvent) {
        // 处理最终结果（无 type 但有 result）
        if event.isFinalResult, let fullResult = event.result {
            flushThinkingLine()
            displaySSEFinalResult(fullResult)
            return
        }

        guard let eventType = event.type else { return }

        switch eventType {
        case .phase:
            flushThinkingLine()
            if let content = event.content {
                appendLine("", type: .output)
                appendLine(content, type: .phase)
            }

        case .clue:
            if let content = event.content {
                appendLine(content, type: .clue)
            }

        case .feature:
            if let content = event.content {
                appendLine(content, type: .feature)
            }

        case .thinking:
            // 流式显示思考过程，遇到换行符时提交当前行
            if let content = event.content {
                // 处理内容中的换行符
                let parts = content.components(separatedBy: "\n")
                for (index, part) in parts.enumerated() {
                    currentThinkingLine += part
                    // 如果不是最后一部分，说明有换行，提交当前行
                    if index < parts.count - 1 {
                        if !currentThinkingLine.isEmpty {
                            appendLine("   " + currentThinkingLine, type: .thinking)
                        }
                        currentThinkingLine = ""
                    }
                }
            }

        case .conclusion:
            flushThinkingLine()
            if let content = event.content {
                appendLine(content, type: .conclusion)
            }

        case .done:
            flushThinkingLine()
            if let fullResult = event.result {
                displaySSEFinalResult(fullResult)
            }

        case .error:
            if let content = event.content {
                appendLine("[ERROR] \(content)", type: .error)
            }

        // Holmes 2.0 新增事件类型
        case .context:
            if let content = event.content {
                appendLine(content, type: .clue)
            }

        case .anomaly:
            if let content = event.content {
                appendLine(content, type: .thinking)
            }

        case .narrative:
            // 流式显示叙事推理过程
            if let content = event.content {
                let parts = content.components(separatedBy: "\n")
                for (index, part) in parts.enumerated() {
                    currentThinkingLine += part
                    if index < parts.count - 1 {
                        if !currentThinkingLine.isEmpty {
                            appendLine("   " + currentThinkingLine, type: .thinking)
                        }
                        currentThinkingLine = ""
                    }
                }
            }
        }
    }

    // MARK: - Helper Methods

    private func flushThinkingLine() {
        if !currentThinkingLine.isEmpty {
            appendLine("   " + currentThinkingLine, type: .thinking)
            currentThinkingLine = ""
        }
    }

    private func displaySSEFinalResult(_ fullResult: SSEFullResult) {
        appendLine("", type: .output)
        appendLine("═══════════════════════════════════════", type: .separator)
        appendLine("           📊 最终判断结果", type: .header)
        appendLine("═══════════════════════════════════════", type: .separator)
        appendLine("", type: .output)

        if let result = fullResult.result {
            let available = result.available ?? false
            let statusEmoji = available ? "🟢" : "🔴"
            let statusText = available ? "有空" : "忙碌"
            appendLine("   \(statusEmoji) 状态: \(statusText) \(result.emoji ?? "")", type: .result)
            appendLine("   📈 概率: \(result.probability ?? 0)%", type: .result)
            appendLine("   🎯 置信度: \(result.confidence ?? "unknown")", type: .result)
            appendLine("   💬 \(result.summary ?? "")", type: .result)

            if let reason = result.reason, !reason.isEmpty {
                appendLine("", type: .output)
                appendLine("   \(reason)", type: .info)
            }
        }

        // 显示推理结论（如果有）
        if let reasoning = fullResult.reasoning, let conclusion = reasoning.conclusion, !conclusion.isEmpty {
            appendLine("", type: .output)
            appendLine("   📝 \(conclusion)", type: .info)
        }

        appendLine("", type: .output)
        appendLine("═══════════════════════════════════════", type: .separator)
    }

    // MARK: - Data Loading

    private func loadData() {
        locationStatus = LocationDataCollector.shared.currentStatus
        deviceStatus = DeviceStatusCollector.shared.getCurrentStatus()
    }

    private func buildRequest(calendarStatus: CalendarStatus, movementStatus: MovementStatus) -> StatusReportRequest {
        // 屏幕数据已移除
        let screenData: ScreenRequestData? = nil

        let locationData = LocationRequestData(
            placeType: locationStatus.placeType.rawValue,
            atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
            city: locationStatus.city
        )

        let extendedLocationData = ExtendedLocationRequestData(
            placeType: locationStatus.placeType.rawValue,
            placeName: locationStatus.placeName,
            atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
            latitude: locationStatus.latitude,
            longitude: locationStatus.longitude
        )

        let batteryData = BatteryRequestData(
            batteryLevel: Int(deviceStatus.batteryLevel * 100),
            batteryState: deviceStatus.batteryState.rawValue,
            isCharging: deviceStatus.isCharging
        )

        let modeData = ModeRequestData(
            isLowPowerMode: deviceStatus.isLowPowerMode,
            isFocusModeOn: deviceStatus.isFocusModeOn
        )

        let connectionData = ConnectionRequestData(
            isHeadphonesConnected: deviceStatus.isHeadphonesConnected,
            networkType: deviceStatus.networkType.rawValue
        )

        let displayData = DisplayRequestData(
            screenBrightness: deviceStatus.screenBrightness
        )

        let calendarData: CalendarRequestData? = calendarStatus.todayRemainingCount >= 0 ?
            CalendarRequestData(
                hasCurrentEvent: calendarStatus.hasCurrentEvent,
                currentEventTitle: calendarStatus.currentEventTitle,
                eventEndMinutes: calendarStatus.eventEndMinutes,
                nextEventInMinutes: calendarStatus.nextEventInMinutes,
                todayRemainingCount: calendarStatus.todayRemainingCount
            ) : nil

        let movementData: MovementRequestData? = movementStatus.stepsToday >= 0 ?
            MovementRequestData(
                isMoving: movementStatus.isMoving,
                movementType: movementStatus.movementType.rawValue,
                stepsToday: movementStatus.stepsToday,
                stepsLastHour: movementStatus.stepsLastHour,
                stationaryMinutes: movementStatus.stationaryMinutes
            ) : nil

        return StatusReportRequest(
            screen: screenData,
            location: locationData,
            extendedLocation: extendedLocationData,
            battery: batteryData,
            mode: modeData,
            connection: connectionData,
            display: displayData,
            calendar: calendarData,
            movement: movementData
        )
    }
}

// MARK: - Terminal Line Model

struct TerminalLine: Identifiable {
    let id = UUID()
    let text: String
    let type: LineType
    let timestamp = Date()

    enum LineType {
        // 基础类型
        case command        // 命令行 (绿色 $)
        case output         // 普通输出 (灰白)
        case info           // 信息 (青色)
        case success        // 成功 (绿色)
        case warning        // 警告 (黄色)
        case error          // 错误 (红色)
        case section        // 段落标题 (黄色加粗)
        case header         // 大标题 (白色加粗)
        case separator      // 分隔线 (灰色)
        case result         // 结果 (白色加粗)

        // SSE 流式类型
        case phase          // 阶段提示 (紫色)
        case clue           // 线索 (绿色)
        case feature        // 特征 (蓝色)
        case thinking       // 思考过程 (青色斜体)
        case conclusion     // 结论 (绿色)
    }
}

// MARK: - Terminal Line View

struct TerminalLineView: View {
    let line: TerminalLine

    var body: some View {
        HStack(alignment: .top, spacing: 0) {
            // 命令行前缀
            if line.type == .command {
                Text("$ ")
                    .foregroundColor(.green)
                    .fontWeight(.bold)
            }

            // 内容
            Text(line.text)
                .foregroundColor(textColor)
                .fontWeight(fontWeight)
                .italic(line.type == .thinking)
        }
        .font(.system(size: fontSize, design: .monospaced))
    }

    private var textColor: Color {
        switch line.type {
        case .command: return .green
        case .output: return Color(white: 0.75)
        case .info: return .cyan
        case .success: return .green
        case .warning: return .yellow
        case .error: return .red
        case .section: return .yellow
        case .header: return .white
        case .separator: return Color(white: 0.4)
        case .result: return .white

        // SSE 流式类型
        case .phase: return .purple
        case .clue: return Color(red: 0.4, green: 0.8, blue: 0.4)
        case .feature: return Color(red: 0.4, green: 0.6, blue: 1.0)
        case .thinking: return .cyan.opacity(0.85)
        case .conclusion: return .green
        }
    }

    private var fontWeight: Font.Weight {
        switch line.type {
        case .header, .result: return .bold
        case .section, .phase, .command: return .semibold
        case .conclusion: return .medium
        default: return .regular
        }
    }

    private var fontSize: CGFloat {
        switch line.type {
        case .header: return 13
        case .separator: return 11
        default: return 12
        }
    }
}

#Preview {
    FriendsListView()
        .environmentObject(AuthManager.shared)
}
