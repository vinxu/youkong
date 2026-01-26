import Foundation
import Combine
import Factory

@MainActor
final class MyAvailabilitiesViewModel: ObservableObject {
    @Published var availabilities: [Availability] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Injected(\.availabilityRepository) private var repository

    func loadAvailabilities() async {
        isLoading = true
        defer { isLoading = false }

        do {
            availabilities = try await repository.getMyAvailabilities()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func refresh() async {
        await loadAvailabilities()
    }

    func cancel(id: String) async {
        do {
            try await repository.cancelAvailability(id: id)
            availabilities.removeAll { $0.id == id }
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
