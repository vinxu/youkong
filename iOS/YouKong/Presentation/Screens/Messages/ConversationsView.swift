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
        HStack(spacing: 0) {
            // 状态指示符
            Text(conversation.unreadCount > 0 ? CLIConstants.bullet : CLIConstants.hollowBullet)
                .font(.system(size: 18, design: .monospaced))
                .foregroundColor(conversation.unreadCount > 0 ? .green : .gray)
                .frame(width: 30)

            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text(conversation.partner.nickname)
                        .font(.system(size: 16, design: .monospaced))
                        .fontWeight(.medium)

                    Spacer()

                    if let lastMessage = conversation.lastMessage {
                        Text(lastMessage.createdAt.relativeDescription)
                            .font(.system(size: 12, design: .monospaced))
                            .foregroundColor(.secondary)
                    }
                }

                // 消息预览
                HStack(spacing: 4) {
                    Text(CLIConstants.dash)
                        .font(.system(size: 12, design: .monospaced))
                        .foregroundColor(.secondary)

                    if let lastMessage = conversation.lastMessage {
                        Text(lastMessagePreview(lastMessage))
                            .font(.system(size: 14, design: .monospaced))
                            .foregroundColor(.secondary)
                            .lineLimit(1)
                    }
                }

                // 未读数量
                if conversation.unreadCount > 0 {
                    Text("[\(conversation.unreadCount)]")
                        .font(.system(size: 12, design: .monospaced))
                        .foregroundColor(.red)
                }
            }

            Spacer()
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
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
