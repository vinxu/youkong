import SwiftUI

#if DEBUG
struct DebugPanelView: View {
    @ObservedObject var debugTool = DebugTool.shared
    @Environment(\.dismiss) private var dismiss
    @State private var selectedTab = 0

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Tab Picker
                Picker("", selection: $selectedTab) {
                    Text("Agent").tag(0)
                    Text("网络").tag(1)
                    Text("日志").tag(2)
                    Text("设置").tag(3)
                    Text("信息").tag(4)
                }
                .pickerStyle(.segmented)
                .padding()

                // Content
                TabView(selection: $selectedTab) {
                    AgentDataView()
                        .tag(0)

                    NetworkLogsView()
                        .tag(1)

                    ConsoleLogsView()
                        .tag(2)

                    DebugSettingsView()
                        .tag(3)

                    AppInfoView()
                        .tag(4)
                }
                .tabViewStyle(.page(indexDisplayMode: .never))
            }
            .navigationTitle("调试工具")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("关闭") {
                        dismiss()
                    }
                }
            }
        }
    }
}

// MARK: - Network Logs View
struct NetworkLogsView: View {
    @ObservedObject var debugTool = DebugTool.shared
    @State private var selectedLog: NetworkLog?

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text("共 \(debugTool.networkLogs.count) 条请求")
                    .font(.caption)
                    .foregroundColor(.secondary)

                Spacer()

                Button("清空") {
                    debugTool.clearNetworkLogs()
                }
                .font(.caption)
            }
            .padding(.horizontal)
            .padding(.vertical, 8)

            if debugTool.networkLogs.isEmpty {
                VStack(spacing: 8) {
                    Image(systemName: "network.slash")
                        .font(.system(size: 40))
                        .foregroundColor(.secondary)
                    Text("暂无网络请求")
                        .font(.headline)
                        .foregroundColor(.secondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List {
                    ForEach(debugTool.networkLogs) { log in
                        NetworkLogRow(log: log)
                            .contentShape(Rectangle())
                            .onTapGesture {
                                selectedLog = log
                            }
                    }
                }
                .listStyle(.plain)
            }
        }
        .sheet(item: $selectedLog) { log in
            NetworkLogDetailView(log: log)
        }
    }
}

struct NetworkLogRow: View {
    let log: NetworkLog

    var body: some View {
        HStack(spacing: 12) {
            // Status indicator
            Circle()
                .fill(log.status.color)
                .frame(width: 8, height: 8)

            // Method
            Text(log.method)
                .font(.system(.caption, design: .monospaced))
                .fontWeight(.bold)
                .foregroundColor(.white)
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(methodColor(log.method))
                .cornerRadius(4)

            // URL path
            VStack(alignment: .leading, spacing: 2) {
                Text(urlPath(log.url))
                    .font(.system(.caption, design: .monospaced))
                    .lineLimit(1)

                HStack(spacing: 8) {
                    if let statusCode = log.statusCode {
                        Text("\(statusCode)")
                            .font(.system(.caption2, design: .monospaced))
                            .foregroundColor(statusCodeColor(statusCode))
                    }

                    Text(log.durationString)
                        .font(.caption2)
                        .foregroundColor(.secondary)

                    Text(log.timestamp.formatted(.dateTime.hour().minute().second()))
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }
            }

            Spacer()

            Image(systemName: "chevron.right")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 4)
    }

    private func urlPath(_ url: String) -> String {
        guard let urlObj = URL(string: url) else { return url }
        return urlObj.path + (urlObj.query.map { "?\($0)" } ?? "")
    }

    private func methodColor(_ method: String) -> Color {
        switch method {
        case "GET": return .blue
        case "POST": return .green
        case "PUT": return .orange
        case "DELETE": return .red
        case "PATCH": return .purple
        default: return .gray
        }
    }

    private func statusCodeColor(_ code: Int) -> Color {
        switch code {
        case 200..<300: return .green
        case 300..<400: return .orange
        case 400..<500: return .red
        case 500..<600: return .red
        default: return .secondary
        }
    }
}

