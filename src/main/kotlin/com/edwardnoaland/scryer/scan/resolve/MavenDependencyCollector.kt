package com.edwardnoaland.scryer.scan.resolve

import com.edwardnoaland.scryer.scan.model.ConfigurationGraph
import com.edwardnoaland.scryer.scan.model.GraphEdge
import com.edwardnoaland.scryer.scan.model.ModuleFacts
import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.model.ResolutionFacts
import com.edwardnoaland.scryer.serialization.jsonMapper
import java.nio.file.Path
import kotlin.io.path.isRegularFile

internal class MavenDependencyCollector(private val process: BuildToolProcess) {
    fun collect(facts: RepositoryFacts, temp: Path): ResolutionFacts {
        val wrapper = facts.root.resolve("mvnw")
        if (!wrapper.isRegularFile()) {
            return ResolutionFacts(
                status = "unavailable",
                collector = "Maven",
                notes = listOf("Maven Wrapper script missing; no global Maven fallback."),
            )
        }

        val modules = facts.modules.filter { it.definition == "pom.xml" }
        // Each module gets its own output, avoiding reactor modules overwriting one JSON file.
        val graphs = modules.mapIndexed { index, module ->
            collectModule(facts, module, wrapper, temp, index)
        }
        val hasCollectedTrees = graphs.any { it.nodes.isNotEmpty() }
        val failures = graphs.mapNotNull { graph ->
            graph.error?.let { error -> "Maven module ${graph.module}: $error" }
        }
        val limitations = if (hasCollectedTrees) {
            listOf("Resolved Maven trees collected. Exact conflict/override reasons and effective BOM provenance are unavailable from JSON tree; null counts do not mean zero.")
        } else {
            emptyList()
        }

        return ResolutionFacts(
            status = if (hasCollectedTrees) "partial" else "unavailable",
            collector = "Maven dependency:tree 3.8.1",
            configurations = graphs,
            notes = failures + limitations,
            metadata = mapOf("conflictReasons" to "unavailable", "overrideReasons" to "unavailable"),
        )
    }

    private fun collectModule(
        facts: RepositoryFacts,
        module: ModuleFacts,
        wrapper: Path,
        temp: Path,
        index: Int,
    ): ConfigurationGraph {
        process.report("Resolving Maven dependencies for ${module.id} (first scan may download dependencies; timeout ${process.timeoutSeconds}s)…")
        val output = temp.resolve("maven-$index.json")
        val cache = Path.of(
            System.getenv("SCRYER_CACHE_HOME") ?: System.getProperty("java.io.tmpdir") + "/scryer-cache",
        ).resolve("maven")
        val pom = facts.root.resolve(module.directory).resolve("pom.xml")
        val command = listOf(
            "sh", wrapper.toString(), "-B", "-N", "-f", pom.toString(),
            "-Dmaven.repo.local=$cache",
            "org.apache.maven.plugins:maven-dependency-plugin:3.8.1:tree",
            "-DoutputType=json", "-DoutputFile=$output",
        )

        return try {
            process.execute(command, facts.root, temp, process.chooseJava(facts))
            val graph = readMavenTree(jsonMapper.readTree(output.toFile()), module.id)
            graph.copy(edges = graph.edges.map { edge -> enrichRequestedVersion(edge, facts, module) })
        } catch (exception: Exception) {
            ConfigurationGraph(
                module = module.id,
                configuration = "Maven dependency tree",
                root = "project:${module.id}",
                nodes = emptyList(),
                edges = emptyList(),
                status = "failed",
                error = exception.message,
            )
        }
    }

    private fun enrichRequestedVersion(
        edge: GraphEdge,
        facts: RepositoryFacts,
        module: ModuleFacts,
    ): GraphEdge {
        val declaration = facts.dependencies.firstOrNull {
            it.module == module.id && it.notation == edge.requested && it.kind != "managed declaration"
        }
        val requestedVersion = declaration?.version
        if (!edge.direct || requestedVersion == null) {
            return edge
        }

        return edge.copy(
            requestedVersion = requestedVersion,
            changedSelection = '$' !in requestedVersion && requestedVersion != edge.selectedVersion,
            reasons = edge.reasons + "Requested version from local root declaration/management",
        )
    }
}
