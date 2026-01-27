package com.youkong.core.data.mapper

import com.youkong.core.domain.model.Availability
import com.youkong.core.domain.model.AvailabilityStatus
import com.youkong.core.domain.model.Location
import com.youkong.core.domain.model.LocationType
import com.youkong.core.network.model.AvailabilityResponse
import kotlinx.datetime.Instant

fun AvailabilityResponse.toModel(): Availability = Availability(
    id = id,
    user = user.toModel(),
    startTime = Instant.parse(startTime),
    endTime = Instant.parse(endTime),
    location = Location(
        type = LocationType.valueOf(location.type),
        name = location.name,
        latitude = location.latitude,
        longitude = location.longitude,
    ),
    status = AvailabilityStatus.valueOf(status),
    circleIds = circleIds,
    createdAt = Instant.parse(createdAt),
)
