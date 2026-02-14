# Consumer ProGuard rules for core-network module

# 保留泛型签名（Retrofit 泛型参数必需）
-keepattributes Signature

# 保留网络模型类（确保序列化器不被删除）
-keep class com.youkong.core.network.model.** { *; }

# 保留 API 接口
-keep interface com.youkong.core.network.api.** { *; }
