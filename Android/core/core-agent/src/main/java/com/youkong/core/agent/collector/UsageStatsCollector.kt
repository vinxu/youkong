package com.youkong.core.agent.collector

import android.app.usage.UsageEvents
import android.app.usage.UsageStatsManager
import android.content.Context
import com.youkong.core.agent.model.LocalScreenData
import com.youkong.core.permission.PermissionManager
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import java.util.Calendar
import javax.inject.Inject
import javax.inject.Singleton

/**
 * 屏幕使用数据收集器
 * 使用 UsageStatsManager 收集应用使用情况
 */
@Singleton
class UsageStatsCollector @Inject constructor(
    @ApplicationContext private val context: Context,
    private val permissionManager: PermissionManager,
) : ScreenDataCollector {

    private val usageStatsManager: UsageStatsManager by lazy {
        context.getSystemService(Context.USAGE_STATS_SERVICE) as UsageStatsManager
    }

    override suspend fun hasPermission(): Boolean {
        return permissionManager.hasUsageStatsPermission()
    }

    override suspend fun collect(): LocalScreenData? {
        if (!hasPermission()) return null

        val now = System.currentTimeMillis()
        val todayStart = getTodayStartMillis()

        // 获取今天的使用事件
        val events = usageStatsManager.queryEvents(todayStart, now)
        val event = UsageEvents.Event()

        var lastActiveTime: Long? = null
        var currentApp: String? = null
        var sessionStartTime: Long? = null
        var isScreenOn = false
        var totalForegroundTime = 0L

        // 用于计算前台时间的 map: packageName -> lastForegroundTime
        val foregroundStartTimes = mutableMapOf<String, Long>()

        while (events.hasNextEvent()) {
            events.getNextEvent(event)

            when (event.eventType) {
                UsageEvents.Event.ACTIVITY_RESUMED -> {
                    // 应用进入前台
                    foregroundStartTimes[event.packageName] = event.timeStamp
                    currentApp = event.packageName
                    lastActiveTime = event.timeStamp
                    isScreenOn = true
                    if (sessionStartTime == null) {
                        sessionStartTime = event.timeStamp
                    }
                }
                UsageEvents.Event.ACTIVITY_PAUSED -> {
                    // 应用离开前台
                    foregroundStartTimes[event.packageName]?.let { startTime ->
                        totalForegroundTime += event.timeStamp - startTime
                    }
                    foregroundStartTimes.remove(event.packageName)
                }
                UsageEvents.Event.SCREEN_NON_INTERACTIVE -> {
                    // 屏幕关闭
                    isScreenOn = false
                    sessionStartTime = null
                    // 结算所有正在前台的应用时间
                    foregroundStartTimes.forEach { (_, startTime) ->
                        totalForegroundTime += event.timeStamp - startTime
                    }
                    foregroundStartTimes.clear()
                }
                UsageEvents.Event.SCREEN_INTERACTIVE -> {
                    // 屏幕打开
                    isScreenOn = true
                    sessionStartTime = event.timeStamp
                }
            }
        }

        // 如果当前有应用在前台，结算到当前时间
        foregroundStartTimes.forEach { (_, startTime) ->
            totalForegroundTime += now - startTime
        }

        return LocalScreenData(
            isScreenOn = isScreenOn,
            lastActiveTime = lastActiveTime?.let { Instant.fromEpochMilliseconds(it) },
            currentApp = currentApp,
            sessionStartTime = sessionStartTime?.let { Instant.fromEpochMilliseconds(it) },
            totalScreenTimeToday = totalForegroundTime,
        )
    }

    private fun getTodayStartMillis(): Long {
        val calendar = Calendar.getInstance().apply {
            set(Calendar.HOUR_OF_DAY, 0)
            set(Calendar.MINUTE, 0)
            set(Calendar.SECOND, 0)
            set(Calendar.MILLISECOND, 0)
        }
        return calendar.timeInMillis
    }
}
