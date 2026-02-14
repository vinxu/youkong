plugins {
    id("youkong.android.feature")
}

android {
    namespace = "com.youkong.feature.home"
}

dependencies {
    // Core modules
    implementation(project(":core:core-network"))
    implementation(project(":core:core-agent"))
    implementation(project(":core:core-datastore"))
    implementation(project(":core:core-permission"))

    // SwipeRefresh for pull-to-refresh
    implementation("com.google.accompanist:accompanist-swiperefresh:0.32.0")
}
