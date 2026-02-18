# Add project specific ProGuard rules here.

# Keep data classes for serialization
-keep class com.youkong.core.domain.model.** { *; }
-keep class com.youkong.core.network.model.** { *; }

# Kotlinx Serialization
-keepattributes *Annotation*, InnerClasses, AnnotationDefault
-dontnote kotlinx.serialization.AnnotationsKt

-keepclassmembers class kotlinx.serialization.json.** {
    *** Companion;
}
-keepclasseswithmembers class kotlinx.serialization.json.** {
    kotlinx.serialization.KSerializer serializer(...);
}

# R8 full mode strips generic signatures from classes not kept — kotlinx.serialization needs them
-keep,allowobfuscation,allowshrinking class kotlinx.serialization.** { *; }
-keep,allowobfuscation,allowshrinking class * implements kotlinx.serialization.KSerializer

# Retrofit — basic rules
-keepattributes Signature, InnerClasses, EnclosingMethod
-keepattributes RuntimeVisibleAnnotations, RuntimeVisibleParameterAnnotations
-keepclassmembers,allowshrinking,allowobfuscation interface * {
    @retrofit2.http.* <methods>;
}
-dontwarn org.codehaus.mojo.animal_sniffer.IgnoreJRERequirement
-dontwarn javax.annotation.**
-dontwarn kotlin.Unit
-dontwarn retrofit2.KotlinExtensions
-dontwarn retrofit2.KotlinExtensions$*

# Retrofit — R8 full mode rules (CRITICAL: 防止泛型签名被擦除)
# R8 full mode 看不到 Retrofit 接口的子类（它们是 Proxy 动态创建的），会把接口优化掉
-if interface * { @retrofit2.http.* <methods>; }
-keep,allowobfuscation interface <1>

# 保留继承的 Retrofit 接口
-if interface * { @retrofit2.http.* <methods>; }
-keep,allowobfuscation interface * extends <1>

# R8 full mode 会擦除 retrofit2.Call 和 retrofit2.Response 的泛型签名
-keep,allowobfuscation,allowshrinking interface retrofit2.Call
-keep,allowobfuscation,allowshrinking class retrofit2.Response

# R8 full mode 擦除方法返回类型的泛型签名 — 这是 ParameterizedType 崩溃的根因
-if interface * { @retrofit2.http.* public *** *(...); }
-keep,allowoptimization,allowshrinking,allowobfuscation class <3>

# OkHttp
-dontwarn okhttp3.**
-dontwarn okio.**

# TPNS (腾讯推送) + 华为 HMS Push
-dontwarn com.jg.**
-dontwarn com.huawei.**
-keep class com.tencent.android.tpush.** { *; }
-keep class com.huawei.** { *; }

# Coil (图片加载 + GIF 解码)
-dontwarn coil.**
-keep class coil.** { *; }
-keep class coil.decode.** { *; }

# Coroutines
-keepnames class kotlinx.coroutines.internal.MainDispatcherFactory {}
-keepnames class kotlinx.coroutines.CoroutineExceptionHandler {}
-keepclassmembernames class kotlinx.** {
    volatile <fields>;
}
