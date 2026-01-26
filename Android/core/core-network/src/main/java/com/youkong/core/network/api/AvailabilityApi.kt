package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.AvailabilityResponse
import com.youkong.core.network.model.CreateAvailabilityRequest
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path

interface AvailabilityApi {

    @GET("availabilities/friends")
    suspend fun getFriendsAvailabilities(): ApiResponse<List<AvailabilityResponse>>

    @GET("availabilities/mine")
    suspend fun getMyAvailabilities(): ApiResponse<List<AvailabilityResponse>>

    @POST("availabilities")
    suspend fun createAvailability(@Body request: CreateAvailabilityRequest): ApiResponse<AvailabilityResponse>

    @DELETE("availabilities/{id}")
    suspend fun cancelAvailability(@Path("id") availabilityId: String): ApiResponse<Unit>
}
