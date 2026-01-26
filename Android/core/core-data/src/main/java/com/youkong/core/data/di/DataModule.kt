package com.youkong.core.data.di

import com.youkong.core.data.repository.AgentRepositoryImpl
import com.youkong.core.data.repository.AuthRepositoryImpl
import com.youkong.core.data.repository.AvailabilityRepositoryImpl
import com.youkong.core.data.repository.CircleRepositoryImpl
import com.youkong.core.data.repository.FriendRepositoryImpl
import com.youkong.core.data.repository.InvitationRepositoryImpl
import com.youkong.core.data.repository.MessageRepositoryImpl
import com.youkong.core.data.repository.UserRepositoryImpl
import com.youkong.core.domain.repository.AgentRepository
import com.youkong.core.domain.repository.AuthRepository
import com.youkong.core.domain.repository.AvailabilityRepository
import com.youkong.core.domain.repository.CircleRepository
import com.youkong.core.domain.repository.FriendRepository
import com.youkong.core.domain.repository.InvitationRepository
import com.youkong.core.domain.repository.MessageRepository
import com.youkong.core.domain.repository.UserRepository
import dagger.Binds
import dagger.Module
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
abstract class DataModule {

    @Binds
    @Singleton
    abstract fun bindAuthRepository(impl: AuthRepositoryImpl): AuthRepository

    @Binds
    @Singleton
    abstract fun bindUserRepository(impl: UserRepositoryImpl): UserRepository

    @Binds
    @Singleton
    abstract fun bindCircleRepository(impl: CircleRepositoryImpl): CircleRepository

    @Binds
    @Singleton
    abstract fun bindAvailabilityRepository(impl: AvailabilityRepositoryImpl): AvailabilityRepository

    @Binds
    @Singleton
    abstract fun bindMessageRepository(impl: MessageRepositoryImpl): MessageRepository

    @Binds
    @Singleton
    abstract fun bindInvitationRepository(impl: InvitationRepositoryImpl): InvitationRepository

    @Binds
    @Singleton
    abstract fun bindFriendRepository(impl: FriendRepositoryImpl): FriendRepository

    @Binds
    @Singleton
    abstract fun bindAgentRepository(impl: AgentRepositoryImpl): AgentRepository
}
