plugins {
    id("youkong.android.library")
    id("youkong.android.hilt")
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "com.youkong.core.websocket"
}

dependencies {
    implementation(project(":core:core-domain"))
    implementation(project(":core:core-datastore"))

    // OkHttp for WebSocket
    implementation(libs.okhttp)

    // Serialization
    implementation(libs.kotlinx.serialization.json)

    // Coroutines
    implementation(libs.kotlinx.coroutines.core)
    implementation(libs.kotlinx.coroutines.android)
}