struct NetworkLogDetailView: View {
    let log: NetworkLog
    @Environment(\.dismiss) private var dismiss
    @State private var selectedSection = 0

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Summary
                VStack(alignment: .leading, spacing: 8) {
                    HStack {
                        Text(log.method)
                            .font(.headline)
                            .fontWeight(.bold)

                        if let statusCode = log.statusCode {
                            Text("\(statusCode)")
                                .font(.headline)
                                .foregroundColor(statusCode < 400 ? .green : .red)
                        }

                        Spacer()

                        Text(log.durationString)
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }

                    Text(log.url)
                        .font(.system(.caption, design: .monospaced))
                        .foregroundColor(.secondary)
                        .lineLimit(3)
                }
                .padding()
                .background(Color(.systemGroupedBackground))

                Picker("", selection: $selectedSection) {
                    Text("请求").tag(0)
                    Text("响应").tag(1)
                }
                .pickerStyle(.segmented)
                .padding()

                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if selectedSection == 0 {
                            // Request Headers
                            DebugSection(title: "请求头") {
                                ForEach(Array(log.requestHeaders.keys.sorted()), id: \.self) { key in
                                    DebugKeyValue(key: key, value: log.requestHeaders[key] ?? "")
                                }
                            }

                            // Request Body
                            if let body = log.requestBody {
                                DebugSection(title: "请求体") {
                                    CodeBlock(content: formatJSON(body))
                                }
                            }
                        } else {
                            // Response Headers
                            if !log.responseHeaders.isEmpty {
                                DebugSection(title: "响应头") {
                                    ForEach(Array(log.responseHeaders.keys.sorted()), id: \.self) { key in
                                        DebugKeyValue(key: key, value: log.responseHeaders[key] ?? "")
                                    }
                                }
                            }

                            // Response Body
                            if let body = log.responseBody {
                                DebugSection(title: "响应体") {
                                    CodeBlock(content: formatJSON(body))
                                }
                            }

                            // Error
                            if let error = log.errorMessage {
                                DebugSection(title: "错误") {
                                    Text(error)
                                        .font(.system(.caption, design: .monospaced))
                                        .foregroundColor(.red)
                                }
                            }
                        }
                    }
                    .padding()
                }
            }
            .navigationTitle("请求详情")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("关闭") {
                        dismiss()
                    }
                }

                ToolbarItem(placement: .topBarLeading) {
                    Menu {
                        Button {
                            UIPasteboard.general.string = log.url
                        } label: {
                            Label("复制 URL", systemImage: "link")
                        }

                        if let body = log.requestBody {
                            Button {
                                UIPasteboard.general.string = body
                            } label: {
                                Label("复制请求体", systemImage: "doc.on.doc")
                            }
                        }

                        if let body = log.responseBody {
                            Button {
                                UIPasteboard.general.string = body
                            } label: {
                                Label("复制响应体", systemImage: "doc.on.doc")
                            }
                        }

                        Button {
                            UIPasteboard.general.string = generateCurlCommand(log)
                        } label: {
                            Label("复制 cURL", systemImage: "terminal")
                        }
                    } label: {
                        Image(systemName: "square.and.arrow.up")
                    }
                }
            }
        }
    }

    private func formatJSON(_ string: String) -> String {
        guard let data = string.data(using: .utf8),
              let json = try? JSONSerialization.jsonObject(with: data),
              let formatted = try? JSONSerialization.data(withJSONObject: json, options: [.prettyPrinted, .sortedKeys]),
              let result = String(data: formatted, encoding: .utf8) else {
            return string
        }
        return result
    }

    private func generateCurlCommand(_ log: NetworkLog) -> String {
        var curl = "curl -X \(log.method)"

        for (key, value) in log.requestHeaders {
            curl += " \\\n  -H '\(key): \(value)'"
        }

        if let body = log.requestBody {
            curl += " \\\n  -d '\(body)'"
        }

        curl += " \\\n  '\(log.url)'"

        return curl
    }
}

