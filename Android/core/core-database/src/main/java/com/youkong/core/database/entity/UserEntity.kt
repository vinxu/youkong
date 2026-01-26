package com.youkong.core.database.entity

import androidx.room.Entity
import androidx.room.PrimaryKey
import com.youkong.core.domain.model.User
import com.youkong.core.domain.model.UserProfile
import kotlinx.datetime.Instant

@Entity(tableName = "users")
data class UserEntity(
    @PrimaryKey
    val id: String,
    val phone: String?,
    val nickname: String,
    val avatar: String?,
    val createdAt: Long,
    val updatedAt: Long,
)

fun UserEntity.toDomain(): User = User(
    id = id,
    phone = phone ?: "",
    nickname = nickname,
    avatar = avatar,
    createdAt = Instant.fromEpochMilliseconds(createdAt),
    updatedAt = Instant.fromEpochMilliseconds(updatedAt),
)

fun UserEntity.toProfile(): UserProfile = UserProfile(
    id = id,
    nickname = nickname,
    avatar = avatar,
)

fun User.toEntity(): UserEntity = UserEntity(
    id = id,
    phone = phone,
    nickname = nickname,
    avatar = avatar,
    createdAt = createdAt.toEpochMilliseconds(),
    updatedAt = updatedAt.toEpochMilliseconds(),
)
