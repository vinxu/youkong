package com.youkong.core.database.entity

import androidx.room.Entity
import androidx.room.PrimaryKey
import com.youkong.core.domain.model.Circle
import kotlinx.datetime.Instant

@Entity(tableName = "circles")
data class CircleEntity(
    @PrimaryKey
    val id: String,
    val name: String,
    val emoji: String,
    val color: String,
    val ownerId: String,
    val createdAt: Long,
)

fun CircleEntity.toDomain(): Circle = Circle(
    id = id,
    name = name,
    emoji = emoji,
    color = color,
    ownerId = ownerId,
    createdAt = Instant.fromEpochMilliseconds(createdAt),
)

fun Circle.toEntity(): CircleEntity = CircleEntity(
    id = id,
    name = name,
    emoji = emoji,
    color = color,
    ownerId = ownerId,
    createdAt = createdAt.toEpochMilliseconds(),
)
