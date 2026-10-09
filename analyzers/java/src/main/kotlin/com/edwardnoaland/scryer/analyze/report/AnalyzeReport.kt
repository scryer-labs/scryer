package com.edwardnoaland.scryer.analyze.report

import com.edwardnoaland.scryer.analyze.ChangedFile
import com.edwardnoaland.scryer.analyze.LineRange
import com.edwardnoaland.scryer.analyze.evidence.CoverageCounter
import com.edwardnoaland.scryer.analyze.evidence.TestCaseRecord

/** Versioned output model shared by terminal, JSON, Markdown and HTML. No Path/map-key serializers needed. */
data class AnalyzeReport(
    val schemaVersion: Int,
    val generatedAt: String,
    val repository: String,
    val beforeSha: String,
    val afterSha: String,
    val files: List<ChangedFile>,
    val changes: List<ReportChange>,
    val before: ReportSnapshot,
    val after: ReportSnapshot,
    val execution: ReportExecution,
    val evidence: ReportEvidence,
    val matching: ReportMatching,
    val notes: List<String>,
)
data class ReportMethodVersion(val signature: String, val path: String?, val lines: LineRange, val source: String)
data class ReportChange(val kind: String, val before: ReportMethodVersion?, val after: ReportMethodVersion?)
data class ReportNode(
    val id: String, val owner: String, val name: String, val descriptor: String,
    val signature: String, val path: String, val line: Int, val role: String,
    val impact: String, val module: String?,
)
data class ReportEdge(val caller: String, val callee: String, val kind: String)
data class ReportBoundary(val caller: String?, val path: String, val line: Long, val expression: String, val reason: String)
data class ReportSnapshot(
    val nodes: List<ReportNode>, val edges: List<ReportEdge>, val boundaries: List<ReportBoundary>,
    val totalBoundaryCount: Int, val unresolvedChanges: List<String>, val notes: List<String>,
)
data class ReportExecution(
    val afterSha: String, val status: String, val command: List<String>, val javaHome: String?,
    val exitCode: Int?, val durationMillis: Long, val log: String?, val notes: List<String>,
)
data class ReportArtifact(val source: String, val retained: String, val sha256: String)
data class ReportTestReport(val artifact: ReportArtifact, val cases: List<TestCaseRecord>)
data class ReportCoverageMethod(val symbol: String, val firstLine: Int?, val instructions: CoverageCounter, val branches: CoverageCounter, val lines: CoverageCounter)
data class ReportCoverageClass(val name: String, val compiledFile: String?, val match: String, val classId: String?, val methods: List<ReportCoverageMethod>)
data class ReportCoverageDataset(val artifact: ReportArtifact, val format: String, val classes: List<ReportCoverageClass>, val notes: List<String>)
data class ReportEvidence(
    val afterSha: String, val commandStatus: String,
    val testRecords: Map<String, Int>?, val tests: List<ReportTestReport>,
    val coverage: List<ReportCoverageDataset>, val artifacts: List<ReportArtifact>,
    val manifest: String?, val notes: List<String>,
)
data class ReportTestRoute(val test: String, val kind: String, val passedClassRecords: Int)
data class ReportMethodEvidence(
    val symbol: String, val signature: String, val path: String, val line: Int, val role: String,
    val impact: String, val status: String, val reason: String,
    val instructions: CoverageCounter?, val branches: CoverageCounter?, val routes: List<ReportTestRoute>,
)
data class ReportDatasetMatch(
    val artifact: ReportArtifact?, val methods: List<ReportMethodEvidence>,
    val counts: Map<String, Int>, val assessed: Int, val withHits: Int, val methodExecutionPercent: Double?,
)
data class ReportMatching(
    val afterSha: String, val datasets: List<ReportDatasetMatch>, val removed: List<String>,
    val unresolvedChanges: List<String>, val notes: List<String>,
)
