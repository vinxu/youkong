import Foundation

protocol MessageRepositoryProtocol {
    func getConversations() async throws -> [Conversation]
    func createConversation(partnerId: String) async throws -> Conversation
    func getMessages(conversationId: String) async throws -> [Message]
    func sendMessage(conversationId: String, request: SendMessageRequest) async throws -> Message
}
