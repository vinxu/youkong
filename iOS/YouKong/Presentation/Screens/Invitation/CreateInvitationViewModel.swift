//
//  CreateInvitationViewModel.swift
//  YouKong
//
//  创建邀请 ViewModel
//

import Foundation
import Combine
import Factory

@MainActor
final class CreateInvitationViewModel: ObservableObject {
    @Published var circles: [FriendCircle] = []
    @Published var selectedCircleId: String?
    @Published var maxUses: Int = 100
    @Published var expiresDays: Int = 7
    @Published var isLoading = false
    @Published var isCreating = false
    @Published var errorMessage: String?
    @Published var createdInvitation: InvitationDetail?

    @Injected(\.circleRepository) private var circleRepository
    @Injected(\.invitationRepository) private var invitationRepository

    let maxUsesOptions = [10, 50, 100, 500]
    let expiresDaysOptions = [1, 3, 7, 14, 30]

    func loadCircles() async {
        isLoading = true
        defer { isLoading = false }

        do {
            circles = try await circleRepository.getCircles()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func createInvitation() async -> InvitationDetail? {
        isCreating = true
        defer { isCreating = false }

        do {
            let invitation = try await invitationRepository.createInvitation(
                circleId: selectedCircleId,
                maxUses: maxUses,
                expiresDays: expiresDays
            )
            createdInvitation = invitation
            return invitation
        } catch {
            errorMessage = error.localizedDescription
            return nil
        }
    }
}
