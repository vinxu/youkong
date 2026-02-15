import Foundation
import Combine
import Factory

// MARK: - Grid Home View Model

@MainActor
class GridHomeViewModel: ObservableObject {
    @Published var friends: [FriendStatus] = []
    @Published var isLoading = true
    @Published var error: Error?
    @Published var showPosterSheet = false
    @Published var unreadCounts: [String: Int] = [:]

    @Injected(\.agentRepository) private var agentRepository
    @Injected(\.messageRepository) private var messageRepository

    /// friendId → conversationId 映射
    private var friendConversationMap: [String: String] = [:]

    private var cancellables = Set<AnyCancellable>()

    init() {
        // 观察未读消息变化
        observeUnreadCounts()
    }

    // MARK: - Observe Unread Counts

    private func observeUnreadCounts() {
        UnreadMessageManager.shared.$unreadCounts
            .receive(on: DispatchQueue.main)
            .sink { [weak self] counts in
                self?.unreadCounts = counts
            }
            .store(in: &cancellables)
    }

    // MARK: - Get Unread Count

    func getUnreadCount(for friendId: String) -> Int {
        guard let conversationId = friendConversationMap[friendId] else {
            return 0
        }
        return unreadCounts[conversationId] ?? 0
    }

    // MARK: - Load Grid

    func loadGrid() async {
        // 首次加载时显示 loading
        if friends.isEmpty {
            isLoading = true
        }
        error = nil

        // 首次加载时，后台上报状态（确保城市信息被上传）
        Task {
            do {
                try await agentRepository.reportStatus()
                print("📡 [GridHome] 状态上报成功")
            } catch {
                print("❌ [GridHome] 状态上报失败: \(error)")
            }
        }

        do {
            // 先加载宫格数据（主要数据）
            let gridData = try await agentRepository.getGridData()

            // 转换并按更新时间排序（最新更新的在前面）
            let friendsList = gridData.friends.map { friend in
                FriendStatus(
                    id: friend.userId,
                    nickname: friend.nickname,
                    avatar: friend.avatar,
                    emoji: friend.emoji,
                    status: friend.status,
                    updatedAt: ISO8601DateFormatter().date(from: friend.updatedAt) ?? Date(),
                    relativeTime: friend.relativeTime,
                    city: friend.city,
                    isAvailable: friend.isAvailable ?? false,
                    isVisiting: friend.isVisiting ?? false,
                    gifUrl: friend.gifUrl,
                    giphyQuery: friend.giphyQuery,
                    useGif: friend.useGif ?? false,
                    needsSchedule: friend.needsSchedule ?? false,
                    riveState: friend.riveState,
                    sceneConfig: friend.pixelSceneConfig,
                    interactions: friend.interactions ?? [],
                    interactionCount: friend.interactionCount ?? 0
                )
            }.sorted { $0.updatedAt > $1.updatedAt }

            friends = friendsList

            // 会话列表单独加载，失败不影响主流程
            if let conversations = try? await messageRepository.getConversations() {
                buildFriendConversationMap(conversations)
                // 同步未读计数到 UnreadMessageManager（更新本地状态和 badge）
                UnreadMessageManager.shared.syncFromConversations(conversations)
            }

            print("📬 [GridHome] 加载完成，好友: \(friends.count), 会话映射: \(friendConversationMap.count)")
        } catch {
            self.error = error
            print("❌ [GridHome] Load failed: \(error)")
        }

        isLoading = false
    }

    // MARK: - Build Friend Conversation Map

    private func buildFriendConversationMap(_ conversations: [Conversation]) {
        friendConversationMap.removeAll()
        for conversation in conversations {
            friendConversationMap[conversation.partner.id] = conversation.id
        }
    }

    // MARK: - Refresh

