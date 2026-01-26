import SwiftUI

struct ConversationsView: View {
    @StateObject private var viewModel = ConversationsViewModel()

    var body: some View {
        NavigationStack {
            ScrollView {
                LazyVStack(spacing: 0) {
                    if viewModel.conversations.isEmpty && !viewModel.isLoading {
                        VStack(spacing: UIConstants.Spacing.lg) {
                            Image(systemName: "message")
                                .font(.system(size: 50))
                                .foregroundColor(.secondary)
                            Text("暂无消息")
                                .font(.headline)
                            Text("发起一个对话吧")
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                        }
                        .padding(.top, 100)
                    } else {
                        ForEach(viewModel.conversations) { conversation in
                            NavigationLink(value: conversation) {
                                ConversationRowView(conversation: conversation)
                            }
                            .buttonStyle(.plain)

                            Divider()
                                .padding(.leading, 76)
                        }
                    }
                }
            }
            .refreshable {
                await viewModel.refresh()
            }
            .navigationTitle("消息")
            .navigationDestination(for: Conversation.self) { conversation in
                ChatView(partner: conversation.partner, conversationId: conversation.id)
            }
        }
        .task {
            await viewModel.loadConversations()
        }
    }
}

struct ConversationRowView: View {
    let conversation: Conversation

    var body: some View {
        HStack(spacing: UIConstants.Spacing.md) {
            ZStack(alignment: .topTrailing) {
                AvatarView(url: conversation.partner.avatar, size: 52)

                if conversation.unreadCount > 0 {
                    Text("\(min(conversation.unreadCount, 99))")
                        .font(.caption2)
                        .fontWeight(.bold)
                        .foregroundColor(.white)
                        .frame(minWidth: 18, minHeight: 18)
                        .background(Color.red)
                        .clipShape(SwiftUI.Circle())
                        .offset(x: 4, y: -4)
                }
            }

            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text(conversation.partner.nickname)
                        .font(.headline)

                    Spacer()

                    if let lastMessage = conversation.lastMessage {
                        Text(lastMessage.createdAt.relativeDescription)
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                if let lastMessage = conversation.lastMessage {
                    Text(lastMessagePreview(lastMessage))
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .lineLimit(1)
                }
            }
        }
        .padding()
    }

    private func lastMessagePreview(_ message: Message) -> String {
        switch message.type {
        case .text:
            return message.content ?? ""
        case .availabilityCard:
            return "[有空卡片]"
        case .confirmRequest:
            return "[确认请求]"
        case .confirmResponse:
            return "[确认回复]"
        }
    }
}

#Preview {
    ConversationsView()
}
