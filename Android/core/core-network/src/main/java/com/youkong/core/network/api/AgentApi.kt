package com.youkong.core.network.api

import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.AnalysisResult
import com.youkong.core.network.model.ApiResponse
import com.youkong.core.network.model.InferenceV2Result
import com.youkong.core.network.model.SelectStatusRequest
import com.youkong.core.network.model.SelectStatusResponse
import com.youkong.core.network.model.StatusReportResponse
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import okhttp3.MultipartBody
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.Multipart
import retrofit2.http.POST
import retrofit2.http.Part

interface AgentApi {

    @POST("agent/status")
    suspend fun reportStatus(@Body request: AgentStatusRequest): ApiResponse<StatusReportResponse>

    @GET("agent/my-analysis")
    suspend fun getMyAnalysis(): ApiResponse<MyAnalysisResponse>

    @POST("agent/infer-status-v2")
    suspend fun inferStatusV3(@Body request: AgentStatusRequest): ApiResponse<InferenceV3Response>

    @POST("agent/infer-status-v2/respond")
    suspend fun inferStatusV3Respond(@Body request: InferV3RespondRequest): ApiResponse<InferenceV3Response>

    @POST("agent/status-feedback")
    suspend fun submitStatusFeedback(@Body request: StatusFeedbackApiRequest): ApiResponse<Unit>

    @POST("agent/select-status")
    suspend fun selectStatus(@Body request: SelectStatusRequest): ApiResponse<SelectStatusResponse>

    @Multipart
    @POST("agent/upload-gif")
    suspend fun uploadGif(@Part gif: MultipartBody.Part): ApiResponse<UploadGifResponse>

    @GET("agent/sts")
    suspend fun getSTSCredentials(): ApiResponse<STSResponse>
}

@Serializable
data class UploadGifResponse(
    @SerialName("gif_url")
    val gifUrl: String,
)

@Serializable
data class STSResponse(
    val sts: STSCredentials,
)

@Serializable
data class STSCredentials(
    val accessKeyId: String,
    val secretAccessKey: String,
    val sessionToken: String,
    val bucket: String,
    val region: String,
    val prefix: String,
)

@Serializable
data class MyAnalysisResponse(
    val analysis: AnalysisResult?
)

@Serializable
data class InferenceV3Response(
    val phase: String,                        // "completed" | "awaiting_choice"
    val result: InferenceV2Result? = null,   // phase=completed 时有值
    @SerialName("session_id")
    val sessionId: String? = null,           // phase=awaiting_choice 时有值
    val question: String? = null,
    val options: List<InferenceV3Option>? = null,
    @SerialName("default_index")
    val defaultIndex: Int? = null,
)

@Serializable
data class InferenceV3Option(
    val index: Int,
    val emoji: String,
    val activity: String,
    val reason: String? = null,
)

@Serializable
data class InferV3RespondRequest(
    @SerialName("session_id")
    val sessionId: String,
    @SerialName("selected_index")
    val selectedIndex: Int,
)

@Serializable
data class StatusFeedbackApiRequest(
    @SerialName("original_emoji")
    val originalEmoji: String? = null,
    @SerialName("original_activity")
    val originalActivity: String? = null,
    @SerialName("corrected_emoji")
    val correctedEmoji: String,
    @SerialName("corrected_activity")
    val correctedActivity: String,
    @SerialName("corrected_place")
    val correctedPlace: String? = null,
    @SerialName("corrected_is_available")
    val correctedIsAvailable: Boolean = false,
    @SerialName("gif_url")
    val gifUrl: String? = null,
    @SerialName("giphy_query")
    val giphyQuery: String? = null,
    @SerialName("use_gif")
    val useGif: Boolean = false,
)
