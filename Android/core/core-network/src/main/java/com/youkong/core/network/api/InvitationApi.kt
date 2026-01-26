package com.youkong.core.network.api

import com.youkong.core.network.model.AcceptInvitationResponse
import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.CreateInvitationRequest
import com.youkong.core.network.model.InvitationResponse
import com.youkong.core.network.model.PublicInvitationResponse
import okhttp3.ResponseBody
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Streaming

interface InvitationApi {

    @POST("invitations")
    suspend fun createInvitation(@Body request: CreateInvitationRequest): ApiResponse<InvitationResponse>

    @GET("invitations")
    suspend fun getMyInvitations(): ApiResponse<List<InvitationResponse>>

    @GET("invite/{code}")
    suspend fun getPublicInvitation(@Path("code") code: String): ApiResponse<PublicInvitationResponse>

    @GET("invitations/{id}")
    suspend fun getInvitationDetail(@Path("id") id: String): ApiResponse<InvitationResponse>

    @DELETE("invitations/{id}")
    suspend fun disableInvitation(@Path("id") id: String): ApiResponse<Unit>

    @POST("invite/{code}/accept")
    suspend fun acceptInvitation(@Path("code") code: String): ApiResponse<AcceptInvitationResponse>

    @GET("invitations/{id}/poster")
    @Streaming
    suspend fun getInvitationPoster(@Path("id") id: String): ResponseBody

    @GET("invitations/{id}/qrcode")
    @Streaming
    suspend fun getInvitationQrCode(@Path("id") id: String): ResponseBody
}
