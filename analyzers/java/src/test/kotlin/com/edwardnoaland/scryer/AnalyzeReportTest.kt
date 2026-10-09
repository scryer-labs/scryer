package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.output.renderAnalyzeTerminal
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import org.junit.jupiter.api.Test
import kotlin.test.*

class AnalyzeReportTest {
    @Test fun `report preserves change scope raw evidence and separate matching provenance`() {
        val report = exampleAnalyzeReport()
        assertEquals(1, report.schemaVersion)
        assertEquals("after-sha", report.evidence.afterSha)
        assertEquals("after-sha", report.matching.afterSha)
        assertEquals(4, report.after.nodes.size)
        assertEquals(3, report.after.edges.size)
        assertEquals(1, report.after.boundaries.size)
        assertEquals("int changed() { return 1; }", report.changes.single().before?.source)
        assertEquals(1, report.evidence.testRecords?.get("PASSED"))
        val dataset = report.matching.datasets.single()
        assertEquals(3, dataset.assessed)
        assertEquals(2, dataset.withHits)
        assertEquals("NOT_EXECUTED", dataset.methods.single { it.symbol.contains("#caller") }.status)
        assertEquals("DIRECT_CANDIDATE", dataset.methods.single { it.symbol.contains("#changed") }.routes.single().kind)
    }

    @Test fun `terminal puts execution and gaps first while full graph is opt in`() {
        val report = exampleAnalyzeReport()
        val output = ByteArrayOutputStream()
        renderAnalyzeTerminal(report, PrintStream(output))
        val text = output.toString()
        assertContains(text, "After test command: SUCCEEDED")
        assertContains(text, "NOT_EXECUTED [CALLER]")
        assertContains(text, "PARTIALLY_EXECUTED [POTENTIAL_INDIRECT]")
        assertFalse(text.contains("Before impact graph"))
        output.reset()
        renderAnalyzeTerminal(report, PrintStream(output), verbose = true)
        assertContains(output.toString(), "Before impact graph")
        assertContains(output.toString(), "REFLECTION ->")
        assertContains(output.toString(), "attribution unknown")
    }

    @Test fun `concise terminal explicitly indicates additional gaps rather than silently dropping them`() {
        val report = exampleAnalyzeReport()
        val dataset = report.matching.datasets.single()
        val gaps = List(12) { dataset.methods.first().copy(status = "UNKNOWN", signature = "method$it") }
        val expanded = report.copy(matching = report.matching.copy(datasets = listOf(dataset.copy(methods = gaps))))
        val output = ByteArrayOutputStream()
        renderAnalyzeTerminal(expanded, PrintStream(output))
        assertContains(output.toString(), "4 more; use --verbose")
    }
}
