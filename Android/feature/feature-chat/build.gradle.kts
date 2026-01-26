plugins {
    id("youkong.android.feature")
}

android {
    namespace = "com.youkong.feature.chat"
}

dependencies {
    // DataStore for user preferences
    implementation(project(":core:core-datastore"))
}
