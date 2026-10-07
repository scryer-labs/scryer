package com.edwardnoaland.scryer

import com.fasterxml.jackson.databind.JsonNode
import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import java.nio.file.Files
import java.nio.file.Path
import java.util.concurrent.TimeUnit
import kotlin.io.path.*

internal val jsonMapper = jacksonObjectMapper()

/** A build-tool adapter, not a recipe runner. Output logs never contaminate JSON/stdout. */
class DependencyResolver(
    private val timeoutSeconds: Long = System.getenv("SCRYER_RESOLUTION_TIMEOUT_SECONDS")?.toLongOrNull()?.takeIf { it > 0 } ?: 600,
    private val progress: (String) -> Unit = {},
) {
    fun resolve(facts: RepositoryFacts): ResolutionFacts {
        if (facts.builds.map { it.tool }.distinct().size != 1) return ResolutionFacts("unavailable", notes = listOf("Multiple build tools: resolved graph is ambiguous; select a single-build repository."))
        val temporary = Files.createTempDirectory("scryer-resolution-")
        return try {
            when (facts.builds.first().tool) {
                "Gradle" -> gradle(facts, temporary)
                else -> maven(facts, temporary)
            }
        } catch (e: Exception) {
            ResolutionFacts("unavailable", notes = listOf("Dependency resolution unavailable: ${e.message}"))
        } finally {
            Files.walk(temporary).use { paths -> paths.sorted(Comparator.reverseOrder()).forEach { Files.deleteIfExists(it) } }
        }
    }

    private fun gradle(facts: RepositoryFacts, temp: Path): ResolutionFacts {
        val wrapper = facts.root.resolve("gradlew")
        if (!wrapper.isRegularFile()) return ResolutionFacts("unavailable", "Gradle", notes = listOf("Gradle Wrapper script missing; no global Gradle fallback."))
        val script = temp.resolve("collect.gradle")
        javaClass.getResourceAsStream("/gradle-model.gradle")!!.use { Files.copy(it, script) }
        val output = temp.resolve("graph.json")
        val cache = facts.root.resolve(".tooling/gradle").takeIf { it.isDirectory() }
            ?: Path.of(System.getenv("SCRYER_CACHE_HOME") ?: System.getProperty("java.io.tmpdir") + "/scryer-cache").resolve("gradle")
        val command = listOf("sh", wrapper.toString(), "--no-daemon", "--console=plain", "--project-cache-dir", temp.resolve("project-cache").toString(),
            "-g", cache.toString(), "-I", script.toString(), "-Dscryer.output=$output", "scryerCollectFacts")
        val javaHome = chooseJava(facts)
        execute(command, facts.root, temp, javaHome, mapOf("GRADLE_USER_HOME" to cache.toString()))
        val data = jsonMapper.readTree(output.toFile())
        val configurations = data["configurations"].map { jsonMapper.treeToValue(it, ConfigurationGraph::class.java) }
        val projects = data["projects"].map { jsonMapper.treeToValue(it, EvaluatedProject::class.java) }
        val status = if (configurations.any { it.status != "complete" }) "partial" else "complete"
        return ResolutionFacts(status, "Gradle ResolutionResult", configurations,
            notes = listOf("Target Gradle configuration evaluated; compile/tests were not run. Selection reasons do not always identify an exact BOM entry."),
            metadata = mapOf("javaHome" to javaHome, "countSemantics" to "components are unique group:artifact:version across configurations; conflicts/forced overrides are unique module/selected-component across configurations"),
            projects = projects)
    }

    private fun maven(facts: RepositoryFacts, temp: Path): ResolutionFacts {
        val wrapper = facts.root.resolve("mvnw")
        if (!wrapper.isRegularFile()) return ResolutionFacts("unavailable", "Maven", notes = listOf("Maven Wrapper script missing; no global Maven fallback."))
        val graphs = mutableListOf<ConfigurationGraph>()
        // One module invocation per output avoids overwriting a reactor's shared JSON file.
        for (module in facts.modules.filter { it.definition == "pom.xml" }) {
            progress("Resolving Maven dependencies for ${module.id} (first scan may download dependencies; timeout ${timeoutSeconds}s)…")
            val output = temp.resolve("maven-${graphs.size}.json")
            val cache = Path.of(System.getenv("SCRYER_CACHE_HOME") ?: System.getProperty("java.io.tmpdir") + "/scryer-cache").resolve("maven")
            try {
                execute(listOf("sh", wrapper.toString(), "-B", "-N", "-f", facts.root.resolve(module.directory).resolve("pom.xml").toString(),
                    "-Dmaven.repo.local=$cache", "org.apache.maven.plugins:maven-dependency-plugin:3.8.1:tree",
                    "-DoutputType=json", "-DoutputFile=$output"), facts.root, temp, chooseJava(facts))
                val graph = readMavenTree(jsonMapper.readTree(output.toFile()), module.id)
                graphs += graph.copy(edges = graph.edges.map { edge ->
                    val declaration = facts.dependencies.firstOrNull { it.module == module.id && it.notation == edge.requested && it.kind != "managed declaration" }
                    if (!edge.direct || declaration?.version == null) edge else edge.copy(
                        requestedVersion = declaration.version,
                        changedSelection = '$' !in declaration.version && declaration.version != edge.selectedVersion,
                        reasons = edge.reasons + "Requested version from local root declaration/management")
                })
            } catch (e: Exception) {
                graphs += ConfigurationGraph(module.id, "Maven dependency tree", "project:${module.id}", emptyList(), emptyList(), "failed", e.message)
            }

        }
        return ResolutionFacts(if (graphs.any { it.nodes.isNotEmpty() }) "partial" else "unavailable", "Maven dependency:tree 3.8.1", graphs,
            notes = graphs.mapNotNull { it.error?.let { error -> "Maven module ${it.module}: $error" } } +
                if (graphs.any { it.nodes.isNotEmpty() }) listOf("Resolved Maven trees collected. Exact conflict/override reasons and effective BOM provenance are unavailable from JSON tree; null counts do not mean zero.") else emptyList(),
            metadata = mapOf("conflictReasons" to "unavailable", "overrideReasons" to "unavailable"))
    }

    private fun execute(command: List<String>, root: Path, temp: Path, javaHome: String, environment: Map<String, String> = emptyMap()) {
        val log = temp.resolve("resolver.log")
        val builder = ProcessBuilder(command).directory(root.toFile()).redirectErrorStream(true).redirectOutput(log.toFile())
        builder.environment()["JAVA_HOME"] = javaHome
        builder.environment().putAll(environment)
        builder.environment().putIfAbsent("MAVEN_USER_HOME", Path.of(System.getenv("SCRYER_CACHE_HOME") ?: System.getProperty("java.io.tmpdir") + "/scryer-cache").resolve("maven-home").toString())
        val process = builder.start()
        val started = System.nanoTime()
        var completed = false
        while ((System.nanoTime() - started) / 1_000_000_000 < timeoutSeconds) {
            val remaining = timeoutSeconds - (System.nanoTime() - started) / 1_000_000_000
            if (process.waitFor(minOf(15, remaining.coerceAtLeast(1)), TimeUnit.SECONDS)) { completed = true; break }
            progress("Still collecting build/dependency facts (${(System.nanoTime() - started) / 1_000_000_000}s elapsed)…")
        }
        if (!completed) {
            // Some sandboxed macOS processes cannot enumerate descendants; retain the timeout diagnosis.
            runCatching { process.toHandle().descendants().use { children -> children.forEach { it.destroyForcibly() } } }
            process.destroyForcibly()
            process.waitFor(5, TimeUnit.SECONDS)
            throw ScanException("Build-model collection timed out after ${timeoutSeconds}s. " +
                "Downloads are cached; retry or set SCRYER_RESOLUTION_TIMEOUT_SECONDS. Last output: " + logTail(log))
        }
        if (process.exitValue() != 0) throw ScanException("Build-model collection failed (exit ${process.exitValue()}): " +
            logTail(log))
    }

    private fun logTail(log: Path) = log.readText().takeLast(1800).replace(Regex("\\u001B\\[[;\\d]*m"), "")

    private fun chooseJava(facts: RepositoryFacts): String {
        System.getenv("SCRYER_JAVA_HOME")?.let { return it }
        val major = facts.builds.first().version?.substringBefore('.')?.toIntOrNull()
        if (facts.builds.first().tool == "Gradle" && major != null && major <= 6) {
            val installs = facts.root.resolve(".tooling/mise/data/installs/java")
            if (installs.isDirectory()) Files.list(installs).use { dirs ->
                for (dir in dirs.toList().sorted()) for (candidate in listOf(dir.resolve("Contents/Home"), dir)) {
                    if (candidate.resolve("bin/java").isRegularFile() && candidate.resolve("release").isRegularFile() &&
                        Regex("JAVA_VERSION=\"1\\.8\\.").containsMatchIn(candidate.resolve("release").readText())) return candidate.toString()
                }
            }
            throw ScanException("Gradle ${facts.builds.first().version} needs a compatible JVM; set SCRYER_JAVA_HOME (fixture-local Java 8 is auto-detected).")
        }
        return System.getenv("JAVA_HOME")?.takeIf { it.isNotBlank() } ?: System.getProperty("java.home")
    }
}

