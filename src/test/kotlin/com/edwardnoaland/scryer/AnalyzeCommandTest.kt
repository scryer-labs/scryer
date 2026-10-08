package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.runCli
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import kotlin.test.assertContains
import kotlin.test.assertEquals
import org.junit.jupiter.api.Test

class AnalyzeCommandTest {
    private fun invoke(vararg args: String): Triple<Int, String, String> {
        val output = ByteArrayOutputStream()
        val error = ByteArrayOutputStream()
        val code = runCli(args.toList().toTypedArray(), PrintStream(output), PrintStream(error))
        return Triple(code, output.toString(Charsets.UTF_8), error.toString(Charsets.UTF_8))
    }

    @Test fun `valid command accepts either option order without resolving refs`() {
        for (args in listOf(
            arrayOf("analyze", "--before", "HEAD~1", "--after", "HEAD"),
            arrayOf("analyze", "--after", "not-yet-a-real-ref", "--before", "fixture-baseline"),
        )) {
            val (code, output, error) = invoke(*args)
            assertEquals(0, code)
            assertEquals("Analyzing…${System.lineSeparator()}", output)
            assertEquals("", error)
        }
    }

    @Test fun `missing blank duplicate and unsupported arguments are rejected`() {
        val invalid = listOf(
            arrayOf("analyze"),
            arrayOf("analyze", "--before", "HEAD"),
            arrayOf("analyze", "--before", "HEAD", "--after"),
            arrayOf("analyze", "--before", "", "--after", "HEAD"),
            arrayOf("analyze", "--before", "--after", "HEAD"),
            arrayOf("analyze", "--before", "HEAD", "--before", "HEAD~1", "--after", "HEAD"),
            arrayOf("analyze", "--before", "HEAD", "--after", "HEAD", "--after", "HEAD~1"),
            arrayOf("analyze", "--before", "HEAD", "--after", "HEAD", "--json"),
            arrayOf("analyze", "--before", "HEAD", "--after", "HEAD", "extra"),
        )
        for (args in invalid) {
            val (code, output, error) = invoke(*args)
            assertEquals(2, code, args.joinToString(" "))
            assertEquals("", output)
            assertContains(error, "scryer analyze --before <ref> --after <ref>")
        }
    }

    @Test fun `global and command help include analyze usage`() {
        for (args in listOf(arrayOf("--help"), arrayOf("analyze", "--help"), arrayOf("analyze", "-h"))) {
            val (code, output, error) = invoke(*args)
            assertEquals(0, code)
            assertContains(output, "scryer analyze --before <ref> --after <ref>")
            assertEquals("", error)
        }
    }
}
