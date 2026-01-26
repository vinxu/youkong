import Foundation
import SwiftUI
import Combine
import Factory

@MainActor
final class AddMemberViewModel: ObservableObject {
    @Published var searchText = ""
    @Published var searchResults: [UserProfile] = []
    @Published var isSearching = false
    @Published var showError = false
    @Published var errorMessage = ""

    @Injected(\.userRepository) private var userRepository
    @Injected(\.circleRepository) private var circleRepository

    private let circleId: String
    private var searchTask: Task<Void, Never>?

    init(circleId: String) {
        self.circleId = circleId
    }

    func search() {
        searchTask?.cancel()

        guard !searchText.isEmpty else {
            searchResults = []
            return
        }

        searchTask = Task {
            try? await Task.sleep(nanoseconds: 300_000_000)

            guard !Task.isCancelled else { return }

            isSearching = true
            defer { isSearching = false }

            do {
                searchResults = try await userRepository.searchUsers(keyword: searchText)
            } catch {
                if !Task.isCancelled {
                    errorMessage = error.localizedDescription
                    showError = true
                }
            }
        }
    }

    func addMember(userId: String) async {
        do {
            try await circleRepository.addMember(circleId: circleId, userId: userId)
        } catch let error as APIError {
            errorMessage = error.localizedDescription
            showError = true
        } catch {
            errorMessage = "添加失败，请稍后重试"
            showError = true
        }
    }
}
