package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.analyze.report.*

internal fun renderAnalyzeMarkdown(report: AnalyzeReport): String = buildString {
    appendLine("# Scryer analysis")
    appendLine()
    table(listOf("Fact", "Value"), listOf(
        listOf("Repository", report.repository), listOf("Before", report.beforeSha), listOf("After", report.afterSha),
        listOf("Generated", report.generatedAt), listOf("Schema", report.schemaVersion.toString()),
        listOf("Changed files", report.files.size.toString()), listOf("Changed Java methods", report.changes.size.toString()),
        listOf("After command", report.execution.status), listOf("Test records", report.evidence.testRecords?.entries?.joinToString { "${it.key}: ${it.value}" } ?: "unknown"),
    ))
    appendLine("Test record counts are not unique test IDs. Method hits do not prove a particular caller path or individual test attribution. Percentages are not safety scores.")
    appendLine()
    appendLine("## After evidence and gaps")
    appendLine()
    report.matching.datasets.forEach { dataset ->
        appendLine("### Dataset: ${markdownText(dataset.artifact?.source ?: "unavailable")}")
        appendLine()
        appendLine("Method execution: **${formatPercent(dataset.methodExecutionPercent)}** (${dataset.withHits}/${dataset.assessed} assessed; UNKNOWN excluded).")
        appendLine()
        table(listOf("Status", "Methods"), dataset.counts.map { listOf(it.key, it.value.toString()) })
        table(listOf("Production symbol", "Impact", "Evidence", "Instructions hit / missed", "Missed branches", "Source"), dataset.methods.map { method ->
            listOf(method.symbol, method.impact, method.status, method.instructions?.let { "${it.covered} / ${it.missed}" } ?: "unknown",
                method.branches?.missed?.toString() ?: "unknown", "${method.path}:${method.line}")
        })
        dataset.methods.forEach { method ->
            appendLine("- **${markdownText(method.signature)}**: ${markdownText(method.reason)}")
            method.routes.forEach { route -> appendLine("  - ${route.kind}: ${markdownText(route.test)}; ${route.passedClassRecords} passed class records; attribution unknown.") }
        }
        appendLine()
    }
    appendLine("## Before / after impact graphs")
    appendLine()
    appendLine("Arrows point from caller to callee. Changed symbols are red, callers blue, and potential indirect branches amber. TEST/UNKNOWN roles remain explicit. All resolved scope edges, including cycles and shared nodes, are retained; unresolved calls remain boundaries.")
    appendLine()
    listOf("Before" to report.before, "After" to report.after).forEach { (label, snapshot) ->
        appendLine("### $label")
        appendLine()
        if (snapshot.nodes.isEmpty()) appendLine("No resolved changed scope symbols.") else codeBlock("mermaid", renderAnalyzeMermaid(snapshot))
        appendLine()
        appendLine("Boundaries: ${snapshot.boundaries.size} in scope/unknown caller; ${snapshot.totalBoundaryCount} repository sites.")
        appendLine()
        table(listOf("Caller", "Source", "Reason", "Expression"), snapshot.boundaries.map { listOf(it.caller ?: "unknown", "${it.path}:${it.line}", it.reason, it.expression) })
        snapshot.unresolvedChanges.forEach { appendLine("- Unresolved change: ${markdownText(it)}") }
        snapshot.notes.forEach { appendLine("- ${markdownText(it)}") }
        appendLine()
    }
    appendLine("## Changes")
    appendLine()
    table(listOf("Status", "Before path", "After path", "Changed line ranges"), report.files.map { file ->
        listOf(file.status, file.beforePath ?: "added", file.afterPath ?: "deleted", file.lines.joinToString { "before ${it.before.start} +${it.before.count} → after ${it.after.start} +${it.after.count}" })
    })
    report.changes.forEach { change ->
        appendLine("### ${change.kind}: ${markdownText((change.after ?: change.before)?.signature.orEmpty())}")
        appendLine()
        listOf("Before" to change.before, "After" to change.after).forEach { (label, method) ->
            if (method != null) {
                appendLine("$label: ${markdownText(method.path.orEmpty())}, line ${method.lines.start}, ${method.lines.count} line(s).")
                appendLine()
                codeBlock("java", method.source)
                appendLine()
            }
        }
    }
    appendLine("## Execution and artifacts")
    appendLine()
    table(listOf("Fact", "Value"), listOf(listOf("Status", report.execution.status), listOf("After SHA", report.execution.afterSha),
        listOf("JAVA_HOME", report.execution.javaHome ?: "unknown"), listOf("Exit", report.execution.exitCode?.toString() ?: "unknown"),
        listOf("Duration", "${report.execution.durationMillis}ms"), listOf("Log", report.execution.log ?: "unavailable"), listOf("Manifest", report.evidence.manifest ?: "unavailable")))
    codeBlock("text", report.execution.command.joinToString(" "))
    appendLine()
    report.evidence.tests.forEach { test ->
        appendLine("### Test report: ${markdownText(test.artifact.source)}")
        appendLine()
        table(listOf("Class", "Test record", "Status", "Seconds"), test.cases.map { listOf(it.className ?: "unknown", it.name, it.status.name, it.seconds?.toString() ?: "unknown") })
    }
    report.evidence.coverage.forEach { dataset ->
        appendLine("### Coverage: ${markdownText(dataset.artifact.source)}")
        appendLine()
        table(listOf("Class", "Bytecode match", "Class ID", "Compiled output"), dataset.classes.map { listOf(it.name, it.match, it.classId ?: "unknown", it.compiledFile ?: "unknown") })
        dataset.notes.forEach { appendLine("- ${markdownText(it)}") }
        appendLine()
    }
    table(listOf("Artifact", "SHA-256", "Retained path"), report.evidence.artifacts.map { listOf(it.source, it.sha256, it.retained) })
    appendLine("## Limits and unresolved changes")
    appendLine()
    report.matching.removed.forEach { appendLine("- Removed before symbol: ${markdownText(it)}; after coverage not applicable.") }
    report.matching.unresolvedChanges.forEach { appendLine("- Unresolved change: ${markdownText(it)}; excluded from method metrics.") }
    (report.notes + report.execution.notes + report.evidence.notes + report.matching.notes).distinct().forEach { appendLine("- ${markdownText(it)}") }
}

