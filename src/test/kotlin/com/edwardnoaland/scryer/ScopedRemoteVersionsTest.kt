package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.output.MarkdownReport
import com.edwardnoaland.scryer.cli.output.ScanRenderer
import com.edwardnoaland.scryer.scan.model.*
import com.edwardnoaland.scryer.scan.remote.*
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import java.nio.file.Path
import org.junit.jupiter.api.Test
import kotlin.test.*

class ScopedRemoteVersionsTest {
    private fun dependency(scope: String, module: String = ":") = DependencyDeclaration(scope, "example:lib", "0.5", "build file", module = module)
    private fun node(version: String) = GraphNode("lib-$version", "example", "lib", version)
    private fun facts(graphs: List<ConfigurationGraph>) = RepositoryFacts(Path.of("."), emptyList(), resolution = ResolutionFacts(configurations = graphs))

    @Test fun `Maven scopes isolate selected versions and ignore transitive or other module matches`() {
        val nodes = listOf(node("1"), node("2"), node("3"))
        val graph = ConfigurationGraph(":", "Maven dependency tree (scope on nodes)", "root", nodes,
            listOf(GraphEdge("root", "lib-1", "example:lib", null, "1", true, scope = "compile"),
                GraphEdge("root", "lib-2", "example:lib", null, "2", true, scope = "test"),
                GraphEdge("nested", "lib-3", "example:lib", null, "3", false, scope = "runtime")))
        val facts = facts(listOf(graph, graph.copy(module = ":other")))
        assertEquals(listOf("1"), scopedSelectedVersions(facts, dependency("compile")))
        assertEquals(listOf("2"), scopedSelectedVersions(facts, dependency("test")))
        assertTrue(scopedSelectedVersions(facts, dependency("runtime")).isEmpty())
        assertTrue(scopedSelectedVersions(facts, dependency("compile", ":missing")).isEmpty())
    }

    @Test fun `Gradle configurations use observed inheritance rather than names or all module versions`() {
        fun graph(name: String, version: String, declarations: List<String>) = ConfigurationGraph(":", name, "root", listOf(node(version)), emptyList(), declarationConfigurations = declarations)
        val facts = facts(listOf(graph("compileClasspath", "1", listOf("implementation", "compileOnly")),
            graph("testCompileClasspath", "2", listOf("testImplementation")),
            graph("runtimeClasspath", "3", listOf("runtimeOnly")),
            graph("customClasspath", "4", emptyList())))
        assertEquals(listOf("1"), scopedSelectedVersions(facts, dependency("implementation")))
        assertEquals(listOf("2"), scopedSelectedVersions(facts, dependency("testImplementation")))
        assertEquals(listOf("3"), scopedSelectedVersions(facts, dependency("runtimeOnly")))
        assertTrue(scopedSelectedVersions(facts, dependency("customDeclaration")).isEmpty())
        assertEquals(listOf("4"), scopedSelectedVersions(facts, dependency("customClasspath")))
    }

    @Test fun `remote lookup is deduplicated while both renderers retain module and original scope groups`() {
        var calls = 0
        val declarations = listOf(dependency("compile", ":app"), dependency("test", ":app"), dependency("runtimeOnly", ":worker"))
        val remote = RemoteVersionCollector(lookup = { _, _ -> calls++; ReleaseLookup("2", null) }).collect(declarations) { emptyList() }
        assertEquals(1, calls)
        assertEquals(3, remote.dependencies.size)
        val output = ByteArrayOutputStream()
        ScanRenderer(PrintStream(output), false).remoteVersions(remote)
        val terminal = output.toString()
        val markdown = MarkdownReport().render(facts(emptyList()).copy(remoteVersions = remote))
        for (group in listOf(":app / compile", ":app / test", ":worker / runtimeOnly")) {
            assertContains(terminal, group)
            assertContains(markdown, "### $group")
        }
        assertEquals(3, Regex("example:lib").findAll(terminal).count())
        assertEquals(3, Regex("example:lib").findAll(markdown).count())
        assertFalse(terminal.contains('\u001b'))
        assertFalse(markdown.contains('\u001b'))
    }
}
