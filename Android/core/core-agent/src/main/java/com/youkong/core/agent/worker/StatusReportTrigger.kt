package com.youkong.core.agent.worker

import android.util.Log
import androidx.work.Constraints
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import java.util.concurrent.atomic.AtomicLong
import javax.inject.Inject
import javax.inject.Singleton

/**
 * 状态上报触发器（带 60s 冷却）
 * 供 Activity 生命周期、ViewModel 等多处调用
 */
@Singleton
class StatusReportTrigger @Inject constructor(
    private val workManager: WorkManager,
) {
    private val lastTriggerTime = AtomicLong(0)
    private val cooldownMs = 60_000L // 60 秒冷却

    companion object {
        private const val TAG = "StatusReportTrigger"
    }

    /**
     * 带冷却的触发（前台切换、Grid 加载等场景）
     */
    fun triggerIfNeeded() {
        val now = System.currentTimeMillis()
        val last = lastTriggerTime.get()
        if (now - last < cooldownMs) {
            Log.d(TAG, "Skipped (cooldown ${(cooldownMs - (now - last)) / 1000}s)")
            return
        }
        if (!lastTriggerTime.compareAndSet(last, now)) return
        enqueueOneTime()
    }

    /**
     * 立即触发（登录时）
     */
    fun triggerNow() {
        lastTriggerTime.set(System.currentTimeMillis())
        enqueueOneTime()
    }

    private fun enqueueOneTime() {
        val work = OneTimeWorkRequestBuilder<StatusReportWorker>()
            .setConstraints(
                Constraints.Builder()
                    .setRequiredNetworkType(NetworkType.CONNECTED)
                    .build()
            )
            .build()
        workManager.enqueue(work)
        Log.d(TAG, "OneTimeWork enqueued")
    }
}
