package com.youkong.core.agent.collector

import android.content.Context
import android.hardware.Sensor
import android.hardware.SensorManager
import com.google.android.gms.location.ActivityRecognition
import com.google.android.gms.location.ActivityRecognitionClient
import com.google.android.gms.location.DetectedActivity
import com.youkong.core.agent.model.MovementData
import com.youkong.core.agent.model.MovementType
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeoutOrNull
import kotlinx.datetime.Clock
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.coroutines.resume

/**
 * 移动数据收集器
 * 使用 Activity Recognition API 获取活动类型
 */
@Singleton
class MovementCollector @Inject constructor(
    @ApplicationContext private val context: Context,
) {
    private val activityRecognitionClient: ActivityRecognitionClient by lazy {
        ActivityRecognition.getClient(context)
    }

    private val sensorManager: SensorManager by lazy {
        context.getSystemService(Context.SENSOR_SERVICE) as SensorManager
    }

    // 缓存最近的活动状态
    @Volatile
    private var lastDetectedActivity: DetectedActivity? = null

    @Volatile
    private var lastActivityTime: Long = 0

    @Volatile
    private var stationaryStartTime: Long = 0

    /**
     * 收集移动数据
     */
    suspend fun collect(): MovementData {
        val now = System.currentTimeMillis()

        // 尝试获取最新活动
        val activity = getLatestActivity()

        val movementType = when (activity?.type) {
            DetectedActivity.STILL -> MovementType.STATIONARY
            DetectedActivity.WALKING -> MovementType.WALKING
            DetectedActivity.RUNNING -> MovementType.RUNNING
            DetectedActivity.ON_BICYCLE -> MovementType.CYCLING
            DetectedActivity.IN_VEHICLE -> MovementType.DRIVING
            DetectedActivity.ON_FOOT -> MovementType.WALKING
            else -> MovementType.UNKNOWN
        }

        val isMoving = movementType != MovementType.STATIONARY && movementType != MovementType.UNKNOWN

        // 计算静止时间
        val stationaryMinutes = if (!isMoving && stationaryStartTime > 0) {
            ((now - stationaryStartTime) / 60000).toInt()
        } else {
            if (!isMoving) {
                stationaryStartTime = now
            }
            null
        }

        // 更新静止开始时间
        if (isMoving) {
            stationaryStartTime = 0
        }

        return MovementData(
            isMoving = isMoving,
            movementType = movementType,
            stepsToday = getStepsToday(),
            stepsLastHour = null, // 需要更复杂的实现
            stationaryMinutes = stationaryMinutes,
            timestamp = Clock.System.now(),
        )
    }

    /**
     * 获取最新的活动类型
     */
    private suspend fun getLatestActivity(): DetectedActivity? {
        // 如果缓存的活动数据不超过 5 分钟，直接返回
        val now = System.currentTimeMillis()
        if (lastDetectedActivity != null && now - lastActivityTime < 5 * 60 * 1000) {
            return lastDetectedActivity
        }

        // 尝试获取新的活动数据（带超时）
        return withTimeoutOrNull(3000L) {
            try {
                requestActivityUpdate()
            } catch (e: SecurityException) {
                null
            } catch (e: Exception) {
                null
            }
        } ?: lastDetectedActivity
    }

    /**
     * 请求活动更新
     */
    private suspend fun requestActivityUpdate(): DetectedActivity? {
        return suspendCancellableCoroutine { continuation ->
            try {
                // 使用 Activity Transition API 获取当前活动
                // 注意：这是简化实现，实际应该注册 PendingIntent 接收更新
                val task = activityRecognitionClient.requestActivityUpdates(
                    0L, // 立即
                    android.app.PendingIntent.getBroadcast(
                        context,
                        0,
                        android.content.Intent(),
                        android.app.PendingIntent.FLAG_IMMUTABLE or android.app.PendingIntent.FLAG_NO_CREATE
                    )
                )

                // 由于 Activity Recognition 是异步的，这里返回缓存值
                if (continuation.isActive) {
                    continuation.resume(lastDetectedActivity)
                }
            } catch (e: Exception) {
                if (continuation.isActive) {
                    continuation.resume(null)
                }
            }
        }
    }

    /**
     * 更新检测到的活动（由外部广播接收器调用）
     */
    fun updateDetectedActivity(activity: DetectedActivity) {
        lastDetectedActivity = activity
        lastActivityTime = System.currentTimeMillis()

        if (activity.type == DetectedActivity.STILL) {
            if (stationaryStartTime == 0L) {
                stationaryStartTime = lastActivityTime
            }
        } else {
            stationaryStartTime = 0
        }
    }

    /**
     * 获取今日步数
     * 使用计步器传感器
     */
    private fun getStepsToday(): Int? {
        return try {
            val stepSensor = sensorManager.getDefaultSensor(Sensor.TYPE_STEP_COUNTER)
            if (stepSensor != null) {
                // 计步器传感器返回的是设备启动以来的总步数
                // 需要配合存储来计算今日步数
                // 这里简化处理，返回 null
                null
            } else {
                null
            }
        } catch (e: Exception) {
            null
        }
    }
}
