package com.youkong.core.agent.worker

import android.content.Context
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import com.youkong.core.network.api.FriendApi
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import java.util.concurrent.TimeUnit

/**
 * 状态同步 Worker
 * 定期从服务器同步好友的有空概率数据
 */
@HiltWorker
class StatusSyncWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted workerParams: WorkerParameters,
    private val friendApi: FriendApi,
) : CoroutineWorker(appContext, workerParams) {

    override suspend fun doWork(): Result {
        return try {
            // 同步好友有空概率数据
            friendApi.getFriendsProbability()
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

    companion object {
        private const val WORK_NAME = "status_sync_worker"

        /**
         * 调度定期状态同步任务
         * @param workManager WorkManager 实例
         * @param intervalHours 同步间隔（小时），默认 1 小时
         */
        fun schedule(
            workManager: WorkManager,
            intervalHours: Long = 1,
        ) {
            val workRequest = PeriodicWorkRequestBuilder<StatusSyncWorker>(
                intervalHours,
                TimeUnit.HOURS
            ).build()

            workManager.enqueueUniquePeriodicWork(
                WORK_NAME,
                ExistingPeriodicWorkPolicy.KEEP,
                workRequest
            )
        }

        /**
         * 取消状态同步任务
         */
        fun cancel(workManager: WorkManager) {
            workManager.cancelUniqueWork(WORK_NAME)
        }
    }
}
