package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.runCli
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import java.nio.file.Path
import kotlin.io.path.createDirectories
import kotlin.io.path.writeText
import kotlin.test.*
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir

class CliTest {
    @TempDir lateinit var root: Path

    private fun invoke(vararg args: String): Triple<Int, String, String> {
        val output = ByteArrayOutputStream()
        val error = ByteArrayOutputStream()
        val code = runCli(arrayOf(*args), PrintStream(output), PrintStream(error))
        return Triple(code, output.toString(Charsets.UTF_8), error.toString(Charsets.UTF_8))
    }

    @Test fun `help succeeds and invalid flags are rejected`() {
        assertEquals(0, invoke("--help").first)
        assertEquals(2, invoke("scan", ".", "--unknown-option").first)
        assertEquals(2, invoke("scan", ".", "-o").first)
    }

    @Test fun `scan accepts paths containing spaces and reports versions`() {
        val directory = root.resolve("legacy project").createDirectories()
        directory.resolve("build.gradle").writeText("dependencies { compile 'commons-lang:commons-lang:2.6' }")
        val (code, output, error) = invoke("scan", directory.toString(), "--static", "--dependencies")
        assertEquals(0, code)
        assertEquals("", error)
        assertContains(output, "commons-lang:commons-lang:2.6")
        assertContains(output, "Direct dependencies")
    }

    @Test fun `missing repository exits with useful error`() {
        val (code, output, error) = invoke("scan", root.resolve("missing").toString())
        assertEquals(1, code)
        assertEquals("", output)
        assertContains(error, "Not a directory")
    }
}
