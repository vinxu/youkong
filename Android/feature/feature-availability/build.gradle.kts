plugins {
    id("youkong.android.feature")
}

android {
    namespace = "com.youkong.feature.availability"
}

dependencies {
    // Date/Time picker
    implementation(libs.kotlinx.datetime)
}
