package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.scan.model.DependencyDeclaration
import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.github.ajalt.mordant.rendering.TextColors
import com.github.ajalt.mordant.rendering.TextStyles
import java.io.PrintStream

class ScanRenderer(private val out: PrintStream, private val color: Boolean) {
    private fun header(text: String) { out.println(if (color) (TextColors.cyan + TextStyles.bold)(text) else text) }
    private fun dim(text: String) = if (color) TextStyles.dim(text) else text
    private fun warning(text: String) = if (color) TextColors.yellow(text) else text
    private fun bad(text: String) = if (color) TextColors.red(text) else text
    private fun good(text: String) = if (color) TextColors.green(text) else text
    private fun row(key: String, value: String) { out.println("  ${dim(key.padEnd(25) + " ")}$value") }
    private fun unknown(value: String?) = value ?: bad("unknown")

    fun remoteVersions(versions: com.edwardnoaland.scryer.scan.model.RemoteVersions) {
        header("\nRemote dependency releases")
        out.println("  ${versions.source}")
        out.println("  Checked: ${versions.checkedAt}")
        for ((group, dependencies) in versions.dependencies.groupBy { it.module to it.configuration }) {
            out.println("\n  ${group.first} / ${group.second}")
            dependencies.forEach { dependency ->
                val comparison = "${dependency.current ?: "unknown"} → ${dependency.latest ?: "unavailable"} [${dependency.status}]"
                val styled = when (dependency.status) {
                    "CURRENT" -> good(comparison)
                    "UPDATE_AVAILABLE", "CURRENT_AHEAD" -> warning(comparison)
                    else -> bad(comparison)
                }
                out.println("    ${dependency.coordinate} (${dependency.currentSource}): $styled")
                dependency.note?.let { out.println("      $it") }
            }
        }
    }

    fun summary(facts: RepositoryFacts) {
        renderProject(facts)
        renderDependencySummary(facts)
        renderTesting(facts)
        renderCodeCharacteristics(facts)
        renderVerification(facts)
        renderNotes(facts)
    }

    private fun renderProject(facts: RepositoryFacts) {
        header("Project")
        row("Repository", dim(facts.root.toString()))
        row("Java source / target", "${unknown(facts.language.sourceVersions.takeIf { it.isNotEmpty() }?.joinToString())} / ${unknown(facts.language.targetVersions.takeIf { it.isNotEmpty() }?.joinToString())}")
        row("Toolchain", unknown(facts.language.toolchainVersions.takeIf { it.isNotEmpty() }?.joinToString()))
        facts.selectedBuildTool?.let { row("Selected build tool", "$it (explicit)") }
        row("Build", facts.builds.joinToString { "${it.tool} ${unknown(it.version)}" })
        row("Wrapper", facts.builds.joinToString { "${it.tool}: ${if (it.wrapperPresent) "present" else "not found"}" })
        row("Modules", (facts.resolution.projects.map { it.id }.distinct().size.takeIf { it > 0 }
            ?: facts.modules.map { it.directory }.distinct().size).toString())
        facts.plugins.firstOrNull { it.id == "org.springframework.boot" }?.let { row("Spring Boot (plugin)", unknown(it.version)) }
    }

