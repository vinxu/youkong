import Foundation
import SwiftUI

@MainActor
class GridHomeViewModel: ObservableObject {
    @Published var friends: [FriendGridItem] = []
    @Published var gridSize: Int = 1
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var showPosterSheet = false

    private let apiClient = APIClient.shared

    // MARK: - Load Grid Data

    func loadGrid() async {
        isLoading = true
        errorMessage = nil

        do {
            let response: APIResponse<GridResponse> = try await apiClient.request(.getGridData)
            friends = response.data.friends
            gridSize = response.data.gridSize
        } catch {
            errorMessage = "加载失败: \(error.localizedDescription)"
            print("❌ [GridHome] Load failed: \(error)")
        }

        isLoading = false
    }

    // MARK: - Update Status

    func updateStatus() async {
        // TODO: 触发 Agent 分析
        // 暂时直接刷新宫格
        await loadGrid()
    }

    // MARK: - Generate Poster

    func generatePoster() {
        // TODO: 实现海报生成
        showPosterSheet = true
    }
}
