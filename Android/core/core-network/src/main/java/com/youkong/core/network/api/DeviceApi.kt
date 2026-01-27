package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.RegisterDeviceTokenRequest
import retrofit2.http.Body
import retrofit2.http.POST

interface DeviceApi {

    @POST("devices/token")
    suspend fun registerToken(@Body request: RegisterDeviceTokenRequest): ApiResponse<Unit>
}
