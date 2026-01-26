package com.youkong.core.data.repository

import com.youkong.core.data.mapper.toDomain
import com.youkong.core.database.dao.AvailabilityDao
import com.youkong.core.database.entity.toDomain
import com.youkong.core.database.entity.toEntity
import com.youkong.core.domain.model.Availability
import com.youkong.core.domain.model.LocationType
import com.youkong.core.domain.repository.AvailabilityRepository
import com.youkong.core.network.api.AvailabilityApi
import com.youkong.core.network.model.CreateAvailabilityRequest
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import kotlinx.datetime.Instant
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AvailabilityRepositoryImpl @Inject constructor(
    private val availabilityApi: AvailabilityApi,
    private val availabilityDao: AvailabilityDao,
) : AvailabilityRepository {

    override val friendsAvailabilities: Flow<List<Availability>> =
        availabilityDao.observeFriendsAvailabilities().map { entities ->
            entities.map { it.toDomain() }
        }

    override val myAvailabilities: Flow<List<Availability>> =
        availabilityDao.observeMyAvailabilities().map { entities ->
            entities.map { it.toDomain() }
        }

    override suspend fun getFriendsAvailabilities(): Result<List<Availability>> {
        return try {
            val response = availabilityApi.getFriendsAvailabilities()
            val data = response.data
            if (response.isSuccess && data != null) {
                val availabilities = data.map { it.toDomain() }
                availabilityDao.deleteFriendsAvailabilities()
                availabilityDao.insertAll(availabilities.map { it.toEntity(isMine = false) })
                Result.success(availabilities)
            } else {
                // 返回缓存
                val cached = availabilityDao.getFriendsAvailabilities().map { it.toDomain() }
                if (cached.isNotEmpty()) {
                    Result.success(cached)
                } else {
                    Result.failure(Exception(response.message))
                }
            }
        } catch (e: Exception) {
            val cached = availabilityDao.getFriendsAvailabilities().map { it.toDomain() }
            if (cached.isNotEmpty()) {
                Result.success(cached)
            } else {
                Result.failure(e)
            }
        }
    }

    override suspend fun getMyAvailabilities(): Result<List<Availability>> {
        return try {
            val response = availabilityApi.getMyAvailabilities()
            val data = response.data
            if (response.isSuccess && data != null) {
                val availabilities = data.map { it.toDomain() }
                availabilityDao.deleteMyAvailabilities()
                availabilityDao.insertAll(availabilities.map { it.toEntity(isMine = true) })
                Result.success(availabilities)
            } else {
                val cached = availabilityDao.getMyAvailabilities().map { it.toDomain() }
                if (cached.isNotEmpty()) {
                    Result.success(cached)
                } else {
                    Result.failure(Exception(response.message))
                }
            }
        } catch (e: Exception) {
            val cached = availabilityDao.getMyAvailabilities().map { it.toDomain() }
            if (cached.isNotEmpty()) {
                Result.success(cached)
            } else {
                Result.failure(e)
            }
        }
    }

    override suspend fun createAvailability(
        startTime: Instant,
        endTime: Instant,
        locationType: LocationType,
        locationName: String?,
        latitude: Double?,
        longitude: Double?,
        circleIds: List<String>,
    ): Result<Availability> {
        return try {
            val response = availabilityApi.createAvailability(
                CreateAvailabilityRequest(
                    startTime = startTime.toString(),
                    endTime = endTime.toString(),
                    locationType = locationType.name,
                    locationName = locationName,
                    latitude = latitude,
                    longitude = longitude,
                    circleIds = circleIds,
                )
            )
            val data = response.data
            if (response.isSuccess && data != null) {
                val availability = data.toDomain()
                availabilityDao.insert(availability.toEntity(isMine = true))
                Result.success(availability)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun cancelAvailability(availabilityId: String): Result<Unit> {
        return try {
            val response = availabilityApi.cancelAvailability(availabilityId)
            if (response.isSuccess) {
                availabilityDao.delete(availabilityId)
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
