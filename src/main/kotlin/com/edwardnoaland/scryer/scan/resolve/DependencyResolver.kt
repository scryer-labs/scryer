package com.edwardnoaland.scryer.scan.resolve

import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.model.ResolutionFacts
import java.nio.file.Files

/** A build-tool adapter, not a recipe runner. Output logs never contaminate JSON/stdout. */
class DependencyResolver(
    private val timeoutSeconds: Long = System.getenv("SCRYER_RESOLUTION_TIMEOUT_SECONDS")?.toLongOrNull()?.takeIf { it > 0 } ?: 600,
    private val progress: (String) -> Unit = {},
) {
    private val process = BuildToolProcess(timeoutSeconds, progress)

    fun resolve(facts: RepositoryFacts): ResolutionFacts {
        val buildTools = facts.builds.map { it.tool }.distinct()
        if (buildTools.size != 1) {
            return ResolutionFacts(
                status = "unavailable",
                notes = listOf("Multiple build tools: resolved graph is ambiguous; select a single-build repository."),
            )
        }
        val temporary = Files.createTempDirectory("scryer-resolution-")
        return try {
            when (facts.builds.first().tool) {
                "Gradle" -> GradleModelCollector(process).collect(facts, temporary)
                else -> MavenDependencyCollector(process).collect(facts, temporary)
            }
        } catch (e: Exception) {
            ResolutionFacts("unavailable", notes = listOf("Dependency resolution unavailable: ${e.message}"))
        } finally {
            Files.walk(temporary).use { paths ->
                paths.sorted(Comparator.reverseOrder()).forEach { Files.deleteIfExists(it) }
            }
        }
    }

}