internal fun readMavenTree(root: JsonNode, module: String): ConfigurationGraph {
    val nodes = linkedMapOf<String, GraphNode>()
    val edges = mutableListOf<GraphEdge>()
    fun id(node: JsonNode) = listOf("groupId", "artifactId", "type", "classifier", "version").joinToString(":") { node[it]?.asText().orEmpty() }
    val rootId = "project:$module"
    nodes[rootId] = GraphNode(rootId, null, null, null, "project")
    fun traverse(node: JsonNode, parent: String, depth: Int) {
        val nodeId = id(node)
        nodes[nodeId] = GraphNode(nodeId, node["groupId"]?.asText(), node["artifactId"]?.asText(), node["version"]?.asText(),
            type = node["type"]?.asText(), classifier = node["classifier"]?.asText(), optional = node["optional"]?.asBoolean())
        edges += GraphEdge(parent, nodeId, "${node["groupId"].asText()}:${node["artifactId"].asText()}", null,
            node["version"]?.asText(), depth == 0, conflict = null, forced = null, changedSelection = null, reasons = listOf("Maven selected tree entry; requested version/selection reason not supplied"), scope = node["scope"]?.asText())
        node["children"]?.forEach { traverse(it, nodeId, depth + 1) }
    }
    root["children"]?.forEach { traverse(it, rootId, 0) }
    return ConfigurationGraph(module, "Maven dependency tree (scope on nodes)", rootId, nodes.values.toList(), edges, "partial")
}
