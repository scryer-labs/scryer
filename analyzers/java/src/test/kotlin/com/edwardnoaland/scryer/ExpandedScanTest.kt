package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.output.ScanRenderer
import com.edwardnoaland.scryer.cli.runCli
import com.edwardnoaland.scryer.scan.inspect.RepositoryScanner
import com.edwardnoaland.scryer.scan.inspect.children
import com.edwardnoaland.scryer.scan.inspect.text
import com.edwardnoaland.scryer.scan.model.ConfigurationGraph
import com.edwardnoaland.scryer.scan.model.GraphEdge
import com.edwardnoaland.scryer.scan.model.GraphNode
import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.model.ResolutionFacts
import com.edwardnoaland.scryer.scan.resolve.DependencyResolver
import com.edwardnoaland.scryer.scan.resolve.readMavenTree
import com.edwardnoaland.scryer.scan.withResolution
import com.edwardnoaland.scryer.serialization.jsonMapper
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import java.nio.file.Path
import kotlin.io.path.*
import kotlin.test.*
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir

class ExpandedScanTest {
    @TempDir lateinit var root: Path
    private fun write(path: String, text: String) { root.resolve(path).parent.createDirectories(); root.resolve(path).writeText(text) }
    private fun cli(vararg flags: String): Pair<Int, String> {
        val out = ByteArrayOutputStream(); val err = ByteArrayOutputStream()
        val result = runCli(arrayOf("scan", root.toString(), *flags), PrintStream(out), PrintStream(err))
        assertEquals("", err.toString())
        return result to out.toString()
    }

    @Test fun `summary is concise while JSON retains all declarations`() {
        write("build.gradle", "dependencies {\n" + (1..50).joinToString("\n") { "implementation 'example:lib$it:1'" } + "\n}")
        val summary = cli("--static").second
        assertFalse("example:lib50" in summary)
        val json = jsonMapper.readTree(cli("--static", "--json").second)
        assertEquals(50, json["dependencies"].size())
        assertEquals("not_requested", json["resolution"]["status"].asText())
        assertTrue(json["resolution"]["resolvedComponentCount"].isNull)
        assertEquals(root.toString(), json["root"].asText())
        assertFalse("\u001b" in cli("--static", "--json", "--color", "always").second)
    }

    @Test fun `direct flag prints all declarations and supports combinations`() {
        write("build.gradle", "dependencies { implementation 'example:lib:1' }")
        val text = cli("--static", "--dependencies", "--dependency-tree").second
        assertContains(text, "example:lib:1")
        assertContains(text, "Resolved dependency tree")
        assertContains(text, "Unavailable")
    }

    @Test fun `dependency detail preserves multiple requested versions of the same coordinate`() {
        write("build.gradle", "dependencies { implementation 'junit:junit:4.12'; implementation 'junit:junit:4.13.2' }")
        val output = cli("--static", "--dependencies").second
        assertContains(output, "junit:junit:4.12")
        assertContains(output, "junit:junit:4.13.2")
    }

    @Test fun `color controls use styled output only when requested`() {
        write("build.gradle", "")
        assertContains(cli("--static", "--color", "always").second, "\u001b[")
        assertFalse("\u001b" in cli("--static", "--color", "never").second)
        assertFalse("\u001b" in cli("--static").second)
    }

    @Test fun `missing wrapper leaves resolution unavailable not a fake zero graph`() {
        write("build.gradle", "")
        val facts = RepositoryScanner().scan(root)
        val resolved = DependencyResolver().resolve(facts)
        assertEquals("unavailable", resolved.status)
        assertNull(resolved.resolvedComponentCount)
        assertNull(resolved.conflictCount)
    }

    @Test fun `Maven tree preserves scope relationship and unavailable conflict counts`() {
        val tree = jsonMapper.readTree("""{"groupId":"app","artifactId":"app","version":"1","children":[{"groupId":"g","artifactId":"a","version":"2","scope":"test","children":[{"groupId":"g","artifactId":"b","version":"3","scope":"test"}]}]}""")
        val graph = readMavenTree(tree, ":")
        assertTrue(graph.edges.first().direct)
        assertFalse(graph.edges.last().direct)
        assertEquals("test", graph.edges.first().scope)
        assertEquals(graph.edges.first().to, graph.edges.last().from)
        assertEquals(null, ResolutionFacts(configurations = listOf(graph), metadata = mapOf("conflictReasons" to "unavailable")).conflictCount)
    }

    @Test fun `failed Maven collection exposes error without claiming success`() {
        write("pom.xml", "<project><modelVersion>4.0.0</modelVersion><groupId>example</groupId><artifactId>app</artifactId><version>1</version></project>")
        write("mvnw", "#!/bin/sh\necho 'Repository unavailable'\nexit 2\n")
        val facts = RepositoryScanner().scan(root)
        val progress = mutableListOf<String>()
        val resolution = DependencyResolver(5, progress::add).resolve(facts)
        assertEquals("unavailable", resolution.status)
        assertNull(resolution.resolvedComponentCount)
        assertTrue(progress.isNotEmpty())
        assertTrue(resolution.notes.none { "trees collected" in it })
        val output = ByteArrayOutputStream()
        ScanRenderer(PrintStream(output), false).summary(withResolution(facts, resolution))
        assertContains(output.toString(), "Resolution failure")
        assertContains(output.toString(), "Repository unavailable")
    }

    @Test fun `Maven timeout preserves last activity and retry guidance`() {
        write("pom.xml", "<project><modelVersion>4.0.0</modelVersion><groupId>example</groupId><artifactId>app</artifactId><version>1</version></project>")
        write("mvnw", "#!/bin/sh\necho 'Downloading dependency descriptor'\nexec sleep 30\n")
        val resolution = DependencyResolver(1).resolve(RepositoryScanner().scan(root))
        assertEquals("unavailable", resolution.status)
        val error = resolution.configurations.single().error!!
        assertContains(error, "timed out after 1s")
        assertContains(error, "Downloading dependency descriptor")
        assertContains(error, "SCRYER_RESOLUTION_TIMEOUT_SECONDS")
    }

    @Test fun `tree renders shared nodes and conflicts while keeping raw JSON edges`() {
        val graph = ConfigurationGraph(":", "testRuntimeClasspath", "project :", listOf(
            GraphNode("project :", null, null, null, "project"), GraphNode("g:a:2", "g", "a", "2")), listOf(
            GraphEdge("project :", "g:a:2", "g:a:1", "1", "2", true, listOf("between versions 1 and 2"), conflict = true, changedSelection = true),
            GraphEdge("g:a:2", "g:a:2", "g:a:2", "2", "2", false)))
        val facts = RepositoryFacts(root, emptyList(), resolution = ResolutionFacts("complete", "test", listOf(graph)))
        val out = ByteArrayOutputStream()
        ScanRenderer(PrintStream(out), false).tree(facts)
        assertContains(out.toString(), "└──")
        assertContains(out.toString(), "requested 1; selected 2")
        assertContains(out.toString(), "already shown")
        assertEquals(2, jsonMapper.readTree(jsonMapper.writeValueAsString(graph))["edges"].size())
    }
}