// MARK: - Console Logs View
struct ConsoleLogsView: View {
    @ObservedObject var debugTool = DebugTool.shared
    @State private var filterLevel: ConsoleLog.Level?

    var filteredLogs: [ConsoleLog] {
        if let level = filterLevel {
            return debugTool.consoleLogs.filter { $0.level == level }
        }
        return debugTool.consoleLogs
    }

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                // Filter
                Menu {
                    Button("全部") { filterLevel = nil }
                    ForEach([ConsoleLog.Level.debug, .info, .warning, .error], id: \.self) { level in
                        Button {
                            filterLevel = level
                        } label: {
                            Label(level.rawValue.capitalized, systemImage: level.icon)
                        }
                    }
                } label: {
                    HStack {
                        Image(systemName: "line.3.horizontal.decrease.circle")
                        Text(filterLevel?.rawValue.capitalized ?? "全部")
                            .font(.caption)
                    }
                }

                Spacer()

                Text("共 \(filteredLogs.count) 条")
                    .font(.caption)
                    .foregroundColor(.secondary)

                Button("清空") {
                    debugTool.clearConsoleLogs()
                }
                .font(.caption)
            }
            .padding(.horizontal)
            .padding(.vertical, 8)

            if filteredLogs.isEmpty {
                VStack(spacing: 8) {
                    Image(systemName: "doc.text")
                        .font(.system(size: 40))
                        .foregroundColor(.secondary)
                    Text("暂无日志")
                        .font(.headline)
                        .foregroundColor(.secondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List {
                    ForEach(filteredLogs) { log in
                        ConsoleLogRow(log: log)
                    }
                }
                .listStyle(.plain)
            }
        }
    }
}

struct ConsoleLogRow: View {
    let log: ConsoleLog

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Image(systemName: log.level.icon)
                    .font(.caption)
                    .foregroundColor(log.level.color)

                Text(log.file)
                    .font(.system(.caption2, design: .monospaced))
                    .foregroundColor(.secondary)

                Text(":\(log.line)")
                    .font(.system(.caption2, design: .monospaced))
                    .foregroundColor(.secondary)

                Spacer()

                Text(log.timestamp.formatted(.dateTime.hour().minute().second()))
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }

            Text(log.message)
                .font(.system(.caption, design: .monospaced))
                .foregroundColor(log.level.color)
        }
        .padding(.vertical, 4)
        .contextMenu {
            Button {
                UIPasteboard.general.string = log.message
            } label: {
                Label("复制消息", systemImage: "doc.on.doc")
            }
        }
    }
}

// MARK: - Debug Settings View
struct DebugSettingsView: View {
    @ObservedObject var debugTool = DebugTool.shared
    @State private var showClearAlert = false

