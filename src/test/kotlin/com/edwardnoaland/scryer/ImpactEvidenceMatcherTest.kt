package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.*
import com.edwardnoaland.scryer.analyze.evidence.*
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus
import com.edwardnoaland.scryer.analyze.match.*
import com.edwardnoaland.scryer.cli.output.renderImpactEvidence
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import java.nio.file.Path
import org.junit.jupiter.api.Test
import kotlin.test.*

class ImpactEvidenceMatcherTest {
    private val changed = SymbolId("app.Service", "changed", "()I")
    private val caller = SymbolId("app.Service", "uncoveredCaller", "()I")
    private val sibling = SymbolId("app.Service", "sibling", "()V")
    private val test = SymbolId("app.ServiceTest", "runsOtherPath", "()V")
    private val artifact = EvidenceArtifact("build/jacoco/test.exec", Path.of("/retained/test.exec"), "hash")

    private fun analysis(roles: Map<SymbolId, SourceRole> = emptyMap()): ImpactAnalysis {
        val locations = listOf(changed, caller, sibling, test).map { id ->
            val role = roles[id] ?: if (id == test) SourceRole.TEST else SourceRole.PRODUCTION
            SymbolLocation(id, "src/${if (role == SourceRole.TEST) "test" else "main"}/java/${id.owner.replace('.', '/')}.java", 1, id.toString(), role)
        }
        val graph = CallGraph(locations, listOf(CallEdge(caller, changed, CallKind.DIRECT), CallEdge(caller, sibling, CallKind.DIRECT), CallEdge(test, sibling, CallKind.DIRECT)), emptyList(), emptyList())
        val snapshot = SnapshotImpact(graph, setOf(changed), setOf(changed, caller), emptyList(), setOf(sibling), emptyMap())
        return ImpactAnalysis(MethodAnalysis(GitComparison(Path.of("/repo"), "before", "after", emptyList()), emptyList(), emptyList()), snapshot, snapshot)
    }

    private fun evidence(match: ClassDataMatch = ClassDataMatch.MATCHED, sha: String = "after", status: TestRunStatus = TestRunStatus.SUCCEEDED): ExecutionEvidence {
        fun coverage(symbol: SymbolId, hits: Int, missed: Int, branches: Int = 0) = MethodCoverage(symbol, 1, CoverageCounter(missed, hits), CoverageCounter(branches, 0), CoverageCounter(0, 0))
        val owner = ClassCoverage("app.Service", "build/classes/java/main/app/Service.class", match,
            listOf(coverage(changed, 3, 0), coverage(caller, 0, 5), coverage(sibling, 1, 2, 1)))
        val reports = listOf(TestReport(artifact, listOf(TestCaseRecord("app.ServiceTest", "runsOtherPath", TestCaseStatus.PASSED, 0.1))))
        return ExecutionEvidence(sha, status, reports, listOf(CoverageDataset(artifact, "JACOCO_EXEC", listOf(owner), emptyList())))
    }

    @Test fun `covered change still exposes an uncovered caller and partial indirect branch`() {
        val result = ImpactEvidenceMatcher().match(analysis(), evidence()).datasets.single()
        assertEquals(MethodEvidenceStatus.EXECUTED, result.methods.single { it.location.id == changed }.status)
        assertEquals(MethodEvidenceStatus.NOT_EXECUTED, result.methods.single { it.location.id == caller }.status)
        assertEquals(MethodEvidenceStatus.PARTIALLY_EXECUTED, result.methods.single { it.location.id == sibling }.status)
        assertEquals(ImpactKind.POTENTIAL_INDIRECT, result.methods.single { it.location.id == sibling }.impact)
        assertEquals(3, result.assessed)
        assertEquals(2, result.withHits)
        assertEquals(200.0 / 3, result.methodExecutionPercent)
        assertTrue(result.methods.none { it.location.role == SourceRole.TEST })
    }

    @Test fun `existing passed test class is not evidence for its uncalled changed method`() {
        val result = ImpactEvidenceMatcher().match(analysis(), evidence()).datasets.single()
        assertTrue(result.methods.single { it.location.id == changed }.routes.isEmpty())
        val route = result.methods.single { it.location.id == sibling }.routes.single()
        assertEquals(TestRouteKind.DIRECT_CANDIDATE, route.kind)
        assertEquals(1, route.passedRecords)
    }

    @Test fun `static direct and integration candidates never become causal execution evidence`() {
        val analysis = analysis()
        val graph = analysis.after.graph.copy(edges = analysis.after.graph.edges + CallEdge(test, caller, CallKind.DIRECT))
        val updated = analysis.copy(after = analysis.after.copy(graph = graph))
        val rows = ImpactEvidenceMatcher().match(updated, evidence()).datasets.single().methods
        assertEquals(TestRouteKind.INDIRECT_CANDIDATE, rows.single { it.location.id == changed }.routes.single().kind)
        val callerRow = rows.single { it.location.id == caller }
        assertEquals(TestRouteKind.DIRECT_CANDIDATE, callerRow.routes.single().kind)
        assertEquals(MethodEvidenceStatus.NOT_EXECUTED, callerRow.status)
    }

