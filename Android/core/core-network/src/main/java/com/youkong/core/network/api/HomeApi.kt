package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.CardRoomMessageResponse
import com.youkong.core.network.model.GridResponse
import com.youkong.core.network.model.JoinCardRequest
import com.youkong.core.network.model.JoinCardResponse
import com.youkong.core.network.model.SendInteractionRequest
import com.youkong.core.network.model.SendRoomMessageRequest
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

interface HomeApi {

    @GET("home/grid")
    suspend fun getGridData(): ApiResponse<GridResponse>

    @POST("interact")
    suspend fun sendInteraction(@Body request: SendInteractionRequest): ApiResponse<Unit?>

    @POST("card/join")
    suspend fun joinCard(@Body request: JoinCardRequest): ApiResponse<JoinCardResponse>

    @GET("card/rooms/{roomId}/messages")
    suspend fun getCardRoomMessages(
        @Path("roomId") roomId: String,
        @Query("limit") limit: Int = 50,
        @Query("offset") offset: Int = 0,
    ): ApiResponse<List<CardRoomMessageResponse>>

    @POST("card/rooms/{roomId}/messages")
    suspend fun sendCardRoomMessage(
        @Path("roomId") roomId: String,
        @Body request: SendRoomMessageRequest,
    ): ApiResponse<CardRoomMessageResponse>
}
