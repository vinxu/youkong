package com.youkong.core.network.api

import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.AnalysisResult
import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.StatusReportResponse
import kotlinx.serialization.Serializable
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST

interface AgentApi {

    @POST("agent/status")
    suspend fun reportStatus(@Body request: AgentStatusRequest): ApiResponse<StatusReportResponse>

    @GET("agent/my-analysis")
    suspend fun getMyAnalysis(): ApiResponse<MyAnalysisResponse>
}

@Serializable
data class MyAnalysisResponse(
    val analysis: AnalysisResult?
)
