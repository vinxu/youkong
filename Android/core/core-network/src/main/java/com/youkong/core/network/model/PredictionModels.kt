package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * AI 状态推测 - 开始推测请求
 */
@Serializable
data class StartPredictionRequest(
    @SerialName("target_date") val targetDate: String? = null,
)

/**
 * AI 状态推测 - 推测任务
 */
@Serializable
data class PredictionTask(
    val id: String,
    val status: String,
    @SerialName("start_time") val startTime: String? = null,
    @SerialName("end_time") val endTime: String? = null,
    @SerialName("target_date") val targetDate: String? = null,
    @SerialName("predicted_items") val predictedItems: List<ScheduleItem>? = null,
    @SerialName("existing_items") val existingItems: List<ScheduleItem>? = null,
    @SerialName("merged_preview") val mergedPreview: List<ScheduleItem>? = null,
    @SerialName("reasoning_content") val reasoningContent: String? = null,
    @SerialName("reasoning_steps") val reasoningSteps: List<String>? = null,
    @SerialName("error_message") val errorMessage: String? = null,
    val progress: Int = 0,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("completed_at") val completedAt: String? = null,
    @SerialName("confirmed_at") val confirmedAt: String? = null,
)

/**
 * AI 状态推测 - 确认推测请求
 */
@Serializable
data class ConfirmPredictionRequest(
    @SerialName("modified_items") val modifiedItems: List<ScheduleItem>? = null,
)
