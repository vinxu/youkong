import SwiftUI
import Factory

// MARK: - Add Friend View (发送好友请求) - 作为 NavigationLink 目标使用

struct AddFriendView: View {
    @StateObject private var viewModel = AddFriendViewModel()

    var body: some View {
        AddFriendContentView(viewModel: viewModel)
            .navigationTitle("")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .principal) {
                    Text("$ add --friend")
                        .font(.cliHeadline)
                        .foregroundColor(CLIColors.textPrimary)
                }
            }
            .toolbarBackground(CLIColors.background, for: .navigationBar)
            .toolbarBackground(.visible, for: .navigationBar)
    }
}

// MARK: - Add Friend Sheet View - 作为 Sheet 使用（带 NavigationStack）

struct AddFriendSheetView: View {
    @StateObject private var viewModel = AddFriendViewModel()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            AddFriendContentView(viewModel: viewModel)
                .navigationTitle("")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .principal) {
                        Text("$ add --friend")
                            .font(.cliHeadline)
                            .foregroundColor(CLIColors.textPrimary)
                    }
                    ToolbarItem(placement: .topBarLeading) {
                        Button("取消") {
                            dismiss()
                        }
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                    }
                }
                .toolbarBackground(CLIColors.background, for: .navigationBar)
                .toolbarBackground(.visible, for: .navigationBar)
        }
    }
}

// MARK: - Add Friend Content View - 内容视图

struct AddFriendContentView: View {
    @ObservedObject var viewModel: AddFriendViewModel

    var body: some View {
        ScrollView {
            VStack(spacing: 20) {
                // 说明文字
                VStack(spacing: 8) {
                    Text("[+]")
                        .font(.system(size: 36, weight: .bold, design: .monospaced))
                        .foregroundColor(CLIColors.green)

                    Text("输入好友的手机号码")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                }
                .padding(.top, 32)

                // 手机号输入框
                VStack(alignment: .leading, spacing: 6) {
                    Text("手机号")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textSecondary)

                    HStack(spacing: 6) {
                        Text("$")
                            .font(.system(size: 14, weight: .medium, design: .monospaced))
                            .foregroundColor(CLIColors.green)

                        TextField("", text: $viewModel.phoneNumber, prompt: Text("请输入手机号").foregroundColor(CLIColors.textWeak))
                            .font(.system(size: 14, design: .monospaced))
                            .foregroundColor(CLIColors.textPrimary)
                            .keyboardType(.phonePad)
                            .textContentType(.telephoneNumber)
                    }
                    .padding(12)
                    .background(CLIColors.backgroundSecondary)
                    .overlay(
                        Rectangle()
                            .stroke(CLIColors.border, lineWidth: 1)
                    )
                }
                .padding(.horizontal, 16)

                // 验证消息（可选）
                VStack(alignment: .leading, spacing: 6) {
                    Text("验证消息（可选）")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textSecondary)

                    HStack(spacing: 6) {
                        Text(">")
                            .font(.system(size: 14, weight: .medium, design: .monospaced))
                            .foregroundColor(CLIColors.textSecondary)

                        TextField("", text: $viewModel.message, prompt: Text("我是...").foregroundColor(CLIColors.textWeak))
                            .font(.system(size: 14, design: .monospaced))
                            .foregroundColor(CLIColors.textPrimary)
                    }
                    .padding(12)
                    .background(CLIColors.backgroundSecondary)
                    .overlay(
                        Rectangle()
                            .stroke(CLIColors.border, lineWidth: 1)
                    )
                }
                .padding(.horizontal, 16)

                // 请求结果
                if let result = viewModel.requestResult {
                    resultView(result)
                        .padding(.horizontal, 16)
                }

                // 错误提示
                if let error = viewModel.errorMessage {
                    Text("> \(error)")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.red)
                        .padding(.horizontal, 16)
                }

