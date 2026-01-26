package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class AvailabilityResponse(
    val id: String,
    val user: UserProfileResponse,
    @SerialName("startTime")
    val startTime: String,
    @SerialName("endTime")
    val endTime: String,
    val location: LocationResponse,
    val status: String,
    @SerialName("circleIds")
    val circleIds: List<String>,
    @SerialName("createdAt")
    val createdAt: String,
)

@Serializable
data class LocationResponse(
    val type: String,
    val name: String? = null,
    val latitude: Double? = null,
    val longitude: Double? = null,
)

@Serializable
data class CreateAvailabilityRequest(
    @SerialName("startTime")
    val startTime: String,
    @SerialName("endTime")
    val endTime: String,
    @SerialName("locationType")
    val locationType: String,
    @SerialName("locationName")
    val locationName: String? = null,
    val latitude: Double? = null,
    val longitude: Double? = null,
    @SerialName("circleIds")
    val circleIds: List<String>,
)
