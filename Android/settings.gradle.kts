pluginManagement {
    includeBuild("build-logic")
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
        // 腾讯 TPNS 推送 SDK
        maven { url = uri("https://mirrors.tencent.com/nexus/repository/maven-public/") }
    }
}

rootProject.name = "YouKong"

include(":app")

// Core modules
include(":core:core-ui")
include(":core:core-network")
include(":core:core-data")
include(":core:core-domain")
include(":core:core-database")
include(":core:core-datastore")
include(":core:core-agent")
include(":core:core-permission")

// Feature modules
include(":feature:feature-auth")
include(":feature:feature-home")
include(":feature:feature-message")
include(":feature:feature-profile")
include(":feature:feature-friends")
include(":feature:feature-settings")