                // 发送请求按钮
                Button {
                    Task {
                        await viewModel.sendRequest()
                    }
                } label: {
                    HStack(spacing: 6) {
                        if viewModel.isLoading {
                            ProgressView()
                                .tint(.white)
                                .scaleEffect(0.8)
                        } else {
                            Text("[发送好友请求]")
                                .font(.system(size: 14, weight: .medium, design: .monospaced))
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
                    .background(viewModel.isValidPhone ? CLIColors.green : CLIColors.backgroundSecondary)
                    .foregroundColor(viewModel.isValidPhone ? .white : CLIColors.textWeak)
                    .overlay(
                        Rectangle()
                            .stroke(viewModel.isValidPhone ? Color.clear : CLIColors.border, lineWidth: 1)
                    )
                }
                .disabled(!viewModel.isValidPhone || viewModel.isLoading)
                .padding(.horizontal, 16)
            }
            .padding(.bottom, 20)
        }
        .background(CLIColors.background)
    }

    @ViewBuilder
    private func resultView(_ result: SendFriendRequestResponse) -> some View {
        HStack(spacing: 12) {
            if let user = result.user {
                Text(CLIConstants.bullet)
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.green)
                    .frame(width: 24)

                VStack(alignment: .leading, spacing: 4) {
                    Text(user.nickname)
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textPrimary)
                    Text(statusText(result.status))
                        .font(.cliCaption)
                        .foregroundColor(statusColor(result.status))
                }
            } else {
                Text("x")
                    .font(.system(size: 16, weight: .bold, design: .monospaced))
                    .foregroundColor(CLIColors.textSecondary)
                    .frame(width: 24)

                VStack(alignment: .leading, spacing: 4) {
                    Text("未找到用户")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textPrimary)
                    Text(result.message ?? "该手机号未注册")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textSecondary)
                }
            }

            Spacer()

            Text(statusBracketText(result.status))
                .font(.cliCaption)
                .foregroundColor(statusColor(result.status))
        }
        .padding(12)
        .background(CLIColors.backgroundSecondary)
        .overlay(
            Rectangle()
                .stroke(CLIColors.border, lineWidth: 1)
        )
    }

    private func statusText(_ status: FriendRequestStatus) -> String {
        switch status {
        case .pending:
            return "请求已发送，等待对方确认"
        case .accepted:
            return "已成为好友"
        case .alreadyFriends:
            return "你们已经是好友了"
        case .alreadyRequested:
            return "已发送过请求，等待对方确认"
        case .rejected:
            return "请求被拒绝"
        case .cancelled:
            return "请求已取消"
        }
    }

    private func statusBracketText(_ status: FriendRequestStatus) -> String {
        switch status {
        case .pending, .alreadyRequested:
            return "[等待中]"
        case .accepted, .alreadyFriends:
            return "[已通过]"
        case .rejected, .cancelled:
            return "[已拒绝]"
        }
    }

    private func statusColor(_ status: FriendRequestStatus) -> Color {
        switch status {
        case .pending, .alreadyRequested:
            return CLIColors.yellow
        case .accepted, .alreadyFriends:
            return CLIColors.green
        case .rejected, .cancelled:
            return CLIColors.red
        }
    }
}

// MARK: - Add Friend ViewModel

@MainActor
class AddFriendViewModel: ObservableObject {
    @Published var phoneNumber: String = ""
    @Published var message: String = ""
    @Published var isLoading = false
    @Published var requestResult: SendFriendRequestResponse?
    @Published var errorMessage: String?

    @Injected(\.contactRepository) private var contactRepository

    var isValidPhone: Bool {
        let pattern = "^1[3-9]\\d{9}$"
        return phoneNumber.range(of: pattern, options: .regularExpression) != nil
    }

    func sendRequest() async {
        guard isValidPhone else { return }

        isLoading = true
        errorMessage = nil
        requestResult = nil

        do {
            requestResult = try await contactRepository.sendFriendRequest(
                phone: phoneNumber,
                message: message.isEmpty ? nil : message
            )
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }
}

// MARK: - Friend Requests View (好友请求列表)

struct FriendRequestsView: View {
    @StateObject private var viewModel = FriendRequestsViewModel()
    @State private var selectedTab = 0

