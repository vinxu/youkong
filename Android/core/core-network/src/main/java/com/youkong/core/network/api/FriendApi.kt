package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.FreeProbabilityResponse
import com.youkong.core.network.model.FriendResponse
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Path

interface FriendApi {

    @GET("friends")
    suspend fun getFriends(): ApiResponse<List<FriendResponse>>

    @DELETE("friends/{userId}")
    suspend fun deleteFriend(@Path("userId") userId: String): ApiResponse<Unit>

    @GET("friends/invited-by-me")
    suspend fun getInvitedByMe(): ApiResponse<List<FriendResponse>>

    @GET("friends/invited-me")
    suspend fun getInvitedMe(): ApiResponse<List<FriendResponse>>

    @GET("friends/free-probability")
    suspend fun getFriendsProbability(): ApiResponse<FreeProbabilityResponse>
}
