package com.youkong.core.agent.collector

import android.content.Context
import android.hardware.Sensor
import android.hardware.SensorEvent
import android.hardware.SensorEventListener
import android.hardware.SensorManager
import com.google.android.gms.location.DetectedActivity
import com.youkong.core.agent.model.MovementData
import com.youkong.core.agent.model.MovementType
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeoutOrNull
import kotlinx.datetime.Clock
import java.util.Calendar
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
     * 请求活动更新 - 使用步数传感器推断运动状态
     * Activity Recognition API 需要 BroadcastReceiver，这里用加速度传感器做轻量检测
     */
    private suspend fun requestActivityUpdate(): DetectedActivity? {
        val stepSensor = sensorManager.getDefaultSensor(Sensor.TYPE_STEP_DETECTOR)
        if (stepSensor != null) {
            // 用步数检测器：3秒内有步数事件 → walking，否则 → stationary
            val hasSteps = withTimeoutOrNull(3000L) {
                suspendCancellableCoroutine<Boolean> { cont ->
                    val listener = object : SensorEventListener {
                        override fun onSensorChanged(event: SensorEvent) {
                            sensorManager.unregisterListener(this)
                            if (cont.isActive) cont.resume(true)
                        }
                        override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) {}
                    }
                    sensorManager.registerListener(listener, stepSensor, SensorManager.SENSOR_DELAY_FASTEST)
                    cont.invokeOnCancellation { sensorManager.unregisterListener(listener) }
                }
            } ?: false

            val activity = if (hasSteps) {
                DetectedActivity(DetectedActivity.WALKING, 80)
            } else {
                DetectedActivity(DetectedActivity.STILL, 80)
            }
            updateDetectedActivity(activity)
            return activity
        }

        // 回退：用加速度传感器判断
        val accelerometer = sensorManager.getDefaultSensor(Sensor.TYPE_ACCELEROMETER)
        if (accelerometer != null) {
            val accelMag = withTimeoutOrNull(2000L) {
                suspendCancellableCoroutine<Float?> { cont ->
                    val listener = object : SensorEventListener {
                        override fun onSensorChanged(event: SensorEvent) {
                            sensorManager.unregisterListener(this)
                            val mag = Math.sqrt(
                                (event.values[0] * event.values[0] +
                                 event.values[1] * event.values[1] +
                                 event.values[2] * event.values[2]).toDouble()
                            ).toFloat()
                            if (cont.isActive) cont.resume(mag)
                        }
                        override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) {}
                    }
                    sensorManager.registerListener(listener, accelerometer, SensorManager.SENSOR_DELAY_NORMAL)
                    cont.invokeOnCancellation { sensorManager.unregisterListener(listener) }
                }
            }

            if (accelMag != null) {
                // 重力约 9.8，超过 11 认为在运动中
                val activity = if (accelMag > 11f) {
                    DetectedActivity(DetectedActivity.WALKING, 60)
                } else {
                    DetectedActivity(DetectedActivity.STILL, 60)
                }
                updateDetectedActivity(activity)
                return activity
            }
        }

        return lastDetectedActivity
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
     * TYPE_STEP_COUNTER 返回开机以来总步数，需减去今日零点的基线值
     */
    private suspend fun getStepsToday(): Int? {
        val stepSensor = sensorManager.getDefaultSensor(Sensor.TYPE_STEP_COUNTER) ?: return null

        // 读取当前总步数（注册监听器获取一次读数）
        val totalSteps = withTimeoutOrNull(3000L) {
            suspendCancellableCoroutine<Float?> { cont ->
                val listener = object : SensorEventListener {
                    override fun onSensorChanged(event: SensorEvent) {
                        sensorManager.unregisterListener(this)
                        if (cont.isActive) cont.resume(event.values[0])
                    }
                    override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) {}
                }
                sensorManager.registerListener(listener, stepSensor, SensorManager.SENSOR_DELAY_FASTEST)
                cont.invokeOnCancellation { sensorManager.unregisterListener(listener) }
            }
        }?.toInt() ?: return null

        // 用 SharedPreferences 存今日基线
        val prefs = context.getSharedPreferences("youkong_steps", Context.MODE_PRIVATE)
        val today = todayDateKey()
        val savedDate = prefs.getString("date", null)
        val baseline = prefs.getInt("baseline", -1)

        return if (savedDate == today && baseline >= 0) {
            (totalSteps - baseline).coerceAtLeast(0)
        } else {
            // 新的一天或首次，将当前值设为基线
            prefs.edit().putString("date", today).putInt("baseline", totalSteps).apply()
            0
        }
    }

    private fun todayDateKey(): String {
        val cal = Calendar.getInstance()
        return "${cal.get(Calendar.YEAR)}-${cal.get(Calendar.MONTH) + 1}-${cal.get(Calendar.DAY_OF_MONTH)}"
    }
}
