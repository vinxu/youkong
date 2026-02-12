import SwiftUI

struct ChatView: View {
    let partner: UserProfile?
    let conversationId: String?
    @StateObject private var viewModel: ChatViewModel
    @Environment(\.dismiss) private var dismiss

    // 键盘高度状态
    @State private var keyboardHeight: CGFloat = 0

    // 输入框焦点状态
    @State private var isInputFocused: Bool = false

    // 好友行程表弹窗
    @State private var showFriendSchedule = false

    init(partner: UserProfile, conversationId: String? = nil) {
        self.partner = partner
        self.conversationId = conversationId
        _viewModel = StateObject(wrappedValue: ChatViewModel(
            partner: partner,
            conversationId: conversationId
        ))
    }

    init(partnerId: String, partnerName: String, partnerAvatar: String?) {
        let profile = UserProfile(id: partnerId, nickname: partnerName, avatar: partnerAvatar)
        self.partner = profile
        self.conversationId = nil
        _viewModel = StateObject(wrappedValue: ChatViewModel(
            partner: profile,
            conversationId: nil
        ))
    }

    /// 从通知跳转时使用，只有 conversationId
    init(conversationId: String) {
        self.partner = nil
        self.conversationId = conversationId
        _viewModel = StateObject(wrappedValue: ChatViewModel(conversationId: conversationId))
    }

    var body: some View {
        ZStack {
            // CLI 背景色
            Color(hex: "#0D1117")
                .ignoresSafeArea()

            VStack(spacing: 0) {
                // 终端风格头部（带返回按钮）
                chatHeader

                // 查看好友行程表入口条
                if let p = partner {
                    friendScheduleBar(name: p.nickname)
                }

                // 消息列表
                ScrollViewReader { proxy in
                    ScrollView {
                        LazyVStack(spacing: 12) {
                            ForEach(viewModel.messages) { message in
                                TerminalMessageRow(
                                    message: message,
                                    isMe: !viewModel.isFromPartner(message),
                                    respondingBookingId: viewModel.respondingBookingId,
                                    onRespondBooking: { bookingId, action in
                                        Task { await viewModel.respondToBooking(bookingId: bookingId, action: action) }
                                    }
                                )
                                .id(message.id)
                            }

                            // 底部占位符，用于键盘弹出时的空间
                            Color.clear
                                .frame(height: keyboardHeight > 0 ? 20 : 0)
                                .id("bottom")
                        }
                        .padding()
                        .padding(.bottom, keyboardHeight > 0 ? keyboardHeight - 100 : 0)
                    }
                    .simultaneousGesture(
                        DragGesture(minimumDistance: 0)
                            .onChanged { _ in
                                // 手指按下立即收起键盘
                                isInputFocused = false
                            }
                    )
                    .onChange(of: viewModel.messages.count) { _ in
                        scrollToBottom(proxy: proxy)
                    }
                    .onChange(of: keyboardHeight) { _ in
                        // 键盘高度变化时滚动到底部
                        scrollToBottom(proxy: proxy)
                    }
                    .onAppear {
                        setupKeyboardNotifications()
                    }
                    .onDisappear {
                        removeKeyboardNotifications()
                    }
                }

                // CLI 风格输入框
                terminalInputBar
            }
        }
        .navigationBarHidden(true)
        .navigationBarBackButtonHidden(true)
        .background(EnableSwipeBack())
        .task {
            await viewModel.loadMessages()
        }
        .onAppear {
            print("💬💬💬 [ChatView] onAppear 被调用")
            // ✅ 设置当前会话 ID，避免显示当前聊天的通知
            if let conversationId = conversationId {
                print("💬 会话ID: \(conversationId)")
                NotificationManager.shared.currentConversationId = conversationId
                // ✅ 清除未读计数
                print("💬 正在清除未读计数...")
                UnreadMessageManager.shared.clearUnread(for: conversationId)
            } else {
                print("💬 ⚠️ 会话ID为空")
            }
        }
        .onDisappear {
            // ✅ 离开聊天页面时清除
            NotificationManager.shared.currentConversationId = nil
        }
        .toolbar {
            ToolbarItem(placement: .navigationBarLeading) {
                Button {
                    dismiss()
                } label: {
                    HStack(spacing: 4) {
                        Text(CLIConstants.arrow)
                            .font(.system(size: 14, design: .monospaced))
                        Text("BACK")
                            .font(.system(size: 14, design: .monospaced))
                    }
                    .foregroundColor(.green)
                }
            }
        }
        .sheet(isPresented: $showFriendSchedule) {
            if let p = partner {
                FriendScheduleView(friendId: p.id, friendName: p.nickname)
            }
        }
    }

