package com.youkong.core.database.dao

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import com.youkong.core.database.entity.AvailabilityEntity
import kotlinx.coroutines.flow.Flow

@Dao
interface AvailabilityDao {

    @Query("SELECT * FROM availabilities WHERE isMine = 0 AND status = 'ACTIVE' ORDER BY startTime ASC")
    fun observeFriendsAvailabilities(): Flow<List<AvailabilityEntity>>

    @Query("SELECT * FROM availabilities WHERE isMine = 1 ORDER BY startTime ASC")
    fun observeMyAvailabilities(): Flow<List<AvailabilityEntity>>

    @Query("SELECT * FROM availabilities WHERE isMine = 0 AND status = 'ACTIVE' ORDER BY startTime ASC")
    suspend fun getFriendsAvailabilities(): List<AvailabilityEntity>

    @Query("SELECT * FROM availabilities WHERE isMine = 1 ORDER BY startTime ASC")
    suspend fun getMyAvailabilities(): List<AvailabilityEntity>

    @Query("SELECT * FROM availabilities WHERE id = :availabilityId")
    suspend fun getAvailability(availabilityId: String): AvailabilityEntity?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(availability: AvailabilityEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAll(availabilities: List<AvailabilityEntity>)

    @Query("DELETE FROM availabilities WHERE id = :availabilityId")
    suspend fun delete(availabilityId: String)

    @Query("DELETE FROM availabilities WHERE isMine = 0")
    suspend fun deleteFriendsAvailabilities()

    @Query("DELETE FROM availabilities WHERE isMine = 1")
    suspend fun deleteMyAvailabilities()

    @Query("DELETE FROM availabilities")
    suspend fun deleteAll()
}
