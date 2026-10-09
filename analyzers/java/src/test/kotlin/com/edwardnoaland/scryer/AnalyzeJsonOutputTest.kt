package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.runCli
import com.edwardnoaland.scryer.cli.output.renderAnalyzeJson
import com.edwardnoaland.scryer.cli.output.writeReportFile
import com.edwardnoaland.scryer.serialization.jsonMapper
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import java.nio.file.Files
import java.nio.file.Path
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.*

class AnalyzeJsonOutputTest {
    @TempDir lateinit var root: Path

    @Test fun `JSON output preserves complete scope method status counters and null unknowns`() {
        val report = exampleAnalyzeReport()
        val data = jsonMapper.readTree(renderAnalyzeJson(report))
        assertEquals(1, data["schemaVersion"].asInt())
        assertEquals("after-sha", data["afterSha"].asText())
        assertEquals(3, data["after"]["edges"].size())
        assertEquals(1, data["matching"]["datasets"][0]["counts"]["NOT_EXECUTED"].asInt())
        val unknown = report.matching.datasets.single().copy(assessed = 0, withHits = 0, methodExecutionPercent = null)
        val changed = jsonMapper.readTree(renderAnalyzeJson(report.copy(matching = report.matching.copy(datasets = listOf(unknown)), evidence = report.evidence.copy(testRecords = null))))
        assertTrue(changed["matching"]["datasets"][0]["methodExecutionPercent"].isNull)
        assertTrue(changed["evidence"]["testRecords"].isNull)
        assertEquals("/retained/test.exec", data["evidence"]["artifacts"][0]["retained"].asText())
    }

    @Test fun `CLI JSON stdout is parseable and file content matches stdout`() {
        val output = ByteArrayOutputStream()
        val error = ByteArrayOutputStream()
        val file = root.resolve("nested/report.json")
        val code = runCli(arrayOf("analyze", "--before", "HEAD", "--after", "HEAD", "--skip-tests", "--json", "-o", file.toString()), PrintStream(output), PrintStream(error))
        assertEquals(0, code)
        val json = jsonMapper.readTree(output.toString())
        assertEquals("SKIPPED", json["execution"]["status"].asText())
        assertEquals(output.toString(), Files.readString(file))
        assertContains(error.toString(), "Report saved")
    }

    @Test fun `failed rendering preserves existing output and relative paths are supported`() {
        val file = root.resolve("report.json")
        Files.writeString(file, "previous")
        assertFailsWith<IllegalStateException> { writeReportFile(file) { error("cannot render") } }
        assertEquals("previous", Files.readString(file))
        assertFailsWith<IllegalArgumentException> { writeReportFile(root) { "content" } }
        Files.list(root).use { assertEquals(listOf(file), it.toList()) }
    }
}