    @Test fun `mismatched absent execution and XML unverified class data remain unknown`() {
        for (match in listOf(ClassDataMatch.CLASS_ID_MISMATCH, ClassDataMatch.NO_EXECUTION_RECORD, ClassDataMatch.XML_UNVERIFIED)) {
            val dataset = ImpactEvidenceMatcher().match(analysis(), evidence(match)).datasets.single()
            assertTrue(dataset.methods.all { it.status == MethodEvidenceStatus.UNKNOWN })
            assertNull(dataset.methodExecutionPercent)
        }
    }

    @Test fun `wrong SHA skipped and changed snapshot results cannot supply evidence`() {
        val invalid = listOf(evidence(sha = "other"), evidence(status = TestRunStatus.SKIPPED), evidence(status = TestRunStatus.SNAPSHOT_CHANGED))
        for (data in invalid) {
            val result = ImpactEvidenceMatcher().match(analysis(), data)
            assertTrue(result.datasets.single().methods.all { it.status == MethodEvidenceStatus.UNKNOWN })
            assertNull(result.datasets.single().artifact)
            assertTrue(result.datasets.single().methods.flatMap { it.routes }.all { it.passedRecords == 0 })
        }
        assertEquals(MethodEvidenceStatus.EXECUTED, ImpactEvidenceMatcher().match(analysis(), evidence(status = TestRunStatus.FAILED)).datasets.single().methods.first().status)
    }

    @Test fun `wrong module output cannot protect an identically named source symbol`() {
        val analysis = analysis()
        val graph = analysis.after.graph.copy(symbols = analysis.after.graph.symbols.map { it.copy(module = ":other", moduleDirectory = "other") })
        val result = ImpactEvidenceMatcher().match(analysis.copy(after = analysis.after.copy(graph = graph)), evidence())
        assertTrue(result.datasets.single().methods.all { it.status == MethodEvidenceStatus.UNKNOWN })
    }

    @Test fun `independent coverage datasets are not merged into a reassuring union`() {
        val first = evidence()
        val second = first.coverage.single().copy(artifact = artifact.copy(source = "build/jacoco/second.exec"), classes = first.coverage.single().classes.map { owner ->
            owner.copy(methods = owner.methods.map { method -> method.copy(instructions = CoverageCounter(5, 0)) })
        })
        val result = ImpactEvidenceMatcher().match(analysis(), first.copy(coverage = first.coverage + second))
        assertEquals(2, result.datasets.size)
        assertEquals(0.0, result.datasets[1].methodExecutionPercent)
        assertEquals(2, result.datasets[0].withHits)
    }

    @Test fun `removed and unresolved symbols stay outside after method denominator`() {
        val analysis = analysis()
        val removed = SymbolId("app.Service", "removed", "()V")
        val result = ImpactEvidenceMatcher().match(analysis.copy(before = analysis.before.copy(changed = setOf(removed)), after = analysis.after.copy(unmatched = listOf("unresolved"))), evidence())
        assertEquals(listOf(removed), result.removed)
        assertEquals(listOf("unresolved"), result.unresolvedChanges)
        assertEquals(3, result.datasets.single().assessed)
    }

    @Test fun `unknown source roles are excluded and reported rather than promoted to production`() {
        val result = ImpactEvidenceMatcher().match(analysis(mapOf(changed to SourceRole.UNKNOWN)), evidence())
        assertEquals(2, result.datasets.single().methods.size)
        assertTrue(result.notes.any { it.contains("Unknown-role") })
    }

    @Test fun `abstract methods and absent method records have unknown execution`() {
        val data = evidence()
        val dataset = data.coverage.single()
        val owner = dataset.classes.single().copy(methods = listOf(
            MethodCoverage(changed, null, CoverageCounter(0, 0), CoverageCounter(0, 0), CoverageCounter(0, 0)),
        ))
        val result = ImpactEvidenceMatcher().match(analysis(), data.copy(coverage = listOf(dataset.copy(classes = listOf(owner)))))
        assertTrue(result.datasets.single().methods.all { it.status == MethodEvidenceStatus.UNKNOWN })
        assertNull(result.datasets.single().methodExecutionPercent)
    }

    @Test fun `renderer exposes gaps unknown attribution and percentage denominator`() {
        val bytes = ByteArrayOutputStream()
        renderImpactEvidence(ImpactEvidenceMatcher().match(analysis(), evidence()), PrintStream(bytes))
        val text = bytes.toString()
        assertContains(text, "NOT_EXECUTED [CALLER]")
        assertContains(text, "66.7% (2/3 assessed; UNKNOWN excluded; not path coverage or safety)")
        assertContains(text, "attribution unknown")
    }
}
