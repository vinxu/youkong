plugins {
    id("youkong.android.feature")
}

android {
    namespace = "com.youkong.feature.home"
}

dependencies {
    // SwipeRefresh for pull-to-refresh
    implementation("com.google.accompanist:accompanist-swiperefresh:0.32.0")
}
