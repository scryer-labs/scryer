package com.edwardnoaland.scryer.analyze.model

import com.edwardnoaland.scryer.analyze.GitComparison
import com.edwardnoaland.scryer.analyze.GitProcess
import com.edwardnoaland.scryer.analyze.SourceRole
import com.edwardnoaland.scryer.analyze.execute.TargetTestRunner
import com.edwardnoaland.scryer.analyze.execute.TestRunPlanner
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus
import com.edwardnoaland.scryer.serialization.jsonMapper
import java.nio.file.Files
import java.nio.file.Path

/** Snapshot-local build metadata; dependency downloads share the execution cache. */
internal class AnalysisInputCollector(private val progress: (String) -> Unit = {}) {
    fun collect(root: Path, comparison: GitComparison, sha: String): AnalysisInputs {
        return try {
            val cache = Path.of(System.getenv("SCRYER_CACHE_HOME") ?: "${System.getProperty("java.io.tmpdir")}/scryer-cache")
            Files.createDirectories(cache.resolve("runs"))
            val run = Files.createTempDirectory(cache.resolve("runs"), "model-")
            val plan = TestRunPlanner(System.getenv()).plan(root, comparison.repository, cache, run)
            val timeout = System.getenv("SCRYER_RESOLUTION_TIMEOUT_SECONDS")?.toLongOrNull() ?: 180L
            check(timeout > 0) { "Resolution timeout must be positive" }
            val gradle = Files.isRegularFile(root.resolve("gradlew"))
            val output = run.resolve("inputs.json")
            val command = if (gradle) {
                val script = run.resolve("inputs.gradle")
                javaClass.getResourceAsStream("/analyze-classpath.gradle")!!.use { Files.copy(it, script) }
                plan.command.dropLast(2) + listOf("-I", script.toString(), "-Dscryer.output=$output", "scryerAnalyzeInputs")
            } else {
                // An effective POM captures inherited/custom source roots and reactor modules.
                plan.command.dropLast(2) + listOf("help:effective-pom", "-Doutput=$output")
            }
            progress("Collecting $sha module/classpath inputs")
            val runner = TargetTestRunner(progress = progress)
            val result = runner.execute(root, plan.copy(command = command), run.resolve("model.log"), sha, timeout, "build model")
            check(result.status == TestRunStatus.SUCCEEDED) { "Build model ${result.status}; log: ${result.log}" }
            check(GitProcess.run(root, "rev-parse", "HEAD").trim() == sha && GitProcess.run(root, "diff", "HEAD", "--").isBlank() &&
                GitProcess.run(root, "diff", "--cached", "HEAD", "--").isBlank()) { "Build model changed the requested snapshot" }
            val inputs = if (gradle) readGradle(output, root) else MavenAnalysisInputs(runner, plan, run, timeout, sha).read(output, root)
            check(GitProcess.run(root, "rev-parse", "HEAD").trim() == sha && GitProcess.run(root, "diff", "HEAD", "--").isBlank() &&
                GitProcess.run(root, "diff", "--cached", "HEAD", "--").isBlank()) { "Module collection changed the requested snapshot" }
            inputs
        } catch (exception: Exception) {
            val intact = runCatching {
                GitProcess.run(root, "rev-parse", "HEAD").trim() == sha &&
                    GitProcess.run(root, "diff", "HEAD", "--").isBlank() &&
                    GitProcess.run(root, "diff", "--cached", "HEAD", "--").isBlank()
            }.getOrDefault(false)
            check(intact) { "Cannot analyze requested snapshot after build-model source/index/HEAD mutation: ${exception.message}" }
            AnalysisInputs(notes = listOf("Module/classpath resolution unavailable: ${exception.message}. Falling back to partial source attribution."))
        }
    }

    internal fun readGradle(output: Path, root: Path): AnalysisInputs {
        val modules = jsonMapper.readTree(output.toFile())["modules"].map { node ->
            val directory = Path.of(node["directory"].asText()).normalize()
            check(directory.startsWith(root.normalize())) { "External module directory unsupported: $directory" }
            val sources = node["sources"].flatMap { source ->
                val role = when (source["name"].asText()) {
                    "main" -> SourceRole.PRODUCTION
                    "test", "integrationTest", "testFixtures" -> SourceRole.TEST
                    else -> SourceRole.UNKNOWN
                }
                source["roots"].map { AnalysisSourceRoot(Path.of(it.asText()), role) }
            }
            AnalysisModule(node["id"].asText(), directory, sources,
                node["classpath"].map { Path.of(it.asText()) }, node["dependencies"].map { it.asText() }, node["notes"].map { it.asText() })
        }
        return AnalysisInputs(modules, listOf("Build-native module inputs; source-set compile classpaths are combined within each module. Annotation processing, generated sources and target JDK boot APIs are not reproduced."))
    }
}
