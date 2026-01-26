import Foundation
import Factory

// MARK: - Invitation List ViewModel

@MainActor
final class InvitationListViewModel: ObservableObject {
    @Published var invitations: [Invitation] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Injected(\.invitationRepository) private var invitationRepository

    func loadInvitations() async {
        isLoading = true
        errorMessage = nil

        do {
            invitations = try await invitationRepository.getMyInvitations()
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    func refresh() async {
        await loadInvitations()
    }

    func disableInvitation(_ invitation: Invitation) async {
        do {
            try await invitationRepository.disableInvitation(id: invitation.id)
            // 重新加载列表
            await loadInvitations()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

// MARK: - Create Invitation ViewModel

@MainActor
final class CreateInvitationViewModel: ObservableObject {
    @Published var maxUses: Int = 100
    @Published var expiresDays: Int = 7
    @Published var isCreating = false
    @Published var createdInvitation: Invitation?
    @Published var errorMessage: String?

    @Injected(\.invitationRepository) private var invitationRepository

    func createInvitation() async {
        isCreating = true
        errorMessage = nil

        do {
            createdInvitation = try await invitationRepository.createInvitation(
                circleId: nil,
                maxUses: maxUses,
                expiresDays: expiresDays
            )
        } catch {
            errorMessage = error.localizedDescription
        }

        isCreating = false
    }
}

// MARK: - Accept Invitation ViewModel

@MainActor
final class AcceptInvitationViewModel: ObservableObject {
    @Published var invitationInfo: InvitationPublicInfo?
    @Published var isLoading = false
    @Published var isAccepting = false
    @Published var acceptResult: AcceptInvitationResponse?
    @Published var errorMessage: String?

    @Injected(\.invitationRepository) private var invitationRepository

    let code: String

    init(code: String) {
        self.code = code
    }

    func loadInvitationInfo() async {
        isLoading = true
        errorMessage = nil

        do {
            invitationInfo = try await invitationRepository.getInvitationByCode(code: code)
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    func acceptInvitation() async {
        isAccepting = true
        errorMessage = nil

        do {
            acceptResult = try await invitationRepository.acceptInvitation(code: code)
        } catch {
            errorMessage = error.localizedDescription
        }

        isAccepting = false
    }
}

// MARK: - Friends Management ViewModel

@MainActor
final class FriendsManagementViewModel: ObservableObject {
    @Published var friends: [FriendInfo] = []
    @Published var invitedByMe: [FriendWithInvitation] = []
    @Published var invitedMe: [FriendWithInvitation] = []
    @Published var pendingRequestCount: Int = 0
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Injected(\.friendshipRepository) private var friendshipRepository
    @Injected(\.contactRepository) private var contactRepository

    func loadFriends() async {
        isLoading = true
        errorMessage = nil

        do {
            friends = try await friendshipRepository.getFriends()
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    func loadInvitedByMe() async {
        do {
            invitedByMe = try await friendshipRepository.getFriendsInvitedByMe()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func loadInvitedMe() async {
        do {
            invitedMe = try await friendshipRepository.getFriendsInvitedMe()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func loadPendingRequestCount() async {
        do {
            pendingRequestCount = try await contactRepository.getPendingRequestCount()
        } catch {
            // 忽略错误，保持 count 为 0
        }
    }

    func deleteFriend(_ friend: FriendInfo) async {
        do {
            try await friendshipRepository.deleteFriend(userId: friend.user.id)
            await loadFriends()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
