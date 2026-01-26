import SwiftUI

struct ChatView: View {
    let partner: UserProfile
    let conversationId: String?
    @StateObject private var viewModel: ChatViewModel
    @Environment(\.dismiss) private var dismiss

    init(partner: UserProfile, conversationId: String? = nil) {
        self.partner = partner
        self.conversationId = conversationId
        _viewModel = StateObject(wrappedValue: ChatViewModel(
            partner: partner,
            conversationId: conversationId
        ))
    }

    /// 从朋友推荐卡片进入聊天的初始化方法
    init(partnerId: String, partnerName: String, partnerAvatar: String?) {
        let profile = UserProfile(id: partnerId, nickname: partnerName, avatar: partnerAvatar)
        self.partner = profile
        self.conversationId = nil
        _viewModel = StateObject(wrappedValue: ChatViewModel(
            partner: profile,
            conversationId: nil
        ))
    }

    var body: some View {
        VStack(spacing: 0) {
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: UIConstants.Spacing.md) {
                        ForEach(viewModel.messages) { message in
                            MessageBubble(
                                message: message,
                                isMe: message.sender.id != partner.id
                            )
                            .id(message.id)
                        }
                    }
                    .padding()
                }
                .onChange(of: viewModel.messages.count) { _ in
                    if let lastId = viewModel.messages.last?.id {
                        withAnimation {
                            proxy.scrollTo(lastId, anchor: .bottom)
                        }
                    }
                }
            }

            ChatInputBar(
                text: $viewModel.inputText,
                isLoading: viewModel.isSending
            ) {
                Task {
                    await viewModel.sendMessage()
                }
            }
        }
        .navigationTitle(partner.nickname)
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await viewModel.loadMessages()
        }
    }
}

struct MessageBubble: View {
    let message: Message
    let isMe: Bool

    var body: some View {
        HStack(alignment: .bottom, spacing: UIConstants.Spacing.sm) {
            if isMe {
                Spacer(minLength: 60)
            } else {
                AvatarView(url: message.sender.avatar, size: 32)
            }

            VStack(alignment: isMe ? .trailing : .leading, spacing: 2) {
                messageContent
                    .padding(.horizontal, UIConstants.Spacing.md)
                    .padding(.vertical, UIConstants.Spacing.sm)
                    .background(isMe ? Color.primaryGreen : Color(.systemGray5))
                    .foregroundColor(isMe ? .white : .primary)
                    .cornerRadius(UIConstants.CornerRadius.lg, corners: isMe ? [.topLeft, .topRight, .bottomLeft] : [.topLeft, .topRight, .bottomRight])

                Text(message.createdAt.timeString)
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }

            if !isMe {
                Spacer(minLength: 60)
            }
        }
    }

    @ViewBuilder
    private var messageContent: some View {
        switch message.type {
        case .text:
            Text(message.content ?? "")

        case .availabilityCard:
            Text("📅 分享了状态")
                .fontWeight(.medium)

        case .confirmRequest:
            Text("🤝 想和你确认见面")
                .fontWeight(.medium)

        case .confirmResponse:
            if let content = message.content, content == "accepted" {
                Text("✅ 已确认见面")
                    .fontWeight(.medium)
            } else {
                Text("❌ 已拒绝")
                    .fontWeight(.medium)
            }
        }
    }
}

struct ChatInputBar: View {
    @Binding var text: String
    let isLoading: Bool
    let onSend: () -> Void

    var body: some View {
        HStack(spacing: UIConstants.Spacing.md) {
            TextField("输入消息...", text: $text)
                .padding(.horizontal, UIConstants.Spacing.md)
                .padding(.vertical, UIConstants.Spacing.sm)
                .background(Color(.systemGray6))
                .cornerRadius(UIConstants.CornerRadius.xl)

            Button(action: onSend) {
                if isLoading {
                    ProgressView()
                        .tint(.white)
                } else {
                    Image(systemName: "paperplane.fill")
                }
            }
            .frame(width: 40, height: 40)
            .background(text.isEmpty ? Color.gray : Color.primaryGreen)
            .foregroundColor(.white)
            .clipShape(SwiftUI.Circle())
            .disabled(text.isEmpty || isLoading)
        }
        .padding()
        .background(Color(.systemBackground))
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
