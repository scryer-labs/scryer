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

    @Test fun `valid command resolves refs in either option order`() {
        for (args in listOf(
            arrayOf("analyze", "--before", "HEAD", "--after", "HEAD", "--skip-tests"),
            arrayOf("analyze", "--skip-tests", "--after", "HEAD", "--before", "HEAD"),
        )) {
            val (code, output, error) = invoke(*args)
            assertEquals(0, code)
            assertContains(output, "Changed files: 0")
            assertEquals("", error)
        }
        val (code, _, error) = invoke("analyze", "--before", "not-a-real-ref", "--after", "HEAD")
        assertEquals(1, code)
        assertContains(error, "Git command failed")
    }

    @Test fun `missing blank duplicate and unsupported arguments are rejected`() {
        val invalid = listOf(
            arrayOf("analyze"),
            arrayOf("analyze", "--before", "HEAD", "--after", "HEAD", "--skip-tests", "--skip-tests"),
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
