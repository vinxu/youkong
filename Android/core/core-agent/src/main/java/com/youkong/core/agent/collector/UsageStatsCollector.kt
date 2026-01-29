package com.youkong.core.agent.collector

import android.app.usage.UsageEvents
import android.app.usage.UsageStatsManager
import android.content.Context
import android.content.pm.PackageManager
import com.youkong.core.agent.model.AppUsageInfo
import com.youkong.core.agent.model.AppUsageSession
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
        // 用于累计每个 App 的使用时长
        val appUsageTimes = mutableMapOf<String, Long>()
        // 用于记录每个 App 的使用时段列表
        val appUsageSessions = mutableMapOf<String, MutableList<AppUsageSession>>()

        while (events.hasNextEvent()) {
            events.getNextEvent(event)

            when (event.eventType) {
                UsageEvents.Event.ACTIVITY_RESUMED -> {
                    // 应用进入前台
                    foregroundStartTimes[event.packageName] = event.timeStamp
                    currentApp = event.packageName
                    lastActiveTime = event.timeStamp
                    isScreenOn = true
                    // 如果 session 还没开始，记录开始时间
                    if (sessionStartTime == null) {
                        sessionStartTime = event.timeStamp
                    }
                }
                UsageEvents.Event.ACTIVITY_PAUSED -> {
                    // 应用离开前台
                    foregroundStartTimes[event.packageName]?.let { startTime ->
                        val duration = event.timeStamp - startTime
                        totalForegroundTime += duration
                        // 累计到 App 使用时长
                        appUsageTimes[event.packageName] =
                            (appUsageTimes[event.packageName] ?: 0L) + duration
                        // 记录使用时段
                        appUsageSessions.getOrPut(event.packageName) { mutableListOf() }
                            .add(AppUsageSession(startTime, event.timeStamp, duration))
                    }
                    foregroundStartTimes.remove(event.packageName)
                }
                UsageEvents.Event.SCREEN_NON_INTERACTIVE -> {
                    // 屏幕关闭 - 重置 session
                    isScreenOn = false
                    sessionStartTime = null
                    // 结算所有正在前台的应用时间
                    foregroundStartTimes.forEach { (pkg, startTime) ->
                        val duration = event.timeStamp - startTime
                        totalForegroundTime += duration
                        appUsageTimes[pkg] = (appUsageTimes[pkg] ?: 0L) + duration
                        // 记录使用时段
                        appUsageSessions.getOrPut(pkg) { mutableListOf() }
                            .add(AppUsageSession(startTime, event.timeStamp, duration))
                    }
                    foregroundStartTimes.clear()
                }
                UsageEvents.Event.SCREEN_INTERACTIVE -> {
                    // 屏幕打开 - 开始新的 session
                    isScreenOn = true
                    sessionStartTime = event.timeStamp
                }
            }
        }

        // 如果当前有应用在前台，结算到当前时间
        foregroundStartTimes.forEach { (pkg, startTime) ->
            val duration = now - startTime
            totalForegroundTime += duration
            appUsageTimes[pkg] = (appUsageTimes[pkg] ?: 0L) + duration
            // 记录使用时段（结束时间为当前时间，表示正在使用中）
            appUsageSessions.getOrPut(pkg) { mutableListOf() }
                .add(AppUsageSession(startTime, now, duration))
        }

        // 转换为 AppUsageInfo 列表，按使用时长降序排序，取前 20 个
        val appUsageList = appUsageTimes
            .filter { it.value > 60000 } // 只显示使用超过 1 分钟的 App
            .map { (packageName, usageTime) ->
                // 获取该 App 的使用时段，过滤掉小于 30 秒的时段，按开始时间排序
                val sessions = appUsageSessions[packageName]
                    ?.filter { it.durationMillis >= 30000 }
                    ?.sortedBy { it.startTime }
                    ?: emptyList()
                AppUsageInfo(
                    packageName = packageName,
                    appName = getAppName(packageName),
                    usageTimeMillis = usageTime,
                    sessions = sessions,
                )
            }
            .sortedByDescending { it.usageTimeMillis }
            .take(20)

        return LocalScreenData(
            isScreenOn = isScreenOn,
            lastActiveTime = lastActiveTime?.let { Instant.fromEpochMilliseconds(it) },
            currentApp = currentApp,
            sessionStartTime = sessionStartTime?.let { Instant.fromEpochMilliseconds(it) },
            totalScreenTimeToday = totalForegroundTime,
            appUsageList = appUsageList,
        )
    }

    /**
     * 获取 App 显示名称
     */
    private fun getAppName(packageName: String): String {
        return try {
            val packageManager = context.packageManager
            val appInfo = packageManager.getApplicationInfo(packageName, 0)
            packageManager.getApplicationLabel(appInfo).toString()
        } catch (e: PackageManager.NameNotFoundException) {
            // 如果找不到，返回包名的最后一部分
            packageName.substringAfterLast('.')
        }
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
