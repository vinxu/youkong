import Foundation
import Combine
import Factory

@MainActor
final class ConversationsViewModel: ObservableObject {
    @Published var conversations: [Conversation] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Injected(\.messageRepository) private var repository

    func loadConversations() async {
        isLoading = true
        defer { isLoading = false }

        do {
            conversations = try await repository.getConversations()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func refresh() async {
        await loadConversations()
    }
}
