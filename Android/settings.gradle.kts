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
    }
}

rootProject.name = "YouKong"

include(":app")

// Core modules
include(":core:core-ui")
include(":core:core-network")
include(":core:core-websocket")
include(":core:core-data")
include(":core:core-domain")
include(":core:core-database")
include(":core:core-datastore")

// Feature modules
include(":feature:feature-auth")
include(":feature:feature-home")
include(":feature:feature-availability")
include(":feature:feature-circle")
include(":feature:feature-message")
include(":feature:feature-profile")
include(":feature:feature-invitation")
include(":feature:feature-friend")
