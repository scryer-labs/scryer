package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.runCli
import com.edwardnoaland.scryer.cli.output.renderAnalyzeHtml
import com.edwardnoaland.scryer.cli.output.renderAnalyzeJson
import com.edwardnoaland.scryer.serialization.jsonMapper
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import java.nio.file.Files
import java.nio.file.Path
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.*

class AnalyzeHtmlOutputTest {
    @TempDir lateinit var root: Path

    private fun embeddedJson(html: String): String = Regex("<script id=\"scryer-data\" type=\"application/json\">([\\s\\S]*?)</script>").find(html)!!.groupValues[1]

    @Test fun `HTML is self contained and embeds the same complete JSON model`() {
        val report = exampleAnalyzeReport()
        val html = renderAnalyzeHtml(report)
        val data = jsonMapper.readTree(embeddedJson(html))
        assertEquals(jsonMapper.readTree(renderAnalyzeJson(report)), data)
        assertFalse(html.contains("__SCRYER_"))
        assertFalse(Regex("<(script|link|img)[^>]+(src|href)=\"https?://").containsMatchIn(html))
        assertContains(html, "connect-src 'none'")
        assertContains(html, "Search symbols or source paths")
        assertContains(html, "<dialog")
        val logo = Regex("src=\"data:image/png;base64,([^\"]+)\"").find(html)!!.groupValues[1]
        val bundledLogo = checkNotNull(javaClass.getResourceAsStream("/analyze-report/scryer-logo.png")).use { it.readBytes() }
        assertContentEquals(bundledLogo, java.util.Base64.getDecoder().decode(logo))
    }

    @Test fun `repository content cannot break script boundaries and round trips without loss`() {
        val hostile = "</script><script>alert('unsafe')</script>&\u2028\u2029"
        val report = exampleAnalyzeReport().copy(repository = hostile)
        val html = renderAnalyzeHtml(report)
        assertFalse(html.contains(hostile))
        val data = jsonMapper.readTree(embeddedJson(html))
        assertEquals(hostile, data["repository"].asText())
        assertContains(embeddedJson(html), "\\u003c")
        assertContains(html, "textContent")
    }

    @Test fun `HTML export and JSON stdout share SHA and failed evidence remains visible`() {
        val output = ByteArrayOutputStream()
        val error = ByteArrayOutputStream()
        val file = root.resolve("report.HTML")
        assertEquals(0, runCli(arrayOf("analyze", "--before", "HEAD", "--after", "HEAD", "--skip-tests", "--json", "-o", file.toString()), PrintStream(output), PrintStream(error)))
        val json = jsonMapper.readTree(output.toString())
        val embedded = jsonMapper.readTree(embeddedJson(Files.readString(file)))
        assertEquals(json, embedded)
        assertEquals("SKIPPED", embedded["execution"]["status"].asText())
    }
}
