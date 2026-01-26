package com.youkong.core.data.repository

import com.youkong.core.data.mapper.toDomain
import com.youkong.core.database.dao.CircleDao
import com.youkong.core.database.entity.toDomain
import com.youkong.core.database.entity.toEntity
import com.youkong.core.domain.model.Circle
import com.youkong.core.domain.model.CircleDetail
import com.youkong.core.domain.repository.CircleRepository
import com.youkong.core.network.api.CircleApi
import com.youkong.core.network.model.AddMemberRequest
import com.youkong.core.network.model.CreateCircleRequest
import com.youkong.core.network.model.UpdateCircleRequest
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class CircleRepositoryImpl @Inject constructor(
    private val circleApi: CircleApi,
    private val circleDao: CircleDao,
) : CircleRepository {

    override val myCircles: Flow<List<Circle>> = circleDao.observeAll().map { entities ->
        entities.map { it.toDomain() }
    }

    override suspend fun getMyCircles(): Result<List<Circle>> {
        return try {
            val response = circleApi.getMyCircles()
            val data = response.data
            if (response.isSuccess && data != null) {
                val circles = data.map { it.toDomain() }
                circleDao.deleteAll()
                circleDao.insertAll(circles.map { it.toEntity() })
                Result.success(circles)
            } else {
                // 返回缓存数据
                val cached = circleDao.getAll().map { it.toDomain() }
                if (cached.isNotEmpty()) {
                    Result.success(cached)
                } else {
                    Result.failure(Exception(response.message))
                }
            }
        } catch (e: Exception) {
            // 返回缓存数据
            val cached = circleDao.getAll().map { it.toDomain() }
            if (cached.isNotEmpty()) {
                Result.success(cached)
            } else {
                Result.failure(e)
            }
        }
    }

    override suspend fun getCircle(circleId: String): Result<CircleDetail> {
        return try {
            val response = circleApi.getCircle(circleId)
            val data = response.data
            if (response.isSuccess && data != null) {
                Result.success(data.toDomain())
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun createCircle(name: String, emoji: String, color: String): Result<Circle> {
        return try {
            val response = circleApi.createCircle(CreateCircleRequest(name, emoji, color))
            val data = response.data
            if (response.isSuccess && data != null) {
                val circle = data.toDomain()
                circleDao.insert(circle.toEntity())
                Result.success(circle)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun updateCircle(
        circleId: String,
        name: String?,
        emoji: String?,
        color: String?
    ): Result<Circle> {
        return try {
            val response = circleApi.updateCircle(circleId, UpdateCircleRequest(name, emoji, color))
            val data = response.data
            if (response.isSuccess && data != null) {
                val circle = data.toDomain()
                circleDao.insert(circle.toEntity())
                Result.success(circle)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun deleteCircle(circleId: String): Result<Unit> {
        return try {
            val response = circleApi.deleteCircle(circleId)
            if (response.isSuccess) {
                circleDao.delete(circleId)
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun addMember(circleId: String, userId: String): Result<Unit> {
        return try {
            val response = circleApi.addMember(circleId, AddMemberRequest(userId))
            if (response.isSuccess) {
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun removeMember(circleId: String, userId: String): Result<Unit> {
        return try {
            val response = circleApi.removeMember(circleId, userId)
            if (response.isSuccess) {
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
