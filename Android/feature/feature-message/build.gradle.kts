plugins {
    id("youkong.android.feature")
}

android {
    namespace = "com.youkong.feature.message"
}

dependencies {
    implementation(project(":core:core-datastore"))
    implementation(project(":core:core-websocket"))
}
