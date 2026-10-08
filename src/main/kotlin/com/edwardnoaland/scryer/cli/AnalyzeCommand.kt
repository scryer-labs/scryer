package com.edwardnoaland.scryer.cli

import com.edwardnoaland.scryer.cli.output.renderAnalyzeTerminal
import com.edwardnoaland.scryer.cli.output.renderAnalyzeJson
import com.edwardnoaland.scryer.cli.output.renderAnalyzeMarkdown
import com.edwardnoaland.scryer.cli.output.renderAnalyzeHtml
import com.edwardnoaland.scryer.cli.output.writeReportFile
import com.edwardnoaland.scryer.analyze.report.buildAnalyzeReport
import java.io.PrintStream
import java.nio.file.Path
import com.edwardnoaland.scryer.analyze.GitComparer
import com.edwardnoaland.scryer.analyze.AnalyzeService
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus

internal const val ANALYZE_USAGE = "Usage: scryer analyze --before <ref> --after <ref> [--skip-tests | --test-command <command>] [--verbose | --json] [-o <report.json|report.md|report.html>]"

private data class AnalyzeOptions(val before: String, val after: String, val skipTests: Boolean, val verbose: Boolean, val json: Boolean, val output: Path?, val testCommand: String?)

internal fun runAnalyzeCommand(args: Array<String>, out: PrintStream, err: PrintStream): Int {
    if (args.contentEquals(arrayOf("--help")) || args.contentEquals(arrayOf("-h"))) {
        out.println(ANALYZE_USAGE)
        return 0
    }

    val options = parseAnalyzeOptions(args)
    if (options == null) {
        err.println(ANALYZE_USAGE)
        return 2
    }

    return try {
        val comparison = GitComparer().compare(Path.of("."), options.before, options.after)
        val result = AnalyzeService(progress = { err.println("scryer: $it") }).analyze(comparison, runTests = !options.skipTests, testCommand = options.testCommand)
        val report = buildAnalyzeReport(result)
        if (options.json) out.print(renderAnalyzeJson(report)) else renderAnalyzeTerminal(report, out, options.verbose)
        options.output?.let { destination ->
            writeReportFile(destination) {
                when (destination.fileName.toString().substringAfterLast('.').lowercase()) {
                    "json" -> renderAnalyzeJson(report)
                    "md" -> renderAnalyzeMarkdown(report)
                    else -> renderAnalyzeHtml(report)
                }
            }
            err.println("scryer: Report saved to $destination")
        }
        val execution = result.execution
        if (execution.status in setOf(TestRunStatus.SUCCEEDED, TestRunStatus.SKIPPED)) 0 else 1
    } catch (exception: Exception) {
        err.println("scryer: ${exception.message ?: exception.javaClass.simpleName}")
        1
    }
}

/** Parse command arguments before performing Git operations. */
private fun parseAnalyzeOptions(args: Array<String>): AnalyzeOptions? {
    var before: String? = null
    var after: String? = null
    var index = 0
    var skipTests = false
    var verbose = false
    var json = false
    var output: Path? = null
    var testCommand: String? = null

    while (index < args.size) {
        val flag = args[index]
        if (flag == "--skip-tests") {
            if (skipTests) return null
            skipTests = true
            index++
            continue
        }
        if (flag == "--verbose") {
            if (verbose) return null
            verbose = true
            index++
            continue
        }
        if (flag == "--json") {
            if (json) return null
            json = true
            index++
            continue
        }
        if (flag !in listOf("--before", "--after", "-o", "--test-command")) {
            return null
        }
        if (index + 1 >= args.size) {
            return null
        }
        val reference = args[index + 1]
        if (reference.isBlank() || reference.startsWith('-')) {
            return null
        }

        when (flag) {
            "--test-command" -> {
                if (testCommand != null || '\u0000' in reference) return null
                testCommand = reference
            }
            "-o" -> {
                if (output != null) return null
                output = runCatching { Path.of(reference).toAbsolutePath().normalize() }.getOrNull() ?: return null
                if (output.fileName.toString().substringAfterLast('.', "").lowercase() !in setOf("json", "md", "html")) return null
            }
            "--before" -> {
                if (before != null) {
                    return null
                }
                before = reference
            }
            "--after" -> {
                if (after != null) {
                    return null
                }
                after = reference
            }
        }
        index += 2
    }

    if (skipTests && testCommand != null) return null
    if (json && verbose) return null
    return AnalyzeOptions(before ?: return null, after ?: return null, skipTests, verbose, json, output, testCommand)
}
