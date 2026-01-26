plugins {
    id("youkong.android.library")
    id("youkong.android.hilt")
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "com.youkong.core.network"

    buildFeatures {
        buildConfig = true
    }

    defaultConfig {
        buildConfigField("String", "BASE_URL", "\"http://49.232.13.41:8080/api/v1/\"")
    }

    buildTypes {
        debug {
            buildConfigField("String", "BASE_URL", "\"http://49.232.13.41:8080/api/v1/\"")
        }
        release {
            buildConfigField("String", "BASE_URL", "\"http://49.232.13.41:8080/api/v1/\"")
        }
    }
}

dependencies {
    implementation(project(":core:core-domain"))
    implementation(project(":core:core-datastore"))

    // Network
    implementation(libs.bundles.network)
    implementation(libs.kotlinx.serialization.json)

    // Coroutines
    implementation(libs.kotlinx.coroutines.core)
    implementation(libs.kotlinx.coroutines.android)

    // DateTime
    implementation(libs.kotlinx.datetime)
}
