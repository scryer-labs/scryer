package com.edwardnoaland.scryer.analyze.report

import com.edwardnoaland.scryer.analyze.*
import com.edwardnoaland.scryer.analyze.evidence.EvidenceArtifact
import com.edwardnoaland.scryer.analyze.evidence.TestCaseStatus
import com.edwardnoaland.scryer.analyze.match.MethodEvidenceStatus
import java.time.Instant

fun buildAnalyzeReport(result: AnalyzeResult, generatedAt: Instant = Instant.now()): AnalyzeReport {
    val comparison = result.impact.methods.comparison
    val execution = result.execution
    val evidence = result.evidence
    val matching = result.matching
    return AnalyzeReport(
        schemaVersion = 1, generatedAt = generatedAt.toString(), repository = comparison.repository.toString(),
        beforeSha = comparison.before, afterSha = comparison.after, files = comparison.files,
        changes = result.impact.methods.methods.map { change ->
            ReportChange(change.kind.name, change.before?.let { ReportMethodVersion(it.signature, change.beforePath, it.lines, it.syntax) },
                change.after?.let { ReportMethodVersion(it.signature, change.afterPath, it.lines, it.syntax) })
        },
        before = snapshotReport(result.impact.before), after = snapshotReport(result.impact.after),
        execution = ReportExecution(execution.afterSha, execution.status.name, execution.command, execution.javaHome,
            execution.exitCode, execution.durationMillis, execution.log?.toString(), execution.notes),
        evidence = ReportEvidence(
            afterSha = evidence.afterSha, commandStatus = evidence.commandStatus.name,
            testRecords = if (evidence.tests.isEmpty()) null else TestCaseStatus.entries.associate { status ->
                status.name to evidence.tests.sumOf { report -> report.cases.count { it.status == status } }
            },
            tests = evidence.tests.map { ReportTestReport(artifactReport(it.artifact), it.cases) },
            coverage = evidence.coverage.map { dataset ->
                ReportCoverageDataset(artifactReport(dataset.artifact), dataset.format, dataset.classes.map { owner ->
                    ReportCoverageClass(owner.name, owner.compiledFile, owner.match.name, owner.classId, owner.methods.map { method ->
                        ReportCoverageMethod(method.symbol.toString(), method.firstLine, method.instructions, method.branches, method.lines)
                    })
                }, dataset.notes)
            },
            artifacts = evidence.artifacts.map(::artifactReport), manifest = evidence.manifest?.toString(), notes = evidence.notes,
        ),
        matching = ReportMatching(matching.afterSha, matching.datasets.map { dataset ->
            ReportDatasetMatch(dataset.artifact?.let(::artifactReport), dataset.methods.map { method ->
                ReportMethodEvidence(method.location.id.toString(), method.location.sourceSignature, method.location.path,
                    method.location.line, method.location.role.name, method.impact.name, method.status.name, method.reason,
                    method.instructions, method.branches, method.routes.map { ReportTestRoute(it.test.toString(), it.kind.name, it.passedRecords) })
            }, MethodEvidenceStatus.entries.associate { status -> status.name to dataset.methods.count { it.status == status } },
                dataset.assessed, dataset.withHits, dataset.methodExecutionPercent)
        }, matching.removed.map { it.toString() }, matching.unresolvedChanges, matching.notes),
        notes = result.impact.methods.notes,
    )
}

internal fun snapshotReport(impact: SnapshotImpact): ReportSnapshot {
    val scope = impact.affected + impact.indirect
    val locations = impact.graph.symbols.associateBy { it.id }
    val nodes = scope.sortedBy { it.toString() }.map { id ->
        val location = locations[id]
        val kind = when (id) {
            in impact.changed -> "CHANGED"
            in impact.affected -> "CALLER"
            else -> "POTENTIAL_INDIRECT"
        }
        ReportNode(id.toString(), id.owner, id.name, id.descriptor,
            location?.sourceSignature ?: id.toString(), location?.path.orEmpty(), location?.line ?: 0,
            location?.role?.name ?: "UNKNOWN", kind, location?.module)
    }
    val edges = impact.graph.edges.filter { it.caller in scope && it.callee in scope }
        .sortedWith(compareBy<CallEdge> { it.caller.toString() }.thenBy { it.callee.toString() }.thenBy { it.kind.name })
        .map { ReportEdge(it.caller.toString(), it.callee.toString(), it.kind.name) }
    val boundaries = impact.graph.boundaries.filter { it.caller == null || it.caller in scope }
        .map { ReportBoundary(it.caller?.toString(), it.path, it.line, it.expression, it.reason) }
    return ReportSnapshot(nodes, edges, boundaries, impact.graph.boundaries.size, impact.unmatched, impact.graph.notes)
}

private fun artifactReport(artifact: EvidenceArtifact) = ReportArtifact(artifact.source, artifact.retained.toString(), artifact.sha256)
