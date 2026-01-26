plugins {
    id("youkong.android.feature")
}

android {
    namespace = "com.youkong.feature.auth"
}

dependencies {
    implementation(project(":core:core-datastore"))
}
