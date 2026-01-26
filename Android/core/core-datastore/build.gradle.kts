plugins {
    id("youkong.android.library")
    id("youkong.android.hilt")
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "com.youkong.core.datastore"
}

dependencies {
    // DataStore
    implementation(libs.datastore.preferences)

    // Security
    implementation(libs.security.crypto)

    // Coroutines
    implementation(libs.kotlinx.coroutines.core)
    implementation(libs.kotlinx.coroutines.android)

    // Serialization
    implementation(libs.kotlinx.serialization.json)
}
