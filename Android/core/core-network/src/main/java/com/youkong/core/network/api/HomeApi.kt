package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.GridResponse
import com.youkong.core.network.model.SendInteractionRequest
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST

interface HomeApi {

    @GET("home/grid")
    suspend fun getGridData(): ApiResponse<GridResponse>

    @POST("interact")
    suspend fun sendInteraction(@Body request: SendInteractionRequest): ApiResponse<Unit?>
}
