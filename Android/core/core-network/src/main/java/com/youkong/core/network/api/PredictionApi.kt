package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.ConfirmPredictionRequest
import com.youkong.core.network.model.PredictionTask
import com.youkong.core.network.model.StartPredictionRequest
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path

/**
 * AI 状态推测 API
 */
interface PredictionApi {

    /**
     * 开始推测任务
     */
    @POST("agent/prediction/start")
    suspend fun startPrediction(@Body request: StartPredictionRequest = StartPredictionRequest()): ApiResponse<PredictionTask>

    /**
     * 获取推测任务状态
     */
    @GET("agent/prediction/{id}")
    suspend fun getPrediction(@Path("id") taskId: String): ApiResponse<PredictionTask>

    /**
     * 获取最近的推测任务
     */
    @GET("agent/prediction/latest")
    suspend fun getLatestPrediction(): ApiResponse<PredictionTask?>

    /**
     * 获取待确认的推测任务
     */
    @GET("agent/prediction/pending")
    suspend fun getPendingPrediction(): ApiResponse<PredictionTask?>

    /**
     * 确认推测结果
     */
    @POST("agent/prediction/{id}/confirm")
    suspend fun confirmPrediction(
        @Path("id") taskId: String,
        @Body request: ConfirmPredictionRequest = ConfirmPredictionRequest()
    ): ApiResponse<GenericSuccessResponse>

    /**
     * 放弃推测结果
     */
    @POST("agent/prediction/{id}/reject")
    suspend fun rejectPrediction(@Path("id") taskId: String): ApiResponse<GenericSuccessResponse>
}

@kotlinx.serialization.Serializable
data class GenericSuccessResponse(
    val success: Boolean,
    val message: String? = null
)
