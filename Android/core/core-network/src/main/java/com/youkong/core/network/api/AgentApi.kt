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

    @POST("agent/infer-options")
    suspend fun inferOptions(@Body request: InferOptionsApiRequest): ApiResponse<InferenceOptionsApiResponse>

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
    @SerialName("inference_session_id")
    val inferenceSessionId: String? = null,
    @SerialName("selected_option_idx")
    val selectedOptionIdx: Int? = null,
)

// MARK: - 4选1 推断选项

@Serializable
data class InferOptionsApiRequest(
    val screen: com.youkong.core.network.model.ScreenDataRequest? = null,
    val location: com.youkong.core.network.model.LocationDataRequest? = null,
    @SerialName("extended_location")
    val extendedLocation: com.youkong.core.network.model.ExtendedLocationDataRequest? = null,
    val battery: com.youkong.core.network.model.BatteryDataRequest? = null,
    val mode: com.youkong.core.network.model.ModeDataRequest? = null,
    val connection: com.youkong.core.network.model.ConnectionDataRequest? = null,
    val display: com.youkong.core.network.model.DisplayDataRequest? = null,
    val calendar: com.youkong.core.network.model.CalendarDataRequest? = null,
    val movement: com.youkong.core.network.model.MovementDataRequest? = null,
    @SerialName("ambient_light")
    val ambientLight: com.youkong.core.network.model.AmbientLightDataRequest? = null,
    @SerialName("exclude_activities")
    val excludeActivities: List<String>? = null,
    @SerialName("session_id")
    val sessionId: String? = null,
)

@Serializable
data class InferenceOptionsApiResponse(
    @SerialName("session_id")
    val sessionId: String,
    val options: List<StatusCardOptionApi>,
)

@Serializable
data class StatusCardOptionApi(
    val index: Int,
    val emoji: String,
    val activity: String,
    val place: String? = null,
    @SerialName("is_available")
    val isAvailable: Boolean = false,
    val confidence: String = "medium",
    @SerialName("giphy_query")
    val giphyQuery: String = "",
    @SerialName("gif_url")
    val gifUrl: String? = null,
)
