package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.runCli
import com.edwardnoaland.scryer.cli.output.renderAnalyzeMarkdown
import com.edwardnoaland.scryer.cli.output.renderAnalyzeMermaid
import com.edwardnoaland.scryer.serialization.jsonMapper
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import java.nio.file.Files
import java.nio.file.Path
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.*

class AnalyzeMarkdownOutputTest {
    @TempDir lateinit var root: Path

    @Test fun `Markdown preserves provenance gaps source changes roles and all Mermaid edges`() {
        val report = exampleAnalyzeReport()
        val markdown = renderAnalyzeMarkdown(report)
        assertContains(markdown, "before-sha")
        assertContains(markdown, "NOT\\_EXECUTED")
        assertContains(markdown, "int changed() { return 2; }")
        assertContains(markdown, "attribution unknown")
        assertContains(markdown, "1234")
        val diagram = renderAnalyzeMermaid(report.after)
        assertEquals(report.after.edges.size, Regex("-->").findAll(diagram).count())
        assertContains(diagram, "REFLECTION")
        assertContains(diagram, "TEST")
        val cycle = report.after.copy(edges = report.after.edges + report.after.edges.first().let { it.copy(caller = it.callee, callee = it.caller) })
        assertEquals(4, Regex("-->").findAll(renderAnalyzeMermaid(cycle)).count())
    }

    @Test fun `untrusted text cannot escape table Mermaid labels or source fences`() {
        val report = exampleAnalyzeReport()
        val hostile = "</script>|[x]`\n# heading"
        val change = report.changes.single()
        val edited = report.copy(repository = hostile, changes = listOf(change.copy(after = change.after!!.copy(source = "String value = \"```\";"))),
            after = report.after.copy(nodes = report.after.nodes.map { it.copy(signature = hostile) }))
        val markdown = renderAnalyzeMarkdown(edited)
        assertFalse(markdown.contains("</script>"))
        assertContains(markdown, "&lt;/script&gt;\\|")
        assertContains(markdown, "````java")
        val diagram = renderAnalyzeMermaid(edited.after)
        assertFalse(diagram.contains("</script>"))
        assertContains(diagram, "#60;")
        assertFalse(diagram.contains("\n# heading"))
    }

    @Test fun `Markdown export can accompany pure JSON stdout`() {
        val output = ByteArrayOutputStream()
        val error = ByteArrayOutputStream()
        val file = root.resolve("report.MD")
        assertEquals(0, runCli(arrayOf("analyze", "--before", "HEAD", "--after", "HEAD", "--skip-tests", "--json", "-o", file.toString()), PrintStream(output), PrintStream(error)))
        assertEquals("SKIPPED", jsonMapper.readTree(output.toString())["execution"]["status"].asText())
        assertContains(Files.readString(file), "# Scryer analysis")
    }
}
