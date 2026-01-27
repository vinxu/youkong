package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class RegisterDeviceTokenRequest(
    val token: String,
    val platform: String = "android",
    @SerialName("deviceName")
    val deviceName: String? = null,
    @SerialName("appVersion")
    val appVersion: String? = null,
)
