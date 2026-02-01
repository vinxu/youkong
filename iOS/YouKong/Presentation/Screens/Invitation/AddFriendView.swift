import SwiftUI
import Factory

// MARK: - Add Friend View (发送好友请求) - 作为 NavigationLink 目标使用

struct AddFriendView: View {
    @StateObject private var viewModel = AddFriendViewModel()

    var body: some View {
        AddFriendContentView(viewModel: viewModel)
            .navigationTitle("添加好友")
            .navigationBarTitleDisplayMode(.inline)
    }
}

// MARK: - Add Friend Sheet View - 作为 Sheet 使用（带 NavigationStack）

struct AddFriendSheetView: View {
    @StateObject private var viewModel = AddFriendViewModel()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            AddFriendContentView(viewModel: viewModel)
                .navigationTitle("添加好友")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .topBarLeading) {
                        Button("取消") {
                            dismiss()
                        }
                    }
                }
        }
    }
}

// MARK: - Add Friend Content View - 内容视图

struct AddFriendContentView: View {
    @ObservedObject var viewModel: AddFriendViewModel

    var body: some View {
        ScrollView {
            VStack(spacing: 24) {
                // 说明文字
                VStack(spacing: 8) {
                    Image(systemName: "person.badge.plus")
                        .font(.system(size: 48))
                        .foregroundColor(.primaryGreen)

                    Text("输入好友的手机号码")
                        .foregroundColor(.secondary)
                }
                .padding(.top, 40)

                // 手机号输入框
                VStack(alignment: .leading, spacing: 8) {
                    Text("手机号")
                        .font(.subheadline)
                        .foregroundColor(.secondary)

                    HStack {
                        Image(systemName: "phone")
                            .foregroundColor(.secondary)

                        TextField("请输入手机号", text: $viewModel.phoneNumber)
                            .keyboardType(.phonePad)
                            .textContentType(.telephoneNumber)
                    }
                    .padding()
                    .background(Color(.systemGray6))
                    .cornerRadius(12)
                }
                .padding(.horizontal)

                // 验证消息（可选）
                VStack(alignment: .leading, spacing: 8) {
                    Text("验证消息（可选）")
                        .font(.subheadline)
                        .foregroundColor(.secondary)

                    TextField("我是...", text: $viewModel.message)
                        .padding()
                        .background(Color(.systemGray6))
                        .cornerRadius(12)
                }
                .padding(.horizontal)

                // 请求结果
                if let result = viewModel.requestResult {
                    resultView(result)
                        .padding(.horizontal)
                }

                // 错误提示
                if let error = viewModel.errorMessage {
                    Text(error)
                        .font(.subheadline)
                        .foregroundColor(.red)
                        .padding(.horizontal)
                }

                // 发送请求按钮
                Button {
                    Task {
                        await viewModel.sendRequest()
                    }
                } label: {
                    HStack {
                        if viewModel.isLoading {
                            ProgressView()
                                .tint(.white)
                        } else {
                            Text("发送好友请求")
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(viewModel.isValidPhone ? Color.primaryGreen : Color.gray)
                    .foregroundColor(.white)
                    .cornerRadius(12)
                }
                .disabled(!viewModel.isValidPhone || viewModel.isLoading)
                .padding()
            }
        }
    }

    @ViewBuilder
    private func resultView(_ result: SendFriendRequestResponse) -> some View {
        VStack(spacing: 12) {
            HStack(spacing: 12) {
                if let user = result.user {
                    Text(CLIConstants.bullet).font(.system(size: 16, design: .monospaced)).foregroundColor(.green).frame(width: 30)

                    VStack(alignment: .leading, spacing: 4) {
                        Text(user.nickname)
                            .fontWeight(.medium)
                        Text(statusText(result.status))
                            .font(.caption)
                            .foregroundColor(statusColor(result.status))
                    }
                } else {
                    Image(systemName: "person.slash")
                        .font(.title)
                        .foregroundColor(.secondary)

                    VStack(alignment: .leading, spacing: 4) {
                        Text("未找到用户")
                            .fontWeight(.medium)
                        Text(result.message ?? "该手机号未注册")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                Spacer()

                Image(systemName: statusIcon(result.status))
                    .foregroundColor(statusColor(result.status))
                    .font(.title2)
            }
            .padding()
            .background(statusColor(result.status).opacity(0.1))
            .cornerRadius(12)
        }
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

    private func statusColor(_ status: FriendRequestStatus) -> Color {
        switch status {
        case .pending, .alreadyRequested:
            return .orange
        case .accepted:
            return .green
        case .alreadyFriends:
            return .blue
        case .rejected, .cancelled:
            return .red
        }
    }

    private func statusIcon(_ status: FriendRequestStatus) -> String {
        switch status {
        case .pending, .alreadyRequested:
            return "clock.fill"
        case .accepted, .alreadyFriends:
            return "checkmark.circle.fill"
        case .rejected, .cancelled:
            return "xmark.circle.fill"
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
            // Tab 选择器
            Picker("", selection: $selectedTab) {
                Text("收到的请求").tag(0)
                Text("发出的请求").tag(1)
            }
            .pickerStyle(.segmented)
            .padding()

            // 内容
            TabView(selection: $selectedTab) {
                receivedRequestsTab
                    .tag(0)

                sentRequestsTab
                    .tag(1)
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
        }
        .navigationTitle("好友请求")
        .refreshable {
            await viewModel.refresh()
        }
        .task {
            await viewModel.loadRequests()
        }
    }

    private var receivedRequestsTab: some View {
        Group {
            if viewModel.isLoading && viewModel.receivedRequests.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if viewModel.receivedRequests.isEmpty {
                emptyView(
                    icon: "tray",
                    title: "没有好友请求",
                    subtitle: "暂时没有人向你发送好友请求"
                )
            } else {
                List {
                    ForEach(viewModel.receivedRequests) { request in
                        ReceivedRequestRow(request: request) { accept in
                            Task {
                                await viewModel.handleRequest(request, accept: accept)
                            }
                        }
                    }
                }
                .listStyle(.plain)
            }
        }
    }

    private var sentRequestsTab: some View {
        Group {
            if viewModel.sentRequests.isEmpty {
                emptyView(
                    icon: "paperplane",
                    title: "没有发出的请求",
                    subtitle: "你还没有发送过好友请求"
                )
            } else {
                List {
                    ForEach(viewModel.sentRequests) { request in
                        SentRequestRow(request: request)
                    }
                }
                .listStyle(.plain)
            }
        }
    }

    private func emptyView(icon: String, title: String, subtitle: String) -> some View {
        VStack(spacing: 16) {
            Image(systemName: icon)
                .font(.system(size: 48))
                .foregroundColor(.secondary)
            Text(title)
                .font(.headline)
                .foregroundColor(.secondary)
            Text(subtitle)
                .font(.subheadline)
                .foregroundColor(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

// MARK: - Received Request Row

struct ReceivedRequestRow: View {
    let request: FriendRequest
    let onHandle: (Bool) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 12) {
                Text(CLIConstants.bullet).font(.system(size: 16, design: .monospaced)).foregroundColor(.green).frame(width: 30)

                VStack(alignment: .leading, spacing: 4) {
                    Text(request.user.nickname)
                        .fontWeight(.medium)

                    if let message = request.message, !message.isEmpty {
                        Text(message)
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                            .lineLimit(2)
                    }

                    Text(request.createdAt, style: .relative)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }

                Spacer()
            }

            if request.status == .pending {
                HStack(spacing: 12) {
                    Button {
                        onHandle(false)
                    } label: {
                        Text("拒绝")
                            .font(.subheadline)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(Color(.systemGray5))
                            .foregroundColor(.primary)
                            .cornerRadius(8)
                    }

                    Button {
                        onHandle(true)
                    } label: {
                        Text("同意")
                            .font(.subheadline)
                            .fontWeight(.medium)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(Color.primaryGreen)
                            .foregroundColor(.white)
                            .cornerRadius(8)
                    }
                }
            } else {
                HStack {
                    Text(statusText)
                        .font(.caption)
                        .foregroundColor(statusColor)
                }
            }
        }
        .padding(.vertical, 8)
    }

    private var statusText: String {
        switch request.status {
        case .accepted:
            return "已同意"
        case .rejected:
            return "已拒绝"
        default:
            return ""
        }
    }

    private var statusColor: Color {
        switch request.status {
        case .accepted:
            return .green
        case .rejected:
            return .red
        default:
            return .secondary
        }
    }
}

// MARK: - Sent Request Row

struct SentRequestRow: View {
    let request: FriendRequest

    var body: some View {
        HStack(spacing: 12) {
            Text(CLIConstants.bullet).font(.system(size: 16, design: .monospaced)).foregroundColor(.green).frame(width: 30)

            VStack(alignment: .leading, spacing: 4) {
                Text(request.user.nickname)
                    .fontWeight(.medium)

                if let message = request.message, !message.isEmpty {
                    Text(message)
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .lineLimit(1)
                }

                Text(request.createdAt, style: .relative)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            Spacer()

            Text(statusText)
                .font(.caption)
                .foregroundColor(.white)
                .padding(.horizontal, 10)
                .padding(.vertical, 4)
                .background(statusColor)
                .cornerRadius(10)
        }
        .padding(.vertical, 4)
    }

    private var statusText: String {
        switch request.status {
        case .pending:
            return "等待确认"
        case .accepted:
            return "已通过"
        case .rejected:
            return "已拒绝"
        case .cancelled:
            return "已取消"
        default:
            return ""
        }
    }

    private var statusColor: Color {
        switch request.status {
        case .pending:
            return .orange
        case .accepted:
            return .green
        case .rejected, .cancelled:
            return .red
        default:
            return .secondary
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
            let response = try await contactRepository.handleFriendRequest(
                requestId: request.id,
                accept: accept
            )

            // 显示成功消息（可选，通过 errorMessage 复用）
            // errorMessage 可以用来显示成功提示

            // 刷新列表以确保数据同步
            await loadRequests()
        } catch {
            // 失败时恢复到列表
            receivedRequests.insert(request, at: 0)
            errorMessage = error.localizedDescription
        }
    }
}

// MARK: - Contact Friends View (通讯录好友)

struct ContactFriendsView: View {
    @StateObject private var viewModel = ContactFriendsViewModel()

    var body: some View {
        Group {
            if viewModel.isLoading {
                ProgressView("正在匹配通讯录...")
            } else if viewModel.matches.isEmpty {
                emptyView
            } else {
                List {
                    ForEach(viewModel.matches) { match in
                        ContactMatchRow(match: match) {
                            Task {
                                await viewModel.sendRequest(to: match)
                            }
                        }
                    }
                }
                .listStyle(.plain)
            }
        }
        .navigationTitle("通讯录好友")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    Task {
                        await viewModel.refresh()
                    }
                } label: {
                    Image(systemName: "arrow.clockwise")
                }
                .disabled(viewModel.isLoading)
            }
        }
        .task {
            await viewModel.loadContacts()
        }
        .alert("错误", isPresented: .constant(viewModel.errorMessage != nil)) {
            Button("确定") {
                viewModel.errorMessage = nil
            }
        } message: {
            Text(viewModel.errorMessage ?? "")
        }
    }

    private var emptyView: some View {
        VStack(spacing: 16) {
            Image(systemName: "person.crop.circle.badge.questionmark")
                .font(.system(size: 48))
                .foregroundColor(.secondary)
            Text("没有找到通讯录好友")
                .font(.headline)
                .foregroundColor(.secondary)
            Text("你的通讯录中暂时没有使用有空的好友")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
        }
        .padding()
    }
}

// MARK: - Contact Match Row

struct ContactMatchRow: View {
    let match: ContactMatch
    let onSendRequest: () -> Void
    @State private var requestSent = false

    var body: some View {
        HStack(spacing: 12) {
            Text(CLIConstants.bullet).font(.system(size: 16, design: .monospaced)).foregroundColor(.green).frame(width: 30)

            VStack(alignment: .leading, spacing: 4) {
                Text(match.user.nickname)
                    .fontWeight(.medium)
            }

            Spacer()

            if match.isFriend {
                Text("已是好友")
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(Color(.systemGray5))
                    .cornerRadius(14)
            } else if requestSent {
                Text("已发送")
                    .font(.caption)
                    .foregroundColor(.orange)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(Color.orange.opacity(0.1))
                    .cornerRadius(14)
            } else {
                Button {
                    onSendRequest()
                    requestSent = true
                } label: {
                    Text("添加")
                        .font(.subheadline)
                        .fontWeight(.medium)
                        .foregroundColor(.white)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 6)
                        .background(Color.primaryGreen)
                        .cornerRadius(14)
                }
            }
        }
        .padding(.vertical, 4)
    }
}

// MARK: - Contact Friends ViewModel

@MainActor
class ContactFriendsViewModel: ObservableObject {
    @Published var matches: [ContactMatch] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Injected(\.contactRepository) private var contactRepository

    private let contactsManager = ContactsManager.shared

    func loadContacts() async {
        isLoading = true
        errorMessage = nil

        do {
            let hasPermission = try await contactsManager.requestPermission()
            guard hasPermission else {
                errorMessage = "需要通讯录权限才能匹配好友"
                isLoading = false
                return
            }

            let phoneNumbers = try await contactsManager.fetchAllPhoneNumbers()
            let hashes = phoneNumbers.map { $0.sha256Hash }

            let response = try await contactRepository.matchContacts(phoneHashes: hashes)
            matches = response.matches
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    func refresh() async {
        await loadContacts()
    }

    func sendRequest(to match: ContactMatch) async {
        // 这里需要通过其他方式获取手机号，或者后端提供通过 userId 发送请求的接口
        // 暂时使用 addFriends 接口
        do {
            _ = try await contactRepository.addFriends(userIds: [match.user.id])
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

#Preview {
    AddFriendView()
}