    var body: some View {
        VStack(spacing: 0) {
            // Tab 选择器 (CLI style)
            HStack(spacing: 8) {
                tabButton("收到的请求", tag: 0)
                tabButton("发出的请求", tag: 1)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)

            Rectangle()
                .fill(CLIColors.border)
                .frame(height: 1)

            // 内容
            TabView(selection: $selectedTab) {
                receivedRequestsTab
                    .tag(0)

                sentRequestsTab
                    .tag(1)
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
        }
        .background(CLIColors.background)
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .principal) {
                Text("$ friend-requests")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.textPrimary)
            }
        }
        .toolbarBackground(CLIColors.background, for: .navigationBar)
        .toolbarBackground(.visible, for: .navigationBar)
        .refreshable {
            await viewModel.refresh()
        }
        .task {
            await viewModel.loadRequests()
        }
    }

    private func tabButton(_ title: String, tag: Int) -> some View {
        Button {
            withAnimation { selectedTab = tag }
        } label: {
            Text(title)
                .font(.cliCaption)
                .foregroundColor(selectedTab == tag ? .white : CLIColors.textSecondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
                .background(selectedTab == tag ? CLIColors.green : CLIColors.backgroundSecondary)
                .overlay(
                    Rectangle()
                        .stroke(CLIColors.border, lineWidth: selectedTab == tag ? 0 : 1)
                )
        }
        .buttonStyle(.plain)
    }

    private var receivedRequestsTab: some View {
        Group {
            if viewModel.isLoading && viewModel.receivedRequests.isEmpty {
                VStack(spacing: 8) {
                    HStack(spacing: 4) {
                        Text("[")
                            .font(.cliBody)
                            .foregroundColor(CLIColors.textSecondary)
                        ProgressView()
                            .tint(CLIColors.green)
                            .scaleEffect(0.8)
                        Text("]")
                            .font(.cliBody)
                            .foregroundColor(CLIColors.textSecondary)
                    }
                    Text("加载中...")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textWeak)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if viewModel.receivedRequests.isEmpty {
                cliEmptyView(
                    icon: "tray",
                    title: "没有好友请求",
                    subtitle: "暂时没有人向你发送好友请求"
                )
            } else {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(viewModel.receivedRequests) { request in
                            ReceivedRequestRow(request: request) { accept in
                                Task {
                                    await viewModel.handleRequest(request, accept: accept)
                                }
                            }

                            Rectangle()
                                .fill(CLIColors.border)
                                .frame(height: 1)
                                .padding(.leading, 40)
                        }
                    }
                }
            }
        }
    }

    private var sentRequestsTab: some View {
        Group {
            if viewModel.sentRequests.isEmpty {
                cliEmptyView(
                    icon: "paperplane",
                    title: "没有发出的请求",
                    subtitle: "你还没有发送过好友请求"
                )
            } else {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(viewModel.sentRequests) { request in
                            SentRequestRow(request: request)

                            Rectangle()
                                .fill(CLIColors.border)
                                .frame(height: 1)
                                .padding(.leading, 40)
                        }
                    }
                }
            }
        }
    }

    private func cliEmptyView(icon: String, title: String, subtitle: String) -> some View {
        VStack(spacing: 12) {
            Image(systemName: icon)
                .font(.system(size: 40))
                .foregroundColor(CLIColors.textWeak)

            Text(title)
                .font(.cliBody)
                .foregroundColor(CLIColors.textSecondary)

            Text(subtitle)
                .font(.cliCaption)
                .foregroundColor(CLIColors.textWeak)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(.top, 60)
    }
}

// MARK: - Received Request Row

