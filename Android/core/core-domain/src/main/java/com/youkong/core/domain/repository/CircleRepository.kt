package com.youkong.core.domain.repository

import com.youkong.core.domain.model.Circle
import com.youkong.core.domain.model.CircleDetail
import kotlinx.coroutines.flow.Flow

interface CircleRepository {
    val myCircles: Flow<List<Circle>>

    suspend fun getMyCircles(): Result<List<Circle>>
    suspend fun getCircle(circleId: String): Result<CircleDetail>
    suspend fun createCircle(name: String, emoji: String, color: String): Result<Circle>
    suspend fun updateCircle(circleId: String, name: String?, emoji: String?, color: String?): Result<Circle>
    suspend fun deleteCircle(circleId: String): Result<Unit>
    suspend fun addMember(circleId: String, userId: String): Result<Unit>
    suspend fun removeMember(circleId: String, userId: String): Result<Unit>
}
