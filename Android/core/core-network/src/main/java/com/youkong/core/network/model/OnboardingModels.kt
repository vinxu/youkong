package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * 状态选项（Onboarding + 默认选项使用）
 */
@Serializable
data class StatusOption(
    val emoji: String,
    val status: String,
    @SerialName("gif_url")
    val gifUrl: String? = null,
)

/**
 * 选择状态请求
 */
@Serializable
data class SelectStatusRequest(
    val emoji: String,
    val status: String,
    @SerialName("device_data")
    val deviceData: AgentStatusRequest? = null,
    @SerialName("gif_url")
    val gifUrl: String? = null,
    @SerialName("giphy_query")
    val giphyQuery: String? = null,
)

/**
 * 选择状态响应
 */
@Serializable
data class SelectStatusResponse(
    val success: Boolean = true,
    val message: String? = null
)
