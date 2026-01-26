package com.youkong.core.database

import androidx.room.Database
import androidx.room.RoomDatabase
import com.youkong.core.database.dao.AvailabilityDao
import com.youkong.core.database.dao.CircleDao
import com.youkong.core.database.dao.ConversationDao
import com.youkong.core.database.dao.MessageDao
import com.youkong.core.database.dao.UserDao
import com.youkong.core.database.entity.AvailabilityEntity
import com.youkong.core.database.entity.CircleEntity
import com.youkong.core.database.entity.ConversationEntity
import com.youkong.core.database.entity.MessageEntity
import com.youkong.core.database.entity.UserEntity

@Database(
    entities = [
        UserEntity::class,
        CircleEntity::class,
        AvailabilityEntity::class,
        ConversationEntity::class,
        MessageEntity::class,
    ],
    version = 1,
    exportSchema = true,
)
abstract class YouKongDatabase : RoomDatabase() {
    abstract fun userDao(): UserDao
    abstract fun circleDao(): CircleDao
    abstract fun availabilityDao(): AvailabilityDao
    abstract fun conversationDao(): ConversationDao
    abstract fun messageDao(): MessageDao
}
