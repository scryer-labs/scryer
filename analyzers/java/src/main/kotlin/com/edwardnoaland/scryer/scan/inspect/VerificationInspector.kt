package com.edwardnoaland.scryer.scan.inspect

import com.edwardnoaland.scryer.scan.model.ModuleFacts
import com.edwardnoaland.scryer.scan.model.VerificationFacts
import java.nio.file.Path
import kotlin.io.path.isRegularFile

/** Conventional suggestions only; file presence does not verify that these commands work. */
internal fun inspectVerification(
    root: Path,
    modules: List<ModuleFacts>,
    paths: List<Path>,
    tools: List<String>,
): VerificationFacts {
    val hasGradle = modules.any { it.definition.endsWith("gradle") || it.definition.endsWith("gradle.kts") }
    val hasMaven = modules.any { it.definition == "pom.xml" }
    val gradleCommand = when {
        // Retained fixture convention, not general script discovery.
        root.resolve("scripts/gradle-java8").isRegularFile() -> "./scripts/gradle-java8"
        root.resolve("gradlew").isRegularFile() -> "./gradlew"
        else -> "gradle"
    }
    val mavenCommand = if (root.resolve("mvnw").isRegularFile()) "./mvnw" else "mvn"
    val knownCiFiles = listOf(".gitlab-ci.yml", "Jenkinsfile", "azure-pipelines.yml", ".circleci/config.yml")
    val ciFiles = paths.map { root.relativize(it).toString().replace('\\', '/') }
        .filter { it.startsWith(".github/workflows/") || it in knownCiFiles }

    return VerificationFacts(
        buildCommands = listOfNotNull(
            if (hasGradle) "$gradleCommand build" else null,
            if (hasMaven) "$mavenCommand verify" else null,
        ),
        testCommands = listOfNotNull(
            if (hasGradle) "$gradleCommand test" else null,
            if (hasMaven) "$mavenCommand test" else null,
        ),
        packageCommands = listOfNotNull(
            if (hasGradle) "$gradleCommand assemble" else null,
            if (hasMaven) "$mavenCommand package" else null,
        ),
        ciFiles = ciFiles,
        qualityTools = tools.filter { it !in listOf("jacoco", "surefire", "failsafe") },
    )
}
