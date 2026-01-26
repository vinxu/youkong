package com.youkong.core.domain.repository

import com.youkong.core.domain.model.FriendWithProbability

/**
 * Agent 数据仓库接口
 */
interface AgentRepository {

    /**
     * 上报状态
     */
    suspend fun reportStatus(
        isActive: Boolean,
        activityType: String,
        sessionDurationMinutes: Int,
        lastActiveMinutesAgo: Int,
        placeType: String?,
        atPlaceSinceMinutes: Int?,
    ): Result<Unit>

    /**
     * 获取好友有空概率列表
     */
    suspend fun getFriendsProbability(): Result<List<FriendWithProbability>>
}
