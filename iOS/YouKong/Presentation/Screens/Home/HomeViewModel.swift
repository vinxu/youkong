import Foundation
import Combine
import Factory

@MainActor
final class HomeViewModel: ObservableObject {
    @Published var availabilities: [Availability] = []
    @Published var isLoading = false
    @Published var showMyAvailabilities = false
    @Published var errorMessage: String?

    @Injected(\.availabilityRepository) private var repository

    func loadAvailabilities() async {
        isLoading = true
        defer { isLoading = false }

        do {
            if showMyAvailabilities {
                availabilities = try await repository.getMyAvailabilities()
            } else {
                availabilities = try await repository.getFriendsAvailabilities()
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func refresh() async {
        await loadAvailabilities()
    }
}
