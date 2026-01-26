package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.ConversationResponse
import com.youkong.core.network.model.MessageResponse
import com.youkong.core.network.model.SendMessageRequest
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path

interface MessageApi {

    @GET("conversations")
    suspend fun getConversations(): ApiResponse<List<ConversationResponse>>

    @GET("conversations/{id}/messages")
    suspend fun getMessages(@Path("id") conversationId: String): ApiResponse<List<MessageResponse>>

    @POST("conversations/{id}/messages")
    suspend fun sendMessage(
        @Path("id") conversationId: String,
        @Body request: SendMessageRequest
    ): ApiResponse<MessageResponse>
}
