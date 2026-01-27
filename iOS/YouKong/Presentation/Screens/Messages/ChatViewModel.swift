import Foundation
import Combine
import Factory

@MainActor
final class ChatViewModel: ObservableObject {
    @Published var messages: [Message] = []
    @Published var isLoading = false
    @Published var isAgentThinking = false
    @Published var errorMessage: String?
    @Published private(set) var partnerName: String = "加载中..."

    @Injected(\.messageRepository) private var repository

    private var partner: UserProfile?
    private var conversationId: String?
    private var messageIds = Set<String>()

    init(partner: UserProfile, conversationId: String?) {
        self.partner = partner
        self.partnerName = partner.nickname
        self.conversationId = conversationId
        setupWebSocket()
        updateCurrentConversation()
    }

    /// 从通知跳转时使用，只有 conversationId
    init(conversationId: String) {
        self.partner = nil
        self.conversationId = conversationId
        setupWebSocket()
        updateCurrentConversation()
    }

    deinit {
        // 离开聊天页面时清除当前会话 ID
        Task { @MainActor in
            NotificationManager.shared.currentConversationId = nil
        }
    }

    private func updateCurrentConversation() {
        if let conversationId = conversationId {
            NotificationManager.shared.currentConversationId = conversationId
        }
    }

    /// 判断消息是否来自对方
    func isFromPartner(_ message: Message) -> Bool {
        if let partner = partner {
            return message.sender.id == partner.id
        }
        // 如果没有 partner 信息，通过消息推断
        let currentUserId = AuthManager.shared.currentUser?.id ?? ""
        return message.sender.id != currentUserId
    }

    func loadMessages() async {
        if conversationId == nil {
            await createConversation()
        }

        guard let conversationId = conversationId else { return }

        isLoading = true
        defer { isLoading = false }

        do {
            let fetchedMessages = try await repository.getMessages(conversationId: conversationId)
            for message in fetchedMessages {
                addMessageIfNew(message)
            }

            // 如果没有 partner 信息，从消息中推断
            if partner == nil, let firstMessage = fetchedMessages.first {
                let currentUserId = AuthManager.shared.currentUser?.id ?? ""
                if firstMessage.sender.id != currentUserId {
                    partner = firstMessage.sender
                    partnerName = firstMessage.sender.nickname
                } else {
                    // 尝试从其他消息中找到对方
                    for message in fetchedMessages {
                        if message.sender.id != currentUserId {
                            partner = message.sender
                            partnerName = message.sender.nickname
                            break
                        }
                    }
                }
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func agentReply() async {
        if conversationId == nil {
            await createConversation()
        }

        guard let conversationId = conversationId else {
            errorMessage = "会话创建失败"
            return
        }

        isAgentThinking = true
        errorMessage = nil

        do {
            let message = try await repository.agentReply(conversationId: conversationId)
            addMessageIfNew(message)
        } catch {
            errorMessage = "你的元婴罢工了"
            print("[ChatViewModel] Agent reply error: \(error)")
        }

        isAgentThinking = false
    }

    private func createConversation() async {
        guard let partnerId = partner?.id else {
            errorMessage = "缺少对方信息"
            return
        }
        do {
            let conversation = try await repository.createConversation(partnerId: partnerId)
            self.conversationId = conversation.id
        } catch {
            errorMessage = "创建会话失败"
            print("[ChatViewModel] Create conversation error: \(error)")
        }
    }

    private func setupWebSocket() {
        WebSocketManager.shared.onMessage = { [weak self] convId, message in
            guard let self = self else { return }
            if convId == self.conversationId {
                self.addMessageIfNew(message)
            }
        }
    }

    private func addMessageIfNew(_ message: Message) {
        guard !messageIds.contains(message.id) else { return }
        messageIds.insert(message.id)

        if let index = messages.firstIndex(where: { $0.createdAt > message.createdAt }) {
            messages.insert(message, at: index)
        } else {
            messages.append(message)
        }
    }
}
