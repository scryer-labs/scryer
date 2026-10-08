package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.analyze.evidence.ClassDataMatch
import com.edwardnoaland.scryer.analyze.evidence.ExecutionEvidence
import com.edwardnoaland.scryer.analyze.evidence.TestCaseStatus
import java.io.PrintStream

internal fun renderExecutionEvidence(evidence: ExecutionEvidence, out: PrintStream) {
    out.println("After execution artifacts (SHA ${evidence.afterSha}):")
    val cases = evidence.tests.flatMap { it.cases }
    if (evidence.tests.isEmpty()) out.println("  Test records: unknown")
    else {
        out.println("  Test records: ${cases.size} from ${evidence.tests.size} XML reports (records, not unique test IDs)")
        TestCaseStatus.entries.forEach { status -> out.println("    $status: ${cases.count { it.status == status }}") }
    }
    if (evidence.coverage.isEmpty()) out.println("  Coverage: unavailable")
    for (dataset in evidence.coverage) {
        out.println("  ${dataset.format}: ${dataset.artifact.source}")
        out.println("    Artifact: ${dataset.artifact.retained}")
        val methods = dataset.classes.filter { it.match != ClassDataMatch.CLASS_ID_MISMATCH }.flatMap { it.methods }
        out.println("    Method records: ${methods.size}; with instruction hits: ${methods.count { it.instructions.covered > 0 }}")
        ClassDataMatch.entries.forEach { status ->
            val count = dataset.classes.count { it.match == status }
            if (count > 0) out.println("    $status classes: $count")
        }
        dataset.notes.forEach { out.println("    Note: $it") }
    }
    evidence.manifest?.let { out.println("  Artifact manifest: $it") }
    evidence.notes.forEach { out.println("  Note: $it") }
    evidence.tests.firstOrNull()?.let { out.println("  Retained test report: ${it.artifact.retained}") }
    out.println("  No per-test attribution or safety percentage is inferred from artifact collection alone.")
}
