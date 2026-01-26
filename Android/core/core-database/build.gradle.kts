plugins {
    id("youkong.android.library")
    id("youkong.android.hilt")
    alias(libs.plugins.ksp)
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "com.youkong.core.database"
}

dependencies {
    implementation(project(":core:core-domain"))

    // Room
    implementation(libs.bundles.room)
    ksp(libs.room.compiler)

    // Serialization for TypeConverters
    implementation(libs.kotlinx.serialization.json)

    // DateTime
    implementation(libs.kotlinx.datetime)
}
