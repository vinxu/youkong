plugins {
    id("youkong.android.library")
    id("youkong.android.hilt")
}

android {
    namespace = "com.youkong.core.permission"
}

dependencies {
    // Coroutines
    implementation(libs.kotlinx.coroutines.core)
    implementation(libs.kotlinx.coroutines.android)

    // Location
    implementation(libs.play.services.location)
}
