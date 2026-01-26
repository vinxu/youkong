package com.youkong.core.agent.worker

import android.content.Context
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import com.youkong.core.agent.collector.DeviceStateCollector
import com.youkong.core.agent.collector.LocationCollector
import com.youkong.core.agent.collector.UsageStatsCollector
import com.youkong.core.agent.model.ActivityType
import com.youkong.core.agent.model.DeviceStateData
import com.youkong.core.agent.model.LocalScreenData
import com.youkong.core.agent.model.LocalLocationData
import com.youkong.core.agent.model.PlaceType
import com.youkong.core.network.api.AgentApi
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.DeviceStateRequest
import com.youkong.core.network.model.LocationDataRequest
import com.youkong.core.network.model.ScreenDataRequest
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import kotlinx.datetime.Clock
import java.util.concurrent.TimeUnit

/**
 * 数据收集 Worker
 * 定期收集屏幕使用和位置数据，并上报到服务器
 */
@HiltWorker
class DataCollectWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted workerParams: WorkerParameters,
    private val usageStatsCollector: UsageStatsCollector,
    private val locationCollector: LocationCollector,
    private val deviceStateCollector: DeviceStateCollector,
    private val agentApi: AgentApi,
) : CoroutineWorker(appContext, workerParams) {

    override suspend fun doWork(): Result {
        return try {
            // 收集数据
            val screenData = usageStatsCollector.collect()
            val locationData = locationCollector.collect()
            val deviceStateData = deviceStateCollector.collect()

            // 转换并上报
            val request = AgentStatusRequest(
                screen = screenData?.let { convertScreenData(it) },
                location = locationData?.let { convertLocationData(it) },
                device = convertDeviceState(deviceStateData),
            )

            // 只有当有数据时才上报
            if (request.screen != null || request.location != null || request.device != null) {
                agentApi.reportStatus(request)
            }

            Result.success()
        } catch (e: Exception) {
            // 如果失败，重试
            if (runAttemptCount < 3) {
                Result.retry()
            } else {
                Result.failure()
            }
        }
    }

    private fun convertScreenData(local: LocalScreenData): ScreenDataRequest {
        val now = Clock.System.now()
        val lastActiveMinutesAgo = local.lastActiveTime?.let {
            ((now - it).inWholeMinutes).toInt()
        } ?: 0

        val sessionDurationMinutes = local.sessionStartTime?.let {
            ((now - it).inWholeMinutes).toInt()
        } ?: 0

        // 简单的活动类型判断（可以根据 currentApp 进一步细化）
        val activityType = when {
            !local.isScreenOn -> ActivityType.IDLE
            local.currentApp?.contains("game", ignoreCase = true) == true -> ActivityType.ENTERTAINMENT
            local.currentApp?.contains("video", ignoreCase = true) == true -> ActivityType.ENTERTAINMENT
            local.currentApp?.contains("music", ignoreCase = true) == true -> ActivityType.ENTERTAINMENT
            local.currentApp?.contains("chat", ignoreCase = true) == true -> ActivityType.COMMUNICATION
            local.currentApp?.contains("message", ignoreCase = true) == true -> ActivityType.COMMUNICATION
            local.currentApp?.contains("wechat", ignoreCase = true) == true -> ActivityType.COMMUNICATION
            else -> ActivityType.PRODUCTIVITY
        }

        return ScreenDataRequest(
            isActive = local.isScreenOn,
            activityType = activityType.name.lowercase(),
            sessionDurationMinutes = sessionDurationMinutes,
            lastActiveMinutesAgo = lastActiveMinutesAgo,
        )
    }

    private fun convertLocationData(local: LocalLocationData): LocationDataRequest {
        // 简单的地点类型判断（实际应用中可以使用地理围栏或 ML 模型）
        // 这里暂时返回 UNKNOWN，后续可以根据用户标记的家/公司位置判断
        return LocationDataRequest(
            placeType = PlaceType.UNKNOWN.name.lowercase(),
            atPlaceSinceMinutes = 0,
        )
    }

    private fun convertDeviceState(data: DeviceStateData): DeviceStateRequest {
        return DeviceStateRequest(
            isDndEnabled = data.isDoNotDisturbEnabled,
            isCharging = data.isCharging,
            batteryLevel = data.batteryLevel,
            isPowerSaveMode = data.isPowerSaveMode,
            isHeadphonesConnected = data.isHeadphonesConnected,
            networkType = data.networkType.name.lowercase(),
            ringerMode = data.ringerMode,
        )
    }

    companion object {
        private const val WORK_NAME = "data_collect_worker"

        /**
         * 调度定期数据收集任务
         * @param workManager WorkManager 实例
         * @param intervalMinutes 收集间隔（分钟），默认 15 分钟
         */
        fun schedule(
            workManager: WorkManager,
            intervalMinutes: Long = 15,
        ) {
            val workRequest = PeriodicWorkRequestBuilder<DataCollectWorker>(
                intervalMinutes,
                TimeUnit.MINUTES
            ).build()

            workManager.enqueueUniquePeriodicWork(
                WORK_NAME,
                ExistingPeriodicWorkPolicy.KEEP,
                workRequest
            )
        }

        /**
         * 取消数据收集任务
         */
        fun cancel(workManager: WorkManager) {
            workManager.cancelUniqueWork(WORK_NAME)
        }

        /**
         * 立即执行一次数据收集（用于权限授予后立即上报）
         */
        fun runOnce(workManager: WorkManager) {
            val workRequest = OneTimeWorkRequestBuilder<DataCollectWorker>().build()
            workManager.enqueue(workRequest)
        }
    }
}
