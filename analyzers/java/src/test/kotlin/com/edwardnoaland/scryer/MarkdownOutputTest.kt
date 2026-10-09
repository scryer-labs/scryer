package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.runCli
import com.edwardnoaland.scryer.cli.output.MarkdownReport
import com.edwardnoaland.scryer.scan.model.ConfigurationGraph
import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.model.ResolutionFacts
import com.edwardnoaland.scryer.serialization.jsonMapper
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import java.nio.file.Path
import kotlin.io.path.createDirectories
import kotlin.io.path.readText
import kotlin.io.path.writeText
import kotlin.test.*
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir

class MarkdownOutputTest {
    @TempDir lateinit var root: Path

    private fun invoke(vararg flags: String): Triple<Int, String, String> {
        root.resolve("build.gradle").writeText("dependencies { implementation 'example:lib:1.2' }")
        val output = ByteArrayOutputStream()
        val error = ByteArrayOutputStream()
        val code = runCli(arrayOf("scan", root.toString(), "--static", *flags), PrintStream(output), PrintStream(error))
        return Triple(code, output.toString(Charsets.UTF_8), error.toString(Charsets.UTF_8))
    }

    @Test fun `Markdown export creates parent directories and preserves console output`() {
        val destination = root.resolve("reports with spaces/扫描.md")
        val (code, console, error) = invoke("-o", destination.toString(), "--color", "always")
        assertEquals(0, code)
        assertContains(console, "Project")
        assertContains(error, "Markdown report written")
        val markdown = destination.readText()
        assertContains(markdown, "# Scryer repository scan")
        assertContains(markdown, "example:lib")
        assertContains(markdown, "1.2")
        assertContains(markdown, "not_requested")
        assertContains(markdown, "may be incomplete or inaccurate")
        assertFalse("\u001b" in markdown)
    }

    @Test fun `JSON stdout remains parseable when also exporting Markdown`() {
        val destination = root.resolve("report.md")
        val (code, output, _) = invoke("--json", "--output", destination.toString())
        assertEquals(0, code)
        assertEquals(1, jsonMapper.readTree(output)["schemaVersion"].asInt())
        assertContains(destination.readText(), "## Dependencies")
        assertFalse("Markdown report written" in output)
    }

    @Test fun `missing or blank output argument is rejected`() {
        assertEquals(2, invoke("-o").first)
        assertEquals(2, invoke("-o", "").first)
        assertEquals(2, invoke("-o", "--json").first)
    }

    @Test fun `output errors are explicit and do not replace a directory`() {
        val directory = root.resolve("report.md").createDirectories()
        directory.resolve("keep.txt").writeText("keep")
        val (code, output, error) = invoke("-o", directory.toString())
        assertEquals(1, code)
        assertEquals("", output)
        assertContains(error, "Output path is a directory")
        assertEquals("keep", directory.resolve("keep.txt").readText())
    }

    @Test fun `existing report is replaced with current facts`() {
        val destination = root.resolve("report.md")
        destination.writeText("old report")
        assertEquals(0, invoke("-o", destination.toString()).first)
        assertFalse("old report" in destination.readText())
        assertContains(destination.readText(), "# Scryer repository scan")
    }

    @Test fun `all notes are preserved and table delimiters and HTML are escaped`() {
        val graph = ConfigurationGraph(":", "test", "root", emptyList(), emptyList(), "failed", "cannot fetch <repo>|artifact\nnext line")
        val facts = RepositoryFacts(
            root.resolve("repo|name"), emptyList(),
            notes = (1..8).map { "note $it" },
            resolution = ResolutionFacts("unavailable", configurations = listOf(graph)),
        )
        val markdown = MarkdownReport().render(facts)
        assertContains(markdown, "repo\\|name")
        assertContains(markdown, "note 8")
        assertContains(markdown, "cannot fetch &lt;repo&gt;\\|artifact<br>next line")
        assertContains(markdown, "| Resolved components | unknown |")
    }

    @Test fun `tree flag includes a plain text dependency tree section`() {
        val destination = root.resolve("report.md")
        assertEquals(0, invoke("--dependency-tree", "-o", destination.toString()).first)
        assertContains(destination.readText(), "## Resolved dependency tree")
        assertContains(destination.readText(), "Unavailable: not_requested")
    }
}
