package com.edwardnoaland.scryer.analyze.match

import com.edwardnoaland.scryer.analyze.SymbolLocation
import com.edwardnoaland.scryer.analyze.SymbolId
import com.edwardnoaland.scryer.analyze.evidence.CoverageCounter
import com.edwardnoaland.scryer.analyze.evidence.EvidenceArtifact

enum class ImpactKind { CHANGED, CALLER, POTENTIAL_INDIRECT }
enum class MethodEvidenceStatus { EXECUTED, PARTIALLY_EXECUTED, NOT_EXECUTED, UNKNOWN }
enum class TestRouteKind { DIRECT_CANDIDATE, INDIRECT_CANDIDATE }
/** Static candidate plus independent JUnit class records; never a test-to-method execution claim. */
data class TestRoute(val test: SymbolId, val kind: TestRouteKind, val passedRecords: Int)
data class ImpactMethodEvidence(
    val location: SymbolLocation,
    val impact: ImpactKind,
    val status: MethodEvidenceStatus,
    val reason: String,
    val instructions: CoverageCounter? = null,
    val branches: CoverageCounter? = null,
    val routes: List<TestRoute> = emptyList(),
)
data class DatasetImpactEvidence(val artifact: EvidenceArtifact?, val methods: List<ImpactMethodEvidence>) {
    val assessed: Int get() = methods.count { it.status != MethodEvidenceStatus.UNKNOWN }
    val withHits: Int get() = methods.count { it.status in setOf(MethodEvidenceStatus.EXECUTED, MethodEvidenceStatus.PARTIALLY_EXECUTED) }
    val methodExecutionPercent: Double? get() = if (assessed == 0) null else 100.0 * withHits / assessed
}
data class ImpactEvidenceReport(
    val afterSha: String,
    val datasets: List<DatasetImpactEvidence>,
    val removed: List<SymbolId>,
    val unresolvedChanges: List<String>,
    val notes: List<String>,
)
