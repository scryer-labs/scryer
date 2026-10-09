package com.edwardnoaland.scryer.analyze

import com.edwardnoaland.scryer.repository.BuildTool

import com.edwardnoaland.scryer.analyze.match.ImpactEvidenceMatcher
import com.edwardnoaland.scryer.analyze.match.ImpactEvidenceReport
import com.edwardnoaland.scryer.analyze.model.AnalysisInputCollector
import com.edwardnoaland.scryer.analyze.model.AnalysisInputs
import com.edwardnoaland.scryer.analyze.evidence.ArtifactInventory
import com.edwardnoaland.scryer.analyze.evidence.EvidenceCollector
import com.edwardnoaland.scryer.analyze.evidence.ExecutionEvidence
import com.edwardnoaland.scryer.analyze.execute.TargetTestRunner
import com.edwardnoaland.scryer.analyze.execute.TestExecution
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus

data class AnalyzeResult(val impact: ImpactAnalysis, val execution: TestExecution, val evidence: ExecutionEvidence, val matching: ImpactEvidenceReport)

/** Owns the common snapshot lifetime; execution follows static analysis in the same after checkout. */
class AnalyzeService(private val progress: (String) -> Unit = {}) {
    fun analyze(comparison: GitComparison, runTests: Boolean = true, testCommand: String? = null, buildTool: BuildTool? = null): AnalyzeResult = IsolatedSnapshots.use(comparison) { beforeSnapshot, afterSnapshot ->
        val before = beforeSnapshot.toRealPath()
        val after = afterSnapshot.toRealPath()
        if (buildTool != null) {
            for (snapshot in listOf(before, after)) {
                require(java.nio.file.Files.isRegularFile(snapshot.resolve(buildTool.wrapper))) {
                    "Selected build tool ${buildTool.flag} requires ${buildTool.wrapper} in both snapshots"
                }
            }
        }
        val collector = AnalysisInputCollector(progress)
        val beforeInputs = if (runTests) collector.collect(before, comparison, comparison.before, buildTool) else AnalysisInputs(notes = listOf("--skip-tests also skips build-model commands; classpath unavailable." + (buildTool?.let { " Selected build tool: ${it.flag}." } ?: "")))
        val afterInputs = if (runTests) collector.collect(after, comparison, comparison.after, buildTool) else beforeInputs
        val impact = ImpactAnalyzer().analyzeSnapshots(comparison, before, after, beforeInputs, afterInputs)
        val previousArtifacts = if (runTests) ArtifactInventory.capture(after) else emptyMap()
        val execution = if (runTests) TargetTestRunner(progress = progress).run(after, comparison, testCommand, buildTool)
        else TestExecution(comparison.after, TestRunStatus.SKIPPED, notes = listOf("Explicit --skip-tests: no target build or test command executed."))
        val evidence = runCatching { EvidenceCollector().collect(after, execution, previousArtifacts) }.getOrElse {
            ExecutionEvidence(comparison.after, execution.status, notes = listOf("Cannot collect execution artifacts: ${it.message}"))
        }
        val matching = ImpactEvidenceMatcher().match(impact, evidence)
        AnalyzeResult(impact, execution, evidence, matching)
    }
}
