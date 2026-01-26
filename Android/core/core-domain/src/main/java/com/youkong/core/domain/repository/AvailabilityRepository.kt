package com.youkong.core.domain.repository

import com.youkong.core.domain.model.Availability
import com.youkong.core.domain.model.LocationType
import kotlinx.coroutines.flow.Flow
import kotlinx.datetime.Instant

interface AvailabilityRepository {
    val friendsAvailabilities: Flow<List<Availability>>
    val myAvailabilities: Flow<List<Availability>>

    suspend fun getFriendsAvailabilities(): Result<List<Availability>>
    suspend fun getMyAvailabilities(): Result<List<Availability>>

    suspend fun createAvailability(
        startTime: Instant,
        endTime: Instant,
        locationType: LocationType,
        locationName: String?,
        latitude: Double?,
        longitude: Double?,
        circleIds: List<String>,
    ): Result<Availability>

    suspend fun cancelAvailability(availabilityId: String): Result<Unit>
}
