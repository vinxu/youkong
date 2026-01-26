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

    var invitationRepository: Factory<InvitationRepositoryProtocol> {
        Factory(self) { InvitationRepositoryImpl() }
            .singleton
    }

    var friendshipRepository: Factory<FriendshipRepositoryProtocol> {
        Factory(self) { FriendshipRepositoryImpl() }
            .singleton
    }
}
