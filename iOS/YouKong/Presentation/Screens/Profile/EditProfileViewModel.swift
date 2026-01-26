import Foundation
import Combine
import Factory

@MainActor
final class EditProfileViewModel: ObservableObject {
    @Published var nickname: String
    @Published var avatar: String?
    @Published var isLoading = false
    @Published var showError = false
    @Published var errorMessage = ""

    @Injected(\.userRepository) private var repository

    init(user: User) {
        self.nickname = user.nickname
        self.avatar = user.avatar
    }

    func save() async -> User? {
        guard !nickname.isEmpty else {
            errorMessage = "昵称不能为空"
            showError = true
            return nil
        }

        isLoading = true
        defer { isLoading = false }

        do {
            let user = try await repository.updateCurrentUser(nickname: nickname, avatar: avatar)
            return user
        } catch let error as APIError {
            errorMessage = error.localizedDescription
            showError = true
            return nil
        } catch {
            errorMessage = "保存失败，请稍后重试"
            showError = true
            return nil
        }
    }
}
