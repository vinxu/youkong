import Foundation
import Combine
import Factory

// MARK: - Grid Home View Model

@MainActor
class GridHomeViewModel: ObservableObject {
    @Published var friends: [FriendStatus] = []
    @Published var isLoading = false
    @Published var error: Error?
    @Published var showPosterSheet = false

    @Injected(\.agentRepository) private var agentRepository

    // MARK: - Load Grid

    func loadGrid() async {
        guard !isLoading else { return }

        isLoading = true
        error = nil

        do {
            let gridData = try await agentRepository.getGridData()
            friends = gridData.friends.map { friend in
                FriendStatus(
                    id: friend.userID,
                    nickname: friend.nickname,
                    avatar: friend.avatar,
                    emoji: friend.emoji,
                    status: friend.status,
                    updatedAt: ISO8601DateFormatter().date(from: friend.updatedAt) ?? Date(),
                    relativeTime: friend.relativeTime
                )
            }
        } catch {
            self.error = error
            print("❌ [GridHome] Load failed: \(error)")
        }

        isLoading = false
    }

    // MARK: - Refresh

    func refresh() async {
        await loadGrid()
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

struct FriendStatus: Identifiable {
    let id: String  // user_id
    let nickname: String
    let avatar: String?
    let emoji: String
    let status: String
    let updatedAt: Date
    let relativeTime: String
}
