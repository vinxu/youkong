import Foundation
import SwiftUI

@MainActor
class GridHomeViewModel: ObservableObject {
    @Published var friends: [FriendGridItem] = []
    @Published var gridSize: Int = 1
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var showPosterSheet = false
    @Published var showAnalysisSheet = false

    private let apiClient = APIClient.shared

    // MARK: - Load Grid Data

    func loadGrid() async {
        isLoading = true
        errorMessage = nil

        do {
            let gridData: GridResponse = try await apiClient.request(.getGridData)
            friends = gridData.friends
            gridSize = gridData.gridSize
        } catch {
            errorMessage = "加载失败: \(error.localizedDescription)"
            print("❌ [GridHome] Load failed: \(error)")
        }

        isLoading = false
    }

    // MARK: - Update Status

    func updateStatus() {
        // 显示 Agent 分析页面
        showAnalysisSheet = true
    }

    func onAnalysisComplete() {
        // 分析完成后刷新宫格
        Task {
            await loadGrid()
        }
    }

    // MARK: - Generate Poster

    func generatePoster() {
        // TODO: 实现海报生成
        showPosterSheet = true
    }
}
