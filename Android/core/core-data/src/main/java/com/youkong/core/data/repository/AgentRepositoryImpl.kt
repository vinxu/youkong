package com.youkong.core.data.repository

import com.youkong.core.domain.model.Confidence
import com.youkong.core.domain.model.FriendWithProbability
import com.youkong.core.domain.model.UserProfile
import com.youkong.core.domain.repository.AgentRepository
import com.youkong.core.domain.repository.InferenceOptionsResult
import com.youkong.core.domain.repository.StatusCardOption
import com.youkong.core.domain.repository.StatusFeedback
import com.youkong.core.domain.repository.StatusInferenceResult
import com.youkong.core.domain.repository.V3InferenceOption
import com.youkong.core.domain.repository.V3InferenceResult
import com.youkong.core.network.api.AgentApi
import com.youkong.core.network.api.FriendApi
import com.youkong.core.network.api.InferOptionsApiRequest
import com.youkong.core.network.api.InferV3RespondRequest
import com.youkong.core.network.api.STSCredentials
import com.youkong.core.network.api.StatusFeedbackApiRequest
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.LocationDataRequest
import com.youkong.core.network.model.ScreenDataRequest
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.withContext
import java.net.HttpURLConnection
import java.net.URL
import java.security.MessageDigest
import java.util.UUID
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AgentRepositoryImpl @Inject constructor(
    private val agentApi: AgentApi,
    private val friendApi: FriendApi,
) : AgentRepository {

    private val _friendsProbability = MutableStateFlow<List<FriendWithProbability>>(emptyList())
    val friendsProbability: Flow<List<FriendWithProbability>> = _friendsProbability.asStateFlow()

    override suspend fun reportStatus(
        isActive: Boolean,
        activityType: String,
        sessionDurationMinutes: Int,
        lastActiveMinutesAgo: Int,
        placeType: String?,
        atPlaceSinceMinutes: Int?,
        city: String?,
    ): Result<Unit> {
        return try {
            val request = AgentStatusRequest(
                screen = ScreenDataRequest(
                    isActive = isActive,
                    activityType = activityType,
                    sessionDurationMinutes = sessionDurationMinutes,
                    lastActiveMinutesAgo = lastActiveMinutesAgo,
                ),
                location = if (placeType != null && atPlaceSinceMinutes != null) {
                    LocationDataRequest(
                        placeType = placeType,
                        atPlaceSinceMinutes = atPlaceSinceMinutes,
                        city = city,
                    )
                } else null,
            )
            val response = agentApi.reportStatus(request)
            if (response.isSuccess) {
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getFriendsProbability(): Result<List<FriendWithProbability>> {
        return try {
            val response = friendApi.getFriendsProbability()
            val data = response.data
            if (response.isSuccess && data != null) {
                val friends = data.friends.map { it.toDomain() }
                _friendsProbability.value = friends
                Result.success(friends)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun inferOptions(request: Any, excludeActivities: List<String>?, sessionId: String?): Result<InferenceOptionsResult> {
        return try {
            val sensorData = request as? AgentStatusRequest ?: AgentStatusRequest()
            val apiRequest = InferOptionsApiRequest(
                screen = sensorData.screen,
                location = sensorData.location,
                extendedLocation = sensorData.extendedLocation,
                battery = sensorData.battery,
                mode = sensorData.mode,
                connection = sensorData.connection,
                display = sensorData.display,
                calendar = sensorData.calendar,
                movement = sensorData.movement,
                ambientLight = sensorData.ambientLight,
                excludeActivities = excludeActivities,
                sessionId = sessionId,
            )
            val response = agentApi.inferOptions(apiRequest)
            val data = response.data
            if (response.isSuccess && data != null) {
                Result.success(InferenceOptionsResult(
                    sessionId = data.sessionId,
                    options = data.options.map { o ->
                        StatusCardOption(
                            index = o.index,
                            emoji = o.emoji,
                            activity = o.activity,
                            place = o.place,
                            isAvailable = o.isAvailable,
                            confidence = o.confidence,
                            giphyQuery = o.giphyQuery,
                            gifUrl = o.gifUrl,
                        )
                    }
                ))
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun inferStatus(): Result<StatusInferenceResult> {
        // 通过 V3 接口获取，取 completed 的 result
        return inferStatusV3(AgentStatusRequest()).map { it.result ?: StatusInferenceResult(emoji = "❓", activity = "未知") }
    }

    override suspend fun inferStatusV3(sensorData: Any): Result<V3InferenceResult> {
        return try {
            val request = sensorData as? AgentStatusRequest ?: AgentStatusRequest()
            val response = agentApi.inferStatusV3(request)
            val data = response.data
            if (response.isSuccess && data != null) {
                Result.success(data.toDomain())
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun inferStatusV3Respond(sessionId: String, selectedIndex: Int): Result<V3InferenceResult> {
        return try {
            val response = agentApi.inferStatusV3Respond(InferV3RespondRequest(sessionId, selectedIndex))
            val data = response.data
            if (response.isSuccess && data != null) {
                Result.success(data.toDomain())
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    private fun com.youkong.core.network.api.InferenceV3Response.toDomain(): V3InferenceResult {
        return V3InferenceResult(
            phase = phase,
            result = result?.let { r ->
                StatusInferenceResult(
                    emoji = r.emoji,
                    activity = r.activity,
                    place = r.place,
                    isAvailable = r.isAvailable,
                    durationHint = r.durationHint,
                    confidence = r.confidence,
                    inferredAt = r.inferredAt,
                    reasoning = r.reasoning,
                    gifUrl = r.gifUrl,
                    gifSmallUrl = r.gifSmallUrl,
                    giphyQuery = r.giphyQuery,
                )
            },
            sessionId = sessionId,
            question = question,
            options = options?.map { o ->
                V3InferenceOption(index = o.index, emoji = o.emoji, activity = o.activity, reason = o.reason)
            },
            defaultIndex = defaultIndex,
        )
    }

    override suspend fun submitStatusFeedback(feedback: StatusFeedback): Result<Unit> {
        return try {
            val request = StatusFeedbackApiRequest(
                originalEmoji = feedback.originalEmoji,
                originalActivity = feedback.originalActivity,
                correctedEmoji = feedback.correctedEmoji,
                correctedActivity = feedback.correctedActivity,
                correctedPlace = feedback.correctedPlace,
                correctedIsAvailable = feedback.correctedIsAvailable,
                gifUrl = feedback.gifUrl,
                giphyQuery = feedback.giphyQuery,
                useGif = feedback.useGif,
                inferenceSessionId = feedback.inferenceSessionId,
                selectedOptionIdx = feedback.selectedOptionIdx,
            )
            val response = agentApi.submitStatusFeedback(request)
            if (response.isSuccess) Result.success(Unit)
            else Result.failure(Exception(response.message))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun uploadGifToCOS(gifData: ByteArray): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                // 1. 获取 STS 临时凭证
                val stsResponse = agentApi.getSTSCredentials()
                val cred = stsResponse.data?.sts
                    ?: return@withContext Result.failure(Exception(stsResponse.message))

                // 2. 生成 COS key (MD5 去重)
                val md5 = MessageDigest.getInstance("MD5").digest(gifData)
                val md5Hex = md5.joinToString("") { "%02x".format(it) }
                val objectKey = "${cred.prefix}${md5Hex}.gif"

                // 3. 直传 COS
                val host = "${cred.bucket}.cos.${cred.region}.myqcloud.com"
                val url = "https://$host/$objectKey"
                val timestamp = System.currentTimeMillis() / 1000
                val signature = generateCOSSignature(
                    secretId = cred.accessKeyId,
                    secretKey = cred.secretAccessKey,
                    method = "PUT",
                    path = "/$objectKey",
                    host = host,
                    timestamp = timestamp,
                )

                val conn = URL(url).openConnection() as HttpURLConnection
                conn.requestMethod = "PUT"
                conn.doOutput = true
                conn.connectTimeout = 15000
                conn.readTimeout = 30000
                conn.setRequestProperty("Host", host)
                conn.setRequestProperty("Content-Type", "image/gif")
                conn.setRequestProperty("x-cos-acl", "public-read")
                conn.setRequestProperty("x-cos-security-token", cred.sessionToken)
                conn.setRequestProperty("Authorization", signature)
                conn.setRequestProperty("Content-Length", gifData.size.toString())

                conn.outputStream.use { it.write(gifData) }

                val responseCode = conn.responseCode
                conn.disconnect()

                if (responseCode in 200..299) {
                    Result.success(url)
                } else {
                    Result.failure(Exception("COS 上传失败: HTTP $responseCode"))
                }
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    /** COS 签名算法（参考 CastReader COSUploader） */
    private fun generateCOSSignature(
        secretId: String, secretKey: String,
        method: String, path: String, host: String, timestamp: Long,
    ): String {
        val keyTime = "$timestamp;${timestamp + 3600}"
        val signKey = hmacSha1(secretKey, keyTime)
        val httpString = "${method.lowercase()}\n$path\n\nhost=$host\n"
        val sha1HttpString = sha1Hex(httpString)
        val stringToSign = "sha1\n$keyTime\n$sha1HttpString\n"
        val signature = hmacSha1(signKey, stringToSign)
        return "q-sign-algorithm=sha1&q-ak=$secretId&q-sign-time=$keyTime&q-key-time=$keyTime&q-header-list=host&q-url-param-list=&q-signature=$signature"
    }

    private fun hmacSha1(key: String, data: String): String {
        val mac = Mac.getInstance("HmacSHA1")
        mac.init(SecretKeySpec(key.toByteArray(), "HmacSHA1"))
        return mac.doFinal(data.toByteArray()).joinToString("") { "%02x".format(it) }
    }

    private fun sha1Hex(input: String): String {
        val digest = MessageDigest.getInstance("SHA-1").digest(input.toByteArray())
        return digest.joinToString("") { "%02x".format(it) }
    }

    private fun com.youkong.core.network.model.FriendProbabilityResponse.toDomain(): FriendWithProbability {
        return FriendWithProbability(
            user = UserProfile(
                id = friendId,
                nickname = name,
                avatar = avatar,
            ),
            probability = if (probability < 0) null else probability,
            confidence = Confidence.fromString(confidence),
            reason = reason,
            color = color,
            emoji = emoji,
            activity = activity,
            updatedAt = updatedAt,
        )
    }
}
