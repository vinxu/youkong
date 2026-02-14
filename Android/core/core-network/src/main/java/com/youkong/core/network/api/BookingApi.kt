package com.youkong.core.network.api

import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.BookingRespondRequest
import com.youkong.core.network.model.BookingRespondResponse
import retrofit2.http.Body
import retrofit2.http.POST
import retrofit2.http.Path

interface BookingApi {
    @POST("bookings/{id}/respond")
    suspend fun respondToBooking(
        @Path("id") bookingId: String,
        @Body body: BookingRespondRequest,
    ): ApiResponse<BookingRespondResponse>
}