    // MARK: - Friend Schedule Bar

    private func friendScheduleBar(name: String) -> some View {
        Button {
            showFriendSchedule = true
        } label: {
            HStack(spacing: 8) {
                Text("📅")
                    .font(.system(size: 14))
                Text("查看\(name)的行程表")
                    .font(.system(size: 13, design: .monospaced))
                    .foregroundColor(CLIColors.green)
                Spacer()
                Text(">")
                    .font(.system(size: 14, design: .monospaced))
                    .foregroundColor(.gray)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(Color(white: 0.12))
        }
    }

    private var terminalInputBar: some View {
        VStack(spacing: 0) {
            CLISeparatorView()

            HStack(spacing: 8) {
                // 提示符
                Text(">")
                    .font(.system(size: 16, design: .monospaced))
                    .foregroundColor(.green)

                // 输入框（UIKit wrapper，发送时键盘不收起）
                NonDismissingTextField(
                    text: $viewModel.messageInput,
                    placeholder: "输入消息...",
                    placeholderColor: .gray,
                    textColor: .white,
                    font: .monospacedSystemFont(ofSize: 14, weight: .regular),
                    isFocused: $isInputFocused,
                    onSend: { _ in
                        Task { await viewModel.sendMessage() }
                    }
                )
                .frame(height: 24)

                // 发送按钮（ASCII 箭头）
                Button {
                    Task { await viewModel.sendMessage() }
                } label: {
                    Text(CLIConstants.rightArrow)
                        .font(.system(size: 20, design: .monospaced))
                        .foregroundColor(viewModel.canSendMessage ? .green : .gray)
                        .padding(8)
                }
                .disabled(!viewModel.canSendMessage)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(Color(hex: "#0D1117"))
        }
    }

    // MARK: - Chat Header

    private var chatHeader: some View {
        VStack(spacing: 0) {
            HStack {
                // 返回按钮
                Button {
                    dismiss()
                } label: {
                    HStack(spacing: 4) {
                        Text("<")
                            .font(.system(size: 16, design: .monospaced))
                        Text("BACK")
                            .font(.system(size: 13, design: .monospaced))
                    }
                    .foregroundColor(.green)
                }

                Spacer()

                // 标题
                VStack(spacing: 2) {
                    Text("CHAT: \(viewModel.partnerName)")
                        .font(.system(size: 14, design: .monospaced))
                        .foregroundColor(Color(white: 0.9))

                    Text("\(viewModel.messages.count) messages")
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundColor(.gray)
                }

                Spacer()

                // 占位符保持对称
                HStack(spacing: 4) {
                    Text("<")
                        .font(.system(size: 16, design: .monospaced))
                    Text("BACK")
                        .font(.system(size: 13, design: .monospaced))
                }
                .foregroundColor(.clear)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(Color(white: 0.15))

            // 分隔线
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .frame(height: 1)
        }
    }

    // MARK: - Helper Methods

    private func hideKeyboard() {
        UIApplication.shared.sendAction(#selector(UIResponder.resignFirstResponder), to: nil, from: nil, for: nil)
    }

    private func scrollToBottom(proxy: ScrollViewProxy) {
        if let lastId = viewModel.messages.last?.id {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                withAnimation(.easeOut(duration: 0.25)) {
                    proxy.scrollTo(lastId, anchor: .bottom)
                }
            }
        }
    }

    // MARK: - Keyboard Notifications

    private func setupKeyboardNotifications() {
        NotificationCenter.default.addObserver(
            forName: UIResponder.keyboardWillShowNotification,
            object: nil,
            queue: .main
        ) { notification in
            if let keyboardFrame = notification.userInfo?[UIResponder.keyboardFrameEndUserInfoKey] as? CGRect {
                withAnimation(.easeOut(duration: 0.25)) {
                    keyboardHeight = keyboardFrame.height
                }
            }
        }

        NotificationCenter.default.addObserver(
            forName: UIResponder.keyboardWillHideNotification,
            object: nil,
            queue: .main
        ) { _ in
            withAnimation(.easeOut(duration: 0.25)) {
                keyboardHeight = 0
            }
        }
    }

    private func removeKeyboardNotifications() {
        NotificationCenter.default.removeObserver(
            self,
            name: UIResponder.keyboardWillShowNotification,
            object: nil
        )
        NotificationCenter.default.removeObserver(
            self,
            name: UIResponder.keyboardWillHideNotification,
            object: nil
        )
    }
}

/// CLI 风格消息行组件
struct TerminalMessageRow: View {
    let message: Message
    let isMe: Bool
    var respondingBookingId: String? = nil
    var onRespondBooking: ((String, String) -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            // 消息头: [HH:mm] 用户名
            HStack(spacing: 6) {
                Text("[\(message.createdAt.timeString)]")
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundColor(.gray)

                Text(isMe ? "YOU" : message.sender.nickname.uppercased())
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundColor(isMe ? .green : .cyan)
            }

            // 消息内容: > 文本
            HStack(alignment: .top, spacing: 6) {
                Text(CLIConstants.arrow)
                    .font(.system(size: 14, design: .monospaced))
                    .foregroundColor(isMe ? .green : .gray)

                messageContentView
                    .font(.system(size: 14, design: .monospaced))
                    .foregroundColor(.white)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(
            isMe ?
                Color.green.opacity(0.1) :
                Color.gray.opacity(0.05)
        )
    }

    /// 从 metadata 中安全取 String 值
    private func metaString(_ key: String) -> String? {
        guard let meta = message.metadata, let val = meta[key] else { return nil }
        return val.value as? String
    }

    @ViewBuilder
    private var messageContentView: some View {
        switch message.type {
        case .text:
            Text(message.content ?? "")
                .fixedSize(horizontal: false, vertical: true)

        case .availabilityCard:
            HStack(spacing: 4) {
                Text("[CARD]")
                    .foregroundColor(.yellow)
                Text("分享了状态")
            }

        case .confirmRequest:
            HStack(spacing: 4) {
                Text("[REQUEST]")
                    .foregroundColor(.orange)
                Text("想和你确认见面")
            }

        case .confirmResponse:
            if metaString("action") == "accepted" || (message.content ?? "").contains("接受") {
                HStack(spacing: 4) {
                    Text("[已接受]")
                        .foregroundColor(.green)
                    Text(message.content ?? "已确认见面")
                }
            } else {
                HStack(spacing: 4) {
                    Text("[已拒绝]")
                        .foregroundColor(.red)
                    Text(message.content ?? "已拒绝")
                }
            }

        case .scheduleInvite:
            scheduleInviteView
        }
    }

    // MARK: - Schedule Invite Card

    @ViewBuilder
    private var scheduleInviteView: some View {
        let activity = metaString("activity") ?? "邀约"
        let date = metaString("date") ?? ""
        let startTime = metaString("start_time") ?? ""
        let endTime = metaString("end_time") ?? ""
        let location = metaString("location")
        let status = metaString("status") ?? "pending"
        let bookingId = metaString("booking_id")

        VStack(alignment: .leading, spacing: 6) {
            // 标题行
            HStack(spacing: 4) {
                Text("[INVITE]")
                    .foregroundColor(.cyan)
                Text("📅 \(activity)")
            }

            // 详情
            VStack(alignment: .leading, spacing: 2) {
                Text("  日期: \(date)")
                    .foregroundColor(.gray)
                Text("  时间: \(startTime)-\(endTime)")
                    .foregroundColor(.gray)
                if let loc = location, !loc.isEmpty {
                    Text("  地点: \(loc)")
                        .foregroundColor(.gray)
                }
            }

            // 状态/操作按钮
            if status == "pending" && !isMe {
                // 被邀请方且待响应
                if let bid = bookingId {
                    let isResponding = respondingBookingId == bid
                    HStack(spacing: 12) {
                        Button {
                            onRespondBooking?(bid, "accept")
                        } label: {
                            Text(isResponding ? "[...]" : "[接受]")
                                .foregroundColor(.green)
                        }
                        .disabled(isResponding)

                        Button {
                            onRespondBooking?(bid, "decline")
                        } label: {
                            Text(isResponding ? "[...]" : "[拒绝]")
                                .foregroundColor(.red)
                        }
                        .disabled(isResponding)
                    }
                    .padding(.top, 4)
                }
            } else if status == "accepted" {
                Text("  [已接受]")
                    .foregroundColor(.green)
            } else if status == "declined" {
                Text("  [已拒绝]")
                    .foregroundColor(.red)
            }
        }
    }
}

/// Re-enables the interactive swipe-back gesture when navigationBar is hidden.
private struct EnableSwipeBack: UIViewControllerRepresentable {
    func makeUIViewController(context: Context) -> UIViewController {
        EnableSwipeBackVC()
    }
    func updateUIViewController(_ vc: UIViewController, context: Context) {}

    class EnableSwipeBackVC: UIViewController {
        override func viewWillAppear(_ animated: Bool) {
            super.viewWillAppear(animated)
            navigationController?.interactivePopGestureRecognizer?.isEnabled = true
            navigationController?.interactivePopGestureRecognizer?.delegate = nil
        }
    }
}

#Preview {
    NavigationStack {
        ChatView(
            partner: UserProfile(id: "1", nickname: "小明", avatar: nil),
            conversationId: "1"
        )
    }
}
