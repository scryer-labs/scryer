package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.analyze.match.*
import java.io.PrintStream
import java.util.Locale

internal fun renderImpactEvidence(report: ImpactEvidenceReport, out: PrintStream) {
    out.println("After impact / execution evidence (SHA ${report.afterSha}):")
    report.datasets.forEach { dataset ->
        out.println("  Dataset: ${dataset.artifact?.source ?: "unavailable"}")
        MethodEvidenceStatus.entries.forEach { status -> out.println("    $status: ${dataset.methods.count { it.status == status }} production methods") }
        val percent = dataset.methodExecutionPercent?.let { String.format(Locale.ROOT, "%.1f%%", it) } ?: "unknown"
        out.println("    Method execution: $percent (${dataset.withHits}/${dataset.assessed} assessed; UNKNOWN excluded; not path coverage or safety)")
        dataset.methods.forEach { method ->
            out.println("    ${method.status} [${method.impact}] ${method.location.id}")
            method.instructions?.let { out.println("      Instruction hits: ${it.covered}; missed: ${it.missed}; missed branches: ${method.branches?.missed ?: "unknown"}") }
            out.println("      ${method.reason}")
            if (method.routes.isEmpty()) out.println("      Static test routes: none found")
            else method.routes.forEach { route -> out.println("      ${route.kind}: ${route.test}; passed class records: ${route.passedRecords} (attribution unknown)") }
        }
    }
    report.removed.forEach { out.println("  Removed before symbol: $it (after coverage not applicable)") }
    report.unresolvedChanges.forEach { out.println("  Unresolved change: $it (not included in method metrics)") }
    report.notes.forEach { out.println("  Note: $it") }
}
