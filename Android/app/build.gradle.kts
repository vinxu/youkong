import java.util.Properties

plugins {
    id("youkong.android.application")
    id("youkong.android.application.compose")
    id("youkong.android.hilt")
}

// 加载 local.properties
val localProperties = Properties().apply {
    val localPropertiesFile = rootProject.file("local.properties")
    if (localPropertiesFile.exists()) {
        load(localPropertiesFile.inputStream())
    }
}

android {
    namespace = "com.youkong.app"

    buildFeatures {
        buildConfig = true
    }

    defaultConfig {
        applicationId = "com.youkong.app"
        versionCode = 2
        versionName = "1.5.0"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        vectorDrawables {
            useSupportLibrary = true
        }

        // TPNS 推送配置（从 local.properties 读取）
        val tpnsAccessId = localProperties.getProperty("TPNS_ACCESS_ID") ?: "0"
        val tpnsAccessKey = localProperties.getProperty("TPNS_ACCESS_KEY") ?: ""
        // ACCESS_ID 必须是数字类型
        manifestPlaceholders["TPNS_ACCESS_ID"] = tpnsAccessId.toLongOrNull() ?: 0L
        manifestPlaceholders["TPNS_ACCESS_KEY"] = tpnsAccessKey
    }

    signingConfigs {
        create("release") {
            storeFile = rootProject.file("youkong-release.keystore")
            storePassword = localProperties.getProperty("KEYSTORE_PASSWORD") ?: "youkong123"
            keyAlias = localProperties.getProperty("KEY_ALIAS") ?: "youkong"
            keyPassword = localProperties.getProperty("KEY_PASSWORD") ?: "youkong123"
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            signingConfig = signingConfigs.getByName("release")
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
        debug {
            isMinifyEnabled = false
            applicationIdSuffix = ".debug"
        }
    }

    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
        }
    }
}

dependencies {
    // Core modules
    implementation(project(":core:core-ui"))
    implementation(project(":core:core-network"))
    implementation(project(":core:core-data"))
    implementation(project(":core:core-domain"))
    implementation(project(":core:core-database"))
    implementation(project(":core:core-datastore"))
    implementation(project(":core:core-agent"))
    implementation(project(":core:core-permission"))
    implementation(project(":core:core-websocket"))

    // Feature modules
    implementation(project(":feature:feature-auth"))
    implementation(project(":feature:feature-home"))
    implementation(project(":feature:feature-message"))
    implementation(project(":feature:feature-profile"))
    implementation(project(":feature:feature-friends"))
    implementation(project(":feature:feature-settings"))

    // WorkManager
    implementation(libs.work.runtime)
    implementation(libs.hilt.work)
    ksp(libs.hilt.work.compiler)

    // Android
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.activity.compose)
    implementation(libs.bundles.lifecycle)
    implementation(libs.androidx.lifecycle.process)

    // Navigation
    implementation(libs.navigation.compose)
    implementation(libs.hilt.navigation.compose)

    // Splash
    implementation("androidx.core:core-splashscreen:1.0.1")

    // TPNS 推送
    implementation("com.tencent.tpns:tpns:1.3.9.0-release")
    // 厂商通道（小米、华为）
    implementation("com.tencent.tpns:xiaomi:1.3.9.0-release")
    implementation("com.tencent.tpns:huawei:1.3.9.0-release")

    // Coil (for configuring ImageLoader with auth)
    implementation(libs.coil.compose)

    // Testing
    testImplementation(libs.junit)
    androidTestImplementation(libs.androidx.junit)
    androidTestImplementation(libs.androidx.espresso)
}
