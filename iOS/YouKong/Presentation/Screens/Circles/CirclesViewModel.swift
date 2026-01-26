import Foundation
import Combine
import Factory

@MainActor
final class CirclesViewModel: ObservableObject {
    @Published var circles: [FriendCircle] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Injected(\.circleRepository) private var repository

    func loadCircles() async {
        isLoading = true
        defer { isLoading = false }

        do {
            circles = try await repository.getCircles()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func refresh() async {
        await loadCircles()
    }

    func addCircle(_ circle: FriendCircle) {
        circles.insert(circle, at: 0)
    }

    func removeCircle(id: String) {
        circles.removeAll { $0.id == id }
    }
}
