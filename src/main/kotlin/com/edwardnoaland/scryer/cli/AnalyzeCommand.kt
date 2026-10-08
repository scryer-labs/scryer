package com.edwardnoaland.scryer.cli

import com.edwardnoaland.scryer.cli.output.renderImpactGraph
import java.io.PrintStream
import java.nio.file.Path
import com.edwardnoaland.scryer.analyze.GitComparer
import com.edwardnoaland.scryer.analyze.LineRange
import com.edwardnoaland.scryer.analyze.AnalyzeService
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus
import com.edwardnoaland.scryer.analyze.SnapshotImpact

internal const val ANALYZE_USAGE = "Usage: scryer analyze --before <ref> --after <ref> [--skip-tests]"

private data class AnalyzeOptions(val before: String, val after: String, val skipTests: Boolean)

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
        val result = AnalyzeService(progress = { err.println("scryer: $it") }).analyze(comparison, runTests = !options.skipTests)
        val impact = result.impact
        val analysis = impact.methods
        out.println("Repository: ${comparison.repository}")
        out.println("Before: ${comparison.before}")
        out.println("After: ${comparison.after}")
        out.println("Changed files: ${comparison.files.size}")
        comparison.files.forEach { file ->
            val path = if (file.beforePath != null && file.afterPath != null && file.beforePath != file.afterPath) {
                "${file.beforePath} -> ${file.afterPath}"
            } else file.afterPath ?: file.beforePath
            out.println("${file.status} $path")
            file.lines.forEach { lines ->
                out.println("  before ${formatRange(lines.before)} -> after ${formatRange(lines.after)}")
            }
            if (file.lines.isEmpty()) out.println("  No textual line changes (for example rename, binary or mode change)")
        }
        out.println("Changed Java methods: ${analysis.methods.size}")
        analysis.methods.forEach { change ->
            val method = change.after ?: requireNotNull(change.before)
            out.println("${change.kind} ${method.signature}")
            change.before?.let { out.println("  before ${change.beforePath}: ${formatRange(it.lines)}") }
            change.after?.let { out.println("  after ${change.afterPath}: ${formatRange(it.lines)}") }
        }
        analysis.notes.forEach { out.println("Note: $it") }
        renderImpact("Before", impact.before, out)
        renderImpact("After", impact.after, out)
        val execution = result.execution
        out.println("After test command: ${execution.status}")
        out.println("  After SHA: ${execution.afterSha}")
        if (execution.command.isNotEmpty()) out.println("  Command: ${execution.command.joinToString(" ")}")
        execution.javaHome?.let { out.println("  Target JAVA_HOME: $it") }
        execution.exitCode?.let { out.println("  Exit code: $it") }
        out.println("  Duration: ${execution.durationMillis}ms")
        execution.log?.let { out.println("  Log: $it") }
        execution.notes.forEach { out.println("  Note: $it") }
        out.println("Static potential impact; per-test execution and coverage evidence are not collected yet.")
        if (execution.status in setOf(TestRunStatus.SUCCEEDED, TestRunStatus.SKIPPED)) 0 else 1
    } catch (exception: Exception) {
        err.println("scryer: ${exception.message ?: exception.javaClass.simpleName}")
        1
    }
}

private fun formatRange(range: LineRange): String = when (range.count) {
    0 -> "boundary ${range.start} (0 lines)"
    1 -> "line ${range.start}"
    else -> "lines ${range.start}-${range.start + range.count - 1}"
}

/** Parse command arguments before performing Git operations. */
private fun parseAnalyzeOptions(args: Array<String>): AnalyzeOptions? {
    var before: String? = null
    var after: String? = null
    var index = 0
    var skipTests = false

    while (index < args.size) {
        val flag = args[index]
        if (flag == "--skip-tests") {
            if (skipTests) return null
            skipTests = true
            index++
            continue
        }
        if (flag !in listOf("--before", "--after")) {
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

    return AnalyzeOptions(before ?: return null, after ?: return null, skipTests)
}

private fun renderImpact(label: String, impact: SnapshotImpact, out: PrintStream) {
    out.println("$label symbol impact (partial):")
    out.println("  Roles follow source-path conventions; TEST is not execution evidence")
    out.println("  CHANGED = modified symbol; CALLER = reverse impact; POTENTIAL INDIRECT = conservative branch")
    renderImpactGraph(impact, out)
    impact.unmatched.forEach { out.println("  UNRESOLVED CHANGED SYMBOL $it") }
    out.println("  Call boundaries: ${impact.graph.boundaries.size}")
    impact.graph.boundaries.filter { it.caller == null || it.caller in impact.affected || it.caller in impact.indirect }.forEach {
        out.println("  BOUNDARY ${it.path}:${it.line}: ${it.reason}: ${it.expression.replace('\n', ' ')}")
    }
    impact.graph.notes.forEach { out.println("  Note: $it") }
}