    var body: some View {
        List {
            Section("测试账号") {
                ForEach(Array(DebugTool.testAccounts.enumerated()), id: \.offset) { index, account in
                    Button {
                        debugTool.selectAccount(at: index)
                    } label: {
                        HStack {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(account.name)
                                    .font(.subheadline)
                                    .foregroundColor(.primary)
                                Text("\(account.phone) / \(account.code)")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }

                            Spacer()

                            if debugTool.selectedAccountIndex == index {
                                Image(systemName: "checkmark.circle.fill")
                                    .foregroundColor(.green)
                            }
                        }
                    }
                }
            }

            Section("自定义填充") {
                LabeledContent("手机号") {
                    TextField("手机号", text: $debugTool.autoFillPhone)
                        .textFieldStyle(.roundedBorder)
                        .keyboardType(.phonePad)
                        .frame(width: 150)
                }

                LabeledContent("验证码") {
                    TextField("验证码", text: $debugTool.autoFillCode)
                        .textFieldStyle(.roundedBorder)
                        .keyboardType(.numberPad)
                        .frame(width: 150)
                }
            }

            Section("网络设置") {
                LabeledContent("自定义 BaseURL") {
                    TextField("留空使用默认", text: $debugTool.customBaseURL)
                        .textFieldStyle(.roundedBorder)
                        .autocapitalization(.none)
                        .keyboardType(.URL)
                }

                Text("当前: \(debugTool.getEffectiveBaseURL())")
                    .font(.caption)
                    .foregroundColor(.secondary)

                Toggle("显示网络状态条", isOn: $debugTool.showNetworkOverlay)
            }

            Section("模拟设置") {
                VStack(alignment: .leading) {
                    HStack {
                        Text("请求延迟")
                        Spacer()
                        Text("\(Int(debugTool.mockDelay * 1000))ms")
                            .foregroundColor(.secondary)
                    }
                    Slider(value: $debugTool.mockDelay, in: 0...3, step: 0.1)
                }
            }

            Section("危险操作") {
                Button(role: .destructive) {
                    showClearAlert = true
                } label: {
                    Label("清除所有数据", systemImage: "trash")
                }

                Button(role: .destructive) {
                    KeychainManager.shared.clearAll()
                    debugLog("Token 已清除", level: .warning)
                } label: {
                    Label("清除登录状态", systemImage: "person.slash")
                }
            }
        }
        .alert("确认清除", isPresented: $showClearAlert) {
            Button("取消", role: .cancel) {}
            Button("清除", role: .destructive) {
                debugTool.clearAllData()
            }
        } message: {
            Text("将清除所有本地数据，包括登录状态、缓存和设置。此操作不可撤销。")
        }
    }
}

// MARK: - App Info View
struct AppInfoView: View {
    var body: some View {
        List {
            Section("应用信息") {
                LabeledContent("App 名称", value: Bundle.main.appName)
                LabeledContent("版本", value: Bundle.main.appVersion)
                LabeledContent("Build", value: Bundle.main.buildNumber)
                LabeledContent("Bundle ID", value: Bundle.main.bundleIdentifier ?? "-")
            }

            Section("设备信息") {
                LabeledContent("设备型号", value: UIDevice.current.model)
                LabeledContent("系统版本", value: UIDevice.current.systemVersion)
                LabeledContent("设备名称", value: UIDevice.current.name)
            }

            Section("登录状态") {
                if let token = KeychainManager.shared.getAccessToken() {
                    LabeledContent("Token") {
                        Text(String(token.prefix(20)) + "...")
                            .font(.system(.caption, design: .monospaced))
                    }

                    Button("复制完整 Token") {
                        UIPasteboard.general.string = token
                    }
                } else {
                    Text("未登录")
                        .foregroundColor(.secondary)
                }
            }

            Section("环境") {
                LabeledContent("Base URL", value: DebugTool.shared.getEffectiveBaseURL())
                LabeledContent("编译模式", value: "DEBUG")
                LabeledContent("编译时间", value: compileTime)
            }
        }
    }

    private var compileTime: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd HH:mm"
        return formatter.string(from: Date())
    }
}

// MARK: - Helper Views
struct DebugSection<Content: View>: View {
    let title: String
    @ViewBuilder let content: Content

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.caption)
                .fontWeight(.semibold)
                .foregroundColor(.secondary)

            content
        }
    }
}

struct DebugKeyValue: View {
    let key: String
    let value: String

    var body: some View {
        HStack(alignment: .top) {
            Text(key)
                .font(.system(.caption, design: .monospaced))
                .foregroundColor(.secondary)
                .frame(width: 120, alignment: .leading)

            Text(value)
                .font(.system(.caption, design: .monospaced))
                .lineLimit(2)
        }
    }
}

struct CodeBlock: View {
    let content: String

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            Text(content)
                .font(.system(.caption, design: .monospaced))
                .textSelection(.enabled)
        }
        .padding(8)
        .background(Color(.systemGray6))
        .cornerRadius(8)
    }
}

