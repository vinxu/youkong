package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.GridResponse
import retrofit2.http.GET

interface HomeApi {

    @GET("home/grid")
    suspend fun getGridData(): ApiResponse<GridResponse>
}
