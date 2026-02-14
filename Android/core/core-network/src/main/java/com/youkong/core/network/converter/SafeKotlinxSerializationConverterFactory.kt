package com.youkong.core.network.converter

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.KSerializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.serializer
import okhttp3.MediaType
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.ResponseBody
import retrofit2.Converter
import retrofit2.Retrofit
import java.lang.reflect.ParameterizedType
import java.lang.reflect.Type

/**
 * 自定义 kotlinx.serialization Converter Factory
 *
 * 解决 R8 泛型类型擦除导致的反序列化崩溃问题。
 *
 * 问题根源：
 * - R8 在 Release 构建时会擦除泛型类型信息
 * - 当 Retrofit 返回 ApiResponse<EmptyResponse> 时，R8 可能将其擦除为 ApiResponse<Object>
 * - kotlinx.serialization 找不到 Object 的序列化器，导致崩溃
 *
 * 解决方案：
 * - 在 Converter 中显式使用反射获取完整的泛型类型
 * - 使用 Json.serializersModule.serializer(type) 获取正确的序列化器
 * - 确保 ProGuard 规则保留类型信息 (-keepattributes Signature)
 */
@OptIn(ExperimentalSerializationApi::class)
class SafeKotlinxSerializationConverterFactory private constructor(
    private val json: Json,
    private val contentType: MediaType,
) : Converter.Factory() {

    override fun responseBodyConverter(
        type: Type,
        annotations: Array<out Annotation>,
        retrofit: Retrofit,
    ): Converter<ResponseBody, *> {
        val serializer = json.serializersModule.serializer(type)
        return SafeResponseBodyConverter(json, serializer)
    }

    override fun requestBodyConverter(
        type: Type,
        parameterAnnotations: Array<out Annotation>,
        methodAnnotations: Array<out Annotation>,
        retrofit: Retrofit,
    ): Converter<*, RequestBody> {
        val serializer = json.serializersModule.serializer(type)
        return SafeRequestBodyConverter(json, contentType, serializer)
    }

    private class SafeResponseBodyConverter<T>(
        private val json: Json,
        private val serializer: KSerializer<T>,
    ) : Converter<ResponseBody, T> {
        override fun convert(value: ResponseBody): T {
            val string = value.string()
            return json.decodeFromString(serializer, string)
        }
    }

    private class SafeRequestBodyConverter<T>(
        private val json: Json,
        private val contentType: MediaType,
        private val serializer: KSerializer<T>,
    ) : Converter<T, RequestBody> {
        override fun convert(value: T): RequestBody {
            val string = json.encodeToString(serializer, value)
            return string.toRequestBody(contentType)
        }
    }

    companion object {
        fun create(json: Json, contentType: MediaType): SafeKotlinxSerializationConverterFactory {
            return SafeKotlinxSerializationConverterFactory(json, contentType)
        }
    }
}
