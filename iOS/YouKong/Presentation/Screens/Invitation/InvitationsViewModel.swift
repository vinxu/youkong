//
//  InvitationsViewModel.swift
//  YouKong
//
//  邀请列表 ViewModel
//

import Foundation
import Combine
import Factory

@MainActor
final class InvitationsViewModel: ObservableObject {
    @Published var invitations: [Invitation] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Injected(\.invitationRepository) private var repository

    func loadInvitations() async {
        isLoading = true
        defer { isLoading = false }

        do {
            invitations = try await repository.getMyInvitations()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func refresh() async {
        await loadInvitations()
    }

    func disableInvitation(id: String) async {
        do {
            try await repository.disableInvitation(id: id)
            invitations.removeAll { $0.id == id }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func addInvitation(_ detail: InvitationDetail) {
        let invitation = Invitation(
            id: detail.id,
            code: detail.code,
            inviteUrl: detail.inviteUrl,
            circle: detail.circle,
            maxUses: detail.maxUses,
            useCount: detail.useCount,
            expiresAt: detail.expiresAt,
            status: detail.status,
            isValid: detail.isValid,
            createdAt: detail.createdAt
        )
        invitations.insert(invitation, at: 0)
    }
}
