package com.youkong.core.agent.worker

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.util.Log
import androidx.core.content.ContextCompat
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.youkong.core.agent.collector.CalendarCollector
import com.youkong.core.agent.collector.DeviceStateCollector
import com.youkong.core.agent.collector.LocationCollector
import com.youkong.core.agent.collector.MovementCollector
import com.youkong.core.agent.collector.ScreenUsageCollector
import com.youkong.core.agent.location.PlaceLearner
import com.youkong.core.network.model.ScreenDataRequest
import com.youkong.core.datastore.TokenManager
import com.youkong.core.network.api.AgentApi
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.BatteryDataRequest
import com.youkong.core.network.model.CalendarDataRequest
import com.youkong.core.network.model.ConnectionDataRequest
import com.youkong.core.network.model.DisplayDataRequest
import com.youkong.core.network.model.ExtendedLocationDataRequest
import com.youkong.core.network.model.LocationDataRequest
import com.youkong.core.network.model.ModeDataRequest
import com.youkong.core.network.model.MovementDataRequest
import com.youkong.core.network.model.AmbientLightDataRequest
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import kotlinx.coroutines.flow.first

@HiltWorker
class StatusReportWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted workerParams: WorkerParameters,
    private val agentApi: AgentApi,
    private val tokenManager: TokenManager,
    private val locationCollector: LocationCollector,
    private val calendarCollector: CalendarCollector,
    private val movementCollector: MovementCollector,
    private val deviceStateCollector: DeviceStateCollector,
    private val screenUsageCollector: ScreenUsageCollector,
    private val placeLearner: PlaceLearner,
) : CoroutineWorker(appContext, workerParams) {

    companion object {
        const val TAG = "StatusReportWorker"
        const val WORK_NAME = "status_report_periodic"
    }

    override suspend fun doWork(): Result {
        Log.d(TAG, "=== StatusReportWorker started ===")

        // 1. 未登录则跳过
        val token = tokenManager.accessToken.first()
        if (token.isNullOrBlank()) {
            Log.d(TAG, "Not logged in, skipping")
            return Result.success()
        }

        try {
            // 2. 设备状态（无需权限）
            val deviceState = deviceStateCollector.collect()

            // 3. 位置（需要权限）
            val locationData = if (hasPermission(Manifest.permission.ACCESS_FINE_LOCATION)) {
                try {
                    locationCollector.collect()
                } catch (e: Exception) {
                    Log.w(TAG, "Location failed: ${e.message}")
                    null
                }
            } else null

            // 4. 日历（需要权限）
            val calendarData = if (hasPermission(Manifest.permission.READ_CALENDAR)) {
                try {
                    calendarCollector.collect()
                } catch (e: Exception) {
                    Log.w(TAG, "Calendar failed: ${e.message}")
                    null
                }
            } else null

            // 5. 运动（Android 10+ 需要权限）
            val movementData = if (
                android.os.Build.VERSION.SDK_INT < android.os.Build.VERSION_CODES.Q ||
                hasPermission(Manifest.permission.ACTIVITY_RECOGNITION)
            ) {
                try {
                    movementCollector.collect()
                } catch (e: Exception) {
                    Log.w(TAG, "Movement failed: ${e.message}")
                    null
                }
            } else null

            // 6. 屏幕使用（需要特殊权限）
            val screenData = try {
                screenUsageCollector.collect()
            } catch (e: Exception) {
                Log.w(TAG, "Screen usage failed: ${e.message}")
                null
            }

            // 7. 组装请求
            val placeType = locationData?.let {
                placeLearner.classifyLocation(it.latitude, it.longitude)
            } ?: "unknown"
            Log.d(TAG, "placeType=$placeType, lat=${locationData?.latitude}, lng=${locationData?.longitude}, placeName=${locationData?.placeName}")

            val request = AgentStatusRequest(
                screen = screenData,
                location = locationData?.let {
                    LocationDataRequest(
                        placeType = placeType,
                        atPlaceSinceMinutes = 0,
                        city = it.city,
                    )
                },
                extendedLocation = locationData?.let {
                    ExtendedLocationDataRequest(
                        placeType = placeType,
                        placeName = it.placeName,
                        atPlaceSinceMinutes = 0,
                        latitude = it.latitude,
                        longitude = it.longitude,
                    )
                },
                battery = BatteryDataRequest(
                    batteryLevel = deviceState.batteryLevel,
                    batteryState = if (deviceState.isCharging) "charging" else "unplugged",
                    isCharging = deviceState.isCharging,
                ),
                mode = ModeDataRequest(
                    isLowPowerMode = deviceState.isPowerSaveMode,
                    isFocusModeOn = deviceState.isDoNotDisturbEnabled,
                ),
                connection = ConnectionDataRequest(
                    isHeadphonesConnected = deviceState.isHeadphonesConnected,
                    networkType = deviceState.networkType.name,
                    wifiSSID = deviceState.wifiSSID,
                    bluetoothDeviceType = deviceState.bluetoothDeviceType,
                ),
                display = DisplayDataRequest(
                    screenBrightness = deviceState.screenBrightness,
                ),
                calendar = calendarData?.let {
                    CalendarDataRequest(
                        hasCurrentEvent = it.hasCurrentEvent,
                        currentEventTitle = it.currentEventTitle,
                        eventEndMinutes = it.eventEndMinutes,
                        nextEventInMinutes = it.nextEventInMinutes,
                        todayRemainingCount = it.todayRemainingCount,
                    )
                },
                movement = movementData?.let {
                    MovementDataRequest(
                        isMoving = it.isMoving,
                        movementType = it.movementType.name.lowercase(),
                        stepsToday = it.stepsToday,
                        stepsLastHour = it.stepsLastHour,
                        stationaryMinutes = it.stationaryMinutes,
                    )
                },
                ambientLight = deviceState.ambientLightLux?.let {
                    AmbientLightDataRequest(
                        lux = it,
                        environment = classifyLightEnvironment(it),
                    )
                },
            )

            // 8. 上报
            Log.d(TAG, "Reporting: loc=${locationData != null} cal=${calendarData != null} mov=${movementData != null} scr=${screenData != null}")
            agentApi.reportStatus(request)
            Log.d(TAG, "=== StatusReportWorker success ===")

            return Result.success()
        } catch (e: Exception) {
            Log.e(TAG, "Failed: ${e.message}", e)
            return Result.success() // 不用 retry，下个 15 分钟周期自然重试
        }
    }

    private fun hasPermission(permission: String): Boolean {
        return ContextCompat.checkSelfPermission(
            applicationContext, permission
        ) == PackageManager.PERMISSION_GRANTED
    }

    private fun classifyLightEnvironment(lux: Float): String {
        return when {
            lux < 10 -> "dark"
            lux < 50 -> "dim"
            lux < 300 -> "indoor"
            lux < 1000 -> "bright"
            lux < 10000 -> "outdoor"
            else -> "sunlight"
        }
    }
}
