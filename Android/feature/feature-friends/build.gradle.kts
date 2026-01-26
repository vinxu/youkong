plugins {
    id("youkong.android.feature")
}

android {
    namespace = "com.youkong.feature.friends"
}

dependencies {
    implementation(project(":core:core-agent"))
    implementation(project(":core:core-network"))
}
