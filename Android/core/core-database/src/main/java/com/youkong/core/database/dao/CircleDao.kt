package com.youkong.core.database.dao

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import com.youkong.core.database.entity.CircleEntity
import kotlinx.coroutines.flow.Flow

@Dao
interface CircleDao {

    @Query("SELECT * FROM circles ORDER BY createdAt DESC")
    fun observeAll(): Flow<List<CircleEntity>>

    @Query("SELECT * FROM circles ORDER BY createdAt DESC")
    suspend fun getAll(): List<CircleEntity>

    @Query("SELECT * FROM circles WHERE id = :circleId")
    suspend fun getCircle(circleId: String): CircleEntity?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(circle: CircleEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAll(circles: List<CircleEntity>)

    @Query("DELETE FROM circles WHERE id = :circleId")
    suspend fun delete(circleId: String)

    @Query("DELETE FROM circles")
    suspend fun deleteAll()
}
