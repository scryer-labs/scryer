package com.edwardnoaland.scryer.analyze.execute

import java.nio.file.Files
import java.nio.file.Path
import java.util.Properties
import kotlin.io.path.isRegularFile
import kotlin.io.path.readText

/** Deliberately conventional commands, not discovered custom test-task semantics. */
internal class TestRunPlanner(private val environment: Map<String, String>) {
    fun plan(root: Path, originalRepository: Path, cache: Path, run: Path): TestRunPlan {
        val gradle = root.resolve("gradlew").isRegularFile()
        val maven = root.resolve("mvnw").isRegularFile()
        check(gradle != maven) { "Expected exactly one root Gradle/Maven Wrapper. Custom test commands and nested build roots are not supported yet." }
        check(System.getProperty("os.name").contains("Windows", ignoreCase = true).not()) { "Target test execution currently supports POSIX Wrapper scripts only." }
        val java = chooseJava(root, originalRepository, gradle)
        check(java.resolve("bin/java").isRegularFile()) { "Invalid target JAVA_HOME: $java" }
        val command = if (gradle) listOf("sh", "./gradlew", "--no-daemon", "--console=plain", "--rerun-tasks", "--no-build-cache", "-g", cache.resolve("gradle").toString(),
            "--project-cache-dir", run.resolve("project-cache").toString(), "clean", "test")
        else listOf("sh", "./mvnw", "-B", "-Dmaven.repo.local=${cache.resolve("maven-repository")}", "clean", "verify")
        val overrides = mapOf("JAVA_HOME" to java.toString(),
            "PATH" to "${java.resolve("bin")}:${environment["PATH"].orEmpty()}",
            "GRADLE_USER_HOME" to cache.resolve("gradle").toString(),
            "MAVEN_USER_HOME" to cache.resolve("maven-home").toString())
        return TestRunPlan(command, java, overrides)
    }

    private fun chooseJava(root: Path, original: Path, gradle: Boolean): Path {
        environment["SCRYER_JAVA_HOME"]?.takeIf { it.isNotBlank() }?.let { return Path.of(it).toAbsolutePath().normalize() }
        val properties = Properties()
        val wrapper = root.resolve("gradle/wrapper/gradle-wrapper.properties")
        if (wrapper.isRegularFile()) wrapper.toFile().inputStream().use(properties::load)
        val major = Regex("gradle-(\\d+)\\.").find(properties.getProperty("distributionUrl", ""))?.groupValues?.get(1)?.toIntOrNull()
        if (gradle && major != null && major <= 6) {
            val installs = original.resolve(".tooling/mise/data/installs/java")
            if (Files.isDirectory(installs)) {
                val java8 = Files.list(installs).use { stream ->
                    stream.sorted().toList().asSequence().flatMap { sequenceOf(it.resolve("Contents/Home"), it) }
                        .firstOrNull { candidate ->
                            val release = candidate.resolve("release")
                            candidate.resolve("bin/java").isRegularFile() && release.isRegularFile() &&
                                Regex("JAVA_VERSION=\"1\\.8\\.").containsMatchIn(release.readText())
                        }
                }
                if (java8 != null) return java8.toRealPath()
            }
            error("Gradle $major requires a compatible target JVM; set SCRYER_JAVA_HOME. Fixture-local mise Java 8 can be auto-detected.")
        }
        return Path.of(System.getProperty("java.home"))
    }
}
