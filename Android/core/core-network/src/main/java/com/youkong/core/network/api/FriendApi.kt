package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.FreeProbabilityResponse
import com.youkong.core.network.model.FriendRequestCountResponse
import com.youkong.core.network.model.FriendRequestResponse
import com.youkong.core.network.model.FriendResponse
import com.youkong.core.network.model.HandleFriendRequestRequest
import com.youkong.core.network.model.SendFriendRequestRequest
import com.youkong.core.network.model.SendFriendRequestResponse
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
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

    // 好友请求相关
    @POST("friends/request")
    suspend fun sendFriendRequest(@Body request: SendFriendRequestRequest): ApiResponse<SendFriendRequestResponse>

    @GET("friends/requests/received")
    suspend fun getReceivedRequests(): ApiResponse<List<FriendRequestResponse>>

    @GET("friends/requests/sent")
    suspend fun getSentRequests(): ApiResponse<List<FriendRequestResponse>>

    @POST("friends/requests/{id}/handle")
    suspend fun handleFriendRequest(
        @Path("id") requestId: String,
        @Body request: HandleFriendRequestRequest,
    ): ApiResponse<Unit>

    @GET("friends/requests/count")
    suspend fun getPendingRequestCount(): ApiResponse<FriendRequestCountResponse>
}
