plugins {
    id("youkong.android.library")
    id("youkong.android.hilt")
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "com.youkong.core.domain"
}

dependencies {
    // Coroutines
    api(libs.kotlinx.coroutines.core)

    // Serialization
    api(libs.kotlinx.serialization.json)

    // DateTime
    api(libs.kotlinx.datetime)

    // DataStore (for UnreadMessageManager persistence)
    implementation(project(":core:core-datastore"))

}
