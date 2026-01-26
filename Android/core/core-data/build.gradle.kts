plugins {
    id("youkong.android.library")
    id("youkong.android.hilt")
}

android {
    namespace = "com.youkong.core.data"
}

dependencies {
    implementation(project(":core:core-domain"))
    implementation(project(":core:core-network"))
    implementation(project(":core:core-database"))
    implementation(project(":core:core-datastore"))

    // Coroutines
    implementation(libs.kotlinx.coroutines.core)
    implementation(libs.kotlinx.coroutines.android)
}