// MARK: - Debug Button
struct DebugButton: View {
    @ObservedObject var debugTool = DebugTool.shared

    var body: some View {
        Button {
            debugTool.isDebugPanelPresented = true
        } label: {
            Image(systemName: "ant.fill")
                .font(.title3)
                .foregroundColor(.white)
                .frame(width: 44, height: 44)
                .background(Color.orange)
                .clipShape(Circle())
                .shadow(radius: 4)
        }
    }
}

// MARK: - Floating Debug Button
struct FloatingDebugButton: ViewModifier {
    @ObservedObject var debugTool = DebugTool.shared
    @State private var offset = CGSize.zero
    @State private var lastOffset = CGSize.zero

    func body(content: Content) -> some View {
        content
            .overlay(alignment: .bottomTrailing) {
                DebugButton()
                    .offset(x: offset.width, y: offset.height)
                    .gesture(
                        DragGesture()
                            .onChanged { value in
                                offset = CGSize(
                                    width: lastOffset.width + value.translation.width,
                                    height: lastOffset.height + value.translation.height
                                )
                            }
                            .onEnded { _ in
                                lastOffset = offset
                            }
                    )
                    .padding()
            }
            .sheet(isPresented: $debugTool.isDebugPanelPresented) {
                DebugPanelView()
            }
    }
}

extension View {
    func withDebugButton() -> some View {
        modifier(FloatingDebugButton())
    }
}

// MARK: - Agent Data View

struct AgentDataView: View {
    @StateObject private var screenCollector = ScreenDataCollector.shared
    @StateObject private var locationCollector = LocationDataCollector.shared
    @StateObject private var permissionManager = PermissionManager.shared

