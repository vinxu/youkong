import Foundation
import Combine
import Factory

@MainActor
final class ChatViewModel: ObservableObject {
    @Published var messages: [Message] = []
    @Published var inputText = ""
    @Published var isLoading = false
    @Published var isSending = false
    @Published var errorMessage: String?

    @Injected(\.messageRepository) private var repository

    private let partner: UserProfile
    private var conversationId: String?
    private var refreshTask: Task<Void, Never>?

    init(partner: UserProfile, conversationId: String?) {
        self.partner = partner
        self.conversationId = conversationId
    }

    deinit {
        refreshTask?.cancel()
    }

    func loadMessages() async {
        guard let conversationId = conversationId else { return }

        isLoading = true
        defer { isLoading = false }

        do {
            messages = try await repository.getMessages(conversationId: conversationId)
            startAutoRefresh()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func sendMessage() async {
        guard !inputText.isEmpty else { return }
        guard let conversationId = conversationId else { return }

        let text = inputText
        inputText = ""

        isSending = true
        defer { isSending = false }

        let request = SendMessageRequest(
            type: .text,
            content: text,
            metadata: nil
        )

        do {
            let message = try await repository.sendMessage(
                conversationId: conversationId,
                request: request
            )
            messages.append(message)
        } catch {
            inputText = text
            errorMessage = error.localizedDescription
        }
    }

    private func startAutoRefresh() {
        refreshTask?.cancel()
        refreshTask = Task {
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 5_000_000_000)
                guard !Task.isCancelled else { break }
                await refreshMessages()
            }
        }
    }

    private func refreshMessages() async {
        guard let conversationId = conversationId else { return }

        do {
            let newMessages = try await repository.getMessages(conversationId: conversationId)
            if newMessages.count != messages.count {
                messages = newMessages
            }
        } catch {
            // 静默处理刷新错误
        }
    }
}
