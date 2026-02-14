package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class BookingRespondRequest(val action: String)

@Serializable
data class BookingRespondResponse(
    val booking: BookingInfo? = null,
    @SerialName("schedule_updated") val scheduleUpdated: Boolean? = null,
)

@Serializable
data class BookingInfo(
    val id: String,
    val title: String,
    val status: String,
)
