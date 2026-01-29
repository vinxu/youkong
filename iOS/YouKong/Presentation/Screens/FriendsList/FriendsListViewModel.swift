import Foundation
import Combine
import Factory

// MARK: - Friends List View Model

@MainActor
class FriendsListViewModel: ObservableObject {
    @Published var friends: [FriendRecommendation] = []
    @Published var isLoading = false
    @Published var error: Error?
    @Published var lastUpdated: Date?

    @Injected(\.agentRepository) private var agentRepository
    @Injected(\.messageRepository) private var messageRepository

    /// friendId → conversationId 映射
    private var friendConversationMap: [String: String] = [:]

    private var cancellables = Set<AnyCancellable>()

    init() {
        // 监听未读数变化，触发 UI 刷新
        UnreadMessageManager.shared.$unreadCounts
            .sink { [weak self] _ in
                self?.objectWillChange.send()
            }
            .store(in: &cancellables)
    }

    // MARK: - Load Friends

    func loadFriends() async {
        guard !isLoading else { return }

        isLoading = true
        error = nil

        do {
            // 并行加载好友列表和会话列表
            async let friendsTask = agentRepository.getFriendsFreeProbability()
            async let conversationsTask = messageRepository.getConversations()

            friends = try await friendsTask
            let conversations = try await conversationsTask

            // 建立 friendId → conversationId 映射
            buildFriendConversationMap(conversations)

            lastUpdated = Date()

            print("📬 [FriendsListViewModel] 加载完成")
            print("📬 好友数量: \(friends.count)")
            print("📬 会话数量: \(conversations.count)")
            print("📬 映射数量: \(friendConversationMap.count)")
        } catch {
            self.error = error
            print("Failed to load friends: \(error)")
        }

        isLoading = false
    }

    // MARK: - Refresh

    func refresh() async {
        await loadFriends()
    }

    // MARK: - Unread Count

    /// 获取某个好友的未读消息数
    func getUnreadCount(for friendId: String) -> Int {
        guard let conversationId = friendConversationMap[friendId] else {
            return 0
        }
        let count = UnreadMessageManager.shared.getUnreadCount(for: conversationId)
        return count
    }

    // MARK: - Helper

    var isEmpty: Bool {
        friends.isEmpty && !isLoading
    }

    var lastUpdatedText: String? {
        guard let date = lastUpdated else { return nil }

        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .short
        return formatter.localizedString(for: date, relativeTo: Date())
    }

    // MARK: - Private

    private func buildFriendConversationMap(_ conversations: [Conversation]) {
        friendConversationMap.removeAll()
        for conversation in conversations {
            friendConversationMap[conversation.partner.id] = conversation.id
            print("📬 映射: \(conversation.partner.nickname) (\(conversation.partner.id)) → \(conversation.id)")
        }
    }
}
