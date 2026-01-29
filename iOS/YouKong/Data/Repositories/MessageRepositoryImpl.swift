import Foundation

final class MessageRepositoryImpl: MessageRepositoryProtocol {
    private let apiClient: APIClient

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
    }

    func getConversations() async throws -> [Conversation] {
        try await apiClient.request(.getConversations)
    }

    func createConversation(partnerId: String) async throws -> Conversation {
        try await apiClient.request(.createConversation(partnerId: partnerId))
    }

    func getMessages(conversationId: String) async throws -> [Message] {
        try await apiClient.request(.getMessages(conversationId: conversationId))
    }

    func sendMessage(conversationId: String, request: SendMessageRequest) async throws -> Message {
        try await apiClient.request(.sendMessage(conversationId: conversationId, request: request))
    }
}
