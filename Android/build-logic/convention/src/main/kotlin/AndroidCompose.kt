import com.android.build.api.dsl.CommonExtension
import org.gradle.api.Project
import org.gradle.api.artifacts.VersionCatalogsExtension
import org.gradle.kotlin.dsl.dependencies
import org.gradle.kotlin.dsl.getByType

internal fun Project.configureAndroidCompose(
    commonExtension: CommonExtension<*, *, *, *, *>,
) {
    val libs = extensions.getByType<VersionCatalogsExtension>().named("libs")

    commonExtension.apply {
        buildFeatures {
            compose = true
        }

        composeOptions {
            kotlinCompilerExtensionVersion = libs.findVersion("composeCompiler").get().toString()
        }
    }

    dependencies {
        val bom = libs.findLibrary("compose.bom").get()
        add("implementation", platform(bom))
        add("androidTestImplementation", platform(bom))

        add("implementation", libs.findLibrary("compose.ui").get())
        add("implementation", libs.findLibrary("compose.ui.graphics").get())
        add("implementation", libs.findLibrary("compose.ui.tooling.preview").get())
        add("implementation", libs.findLibrary("compose.material3").get())
        add("implementation", libs.findLibrary("compose.foundation").get())
        add("implementation", libs.findLibrary("compose.runtime").get())

        add("debugImplementation", libs.findLibrary("compose.ui.tooling").get())
        add("debugImplementation", libs.findLibrary("compose.ui.test.manifest").get())
    }
}
