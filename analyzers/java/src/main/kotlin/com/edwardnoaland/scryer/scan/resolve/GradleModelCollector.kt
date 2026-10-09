package com.edwardnoaland.scryer.scan.resolve

import com.edwardnoaland.scryer.scan.model.ConfigurationGraph
import com.edwardnoaland.scryer.scan.model.EvaluatedProject
import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.model.ResolutionFacts
import com.edwardnoaland.scryer.serialization.jsonMapper
import java.nio.file.Files
import java.nio.file.Path
import kotlin.io.path.*

internal class GradleModelCollector(private val process: BuildToolProcess) {
    fun collect(facts: RepositoryFacts, temp: Path): ResolutionFacts {
        val wrapper = facts.root.resolve("gradlew")
        if (!wrapper.isRegularFile()) return ResolutionFacts("unavailable", "Gradle", notes = listOf("Gradle Wrapper script missing; no global Gradle fallback."))
        val script = temp.resolve("collect.gradle")
        javaClass.getResourceAsStream("/gradle-model.gradle")!!.use { Files.copy(it, script) }
        val output = temp.resolve("graph.json")
        val cache = facts.root.resolve(".tooling/gradle").takeIf { it.isDirectory() }
            ?: Path.of(System.getenv("SCRYER_CACHE_HOME") ?: System.getProperty("java.io.tmpdir") + "/scryer-cache").resolve("gradle")
        val command = listOf("sh", wrapper.toString(), "--no-daemon", "--console=plain", "--project-cache-dir", temp.resolve("project-cache").toString(),
            "-g", cache.toString(), "-I", script.toString(), "-Dscryer.output=$output", "scryerCollectFacts")
        val javaHome = process.chooseJava(facts)
        process.execute(command, facts.root, temp, javaHome, mapOf("GRADLE_USER_HOME" to cache.toString()))
        val data = jsonMapper.readTree(output.toFile())
        val configurations = data["configurations"].map { jsonMapper.treeToValue(it, ConfigurationGraph::class.java) }
        val projects = data["projects"].map { jsonMapper.treeToValue(it, EvaluatedProject::class.java) }
        val status = if (configurations.any { it.status != "complete" }) "partial" else "complete"
        return ResolutionFacts(status, "Gradle ResolutionResult", configurations,
            notes = listOf("Target Gradle configuration evaluated; compile/tests were not run. Selection reasons do not always identify an exact BOM entry."),
            metadata = mapOf("javaHome" to javaHome, "countSemantics" to "components are unique group:artifact:version across configurations; conflicts/forced overrides are unique module/selected-component across configurations"),
            projects = projects)
    }

}
