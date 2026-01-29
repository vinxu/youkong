import Foundation

@MainActor
final class UnreadMessageManager: ObservableObject {
    static let shared = UnreadMessageManager()

    /// 每个会话的未读消息数
    @Published private(set) var unreadCounts: [String: Int] = [:]

    /// 总未读数
    @Published private(set) var totalUnreadCount: Int = 0

    private init() {}

    /// 收到新消息，增加未读数
    func incrementUnread(for conversationId: String) {
        let currentCount = unreadCounts[conversationId] ?? 0
        unreadCounts[conversationId] = currentCount + 1
        updateTotalCount()
        updateAppBadge()

        print("📬 [UnreadManager] 会话 \(conversationId) 未读数: \(unreadCounts[conversationId] ?? 0)")
        print("📬 [UnreadManager] 总未读数: \(totalUnreadCount)")
    }

    /// 清除某个会话的未读数（进入聊天页面时调用）
    func clearUnread(for conversationId: String) {
        unreadCounts[conversationId] = 0
        updateTotalCount()
        updateAppBadge()

        print("📬 [UnreadManager] 清除会话 \(conversationId) 的未读数")
        print("📬 [UnreadManager] 总未读数: \(totalUnreadCount)")
    }

    /// 获取某个会话的未读数
    func getUnreadCount(for conversationId: String) -> Int {
        return unreadCounts[conversationId] ?? 0
    }

    /// 更新总未读数
    private func updateTotalCount() {
        totalUnreadCount = unreadCounts.values.reduce(0, +)
    }

    /// 更新 App Badge
    private func updateAppBadge() {
        NotificationManager.shared.updateBadge(count: totalUnreadCount)
    }

    /// 清除所有未读（退出登录时调用）
    func clearAll() {
        unreadCounts.removeAll()
        totalUnreadCount = 0
        updateAppBadge()
    }
}
