package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.GridResponse
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Body
import kotlinx.serialization.Serializable

interface HomeApi {

    @GET("home/grid")
    suspend fun getGridData(): ApiResponse<GridResponse>

    @POST("home/poster")
    suspend fun generatePoster(@Body request: GeneratePosterRequest): ApiResponse<PosterResponse>
}

@Serializable
data class GeneratePosterRequest(
    val user_ids: List<String>
)

@Serializable
data class PosterResponse(
    val poster_url: String
)
