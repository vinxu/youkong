import Foundation
import Factory

@MainActor
class GroupChatViewModel: ObservableObject {
    @Published var messages: [CardRoomMessage] = []
    @Published var messageInput: String = ""
    @Published var isLoading = false

    let roomId: String

    @Injected(\.agentRepository) private var agentRepository

    var canSend: Bool {
        !messageInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    init(roomId: String) {
        self.roomId = roomId
    }

    func loadMessages() async {
        isLoading = true
        do {
            let msgs = try await agentRepository.getCardRoomMessages(roomId: roomId, limit: 50, offset: 0)
            // 按时间正序排列（API 返回倒序）
            messages = msgs.reversed()
        } catch {
            print("❌ [GroupChat] 加载消息失败: \(error)")
        }
        isLoading = false
    }

    func sendMessage() async {
        let text = messageInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty else { return }

        messageInput = ""

        do {
            let msg = try await agentRepository.sendCardRoomMessage(roomId: roomId, content: text)
            messages.append(msg)
        } catch {
            print("❌ [GroupChat] 发送消息失败: \(error)")
            // 恢复输入
            messageInput = text
        }
    }

    func isMe(_ message: CardRoomMessage) -> Bool {
        let currentUserId = AuthManager.shared.currentUser?.id ?? ""
        return message.sender.id == currentUserId
    }
}
