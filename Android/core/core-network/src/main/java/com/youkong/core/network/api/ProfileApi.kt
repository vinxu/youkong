package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import retrofit2.http.Body
import retrofit2.http.POST

/**
 * 用户画像 API
 */
interface ProfileApi {

    /**
     * 获取引导流程的状态选项
     */
    @POST("agent/onboarding-status-options")
    suspend fun getOnboardingStatusOptions(
        @Body request: OnboardingStatusOptionsRequest
    ): ApiResponse<OnboardingStatusOptionsResponse>

    /**
     * 保存简化的用户画像
     */
    @POST("profile/simple")
    suspend fun saveSimpleProfile(
        @Body request: SimpleProfileRequest
    ): ApiResponse<SimpleProfileResponse>
}

/**
 * 引导状态选项请求
 */
@Serializable
data class OnboardingStatusOptionsRequest(
    @SerialName("profile_type")
    val profileType: String,
    val city: String? = null  // 城市名称（如"上海"、"北京"），用于增强推理
)

/**
 * 引导状态选项响应
 */
@Serializable
data class OnboardingStatusOptionsResponse(
    val options: List<StatusOptionData>
)

/**
 * 状态选项数据（API 响应）
 */
@Serializable
data class StatusOptionData(
    val emoji: String,
    val status: String
)

/**
 * 简化的用户画像请求
 */
@Serializable
data class SimpleProfileRequest(
    @SerialName("profile_type")
    val profileType: String,
    val gender: String? = null,
    val mbti: String? = null
)

/**
 * 简化的用户画像响应
 */
@Serializable
data class SimpleProfileResponse(
    val message: String? = null,
    @SerialName("profile_type")
    val profileType: String? = null
)
