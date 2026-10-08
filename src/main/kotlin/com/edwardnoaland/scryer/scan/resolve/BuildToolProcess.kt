package com.edwardnoaland.scryer.scan.resolve

import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.model.ScanException
import java.nio.file.Files
import java.nio.file.Path
import java.util.concurrent.TimeUnit
import kotlin.io.path.*

internal class BuildToolProcess(
    val timeoutSeconds: Long,
    private val progress: (String) -> Unit,
) {
    fun report(message: String) {
        progress(message)
    }

    fun execute(
        command: List<String>,
        root: Path,
        temp: Path,
        javaHome: String,
        environment: Map<String, String> = emptyMap(),
    ) {
        val log = temp.resolve("resolver.log")
        val builder = ProcessBuilder(command)
            .directory(root.toFile())
            .redirectErrorStream(true)
            .redirectOutput(log.toFile())
        builder.environment()["JAVA_HOME"] = javaHome
        builder.environment().putAll(environment)
        builder.environment().putIfAbsent("MAVEN_USER_HOME", Path.of(System.getenv("SCRYER_CACHE_HOME") ?: System.getProperty("java.io.tmpdir") + "/scryer-cache").resolve("maven-home").toString())
        val process = builder.start()
        val started = System.nanoTime()
        var completed = false
        while ((System.nanoTime() - started) / 1_000_000_000 < timeoutSeconds) {
            val remaining = timeoutSeconds - (System.nanoTime() - started) / 1_000_000_000
            if (process.waitFor(minOf(15, remaining.coerceAtLeast(1)), TimeUnit.SECONDS)) {
                completed = true
                break
            }
            progress("Still collecting build/dependency facts (${(System.nanoTime() - started) / 1_000_000_000}s elapsed)…")
        }
        if (!completed) {
            // Some sandboxed macOS processes cannot enumerate descendants; retain the timeout diagnosis.
            runCatching {
                process.toHandle().descendants().use { children ->
                    children.forEach { it.destroyForcibly() }
                }
            }
            process.destroyForcibly()
            process.waitFor(5, TimeUnit.SECONDS)
            throw ScanException("Build-model collection timed out after ${timeoutSeconds}s. " +
                "Downloads are cached; retry or set SCRYER_RESOLUTION_TIMEOUT_SECONDS. Last output: " + logTail(log))
        }
        if (process.exitValue() != 0) throw ScanException("Build-model collection failed (exit ${process.exitValue()}): " +
            logTail(log))
    }

    private fun logTail(log: Path) = log.readText().takeLast(1800).replace(Regex("\\u001B\\[[;\\d]*m"), "")

    fun chooseJava(facts: RepositoryFacts): String {
        System.getenv("SCRYER_JAVA_HOME")?.let { return it }
        val major = facts.builds.first().version?.substringBefore('.')?.toIntOrNull()
        if (facts.builds.first().tool == "Gradle" && major != null && major <= 6) {
            val installs = facts.root.resolve(".tooling/mise/data/installs/java")
            val java8 = findLocalJava8(installs)
            if (java8 != null) {
                return java8.toString()
            }
            throw ScanException("Gradle ${facts.builds.first().version} needs a compatible JVM; set SCRYER_JAVA_HOME (fixture-local Java 8 is auto-detected).")
        }
        return System.getenv("JAVA_HOME")?.takeIf { it.isNotBlank() } ?: System.getProperty("java.home")
    }

    private fun findLocalJava8(installs: Path): Path? {
        if (!installs.isDirectory()) {
            return null
        }
        return Files.list(installs).use { directories ->
            directories.toList().sorted().asSequence()
                .flatMap { directory -> sequenceOf(directory.resolve("Contents/Home"), directory) }
                .firstOrNull { candidate ->
                    val release = candidate.resolve("release")
                    candidate.resolve("bin/java").isRegularFile() && release.isRegularFile() &&
                        Regex("JAVA_VERSION=\"1\\.8\\.").containsMatchIn(release.readText())
                }
        }
    }
}
