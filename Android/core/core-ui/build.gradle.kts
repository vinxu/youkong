plugins {
    id("youkong.android.library")
    id("youkong.android.library.compose")
}

android {
    namespace = "com.youkong.core.ui"
}

dependencies {
    // Compose
    implementation(libs.compose.material.icons.extended)

    // Coil
    api(libs.coil.compose)

    // Android
    implementation(libs.androidx.core.ktx)
}