struct ReceivedRequestRow: View {
    let request: FriendRequest
    let onHandle: (Bool) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 0) {
                Text(CLIConstants.bullet)
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.green)
                    .frame(width: 30)

                VStack(alignment: .leading, spacing: 4) {
                    Text(request.user.nickname)
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textPrimary)

                    if let message = request.message, !message.isEmpty {
                        Text("> \(message)")
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.textSecondary)
                            .lineLimit(2)
                    }

                    Text(request.createdAt, style: .relative)
                        .font(.cliCaptionSmall)
                        .foregroundColor(CLIColors.textWeak)
                }

                Spacer()
            }

            if request.status == .pending {
                HStack(spacing: 8) {
                    Button {
                        onHandle(false)
                    } label: {
                        Text("[拒绝]")
                            .font(.system(size: 13, weight: .medium, design: .monospaced))
                            .foregroundColor(CLIColors.textSecondary)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(CLIColors.backgroundSecondary)
                            .overlay(
                                Rectangle()
                                    .stroke(CLIColors.border, lineWidth: 1)
                            )
                    }
                    .buttonStyle(.plain)

                    Button {
                        onHandle(true)
                    } label: {
                        Text("[同意]")
                            .font(.system(size: 13, weight: .medium, design: .monospaced))
                            .foregroundColor(.white)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(CLIColors.green)
                    }
                    .buttonStyle(.plain)
                }
                .padding(.leading, 30)
            } else {
                HStack {
                    Text(statusText)
                        .font(.cliCaption)
                        .foregroundColor(statusColor)
                }
                .padding(.leading, 30)
            }
        }
        .padding(.vertical, 10)
        .padding(.horizontal, 16)
        .background(CLIColors.background)
    }

    private var statusText: String {
        switch request.status {
        case .accepted:
            return "[已同意]"
        case .rejected:
            return "[已拒绝]"
        default:
            return ""
        }
    }

    private var statusColor: Color {
        switch request.status {
        case .accepted:
            return CLIColors.green
        case .rejected:
            return CLIColors.red
        default:
            return CLIColors.textSecondary
        }
    }
}

// MARK: - Sent Request Row

struct SentRequestRow: View {
    let request: FriendRequest

    var body: some View {
        HStack(spacing: 0) {
            Text(CLIConstants.bullet)
                .font(.cliHeadline)
                .foregroundColor(CLIColors.green)
                .frame(width: 30)

            VStack(alignment: .leading, spacing: 4) {
                Text(request.user.nickname)
                    .font(.cliBody)
                    .foregroundColor(CLIColors.textPrimary)

                if let message = request.message, !message.isEmpty {
                    Text("> \(message)")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textSecondary)
                        .lineLimit(1)
                }

                Text(request.createdAt, style: .relative)
                    .font(.cliCaptionSmall)
                    .foregroundColor(CLIColors.textWeak)
            }

            Spacer()

            Text(statusText)
                .font(.cliCaption)
                .foregroundColor(statusColor)
        }
        .padding(.vertical, 10)
        .padding(.horizontal, 16)
        .background(CLIColors.background)
    }

    private var statusText: String {
        switch request.status {
        case .pending:
            return "[等待确认]"
        case .accepted:
            return "[已通过]"
        case .rejected:
            return "[已拒绝]"
        case .cancelled:
            return "[已取消]"
        default:
            return ""
        }
    }

    private var statusColor: Color {
        switch request.status {
        case .pending:
            return CLIColors.yellow
        case .accepted:
            return CLIColors.green
        case .rejected, .cancelled:
            return CLIColors.red
        default:
            return CLIColors.textSecondary
        }
    }
}

// MARK: - Friend Requests ViewModel

@MainActor
class FriendRequestsViewModel: ObservableObject {
    @Published var receivedRequests: [FriendRequest] = []
    @Published var sentRequests: [FriendRequest] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Injected(\.contactRepository) private var contactRepository

    func loadRequests() async {
        isLoading = true
        errorMessage = nil

        do {
            async let received = contactRepository.getReceivedRequests()
            async let sent = contactRepository.getSentRequests()

            receivedRequests = try await received
            sentRequests = try await sent
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    func refresh() async {
        await loadRequests()
    }

    func handleRequest(_ request: FriendRequest, accept: Bool) async {
        // 乐观更新：立即从列表中移除
        receivedRequests.removeAll { $0.id == request.id }

        do {
            _ = try await contactRepository.handleFriendRequest(
                requestId: request.id,
                accept: accept
            )

            // 刷新列表以确保数据同步
            await loadRequests()
        } catch {
            // 失败时恢复到列表
            receivedRequests.insert(request, at: 0)
            errorMessage = error.localizedDescription
        }
    }
}


#Preview {
    AddFriendView()
}
