import Foundation
import Combine
import Factory
import CryptoKit

// MARK: - Notification

extension Notification.Name {
    static let gridNeedsRefresh = Notification.Name("gridNeedsRefresh")
}

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

    /// GIF 持久化缓存: "friendId:query" → gifUrl（跨 app 启动保留，每天自动清空）
    private var gifCache: [String: String]

    private static let gifCacheKey = "gifCache"
    private static let gifCacheDateKey = "gifCacheDate"

    private var cancellables = Set<AnyCancellable>()

    init() {
        // 加载持久化缓存，日期不同则清空
        let today = Self.todayDateString()
        let storedDate = UserDefaults.standard.string(forKey: Self.gifCacheDateKey)
        if storedDate == today,
           let stored = UserDefaults.standard.dictionary(forKey: Self.gifCacheKey) as? [String: String] {
            gifCache = stored
        } else {
            gifCache = [:]
            UserDefaults.standard.set(today, forKey: Self.gifCacheDateKey)
            UserDefaults.standard.removeObject(forKey: Self.gifCacheKey)
        }

        // 观察未读消息变化
        observeUnreadCounts()

        // 观察状态更新通知 → 自动刷新 Grid
        NotificationCenter.default.publisher(for: .gridNeedsRefresh)
            .debounce(for: .milliseconds(500), scheduler: DispatchQueue.main)
            .sink { [weak self] _ in
                Task { [weak self] in
                    await self?.refresh()
                }
            }
            .store(in: &cancellables)
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
                    sceneConfig: friend.pixelSceneConfig,
                    interactions: friend.interactions ?? [],
                    interactionCount: friend.interactionCount ?? 0,
                    hasPlusOned: friend.hasPlusOned ?? false,
                    isSelf: friend.isSelf ?? false
                )
            }.sorted { $0.updatedAt > $1.updatedAt }

            friends = friendsList

            // 异步解析缺失的 GIF URL
            resolveGifUrls()

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

    // MARK: - Resolve GIF URLs

    private func resolveGifUrls() {
        // 收集需要解析的好友（服务器未返回 gif_url 且本地无缓存）
        var toResolve: [(index: Int, friendId: String, query: String)] = []

        for i in friends.indices {
            let friend = friends[i]
            guard friend.gifUrl == nil || friend.gifUrl?.isEmpty == true,
                  !friend.needsSchedule else { continue }

            let query = (friend.giphyQuery?.isEmpty == false ? friend.giphyQuery : friend.status) ?? ""
            guard !query.isEmpty else { continue }

            let cacheKey = "\(friend.id):\(query)"

            // 本地缓存命中 → 直接使用，不发起网络请求
            if let cachedUrl = gifCache[cacheKey] {
                friends[i].gifUrl = cachedUrl
                friends[i].useGif = true
                continue
            }

            toResolve.append((index: i, friendId: friend.id, query: query))
        }

        guard !toResolve.isEmpty else { return }

        // 并发解析，完成后批量写回服务器缓存
        Task {
            var writeBackItems: [[String: String]] = []

            await withTaskGroup(of: (Int, String, String, String?).self) { group in
                for item in toResolve {
                    group.addTask { [self] in
                        let url = await fetchGifUrl(query: item.query, friendId: item.friendId)
                        return (item.index, item.friendId, item.query, url)
                    }
                }
                for await (index, friendId, query, url) in group {
                    if let url = url {
                        let cacheKey = "\(friendId):\(query)"
                        gifCache[cacheKey] = url
                        friends[index].gifUrl = url
                        friends[index].useGif = true
                        writeBackItems.append([
                            "friend_id": friendId,
                            "query": query,
                            "cos_url": url
                        ])
                    }
                }
            }

            if !writeBackItems.isEmpty {
                Self.persistGifCache(gifCache)
                // 写回服务器：其他客户端下次直接从 Grid API 拿到 cos_url
                await writeBackGifCache(writeBackItems)
            }
        }
    }

    /// 批量写回 GIF cos_url 到服务器缓存
    private func writeBackGifCache(_ items: [[String: String]]) async {
        do {
            try await agentRepository.cacheGifUrls(items: items)
            print("[GridHome] GIF 缓存写回成功: \(items.count) 条")
        } catch {
            // 写回失败不影响主流程，静默忽略
            print("[GridHome] GIF 缓存写回失败: \(error.localizedDescription)")
        }
    }

    private func fetchGifUrl(query: String, friendId: String) async -> String? {
        guard let encoded = query.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) else { return nil }

        // seed = md5(friendId + query + todayDate)，保证同人同状态当天稳定，每天轮换
        let today = Self.todayDateString()
        let seed = Self.md5("\(friendId)\(query)\(today)")

        guard let url = URL(string: "https://gif.playa.cn/api/giphy?q=\(encoded)&seed=\(seed)") else { return nil }
        do {
            let (data, _) = try await URLSession.shared.data(from: url)
            if let json = try JSONSerialization.jsonObject(with: data) as? [String: Any],
               let result = json["result"] as? [String: Any],
               let cosUrl = result["cos_url"] as? String, !cosUrl.isEmpty {
                return cosUrl
            }
        } catch {
            print("[GridHome] GIF 解析失败: \(error.localizedDescription)")
        }
        return nil
    }

    // MARK: - Helpers

    private static func md5(_ string: String) -> String {
        let digest = Insecure.MD5.hash(data: Data(string.utf8))
        return digest.map { String(format: "%02x", $0) }.joined()
    }

    private static func todayDateString() -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: Date())
    }

    private static func persistGifCache(_ cache: [String: String]) {
        UserDefaults.standard.set(cache, forKey: gifCacheKey)
    }

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
                    sceneConfig: friend.pixelSceneConfig,
                    interactions: friend.interactions ?? [],
                    interactionCount: friend.interactionCount ?? 0,
                    hasPlusOned: friend.hasPlusOned ?? false,
                    isSelf: friend.isSelf ?? false
                )
            }.sorted { $0.updatedAt > $1.updatedAt }

            friends = friendsList

            resolveGifUrls()

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

    // MARK: - Send +1

    func sendPlusOne(friend: FriendStatus) async {
        // 乐观更新
        if let index = friends.firstIndex(where: { $0.id == friend.id }) {
            friends[index].hasPlusOned = true
            friends[index].interactionCount += 1
        }

        do {
            try await agentRepository.sendInteraction(
                receiverId: friend.id,
                actionEmoji: friend.emoji,
                actionLabel: "+1",
                actionPushText: "在你的\(friend.status)状态上+1"
            )
            print("✅ [GridHome] +1 发送成功 → \(friend.nickname)")
        } catch {
            // 失败回滚
            if let index = friends.firstIndex(where: { $0.id == friend.id }) {
                friends[index].hasPlusOned = false
                friends[index].interactionCount = max(0, friends[index].interactionCount - 1)
            }
            print("❌ [GridHome] +1 发送失败: \(error)")
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
    var gifUrl: String?       // GIF 动图 URL（仅有空时）
    let giphyQuery: String?   // Giphy 搜索词
    var useGif: Bool          // 是否使用 GIF 显示模式
    let needsSchedule: Bool   // 自己当前无行程，需要设置
    // 像素场景
    let sceneConfig: PixelSceneConfig?
    // AI 互动选项
    let interactions: [InteractionOptionItem]
    // 今日互动计数
    var interactionCount: Int
    // 是否已对该好友当前状态 +1
    var hasPlusOned: Bool
    // 是否是自己
    let isSelf: Bool

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
        hasher.combine(sceneConfig)
        hasher.combine(interactionCount)
        hasher.combine(hasPlusOned)
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
        lhs.sceneConfig == rhs.sceneConfig &&
        lhs.interactionCount == rhs.interactionCount &&
        lhs.hasPlusOned == rhs.hasPlusOned
    }
}
