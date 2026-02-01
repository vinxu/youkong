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
    private var loadTask: Task<Void, Never>?

    // MARK: - Load Grid Data

    func loadGrid() async {
        // 取消之前的请求
        loadTask?.cancel()

        loadTask = Task {
            isLoading = true
            errorMessage = nil

            do {
                let gridData: GridResponse = try await apiClient.request(.getGridData)

                // 检查是否已被取消
                if Task.isCancelled {
                    return
                }

                friends = gridData.friends
                gridSize = gridData.gridSize
            } catch {
                // 忽略取消错误
                if (error as NSError).code == NSURLErrorCancelled {
                    return
                }
                errorMessage = "加载失败: \(error.localizedDescription)"
                print("❌ [GridHome] Load failed: \(error)")
            }

            isLoading = false
        }

        await loadTask?.value
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