    func refresh() async {
        do {
            // 加载宫格数据
            let gridData = try await agentRepository.getGridData()

            // 按更新时间排序
            let friendsList = gridData.friends.map { friend in
                FriendStatus(
                    id: friend.userId,
                    nickname: friend.nickname,
                    avatar: friend.avatar,
                    emoji: friend.emoji,
                    status: friend.status,
                    updatedAt: ISO8601DateFormatter().date(from: friend.updatedAt) ?? Date(),
                    relativeTime: friend.relativeTime,
                    city: friend.city,
                    isAvailable: friend.isAvailable ?? false,
                    isVisiting: friend.isVisiting ?? false,
                    gifUrl: friend.gifUrl,
                    giphyQuery: friend.giphyQuery,
                    useGif: friend.useGif ?? false,
                    needsSchedule: friend.needsSchedule ?? false,
                    riveState: friend.riveState,
                    sceneConfig: friend.pixelSceneConfig,
                    interactions: friend.interactions ?? [],
                    interactionCount: friend.interactionCount ?? 0
                )
            }.sorted { $0.updatedAt > $1.updatedAt }

            friends = friendsList

            // 会话列表单独加载，失败不影响刷新
            if let conversations = try? await messageRepository.getConversations() {
                buildFriendConversationMap(conversations)
                // 同步未读计数到 UnreadMessageManager
                UnreadMessageManager.shared.syncFromConversations(conversations)
            }

            print("🔄 [GridHome] 刷新完成，好友: \(friends.count)")
        } catch {
            // 刷新失败时不显示错误弹窗，只打印日志
            print("❌ [GridHome] Refresh failed: \(error)")
        }
    }

    // MARK: - Send Interaction

    func sendInteraction(to friendId: String, interaction: InteractionOptionItem) async {
        do {
            try await agentRepository.sendInteraction(
                receiverId: friendId,
                actionEmoji: interaction.emoji,
                actionLabel: interaction.label,
                actionPushText: interaction.pushText
            )
            print("🎮 [GridHome] 互动发送成功: \(interaction.label) → \(friendId)")
        } catch {
            print("❌ [GridHome] 互动发送失败: \(error)")
        }
    }

    // MARK: - Update Status

    func updateStatus() async {
        // 触发状态更新（上报设备状态）
        // 这会触发后端 LLM 分析并更新缓存
        do {
            try await agentRepository.reportStatus()
            // 等待一小段时间让后端处理完成
            try? await Task.sleep(nanoseconds: 1_000_000_000) // 1秒
            // 刷新数据
            await loadGrid()
        } catch {
            print("❌ [GridHome] Update status failed: \(error)")
            self.error = error
        }
    }
}

// MARK: - Friend Status Model

struct FriendStatus: Identifiable, Hashable {
    let id: String  // user_id
    let nickname: String
    let avatar: String?
    let emoji: String
    let status: String
    let updatedAt: Date
    let relativeTime: String
    let city: String?  // 城市名称（如"上海"、"北京"）
    let isAvailable: Bool  // 是否有空（用于高亮显示）
    let isVisiting: Bool   // 是否来访（当前城市≠常驻城市）
    let gifUrl: String?       // GIF 动图 URL（仅有空时）
    let giphyQuery: String?   // Giphy 搜索词
    let useGif: Bool          // 是否使用 GIF 显示模式
    let needsSchedule: Bool   // 自己当前无行程，需要设置
    // Rive 动画状态
    let riveState: String?
    // 像素场景
    let sceneConfig: PixelSceneConfig?
    // AI 互动选项
    let interactions: [InteractionOptionItem]
    // 今日互动计数
    let interactionCount: Int

    func hash(into hasher: inout Hasher) {
        hasher.combine(id)
        hasher.combine(emoji)
        hasher.combine(status)
        hasher.combine(city)
        hasher.combine(isAvailable)
        hasher.combine(isVisiting)
        hasher.combine(gifUrl)
        hasher.combine(useGif)
        hasher.combine(needsSchedule)
        hasher.combine(riveState)
        hasher.combine(sceneConfig)
        hasher.combine(interactionCount)
    }

    static func == (lhs: FriendStatus, rhs: FriendStatus) -> Bool {
        lhs.id == rhs.id &&
        lhs.emoji == rhs.emoji &&
        lhs.status == rhs.status &&
        lhs.city == rhs.city &&
        lhs.isAvailable == rhs.isAvailable &&
        lhs.isVisiting == rhs.isVisiting &&
        lhs.gifUrl == rhs.gifUrl &&
        lhs.useGif == rhs.useGif &&
        lhs.needsSchedule == rhs.needsSchedule &&
        lhs.riveState == rhs.riveState &&
        lhs.sceneConfig == rhs.sceneConfig &&
        lhs.interactionCount == rhs.interactionCount
    }
}
