package com.edwardnoaland.scryer.scan.model

/** Unknowns stay null/empty with provenance/status; neither is a claim that a fact is absent. */
data class LanguageFacts(
    val sourceVersions: List<String> = emptyList(),
    val targetVersions: List<String> = emptyList(),
    val toolchainVersions: List<String> = emptyList(),
    val previewSignal: Boolean = false,
    val sourceLanguages: List<String> = emptyList(),
    val jpmsDescriptors: List<String> = emptyList(),
)
data class ModuleFacts(val id: String, val directory: String, val definition: String)
data class SourceFacts(
    val productionRoots: List<String> = emptyList(),
    val testRoots: List<String> = emptyList(),
    val integrationRoots: List<String> = emptyList(),
    val generatedRoots: List<String> = emptyList(),
    val javaSourceFiles: Int = 0,
)
data class TestingFacts(
    val frameworks: List<String> = emptyList(),
    val mockLibraries: List<String> = emptyList(),
    val assertionLibraries: List<String> = emptyList(),
    val testSourceFiles: Int = 0,
    val integrationSourceFiles: Int = 0,
    val classification: String = "Static filename/source-root heuristics; not executed test counts",
    val disabledAnnotationSignals: Int = 0,
    val jacocoConfigured: Boolean = false,
    val coverageReports: List<String> = emptyList(),
    val testTasksAndPlugins: List<String> = emptyList(),
)
data class CompileToolingFacts(
    val annotationProcessors: List<DependencyDeclaration> = emptyList(),
    val codeGenerationSignals: List<String> = emptyList(),
    val lombokSignal: Boolean = false,
    val provenance: String = "build/source declarations; not proof that generation executed",
)
data class ParentPomFact(val module: String, val group: String?, val artifact: String?, val declaredVersion: String?)
data class SignalFact(val name: String, val locations: List<String>, val provenance: String = "source-text signal; not semantic proof")
data class VerificationFacts(
    val buildCommands: List<String> = emptyList(),
    val testCommands: List<String> = emptyList(),
    val packageCommands: List<String> = emptyList(),
    val ciFiles: List<String> = emptyList(),
    val qualityTools: List<String> = emptyList(),
    val commandsStatus: String = "conventional suggestions; not executed/verified by scan",
)
data class GraphNode(val id: String, val group: String?, val artifact: String?, val version: String?, val kind: String = "module", val type: String? = null, val classifier: String? = null, val optional: Boolean? = null)
data class GraphEdge(
    val from: String,
    val to: String?,
    val requested: String,
    val requestedVersion: String?,
    val selectedVersion: String?,
    val direct: Boolean,
    val reasons: List<String> = emptyList(),
    val conflict: Boolean? = false,
    val forced: Boolean? = false,
    val changedSelection: Boolean? = false,
    val unresolved: String? = null,
    val constraint: Boolean = false,
    val scope: String? = null,
)
data class ConfigurationGraph(
    val module: String,
    val configuration: String,
    val root: String,
    val nodes: List<GraphNode>,
    val edges: List<GraphEdge>,
    val status: String = "complete",
    val error: String? = null,
)
data class EvaluatedDependency(val configuration: String, val group: String?, val artifact: String?, val version: String?, val kind: String)
data class EvaluatedProject(
    val id: String, val directory: String, val sourceVersion: String?, val targetVersion: String?,
    val plugins: List<String>, val repositories: List<String>, val sourceSets: Map<String, List<String>>,
    val testTasks: List<String>, val declaredDependencies: List<EvaluatedDependency>, val managementSources: List<String>,
    val toolchainVersion: String? = null,
)
data class ResolutionFacts(
    val status: String = "not_requested",
    val collector: String? = null,
    val configurations: List<ConfigurationGraph> = emptyList(),
    val notes: List<String> = emptyList(),
    val metadata: Map<String, String> = emptyMap(),
    val projects: List<EvaluatedProject> = emptyList(),
) {
    val resolvedComponentCount: Int?
        get() {
            if (configurations.none { it.nodes.isNotEmpty() }) {
                return null
            }
            return configurations.flatMap { it.nodes }
                .filter { it.group != null }
                .map { "${it.group}:${it.artifact}:${it.version}" }
                .distinct()
                .size
        }

    val conflictCount: Int?
        get() {
            if (configurations.none { it.nodes.isNotEmpty() } || metadata["conflictReasons"] == "unavailable") {
                return null
            }
            return configurations.flatMap { graph ->
                graph.edges.filter { it.conflict == true }.map { "${graph.module}|${it.to}" }
            }.distinct().size
        }

    val overrideCount: Int?
        get() {
            if (configurations.none { it.nodes.isNotEmpty() } || metadata["overrideReasons"] == "unavailable") {
                return null
            }
            return configurations.flatMap { graph ->
                graph.edges.filter { it.forced == true }.map { "${graph.module}|${it.to}" }
            }.distinct().size
        }
}
