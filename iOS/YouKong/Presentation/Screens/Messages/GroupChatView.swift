import SwiftUI

struct GroupChatView: View {
    let roomId: String
    let ownerEmoji: String
    let ownerNickname: String
    let members: [CardRoomMemberBrief]

    @StateObject private var viewModel: GroupChatViewModel
    @Environment(\.dismiss) private var dismiss

    @State private var keyboardHeight: CGFloat = 0
    @State private var isInputFocused: Bool = false

    init(roomId: String, ownerEmoji: String, ownerNickname: String, members: [CardRoomMemberBrief]) {
        self.roomId = roomId
        self.ownerEmoji = ownerEmoji
        self.ownerNickname = ownerNickname
        self.members = members
        _viewModel = StateObject(wrappedValue: GroupChatViewModel(roomId: roomId))
    }

    var body: some View {
        ZStack {
            // 背景层：放大版 emoji 半透明背景
            emojiBackground

            // 前景层：聊天内容
            VStack(spacing: 0) {
                chatHeader

                // 消息列表
                ScrollViewReader { proxy in
                    ScrollView {
                        LazyVStack(spacing: 12) {
                            ForEach(viewModel.messages) { message in
                                GroupMessageRow(message: message, isMe: viewModel.isMe(message))
                                    .id(message.id)
                            }

                            Color.clear
                                .frame(height: keyboardHeight > 0 ? 20 : 0)
                                .id("bottom")
                        }
                        .padding()
                        .padding(.bottom, keyboardHeight > 0 ? keyboardHeight - 100 : 0)
                    }
                    .scrollDismissesKeyboard(.immediately)
                    .dismissKeyboardOnTouchDown()
                    .onChange(of: viewModel.messages.count) { _ in
                        withAnimation {
                            proxy.scrollTo("bottom", anchor: .bottom)
                        }
                    }
                }

                // 输入栏
                inputBar
            }
        }
        .background(Color(hex: "#0D1117"))
        .navigationBarHidden(true)
        .navigationBarBackButtonHidden(true)
        .task {
            await viewModel.loadMessages()
        }
        .onAppear {
            NotificationManager.shared.currentConversationId = roomId
        }
        .onDisappear {
            NotificationManager.shared.currentConversationId = nil
        }
    }

    // MARK: - Emoji Background

    private var emojiBackground: some View {
        GeometryReader { geo in
            let emojiSize: CGFloat = 60
            let cols = Int(geo.size.width / emojiSize) + 1
            let rows = Int(geo.size.height / emojiSize) + 1

            VStack(spacing: 0) {
                ForEach(0..<rows, id: \.self) { row in
                    HStack(spacing: 0) {
                        ForEach(0..<cols, id: \.self) { col in
                            Text(ownerEmoji)
                                .font(.system(size: 30))
                                .frame(width: emojiSize, height: emojiSize)
                                .offset(x: row % 2 == 0 ? 0 : emojiSize / 2)
                        }
                    }
                }
            }
            .opacity(0.08)
        }
        .ignoresSafeArea()
    }

    // MARK: - Chat Header

    private var chatHeader: some View {
        VStack(spacing: 0) {
            HStack {
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

                VStack(spacing: 2) {
                    Text("\(ownerEmoji) \(ownerNickname)的状态")
                        .font(.system(size: 14, design: .monospaced))
                        .foregroundColor(Color(white: 0.9))

                    Text("\(members.count) 人已加入")
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundColor(.gray)
                }

                Spacer()

                // 成员头像
                HStack(spacing: -8) {
                    ForEach(members.prefix(4)) { member in
                        Circle()
                            .fill(Color.gray.opacity(0.3))
                            .frame(width: 24, height: 24)
                            .overlay(
                                Text(String(member.nickname.prefix(1)))
                                    .font(.system(size: 11, design: .monospaced))
                                    .foregroundColor(.white)
                            )
                    }
                    if members.count > 4 {
                        Circle()
                            .fill(Color.gray.opacity(0.3))
                            .frame(width: 24, height: 24)
                            .overlay(
                                Text("+\(members.count - 4)")
                                    .font(.system(size: 9, design: .monospaced))
                                    .foregroundColor(.white)
                            )
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)

            CLISeparatorView()
        }
        .background(Color(hex: "#0D1117").opacity(0.95))
    }

    // MARK: - Input Bar

    private var inputBar: some View {
        VStack(spacing: 0) {
            CLISeparatorView()

            HStack(spacing: 8) {
                Text(">")
                    .font(.system(size: 16, design: .monospaced))
                    .foregroundColor(.green)

                NonDismissingTextField(
                    text: $viewModel.messageInput,
                    placeholder: "说点什么...",
                    placeholderColor: .gray,
                    textColor: .white,
                    font: .monospacedSystemFont(ofSize: 14, weight: .regular),
                    isFocused: $isInputFocused,
                    onSend: { _ in
                        Task { await viewModel.sendMessage() }
                    }
                )
                .frame(height: 24)

                Button {
                    Task { await viewModel.sendMessage() }
                } label: {
                    Text(CLIConstants.rightArrow)
                        .font(.system(size: 20, design: .monospaced))
                        .foregroundColor(viewModel.canSend ? .green : .gray)
                        .padding(8)
                }
                .disabled(!viewModel.canSend)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(Color(hex: "#0D1117").opacity(0.95))
        }
    }
}

// MARK: - Group Message Row

struct GroupMessageRow: View {
    let message: CardRoomMessage
    let isMe: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            // 发送者标签
            HStack(spacing: 6) {
                Text("[\(formatTime(message.createdAt))]")
                    .font(.system(size: 11, design: .monospaced))
                    .foregroundColor(.gray)
                Text(message.sender.nickname)
                    .font(.system(size: 12, design: .monospaced))
                    .fontWeight(.bold)
                    .foregroundColor(isMe ? .green : .cyan)
            }

            // 消息内容
            HStack(spacing: 0) {
                Text("> ")
                    .font(.system(size: 13, design: .monospaced))
                    .foregroundColor(isMe ? .green.opacity(0.6) : .cyan.opacity(0.6))
                Text(message.content)
                    .font(.system(size: 13, design: .monospaced))
                    .foregroundColor(Color(white: 0.9))
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.vertical, 4)
        .padding(.horizontal, 8)
        .background(
            RoundedRectangle(cornerRadius: 4)
                .fill(isMe ? Color.green.opacity(0.05) : Color.cyan.opacity(0.05))
        )
    }

    private func formatTime(_ dateString: String) -> String {
        let isoFormatter = ISO8601DateFormatter()
        isoFormatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = isoFormatter.date(from: dateString) {
            let formatter = DateFormatter()
            formatter.dateFormat = "HH:mm"
            return formatter.string(from: date)
        }
        // fallback: try to extract HH:mm from the string
        if dateString.count >= 16 {
            let start = dateString.index(dateString.startIndex, offsetBy: 11)
            let end = dateString.index(dateString.startIndex, offsetBy: 16)
            return String(dateString[start..<end])
        }
        return ""
    }
}
