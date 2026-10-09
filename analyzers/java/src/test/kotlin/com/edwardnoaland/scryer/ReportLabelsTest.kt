package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.output.compactSignature
import com.edwardnoaland.scryer.cli.output.compactSource
import com.edwardnoaland.scryer.cli.output.signatureLabels
import com.edwardnoaland.scryer.cli.output.renderAnalyzeMarkdown
import com.edwardnoaland.scryer.cli.output.renderAnalyzeMermaid
import org.junit.jupiter.api.Test
import kotlin.test.*

class ReportLabelsTest {
    @Test fun `compact labels preserve overloads and qualify colliding owners`() {
        val signatures = listOf("one.Order#submit(int)", "two.Order#submit(int)", "one.Order#submit(String)", "one.Order#submit(int)")
        val labels = signatureLabels(signatures)
        assertEquals("one.Order#submit(int)", labels.getValue(signatures[0]))
        assertEquals("two.Order#submit(int)", labels.getValue(signatures[1]))
        assertEquals("Order#submit(String)", labels.getValue(signatures[2]))
        assertEquals("Outer\$Inner#<init>()", compactSignature("example.Outer\$Inner#<init>()"))
        assertEquals("Order.java", compactSource("src/main/java/example/Order.java"))
        assertEquals("unknown", compactSource(null))
    }

    @Test fun `readable diagrams retain complete identities in Markdown index`() {
        val report = exampleAnalyzeReport()
        val diagram = renderAnalyzeMermaid(report.after)
        val markdown = renderAnalyzeMarkdown(report)
        assertContains(markdown, "## Full symbol index")
        report.after.nodes.forEach { symbol ->
            assertContains(diagram, compactSignature(symbol.signature).substringBefore('#'))
            assertContains(markdown, symbol.path.orEmpty())
        }
        assertContains(markdown, "ambiguous short signatures")
    }
}
