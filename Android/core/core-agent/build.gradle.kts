plugins {
    id("youkong.android.library")
    id("youkong.android.hilt")
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "com.youkong.core.agent"
}

dependencies {
    // Project
    implementation(project(":core:core-permission"))
    implementation(project(":core:core-network"))
    implementation(project(":core:core-datastore"))

    // Coroutines
    implementation(libs.kotlinx.coroutines.core)
    implementation(libs.kotlinx.coroutines.android)

    // Serialization
    implementation(libs.kotlinx.serialization.json)

    // DateTime
    implementation(libs.kotlinx.datetime)

    // WorkManager
    implementation(libs.work.runtime)
    implementation(libs.hilt.work)
    ksp(libs.hilt.work.compiler)

    // Location
    implementation(libs.play.services.location)
}
