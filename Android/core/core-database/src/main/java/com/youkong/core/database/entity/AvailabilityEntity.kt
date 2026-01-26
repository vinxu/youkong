package com.youkong.core.database.entity

import androidx.room.Embedded
import androidx.room.Entity
import androidx.room.PrimaryKey
import com.youkong.core.domain.model.Availability
import com.youkong.core.domain.model.AvailabilityStatus
import com.youkong.core.domain.model.Location
import com.youkong.core.domain.model.LocationType
import com.youkong.core.domain.model.UserProfile
import kotlinx.datetime.Instant

@Entity(tableName = "availabilities")
data class AvailabilityEntity(
    @PrimaryKey
    val id: String,
    val userId: String,
    val userNickname: String,
    val userAvatar: String?,
    val startTime: Long,
    val endTime: Long,
    val locationType: String,
    val locationName: String?,
    val latitude: Double?,
    val longitude: Double?,
    val status: String,
    val circleIds: String, // JSON array
    val createdAt: Long,
    val isMine: Boolean = false,
)

fun AvailabilityEntity.toDomain(): Availability = Availability(
    id = id,
    user = UserProfile(
        id = userId,
        nickname = userNickname,
        avatar = userAvatar,
    ),
    startTime = Instant.fromEpochMilliseconds(startTime),
    endTime = Instant.fromEpochMilliseconds(endTime),
    location = Location(
        type = LocationType.valueOf(locationType),
        name = locationName,
        latitude = latitude,
        longitude = longitude,
    ),
    status = AvailabilityStatus.valueOf(status),
    circleIds = circleIds.split(",").filter { it.isNotBlank() },
    createdAt = Instant.fromEpochMilliseconds(createdAt),
)

fun Availability.toEntity(isMine: Boolean = false): AvailabilityEntity = AvailabilityEntity(
    id = id,
    userId = user.id,
    userNickname = user.nickname,
    userAvatar = user.avatar,
    startTime = startTime.toEpochMilliseconds(),
    endTime = endTime.toEpochMilliseconds(),
    locationType = location.type.name,
    locationName = location.name,
    latitude = location.latitude,
    longitude = location.longitude,
    status = status.name,
    circleIds = circleIds.joinToString(","),
    createdAt = createdAt.toEpochMilliseconds(),
    isMine = isMine,
)
