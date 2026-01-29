package com.youkong.core.agent.collector

import android.content.ContentResolver
import android.content.Context
import android.database.Cursor
import android.provider.CalendarContract
import com.youkong.core.agent.model.CalendarData
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.datetime.Clock
import java.util.Calendar
import javax.inject.Inject
import javax.inject.Singleton

/**
 * 日历数据收集器
 * 收集当前日程、下一个日程、今日剩余日程数
 */
@Singleton
class CalendarCollector @Inject constructor(
    @ApplicationContext private val context: Context,
) {
    private val contentResolver: ContentResolver
        get() = context.contentResolver

    // 敏感词列表，包含这些词的日程标题会被替换
    private val sensitiveWords = listOf(
        "密码", "账号", "薪资", "工资", "面试", "体检", "病",
        "password", "salary", "interview", "medical"
    )

    /**
     * 收集日历数据
     */
    fun collect(): CalendarData? {
        return try {
            val now = System.currentTimeMillis()
            val endOfDay = getEndOfDayMillis()

            val events = queryTodayEvents(now, endOfDay)

            // 查找当前正在进行的日程
            val currentEvent = events.find { it.startTime <= now && it.endTime > now }

            // 查找下一个日程
            val nextEvent = events.find { it.startTime > now }

            // 今日剩余日程数
            val remainingCount = events.count { it.startTime > now }

            CalendarData(
                hasCurrentEvent = currentEvent != null,
                currentEventTitle = currentEvent?.let { sanitizeTitle(it.title) },
                eventEndMinutes = currentEvent?.let {
                    ((it.endTime - now) / 60000).toInt()
                },
                nextEventInMinutes = nextEvent?.let {
                    ((it.startTime - now) / 60000).toInt()
                },
                todayRemainingCount = remainingCount,
                timestamp = Clock.System.now(),
            )
        } catch (e: SecurityException) {
            // 没有日历权限
            null
        } catch (e: Exception) {
            null
        }
    }

    /**
     * 查询今日日程
     */
    private fun queryTodayEvents(startTime: Long, endTime: Long): List<CalendarEvent> {
        val events = mutableListOf<CalendarEvent>()

        val projection = arrayOf(
            CalendarContract.Events._ID,
            CalendarContract.Events.TITLE,
            CalendarContract.Events.DTSTART,
            CalendarContract.Events.DTEND,
        )

        // 查询实例而不是事件，以处理重复事件
        val instancesProjection = arrayOf(
            CalendarContract.Instances.EVENT_ID,
            CalendarContract.Instances.TITLE,
            CalendarContract.Instances.BEGIN,
            CalendarContract.Instances.END,
        )

        val uri = CalendarContract.Instances.CONTENT_URI.buildUpon()
            .appendPath(startTime.toString())
            .appendPath(endTime.toString())
            .build()

        var cursor: Cursor? = null
        try {
            cursor = contentResolver.query(
                uri,
                instancesProjection,
                null,
                null,
                "${CalendarContract.Instances.BEGIN} ASC"
            )

            cursor?.let {
                val titleIndex = it.getColumnIndex(CalendarContract.Instances.TITLE)
                val beginIndex = it.getColumnIndex(CalendarContract.Instances.BEGIN)
                val endIndex = it.getColumnIndex(CalendarContract.Instances.END)

                while (it.moveToNext()) {
                    val title = if (titleIndex >= 0) it.getString(titleIndex) else null
                    val begin = if (beginIndex >= 0) it.getLong(beginIndex) else 0
                    val end = if (endIndex >= 0) it.getLong(endIndex) else 0

                    if (begin > 0 && end > begin) {
                        events.add(CalendarEvent(title, begin, end))
                    }
                }
            }
        } finally {
            cursor?.close()
        }

        return events
    }

    /**
     * 脱敏处理：移除敏感词
     */
    private fun sanitizeTitle(title: String?): String? {
        if (title == null) return null

        for (word in sensitiveWords) {
            if (title.contains(word, ignoreCase = true)) {
                return "私人事务"
            }
        }

        // 限制长度
        return title.take(20)
    }

    /**
     * 获取今天结束的时间戳
     */
    private fun getEndOfDayMillis(): Long {
        val calendar = Calendar.getInstance()
        calendar.set(Calendar.HOUR_OF_DAY, 23)
        calendar.set(Calendar.MINUTE, 59)
        calendar.set(Calendar.SECOND, 59)
        calendar.set(Calendar.MILLISECOND, 999)
        return calendar.timeInMillis
    }

    /**
     * 日历事件
     */
    private data class CalendarEvent(
        val title: String?,
        val startTime: Long,
        val endTime: Long,
    )
}
