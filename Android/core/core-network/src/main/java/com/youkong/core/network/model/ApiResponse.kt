package com.youkong.core.network.model

import kotlinx.serialization.Serializable

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
