plugins {
    id("youkong.android.feature")
}

android {
    namespace = "com.youkong.feature.settings"
}

dependencies {
    implementation(project(":core:core-permission"))
    implementation(project(":core:core-agent"))
    implementation(project(":core:core-datastore"))

    // WorkManager
    implementation(libs.work.runtime)
}