    private fun renderDependencySummary(facts: RepositoryFacts) {
        header("\nDependencies")
        row("Direct (observed)", directDependencies(facts).map { "${it.module}|${it.notation}" }.distinct().size.toString())
        row("Resolved components", unknown(facts.resolution.resolvedComponentCount?.toString()))
        val graphs = facts.resolution.configurations
        val directIds = graphs.flatMap { it.edges.filter { e -> e.direct }.mapNotNull { e -> e.to } }.toSet()
        val transitiveIds = graphs.flatMap { it.nodes.filter { n -> n.group != null }.map { n -> n.id } }.toSet() - directIds
        row("Transitive-only", unknown(if (facts.resolution.resolvedComponentCount != null) transitiveIds.size.toString() else null))
        row("Conflict selections", facts.resolution.conflictCount?.let { if (it > 0) warning(it.toString()) else it.toString() } ?: bad("unavailable"))
        row("Forced overrides", facts.resolution.overrideCount?.let { if (it > 0) warning(it.toString()) else it.toString() } ?: bad("unavailable"))
        row("Resolution", if (facts.resolution.status == "complete") good("complete") else warning(facts.resolution.status))
        graphs.filter { it.error != null }.forEach {
            row("Resolution failure", bad("${it.module}: ${it.error}"))
        }
        val key = directDependencies(facts).distinctBy { it.notation }.filter {
            Regex("spring|hibernate|jackson|servlet|guava|commons-lang").containsMatchIn(it.notation)
        }.take(5)
        if (key.isNotEmpty()) {
            header("\nKey dependencies")
            key.forEach { row(it.artifact, selectedVersions(facts, it).ifEmpty { unknown(it.version) }) }
        }
    }

    private fun renderTesting(facts: RepositoryFacts) {
        header("\nTesting")
        row("Framework", facts.testing.frameworks.joinToString().ifEmpty { "not detected" })
        row("Mockito", resolvedLibrary(facts, "org.mockito", "mockito-core") ?: facts.testing.mockLibraries.joinToString().ifEmpty { "not detected" })
        row("Unit source files", facts.testing.testSourceFiles.toString())
        row("Integration files", facts.testing.integrationSourceFiles.toString())
        row("JaCoCo", if (facts.testing.jacocoConfigured) good("configured") else "not detected")
        row("Disabled signals", facts.testing.disabledAnnotationSignals.toString())
        out.println(dim("  Test counts are static source-file classifications, not execution evidence."))
    }

    private fun renderCodeCharacteristics(facts: RepositoryFacts) {
        header("\nCode Characteristics")
        for (name in listOf("javax usage", "jakarta usage", "Lombok", "Reflection", "ServiceLoader")) {
            val detected = facts.signals.any { it.name == name && it.locations.isNotEmpty() } ||
                (name == "Lombok" && facts.dependencies.any { "lombok" in it.notation })
            row(name, if (detected) warning("detected") else "not detected")
        }
        row("Annotation processors", (facts.compileTooling.annotationProcessors + directDependencies(facts).filter { "annotationProcessor" in it.configuration || "kapt" in it.configuration }).map { it.notation }.distinct().size.toString() + " observed")
        row("Generated roots", facts.sources.generatedRoots.size.toString() + " existing")
    }

    private fun renderVerification(facts: RepositoryFacts) {
        header("\nVerification (suggested commands; may be incomplete or inaccurate)")
        row("Build", facts.verification.buildCommands.joinToString(" / "))
        row("Test", facts.verification.testCommands.joinToString(" / "))
        row("CI configs", facts.verification.ciFiles.size.toString() + " found")
        out.println(dim("  Commands inferred from build conventions; scan does not verify build/test success."))
    }

    private fun renderNotes(facts: RepositoryFacts) {
        val notes = (facts.notes + facts.resolution.notes).distinct()
        if (notes.isNotEmpty()) {
            header("\nCollection notes")
            notes.take(4).forEach { out.println("  " + warning(it.replace('\n', ' ').take(240))) }
            if (notes.size > 4) out.println(dim("  ${notes.size - 4} more notes; --json retains all."))
        }
    }

    fun dependencies(facts: RepositoryFacts) {
        header("\nDirect dependencies")
        for ((key, dependencies) in directDependencies(facts).groupBy { "${it.module} / ${it.configuration}" }) {
            out.println(dim(key))
            for (dep in dependencies) {
                val selected = selectedVersions(facts, dep)
                out.println("  ${dep.notation}:${dep.version ?: "<managed/unspecified>"}" +
                    (dep.classifier?.let { " [classifier $it]" } ?: "") +
                    (dep.type?.takeIf { it != "jar" }?.let { " [type $it]" } ?: "") +
                    if (selected.isNotEmpty()) "  selected: $selected" else "  " + bad("resolved version unavailable"))
                if (dep.managementSource != null) out.println(dim("    managed by ${dep.managementSource}"))
            }
        }
        if (directDependencies(facts).isEmpty()) out.println(bad("  No direct declarations collected; inspect collection notes."))
    }

