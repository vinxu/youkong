import Foundation
import Combine
import Factory

@MainActor
final class CircleDetailViewModel: ObservableObject {
    @Published var circleDetail: FriendCircleDetail?
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Injected(\.circleRepository) private var repository

    private let circleId: String

    init(circleId: String) {
        self.circleId = circleId
    }

    func loadCircleDetail() async {
        isLoading = true
        defer { isLoading = false }

        do {
            circleDetail = try await repository.getCircle(id: circleId)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func deleteCircle() async {
        do {
            try await repository.deleteCircle(id: circleId)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func removeMember(userId: String) async {
        do {
            try await repository.removeMember(circleId: circleId, userId: userId)
            await loadCircleDetail()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
