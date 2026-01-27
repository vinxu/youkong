package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.MyInviteResponse
import com.youkong.core.network.model.UpdateUserRequest
import com.youkong.core.network.model.UserProfileResponse
import com.youkong.core.network.model.UserResponse
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

interface UserApi {

    @GET("users/me")
    suspend fun getCurrentUser(): ApiResponse<UserResponse>

    @PUT("users/me")
    suspend fun updateCurrentUser(@Body request: UpdateUserRequest): ApiResponse<UserResponse>

    @GET("users/search")
    suspend fun searchUsers(@Query("keyword") keyword: String): ApiResponse<List<UserProfileResponse>>

    @GET("users/{id}")
    suspend fun getUser(@Path("id") userId: String): ApiResponse<UserProfileResponse>

    @GET("users/me/invite")
    suspend fun getMyInvite(): ApiResponse<MyInviteResponse>
}
