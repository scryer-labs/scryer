package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.*
import com.edwardnoaland.scryer.analyze.evidence.*
import com.edwardnoaland.scryer.analyze.execute.*
import com.edwardnoaland.scryer.analyze.match.ImpactEvidenceMatcher
import com.edwardnoaland.scryer.analyze.report.*
import java.nio.file.Path
import java.time.Instant

internal fun exampleAnalyzeReport(): AnalyzeReport {
    val changed = SymbolId("example.Service", "changed", "()I")
    val caller = SymbolId("example.Service", "caller", "()I")
    val sibling = SymbolId("example.Service", "sibling", "()V")
    val test = SymbolId("example.ServiceTest", "checksChanged", "()V")
    val ids = listOf(changed, caller, sibling, test)
    val locations = ids.map { id ->
        val role = if (id == test) SourceRole.TEST else SourceRole.PRODUCTION
        SymbolLocation(id, "src/${if (role == SourceRole.TEST) "test" else "main"}/java/Service.java", 2, "${id.owner}#${id.name}()", role)
    }
    val graph = CallGraph(locations, listOf(CallEdge(caller, changed, CallKind.DIRECT), CallEdge(caller, sibling, CallKind.REFLECTION), CallEdge(test, changed, CallKind.DIRECT)),
        listOf(CallBoundary(caller, locations.first().path, 3, "external()", "Unresolved call")), listOf("Partial static graph"))
    val snapshot = SnapshotImpact(graph, setOf(changed), setOf(changed, caller, test), emptyList(), setOf(sibling), emptyMap())
    val comparison = GitComparison(Path.of("/example/project"), "before-sha", "after-sha", listOf(ChangedFile("M", locations.first().path, locations.first().path,
        listOf(ChangedLines(LineRange(2, 1), LineRange(2, 1))))))
    val old = JavaMethod("example.Service", "changed", emptyList(), LineRange(2, 1), "int changed() { return 1; }")
    val new = old.copy(syntax = "int changed() { return 2; }")
    val methods = MethodAnalysis(comparison, listOf(MethodChange(MethodChangeKind.MODIFIED, locations.first().path, locations.first().path, old, new)), listOf("File/class context impact unavailable"))
    val impact = ImpactAnalysis(methods, snapshot, snapshot)
    val artifact = EvidenceArtifact("build/jacoco/test.exec", Path.of("/retained/test.exec"), "1234")
    val records = listOf(MethodCoverage(changed, 2, CoverageCounter(0, 3), CoverageCounter(0, 0), CoverageCounter(0, 1)),
        MethodCoverage(caller, 2, CoverageCounter(4, 0), CoverageCounter(0, 0), CoverageCounter(1, 0)),
        MethodCoverage(sibling, 2, CoverageCounter(1, 1), CoverageCounter(1, 1), CoverageCounter(1, 1)))
    val evidence = ExecutionEvidence("after-sha", TestRunStatus.SUCCEEDED,
        tests = listOf(TestReport(artifact.copy(source = "build/test-results/test/TEST-Service.xml"), listOf(TestCaseRecord("example.ServiceTest", "checksChanged", TestCaseStatus.PASSED, 0.1)))),
        coverage = listOf(CoverageDataset(artifact, "JACOCO_EXEC", listOf(ClassCoverage("example.Service", "build/classes/java/main/example/Service.class", ClassDataMatch.MATCHED, records, "abcd")), listOf("Aggregate scope"))),
        artifacts = listOf(artifact), manifest = Path.of("/retained/manifest.json"))
    val execution = TestExecution("after-sha", TestRunStatus.SUCCEEDED, listOf("sh", "./gradlew", "clean", "test"), "/jdk", 0, 100, Path.of("/retained/test.log"))
    return buildAnalyzeReport(AnalyzeResult(impact, execution, evidence, ImpactEvidenceMatcher().match(impact, evidence)), Instant.parse("2026-10-08T12:00:00Z"))
}
