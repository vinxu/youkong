package com.youkong.core.network.api

import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.StatusReportResponse
import retrofit2.http.Body
import retrofit2.http.POST

interface AgentApi {

    @POST("agent/status")
    suspend fun reportStatus(@Body request: AgentStatusRequest): ApiResponse<StatusReportResponse>
}
