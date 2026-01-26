package com.youkong.core.database.di

import android.content.Context
import androidx.room.Room
import com.youkong.core.database.YouKongDatabase
import com.youkong.core.database.dao.AvailabilityDao
import com.youkong.core.database.dao.CircleDao
import com.youkong.core.database.dao.ConversationDao
import com.youkong.core.database.dao.MessageDao
import com.youkong.core.database.dao.UserDao
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object DatabaseModule {

    @Provides
    @Singleton
    fun provideDatabase(
        @ApplicationContext context: Context
    ): YouKongDatabase {
        return Room.databaseBuilder(
            context,
            YouKongDatabase::class.java,
            "youkong.db"
        ).build()
    }

    @Provides
    fun provideUserDao(database: YouKongDatabase): UserDao = database.userDao()

    @Provides
    fun provideCircleDao(database: YouKongDatabase): CircleDao = database.circleDao()

    @Provides
    fun provideAvailabilityDao(database: YouKongDatabase): AvailabilityDao = database.availabilityDao()

    @Provides
    fun provideConversationDao(database: YouKongDatabase): ConversationDao = database.conversationDao()

    @Provides
    fun provideMessageDao(database: YouKongDatabase): MessageDao = database.messageDao()
}