    @State private var refreshTimer: Timer?
    @State private var lastRefresh = Date()

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                // 刷新按钮
                HStack {
                    Text("Agent 数据")
                        .font(.headline)
                    Spacer()
                    Button {
                        refreshData()
                    } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                    Text(lastRefresh.formatted(.dateTime.hour().minute().second()))
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                .padding(.horizontal)

                // 权限状态
                GroupBox("权限状态") {
                    VStack(alignment: .leading, spacing: 8) {
                        permissionRow("屏幕使用时间", granted: permissionManager.status.screenTime)
                        permissionRow("地理位置", granted: permissionManager.status.location)
                        permissionRow("通讯录", granted: permissionManager.status.contacts)
                    }
                }
                .padding(.horizontal)

                // 屏幕数据
                GroupBox("屏幕使用数据 (ScreenStatus)") {
                    VStack(alignment: .leading, spacing: 8) {
                        let status = screenCollector.getCurrentStatus()
                        dataRow("is_active", value: "\(status.isActive)")
                        dataRow("activity_type", value: status.activityType.rawValue)
                        dataRow("session_duration_minutes", value: "\(status.sessionDurationMinutes)")
                        dataRow("last_active_minutes_ago", value: "\(status.lastActiveMinutesAgo)")

                        Divider()

                        dataRow("isMonitoring", value: "\(screenCollector.isMonitoring)")
                        dataRow("isAuthorized", value: "\(screenCollector.isAuthorized)")
                    }
                }
                .padding(.horizontal)

                // 位置数据
                GroupBox("位置数据 (LocationStatus)") {
                    VStack(alignment: .leading, spacing: 8) {
                        let status = locationCollector.getCurrentStatus()
                        dataRow("place_type", value: status.placeType.rawValue)
                        dataRow("at_place_since_minutes", value: "\(status.atPlaceSinceMinutes)")

                        Divider()

                        dataRow("isMonitoring", value: "\(locationCollector.isMonitoring)")

                        if let location = locationCollector.currentLocation {
                            dataRow("latitude", value: String(format: "%.6f", location.coordinate.latitude))
                            dataRow("longitude", value: String(format: "%.6f", location.coordinate.longitude))
                        } else {
                            dataRow("location", value: "未获取")
                        }

                        // 学习到的位置
                        Divider()
                        Text("学习到的位置:").font(.caption).foregroundColor(.secondary)

                        if let home = locationCollector.homeLocation {
                            dataRow("🏠 家", value: String(format: "%.4f, %.4f", home.coordinate.latitude, home.coordinate.longitude))
                        } else {
                            dataRow("🏠 家", value: "未学习")
                        }

                        if let work = locationCollector.workLocation {
                            dataRow("🏢 公司", value: String(format: "%.4f, %.4f", work.coordinate.latitude, work.coordinate.longitude))
                        } else {
                            dataRow("🏢 公司", value: "未学习")
                        }
                    }
                }
                .padding(.horizontal)

                // App Group 共享数据
                GroupBox("App Group 共享数据") {
                    VStack(alignment: .leading, spacing: 8) {
                        if let defaults = UserDefaults(suiteName: "group.com.youkong.app") {
                            let minutes = defaults.integer(forKey: "screenTimeMinutes")
                            let lastUpdate = defaults.object(forKey: "screenTimeLastUpdate") as? Date

                            dataRow("screenTimeMinutes", value: "\(minutes)")
                            if let lastUpdate = lastUpdate {
                                dataRow("lastUpdate", value: lastUpdate.formatted(.dateTime.hour().minute().second()))
                            } else {
                                dataRow("lastUpdate", value: "无")
                            }
                        } else {
                            Text("无法访问 App Group")
                                .foregroundColor(.red)
                        }
                    }
                }
                .padding(.horizontal)

                // 上报预览
                GroupBox("API 上报预览 (JSON)") {
                    let screenStatus = screenCollector.getCurrentStatus()
                    let locationStatus = locationCollector.getCurrentStatus()

                    let json = """
                    {
                      "screen": {
                        "is_active": \(screenStatus.isActive),
                        "activity_type": "\(screenStatus.activityType.rawValue)",
                        "session_duration_minutes": \(screenStatus.sessionDurationMinutes),
                        "last_active_minutes_ago": \(screenStatus.lastActiveMinutesAgo)
                      },
                      "location": {
                        "place_type": "\(locationStatus.placeType.rawValue)",
                        "at_place_since_minutes": \(locationStatus.atPlaceSinceMinutes)
                      }
                    }
                    """

                    Text(json)
                        .font(.system(.caption, design: .monospaced))
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(8)
                        .background(Color(.systemGray6))
                        .cornerRadius(8)

                    Button("复制 JSON") {
                        UIPasteboard.general.string = json
                    }
                    .font(.caption)
                }
                .padding(.horizontal)

                Spacer(minLength: 50)
            }
            .padding(.vertical)
        }
        .onAppear {
            refreshData()
            // 每 5 秒自动刷新
            refreshTimer = Timer.scheduledTimer(withTimeInterval: 5, repeats: true) { _ in
                refreshData()
            }
        }
        .onDisappear {
            refreshTimer?.invalidate()
        }
    }

    private func refreshData() {
        lastRefresh = Date()
        Task {
            await permissionManager.checkAllPermissions()
        }
        _ = screenCollector.getCurrentStatus()
        _ = locationCollector.getCurrentStatus()
    }

    private func permissionRow(_ name: String, granted: Bool) -> some View {
        HStack {
            Image(systemName: granted ? "checkmark.circle.fill" : "xmark.circle.fill")
                .foregroundColor(granted ? .green : .red)
            Text(name)
            Spacer()
            Text(granted ? "已授权" : "未授权")
                .font(.caption)
                .foregroundColor(granted ? .green : .red)
        }
    }

    private func dataRow(_ key: String, value: String) -> some View {
        HStack {
            Text(key)
                .font(.system(.caption, design: .monospaced))
                .foregroundColor(.secondary)
            Spacer()
            Text(value)
                .font(.system(.caption, design: .monospaced))
                .fontWeight(.medium)
        }
    }
}

#endif
