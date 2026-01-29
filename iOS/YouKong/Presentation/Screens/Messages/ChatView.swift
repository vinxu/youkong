import SwiftUI

struct ChatView: View {
    let partner: UserProfile?
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
        VStack(spacing: 0) {
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: UIConstants.Spacing.md) {
                        ForEach(viewModel.messages) { message in
                            MessageBubble(
                                message: message,
                                isMe: !viewModel.isFromPartner(message)
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

            messageInputBar
        }
        .navigationTitle(viewModel.partnerName)
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await viewModel.loadMessages()
        }
    }

    private var messageInputBar: some View {
        HStack(spacing: UIConstants.Spacing.md) {
            TextField("输入消息...", text: $viewModel.messageInput, axis: .vertical)
                .textFieldStyle(.roundedBorder)
                .lineLimit(1...4)

            Button {
                Task {
                    await viewModel.sendMessage()
                }
            } label: {
                Image(systemName: "arrow.up.circle.fill")
                    .font(.system(size: 32))
                    .foregroundColor(viewModel.canSendMessage ? .primaryGreen : .gray)
            }
            .disabled(!viewModel.canSendMessage)
        }
        .padding(.horizontal)
        .padding(.vertical, UIConstants.Spacing.sm)
        .background(Color(.systemBackground))
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
                    .background(isMe ? Color.primaryGreen : Color(.systemBackground))
                    .foregroundColor(isMe ? .white : .primary)
                    .cornerRadius(UIConstants.CornerRadius.lg, corners: isMe ? [.topLeft, .topRight, .bottomLeft] : [.topLeft, .topRight, .bottomRight])
                    .shadow(color: isMe ? .clear : .black.opacity(0.08), radius: 4, y: 2)

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
            Text("分享了状态")
                .fontWeight(.medium)

        case .confirmRequest:
            Text("想和你确认见面")
                .fontWeight(.medium)

        case .confirmResponse:
            if let content = message.content, content == "accepted" {
                Text("已确认见面")
                    .fontWeight(.medium)
            } else {
                Text("已拒绝")
                    .fontWeight(.medium)
            }
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