    fun tree(facts: RepositoryFacts) {
        header("\nResolved dependency tree")
        if (facts.resolution.configurations.isEmpty()) { out.println(bad("  Unavailable: ${facts.resolution.status}")); return }
        for (graph in facts.resolution.configurations) {
            out.println(dim("${graph.module} / ${graph.configuration} [${graph.status}]"))
            if (graph.error != null) out.println(bad("  ${graph.error}"))
            val expanded = mutableSetOf<String>()
            val byParent = graph.edges.groupBy { it.from }
            fun children(parent: String, prefix: String) {
                val edges = byParent[parent].orEmpty()
                for ((index, edge) in edges.withIndex()) {
                    val last = index == edges.lastIndex
                    val branch = if (last) "└── " else "├── "
                    val node = graph.nodes.firstOrNull { it.id == edge.to }
                    val selected = node?.let { if (it.group != null) "${it.group}:${it.artifact}:${it.version}" else it.id }
                    val label = selected ?: edge.requested
                    val annotation = when {
                        edge.unresolved != null -> bad(" [unresolved: ${edge.unresolved.take(160)}]")
                        edge.conflict == true -> warning(" [conflict; requested ${edge.requestedVersion}; selected ${edge.selectedVersion}; ${edge.reasons.joinToString()}]")
                        edge.forced == true -> warning(" [forced; requested ${edge.requestedVersion}; selected ${edge.selectedVersion}]")
                        edge.changedSelection == true -> warning(" [requested ${edge.requestedVersion}; selected ${edge.selectedVersion}; ${edge.reasons.joinToString()}]")
                        edge.constraint -> dim(" [constraint]")
                        else -> ""
                    }
                    val repeated = edge.to != null && !expanded.add(edge.to)
                    out.println(prefix + branch + label + annotation + if (repeated) dim(" [already shown]") else "")
                    if (!repeated && edge.to != null) children(edge.to, prefix + if (last) "    " else "│   ")
                }
            }
            expanded.add(graph.root)
            children(graph.root, "")
        }
    }
}

internal fun directDependencies(facts: RepositoryFacts): List<DependencyDeclaration> {
    val literal = facts.dependencies.filter { it.kind != "managed declaration" }
    val evaluated = facts.resolution.projects.flatMap { project -> project.declaredDependencies.filter { it.kind == "dependency" && it.group != null }.map { dep ->
        DependencyDeclaration(dep.configuration, "${dep.group}:${dep.artifact}", dep.version, "evaluated Gradle model", module = project.id)
    } }
    // Preserve original declared versions/provenance; evaluated declarations fill unsupported DSL gaps.
    val missing = evaluated.filter { candidate -> literal.none { it.module == candidate.module && it.configuration == candidate.configuration &&
        it.notation == candidate.notation && it.version == candidate.version } }
    return (literal + missing).distinctBy { "${it.module}|${it.configuration}|${it.notation}|${it.version}|${it.type}|${it.classifier}" }
}
internal fun selectedVersions(facts: RepositoryFacts, dep: DependencyDeclaration): String = facts.resolution.configurations
    .filter { it.module == dep.module }.flatMap { it.nodes }.filter { it.group == dep.group && it.artifact == dep.artifact }
    .mapNotNull { it.version }.distinct().sorted().joinToString(" / ")
internal fun resolvedLibrary(facts: RepositoryFacts, group: String, artifact: String): String? = facts.resolution.configurations.flatMap { it.nodes }
    .filter { it.group == group && it.artifact == artifact }.mapNotNull { it.version }.distinct().takeIf { it.isNotEmpty() }?.joinToString(" / ")
