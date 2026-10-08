package com.edwardnoaland.scryer.analyze.evidence

import com.edwardnoaland.scryer.analyze.SymbolId
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus
import java.nio.file.Path

enum class TestCaseStatus { PASSED, FAILED, ERROR, SKIPPED }
data class TestCaseRecord(val className: String?, val name: String, val status: TestCaseStatus, val seconds: Double?)
data class TestReport(val artifact: EvidenceArtifact, val cases: List<TestCaseRecord>)
data class EvidenceArtifact(val source: String, val retained: Path, val sha256: String)
data class CoverageCounter(val missed: Int, val covered: Int)
enum class ClassDataMatch { MATCHED, NO_EXECUTION_RECORD, CLASS_ID_MISMATCH, XML_UNVERIFIED }
data class MethodCoverage(val symbol: SymbolId, val firstLine: Int?, val instructions: CoverageCounter, val branches: CoverageCounter, val lines: CoverageCounter)
data class ClassCoverage(val name: String, val compiledFile: String?, val match: ClassDataMatch, val methods: List<MethodCoverage>, val classId: String? = null)
data class CoverageDataset(val artifact: EvidenceArtifact, val format: String, val classes: List<ClassCoverage>, val notes: List<String>)
data class ExecutionEvidence(
    val afterSha: String,
    val commandStatus: TestRunStatus,
    val tests: List<TestReport> = emptyList(),
    val coverage: List<CoverageDataset> = emptyList(),
    val notes: List<String> = emptyList(),
    val artifacts: List<EvidenceArtifact> = emptyList(),
    val manifest: Path? = null,
)
