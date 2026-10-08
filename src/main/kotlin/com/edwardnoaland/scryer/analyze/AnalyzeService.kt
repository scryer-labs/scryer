package com.edwardnoaland.scryer.analyze

import com.edwardnoaland.scryer.analyze.model.AnalysisInputCollector
import com.edwardnoaland.scryer.analyze.model.AnalysisInputs
import com.edwardnoaland.scryer.analyze.evidence.ArtifactInventory
import com.edwardnoaland.scryer.analyze.evidence.EvidenceCollector
import com.edwardnoaland.scryer.analyze.evidence.ExecutionEvidence
import com.edwardnoaland.scryer.analyze.execute.TargetTestRunner
import com.edwardnoaland.scryer.analyze.execute.TestExecution
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus

data class AnalyzeResult(val impact: ImpactAnalysis, val execution: TestExecution, val evidence: ExecutionEvidence)

/** Owns the common snapshot lifetime; execution follows static analysis in the same after checkout. */
class AnalyzeService(private val progress: (String) -> Unit = {}) {
    fun analyze(comparison: GitComparison, runTests: Boolean = true): AnalyzeResult = IsolatedSnapshots.use(comparison) { beforeSnapshot, afterSnapshot ->
        val before = beforeSnapshot.toRealPath()
        val after = afterSnapshot.toRealPath()
        val collector = AnalysisInputCollector(progress)
        val beforeInputs = if (runTests) collector.collect(before, comparison, comparison.before) else AnalysisInputs(notes = listOf("--skip-tests also skips build-model commands; classpath unavailable."))
        val afterInputs = if (runTests) collector.collect(after, comparison, comparison.after) else beforeInputs
        val impact = ImpactAnalyzer().analyzeSnapshots(comparison, before, after, beforeInputs, afterInputs)
        val previousArtifacts = if (runTests) ArtifactInventory.capture(after) else emptyMap()
        val execution = if (runTests) TargetTestRunner(progress = progress).run(after, comparison)
        else TestExecution(comparison.after, TestRunStatus.SKIPPED, notes = listOf("Explicit --skip-tests: no target build or test command executed."))
        val evidence = runCatching { EvidenceCollector().collect(after, execution, previousArtifacts) }.getOrElse {
            ExecutionEvidence(comparison.after, execution.status, notes = listOf("Cannot collect execution artifacts: ${it.message}"))
        }
        AnalyzeResult(impact, execution, evidence)
    }
}
