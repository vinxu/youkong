import Foundation
import SwiftUI
import Combine
import Factory

@MainActor
final class CreateCircleViewModel: ObservableObject {
    @Published var name = ""
    @Published var emoji = "😊"
    @Published var selectedColorIndex = 0
    @Published var isLoading = false
    @Published var showError = false
    @Published var errorMessage = ""
    @Published var showEmojiPicker = false

    @Injected(\.circleRepository) private var repository

    var selectedColor: Color {
        CircleColors.all[selectedColorIndex].color
    }

    var selectedColorHex: String {
        CircleColors.hexValues[selectedColorIndex]
    }

    func createCircle() async -> FriendCircle? {
        guard !name.isEmpty else {
            errorMessage = "请输入圈子名称"
            showError = true
            return nil
        }

        isLoading = true
        defer { isLoading = false }

        do {
            let circle = try await repository.createCircle(
                name: name,
                emoji: emoji,
                color: selectedColorHex
            )
            return circle
        } catch let error as APIError {
            errorMessage = error.localizedDescription
            showError = true
            return nil
        } catch {
            errorMessage = "创建失败，请稍后重试"
            showError = true
            return nil
        }
    }
}
