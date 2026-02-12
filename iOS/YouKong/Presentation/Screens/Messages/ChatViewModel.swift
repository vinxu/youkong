import Foundation
import Combine
import Factory

@MainActor
final class ChatViewModel: ObservableObject {
    @Published var messages: [Message] = []
    @Published var isLoading = false
    @Published var messageInput = ""
    @Published var isSending = false
    @Published private(set) var partnerName: String = "加载中..."
    @Published var respondingBookingId: String? = nil

    var canSendMessage: Bool {
        !messageInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isSending
    }

    @Injected(\.messageRepository) private var repository
    @Injected(\.apiClient) private var apiClient

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

        // ✅ 会话 ID 获取成功后，更新当前会话 ID（防止收到消息时增加未读）
        updateCurrentConversation()

        // ✅ 清除未读计数
        print("💬 [ChatViewModel] 清除会话 \(conversationId) 的未读计数")
        UnreadMessageManager.shared.clearUnread(for: conversationId)

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
            print("[ChatViewModel] Load messages error: \(error)")
        }
    }

    func sendMessage() async {
        let content = messageInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !content.isEmpty else { return }

        if conversationId == nil {
            await createConversation()
        }

        guard let conversationId = conversationId else {
            print("[ChatViewModel] Failed to create conversation")
            return
        }

        isSending = true
        messageInput = "" // 立即清空输入框

        do {
            let request = SendMessageRequest(
                type: .text,
                content: content,
                metadata: nil
            )
            let message = try await repository.sendMessage(
                conversationId: conversationId,
                request: request
            )
            addMessageIfNew(message)
        } catch {
            // 发送失败，恢复输入内容
            messageInput = content
            print("[ChatViewModel] Send message error: \(error)")
        }

        isSending = false
    }

    func respondToBooking(bookingId: String, action: String) async {
        respondingBookingId = bookingId
        defer { respondingBookingId = nil }

        do {
            let endpoint = APIEndpoint.respondToBooking(bookingId: bookingId, action: action)
            let _: BookingRespondResponse = try await apiClient.request(endpoint)
            await loadMessages()
        } catch {
            print("[ChatViewModel] Respond to booking error: \(error)")
        }
    }

    private func createConversation() async {
        guard let partnerId = partner?.id else {
            print("[ChatViewModel] Missing partner information")
            return
        }
        do {
            let conversation = try await repository.createConversation(partnerId: partnerId)
            self.conversationId = conversation.id
        } catch {
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
