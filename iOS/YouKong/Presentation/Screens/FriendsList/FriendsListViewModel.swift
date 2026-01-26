import Foundation
import Combine
import Factory

// MARK: - Friends List View Model

@MainActor
class FriendsListViewModel: ObservableObject {
    @Published var friends: [FriendRecommendation] = []
    @Published var isLoading = false
    @Published var error: Error?
    @Published var lastUpdated: Date?

    @Injected(\.agentRepository) private var agentRepository

    // MARK: - Load Friends

    func loadFriends() async {
        guard !isLoading else { return }

        isLoading = true
        error = nil

        do {
            friends = try await agentRepository.getFriendsFreeProbability()
            lastUpdated = Date()
        } catch {
            self.error = error
            print("Failed to load friends: \(error)")
        }

        isLoading = false
    }

    // MARK: - Refresh

    func refresh() async {
        await loadFriends()
    }

    // MARK: - Helper

    var isEmpty: Bool {
        friends.isEmpty && !isLoading
    }

    var lastUpdatedText: String? {
        guard let date = lastUpdated else { return nil }

        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .short
        return formatter.localizedString(for: date, relativeTo: Date())
    }
}
