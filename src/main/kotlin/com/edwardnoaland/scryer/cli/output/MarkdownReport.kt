package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import java.io.ByteArrayOutputStream
import java.io.PrintStream

/** A portable facts report, independent of terminal widths and ANSI styling. */
internal class MarkdownReport {
    fun render(facts: RepositoryFacts, includeTree: Boolean = false): String = buildString {
        appendLine("# Scryer repository scan")
        appendLine()
        appendLine("Repository facts and collection limitations; not build/test execution evidence.")
        appendLine()
        section("Project")
        table(
            listOf("Fact", "Value"),
            listOf(
                listOf("Repository", facts.root.toString()),
                listOf("Java source", known(facts.language.sourceVersions)),
                listOf("Java target", known(facts.language.targetVersions)),
                listOf("Toolchain", known(facts.language.toolchainVersions)),
                listOf("Build", facts.builds.joinToString { "${it.tool} ${it.version ?: "unknown"}" }),
                listOf("Selected build tool", facts.selectedBuildTool ?: "automatic"),
                listOf("Wrapper", facts.builds.joinToString { "${it.tool}: ${if (it.wrapperPresent) "present" else "not found"}" }),
                listOf("Frameworks", known(facts.frameworks)),
                listOf("Resolution", facts.resolution.status),
            ),
        )
        section("Modules")
        val modules = if (facts.resolution.projects.isNotEmpty()) {
            facts.resolution.projects.map { listOf(it.id, it.directory) }
        } else {
            facts.modules.map { listOf(it.id, it.directory) }
        }
        table(listOf("Module", "Directory"), modules.distinct())

        section("Dependencies")
        val declarations = directDependencies(facts)
        val graphs = facts.resolution.configurations
        val directIds = graphs.flatMap { it.edges.filter { edge -> edge.direct }.mapNotNull { edge -> edge.to } }.toSet()
        val componentIds = graphs.flatMap { it.nodes.filter { node -> node.group != null }.map { node -> node.id } }.toSet()
        val transitiveCount = facts.resolution.resolvedComponentCount?.let { (componentIds - directIds).size }
        table(
            listOf("Fact", "Value"),
            listOf(
                listOf("Direct (observed)", declarations.map { "${it.module}|${it.notation}" }.distinct().size.toString()),
                listOf("Resolved components", facts.resolution.resolvedComponentCount?.toString() ?: "unknown"),
                listOf("Transitive-only", transitiveCount?.toString() ?: "unknown"),
                listOf("Conflict selections", facts.resolution.conflictCount?.toString() ?: "unavailable"),
                listOf("Forced overrides", facts.resolution.overrideCount?.toString() ?: "unavailable"),
            ),
        )
        appendLine("Declared versions are local observations; selected versions come from build-tool resolution.")
        appendLine()
        table(
            listOf("Module", "Configuration", "Dependency", "Declared version", "Selected version", "Source"),
            declarations.map { dependency ->
                listOf(
                    dependency.module, dependency.configuration, dependency.notation,
                    dependency.declaredVersion ?: "managed/unspecified",
                    selectedVersions(facts, dependency).ifEmpty { "unavailable" },
                    dependency.source,
                )
            },
        )

        facts.remoteVersions?.let { remote ->
            section("Remote dependency releases")
            appendLine(cell(remote.source))
            appendLine()
            appendLine("Checked: ${cell(remote.checkedAt)}. This lookup does not establish upgrade compatibility; metadata releases may include prereleases.")
            appendLine()
            for ((group, dependencies) in remote.dependencies.groupBy { it.module to it.configuration }) {
                appendLine("### ${cell(group.first)} / ${cell(group.second)}")
                appendLine()
                table(listOf("Dependency", "Current", "Current source", "Remote release", "Comparison", "Metadata / limitation"),
                    dependencies.map { dependency -> listOf(dependency.coordinate,
                        dependency.current ?: "unknown", dependency.currentSource, dependency.latest ?: "unavailable", dependency.status,
                        listOfNotNull(dependency.metadataUrl, dependency.note).joinToString("; ")) })
            }
        }

        section("Testing")
        table(
            listOf("Fact", "Value"),
            listOf(
                listOf("Frameworks", known(facts.testing.frameworks)),
                listOf("Mock libraries", known(facts.testing.mockLibraries)),
                listOf("Assertion libraries", known(facts.testing.assertionLibraries)),
                listOf("Unit source files", facts.testing.testSourceFiles.toString()),
                listOf("Integration source files", facts.testing.integrationSourceFiles.toString()),
                listOf("Disabled annotation signals", facts.testing.disabledAnnotationSignals.toString()),
                listOf("JaCoCo", if (facts.testing.jacocoConfigured) "configured" else "not detected"),
                listOf("Existing coverage reports", known(facts.testing.coverageReports)),
            ),
        )
        appendLine("Test counts classify source files; they do not count executed tests or prove coverage.")
        appendLine()
        section("Source layout")
        table(
            listOf("Kind", "Existing roots"),
            listOf(
                listOf("Production", known(facts.sources.productionRoots)),
                listOf("Tests", known(facts.sources.testRoots)),
                listOf("Integration", known(facts.sources.integrationRoots)),
                listOf("Generated", known(facts.sources.generatedRoots)),
            ),
        )
        section("Code characteristics")
        table(
            listOf("Source-text signal", "Locations (not execution evidence)"),
            facts.signals.map { listOf(it.name, it.locations.joinToString(", ").ifEmpty { "not detected" }) },
        )
        section("Verification (suggested commands; may be incomplete or inaccurate)")
        table(
            listOf("Kind", "Suggestion / observation"),
            listOf(
                listOf("Build", known(facts.verification.buildCommands)),
                listOf("Test", known(facts.verification.testCommands)),
                listOf("Package", known(facts.verification.packageCommands)),
                listOf("CI configurations", known(facts.verification.ciFiles)),
            ),
        )
        appendLine("Commands follow build conventions; scan does not verify build/test success.")
        appendLine()
        section("Collection notes and failures")
        val failures = graphs.mapNotNull { graph -> graph.error?.let { "${graph.module} / ${graph.configuration}: $it" } }
        val notes = (facts.notes + facts.resolution.notes + failures).distinct()
        if (notes.isEmpty()) {
            appendLine("No additional collection notes.")
        } else {
            notes.forEach { appendLine("- ${cell(it)}") }
        }
        appendLine()
        if (includeTree) {
            section("Resolved dependency tree")
            val buffer = ByteArrayOutputStream()
            PrintStream(buffer, true, Charsets.UTF_8).use { ScanRenderer(it, color = false).tree(facts) }
            val tree = buffer.toString(Charsets.UTF_8)
            val longestFence = Regex("`+").findAll(tree).maxOfOrNull { it.value.length } ?: 0
            val fence = "`".repeat(maxOf(3, longestFence + 1))
            appendLine("${fence}text")
            append(tree)
            appendLine(fence)
        }
    }

    private fun known(values: List<String>) = values.joinToString(", ").ifEmpty { "unknown/not observed" }

    private fun StringBuilder.section(title: String) {
        appendLine("## $title")
        appendLine()
    }

    private fun StringBuilder.table(headers: List<String>, rows: List<List<String>>) {
        appendLine(headers.joinToString(" | ", prefix = "| ", postfix = " |") { cell(it) })
        appendLine(headers.joinToString(" | ", prefix = "| ", postfix = " |") { "---" })
        rows.forEach { row ->
            appendLine(row.joinToString(" | ", prefix = "| ", postfix = " |") { cell(it) })
        }
        appendLine()
    }

    private fun cell(value: String): String = value
        .replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace("\\", "\\\\")
        .replace("|", "\\|")
        .replace("\r\n", "\n")
        .replace("\r", "\n")
        .replace("\n", "<br>")
}
