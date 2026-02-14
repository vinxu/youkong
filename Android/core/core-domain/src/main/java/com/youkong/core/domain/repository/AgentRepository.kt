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
        city: String? = null,
    ): Result<Unit>

    /**
     * 获取好友有空概率列表
     */
    suspend fun getFriendsProbability(): Result<List<FriendWithProbability>>

    /**
     * AI 推断当前状态（V1 同步版）
     */
    suspend fun inferStatus(): Result<StatusInferenceResult>

    /**
     * AI 推断当前状态（V3 同步版）
     * @param sensorData 传感器数据（由调用方构建 AgentStatusRequest，通过 Any 传递避免 domain 依赖 network）
     */
    suspend fun inferStatusV3(sensorData: Any): Result<V3InferenceResult>

    /**
     * AI 推断 V3 用户选择
     */
    suspend fun inferStatusV3Respond(sessionId: String, selectedIndex: Int): Result<V3InferenceResult>

    /**
     * 提交状态反馈（用户修正）
     */
    suspend fun submitStatusFeedback(feedback: StatusFeedback): Result<Unit>

    /**
     * 上传 GIF 到 COS（STS 直传，返回 COS URL）
     */
    suspend fun uploadGifToCOS(gifData: ByteArray): Result<String>
}

/**
 * 当下状态推断结果
 */
data class StatusInferenceResult(
    val emoji: String,
    val activity: String,
    val place: String? = null,
    val isAvailable: Boolean = false,
    val durationHint: String? = null,
    val confidence: String = "medium",
    val inferredAt: Long = 0,
    val reasoning: String? = null,
    val gifUrl: String? = null,
    val gifSmallUrl: String? = null,
    val giphyQuery: String? = null,
)

/**
 * 状态反馈（用户修正）
 */
data class StatusFeedback(
    val originalEmoji: String? = null,
    val originalActivity: String? = null,
    val correctedEmoji: String,
    val correctedActivity: String,
    val correctedPlace: String? = null,
    val correctedIsAvailable: Boolean = false,
    val gifUrl: String? = null,
    val giphyQuery: String? = null,
    val useGif: Boolean = false,
)

/**
 * V3 推断统一结果（包含 completed 和 awaiting_choice 两种阶段）
 */
data class V3InferenceResult(
    val phase: String,                                  // "completed" | "awaiting_choice"
    val result: StatusInferenceResult? = null,          // phase=completed 时有值
    val sessionId: String? = null,                      // phase=awaiting_choice 时有值
    val question: String? = null,
    val options: List<V3InferenceOption>? = null,
    val defaultIndex: Int? = null,
)

data class V3InferenceOption(
    val index: Int,
    val emoji: String,
    val activity: String,
    val reason: String? = null,
)
