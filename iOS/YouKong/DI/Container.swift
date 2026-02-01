import Foundation
import Combine
import Factory

extension Container {
    // MARK: - API Client

    var apiClient: Factory<APIClient> {
        Factory(self) { APIClient.shared }
            .singleton
    }

    // MARK: - Repositories

    var authRepository: Factory<AuthRepositoryProtocol> {
        Factory(self) { AuthRepositoryImpl() }
            .singleton
    }

    var userRepository: Factory<UserRepositoryProtocol> {
        Factory(self) { UserRepositoryImpl() }
            .singleton
    }

    var messageRepository: Factory<MessageRepositoryProtocol> {
        Factory(self) { MessageRepositoryImpl() }
            .singleton
    }

    var agentRepository: Factory<AgentRepositoryProtocol> {
        Factory(self) { AgentRepositoryImpl() }
            .singleton
    }

    var contactRepository: Factory<ContactRepositoryProtocol> {
        Factory(self) { ContactRepositoryImpl() }
            .singleton
    }

    var inviteRepository: Factory<InviteRepositoryProtocol> {
        Factory(self) { InviteRepositoryImpl() }
            .singleton
    }

    var friendshipRepository: Factory<FriendshipRepositoryProtocol> {
        Factory(self) { FriendshipRepositoryImpl() }
            .singleton
    }

    var userProfileRepository: Factory<UserProfileRepositoryProtocol> {
        Factory(self) { UserProfileRepositoryImpl() }
            .singleton
    }
}
