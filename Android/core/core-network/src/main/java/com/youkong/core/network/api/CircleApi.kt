package com.youkong.core.network.api

import com.youkong.core.network.model.AddMemberRequest
import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.CircleDetailResponse
import com.youkong.core.network.model.CircleResponse
import com.youkong.core.network.model.CreateCircleRequest
import com.youkong.core.network.model.UpdateCircleRequest
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path

interface CircleApi {

    @GET("circles")
    suspend fun getMyCircles(): ApiResponse<List<CircleResponse>>

    @POST("circles")
    suspend fun createCircle(@Body request: CreateCircleRequest): ApiResponse<CircleResponse>

    @GET("circles/{id}")
    suspend fun getCircle(@Path("id") circleId: String): ApiResponse<CircleDetailResponse>

    @PUT("circles/{id}")
    suspend fun updateCircle(
        @Path("id") circleId: String,
        @Body request: UpdateCircleRequest
    ): ApiResponse<CircleResponse>

    @DELETE("circles/{id}")
    suspend fun deleteCircle(@Path("id") circleId: String): ApiResponse<Unit>

    @POST("circles/{id}/members")
    suspend fun addMember(
        @Path("id") circleId: String,
        @Body request: AddMemberRequest
    ): ApiResponse<Unit>

    @DELETE("circles/{id}/members/{userId}")
    suspend fun removeMember(
        @Path("id") circleId: String,
        @Path("userId") userId: String
    ): ApiResponse<Unit>
}
