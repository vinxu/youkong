import Foundation
import Combine
import Factory

extension Container {
    // MARK: - Repositories

    var authRepository: Factory<AuthRepositoryProtocol> {
        Factory(self) { AuthRepositoryImpl() }
            .singleton
    }

    var userRepository: Factory<UserRepositoryProtocol> {
        Factory(self) { UserRepositoryImpl() }
            .singleton
    }

    var circleRepository: Factory<CircleRepositoryProtocol> {
        Factory(self) { CircleRepositoryImpl() }
            .singleton
    }

    var availabilityRepository: Factory<AvailabilityRepositoryProtocol> {
        Factory(self) { AvailabilityRepositoryImpl() }
            .singleton
    }

    var messageRepository: Factory<MessageRepositoryProtocol> {
        Factory(self) { MessageRepositoryImpl() }
            .singleton
    }

    var invitationRepository: Factory<InvitationRepositoryProtocol> {
        Factory(self) { InvitationRepositoryImpl() }
            .singleton
    }

    var friendRepository: Factory<FriendRepositoryProtocol> {
        Factory(self) { FriendRepositoryImpl() }
            .singleton
    }
}