internal fun renderAnalyzeMermaid(snapshot: ReportSnapshot): String = buildString {
    appendLine("flowchart LR")
    val ids = snapshot.nodes.withIndex().associate { it.value.id to "n${it.index}" }
    snapshot.nodes.forEach { node ->
        val label = "${node.signature} [${node.impact}] [${node.role}]"
        appendLine("  ${ids.getValue(node.id)}[\"${mermaidText(label)}\"]")
        val style = when (node.impact) { "CHANGED" -> "changed"; "CALLER" -> "caller"; else -> "indirect" }
        appendLine("  class ${ids.getValue(node.id)} $style")
    }
    snapshot.edges.forEach { edge ->
        val caller = ids.getValue(edge.caller)
        val callee = ids.getValue(edge.callee)
        appendLine("  $caller -->|${edge.kind}| $callee")
    }
    appendLine("  classDef changed fill:#ffe3e5,stroke:#d73952,color:#541325")
    appendLine("  classDef caller fill:#dfedff,stroke:#3978cf,color:#143359")
    appendLine("  classDef indirect fill:#fff0ce,stroke:#bf811a,color:#523805")
}

private fun mermaidText(text: String): String = buildString {
    text.forEach { character ->
        if (character.isLetterOrDigit() || character in " .,_$()-") append(character)
        else append("#${character.code};")
    }
}

private fun markdownText(value: String): String = value.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
    .replace("\\", "\\\\").replace("|", "\\|").replace("`", "\\`").replace("*", "\\*").replace("_", "\\_")
    .replace("[", "\\[").replace("]", "\\]").replace("\r", "").replace("\n", "<br>")

private fun StringBuilder.table(headers: List<String>, rows: List<List<String>>) {
    appendLine("| ${headers.joinToString(" | ") { markdownText(it) }} |")
    appendLine("| ${headers.joinToString(" | ") { "---" }} |")
    rows.forEach { appendLine("| ${it.joinToString(" | ") { cell -> markdownText(cell) }} |") }
    appendLine()
}

private fun StringBuilder.codeBlock(language: String, source: String) {
    val fence = "`".repeat(maxOf(3, (Regex("`+").findAll(source).maxOfOrNull { it.value.length } ?: 0) + 1))
    appendLine("$fence$language")
    appendLine(source.trimEnd())
    appendLine(fence)
}
