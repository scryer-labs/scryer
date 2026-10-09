package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.analyze.report.*
import java.io.PrintStream
import java.util.Locale

internal fun renderAnalyzeTerminal(report: AnalyzeReport, out: PrintStream, verbose: Boolean = false) {
    out.println("Scryer analysis")
    out.println("Repository: ${report.repository}")
    out.println("Before: ${report.beforeSha}")
    out.println("After: ${report.afterSha}")
    out.println("Changed files: ${report.files.size}")
    out.println("Changed Java methods: ${report.changes.size}")
    out.println("After test command: ${report.execution.status}")
    val tests = report.evidence.testRecords
    out.println("Test records: ${tests?.entries?.joinToString { "${it.key}: ${it.value}" } ?: "unknown"} (records, not unique IDs)")
    out.println("After scope: ${report.after.nodes.count { it.role == "PRODUCTION" }} production, ${report.after.nodes.count { it.role == "TEST" }} test, ${report.after.nodes.count { it.role == "UNKNOWN" }} unknown-role symbols")
    out.println("Call boundaries: ${report.after.boundaries.size} in scope/unknown caller (${report.after.totalBoundaryCount} repository sites)")
    out.println("\nChanges")
    val changes = if (verbose) report.changes else report.changes.take(8)
    changes.forEach { out.println("  ${it.kind} ${(it.after ?: it.before)?.signature}") }
    if (changes.isEmpty()) out.println("  No method-body/signature changes identified; file/class context impact is not analyzed yet.")
    if (report.changes.size > changes.size) out.println("  ${report.changes.size - changes.size} more; use --verbose")
    out.println("\nAfter evidence and gaps")
    report.matching.datasets.forEach { dataset ->
        out.println("  Dataset: ${dataset.artifact?.source ?: "unavailable"}")
        out.println("  ${dataset.counts.entries.joinToString { "${it.key}: ${it.value}" }}")
        out.println("  Method execution: ${formatPercent(dataset.methodExecutionPercent)} (${dataset.withHits}/${dataset.assessed} assessed; UNKNOWN excluded)")
        val gaps = dataset.methods.filter { it.status != "EXECUTED" }.sortedBy { impactOrder(it.impact) }
        val displayed = if (verbose) gaps else gaps.take(8)
        displayed.forEach { out.println("    ${it.status} [${it.impact}] ${it.signature}") }
        if (gaps.size > displayed.size) out.println("    ${gaps.size - displayed.size} more; use --verbose")
        if (gaps.isEmpty()) out.println("    No method-level gaps in this dataset; this does not prove all execution paths.")
    }
    report.matching.unresolvedChanges.forEach { out.println("  UNRESOLVED CHANGE $it") }
    report.matching.removed.forEach { out.println("  REMOVED $it (after coverage not applicable)") }
    out.println("\nMethod hits and static test routes do not prove a specific test/caller path. Percentages are not safety scores.")
    report.execution.log?.let { out.println("Run log: $it") }
    report.evidence.manifest?.let { out.println("Evidence manifest: $it") }
    if (verbose) renderAnalyzeDetails(report, out) else out.println("Use --verbose for full changes, before/after graphs, boundaries and evidence details.")
}

private fun renderAnalyzeDetails(report: AnalyzeReport, out: PrintStream) {
    out.println("\nFull file changes")
    report.files.forEach { file ->
        out.println("  ${file.status} ${file.beforePath ?: "(added)"} -> ${file.afterPath ?: "(deleted)"}")
        file.lines.forEach { out.println("    before ${it.before.start} +${it.before.count} -> after ${it.after.start} +${it.after.count}") }
    }
    listOf("Before" to report.before, "After" to report.after).forEach { (name, snapshot) ->
        out.println("\n$name impact graph (partial)")
        renderImpactGraph(snapshot, out)
        snapshot.boundaries.forEach { out.println("  BOUNDARY ${it.path}:${it.line}: ${it.reason}: ${it.expression.replace('\n', ' ')}") }
        snapshot.unresolvedChanges.forEach { out.println("  UNRESOLVED CHANGE $it") }
        snapshot.notes.forEach { out.println("  Note: $it") }
    }
    out.println("\nExecution details")
    out.println("  Command: ${report.execution.command.joinToString(" ")}")
    out.println("  JAVA_HOME: ${report.execution.javaHome ?: "unknown"}; exit: ${report.execution.exitCode ?: "unknown"}; duration: ${report.execution.durationMillis}ms")
    report.matching.datasets.forEach { dataset ->
        out.println("  Dataset: ${dataset.artifact?.source ?: "unavailable"}")
        dataset.methods.forEach { method ->
            out.println("    ${method.status} [${method.impact}] ${method.symbol}: ${method.reason}")
            method.instructions?.let { out.println("      Instructions hit/missed: ${it.covered}/${it.missed}; missed branches: ${method.branches?.missed ?: "unknown"}") }
            method.routes.forEach { out.println("      ${it.kind}: ${it.test}; passed class records: ${it.passedClassRecords} (attribution unknown)") }
        }
    }
    report.evidence.tests.forEach { test ->
        out.println("  Test report: ${test.artifact.source}")
        test.cases.forEach { out.println("    ${it.status} ${it.className ?: "unknown"}#${it.name}") }
    }
    report.evidence.coverage.forEach { dataset ->
        out.println("  Coverage: ${dataset.format} ${dataset.artifact.source}; ${dataset.classes.size} class records")
        dataset.notes.forEach { out.println("    Note: $it") }
    }
    report.evidence.artifacts.forEach { out.println("  Artifact: ${it.source}; SHA-256 ${it.sha256}; ${it.retained}") }
    (report.notes + report.execution.notes + report.evidence.notes + report.matching.notes).distinct().forEach { out.println("  Note: $it") }
}

internal fun formatPercent(value: Double?): String = value?.let { String.format(Locale.ROOT, "%.1f%%", it) } ?: "unknown"
private fun impactOrder(impact: String) = when (impact) { "CHANGED" -> 0; "CALLER" -> 1; else -> 2 }
