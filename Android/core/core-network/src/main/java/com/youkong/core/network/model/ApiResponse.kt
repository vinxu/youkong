package com.youkong.core.network.model

import androidx.annotation.Keep
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

/**
 * 空响应类型，用于不需要 data 内容的 API
 * 使用 @Serializable data class 确保 R8 保留序列化器
 */
@Keep
@Serializable
data class EmptyResponse(
    val message: String? = null,
)

/**
 * 简单 API 响应（非泛型）
 * 用于不需要 data 字段的 API，完全避免 R8 泛型类型擦除问题
 *
 * 适用场景：发送验证码、删除好友等只需要知道成功/失败的 API
 *
 * @Keep 注解确保 R8 不会混淆此类
 * data 字段使用 JsonObject 类型，可以接收任意 JSON 对象，避免类型解析问题
 */
@Keep
@Serializable
data class SimpleApiResponse(
    val code: Int,
    val message: String,
    val data: JsonObject? = null, // 接收任意 JSON 对象，忽略其内容
) {
    val isSuccess: Boolean get() = code == ApiErrorCode.SUCCESS

    fun checkSuccess() {
        if (!isSuccess) {
            throw ApiException(code, message)
        }
    }
}

@Serializable
data class ApiResponse<T>(
    val code: Int,
    val message: String,
    val data: T? = null,
) {
    val isSuccess: Boolean get() = code == ApiErrorCode.SUCCESS

    fun getOrThrow(): T {
        if (isSuccess && data != null) {
            return data
        }
        throw ApiException(code, message)
    }

    fun getOrNull(): T? {
        return if (isSuccess) data else null
    }
}

object ApiErrorCode {
    const val SUCCESS = 0
    const val PARAM_ERROR = 1001
    const val UNAUTHORIZED = 1002
    const val TOKEN_EXPIRED = 1003
    const val NOT_FOUND = 1004
    const val FORBIDDEN = 1005
    const val INTERNAL_ERROR = 5000
}

class ApiException(
    val code: Int,
    override val message: String,
) : Exception(message) {

    val isUnauthorized: Boolean get() = code == ApiErrorCode.UNAUTHORIZED
    val isTokenExpired: Boolean get() = code == ApiErrorCode.TOKEN_EXPIRED
    val isNotFound: Boolean get() = code == ApiErrorCode.NOT_FOUND
    val isForbidden: Boolean get() = code == ApiErrorCode.FORBIDDEN

    companion object {
        fun unauthorized() = ApiException(ApiErrorCode.UNAUTHORIZED, "未授权")
        fun tokenExpired() = ApiException(ApiErrorCode.TOKEN_EXPIRED, "Token已过期")
        fun notFound() = ApiException(ApiErrorCode.NOT_FOUND, "资源不存在")
        fun forbidden() = ApiException(ApiErrorCode.FORBIDDEN, "无权限")
        fun internalError() = ApiException(ApiErrorCode.INTERNAL_ERROR, "服务器内部错误")
    }
}

// MARK: - 用户设置

/**
 * 用户设置请求
 */
@Serializable
data class UserSettingsRequest(
    @SerialName("auto_predict_enabled")
    val autoPredictEnabled: Boolean? = null
)

/**
 * 用户设置响应
 */
@Serializable
data class UserSettingsResponse(
    @SerialName("auto_predict_enabled")
    val autoPredictEnabled: Boolean
)
