package com.edwardnoaland.scryer.analyze

import com.edwardnoaland.scryer.analyze.execute.TargetTestRunner
import com.edwardnoaland.scryer.analyze.execute.TestExecution
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus

data class AnalyzeResult(val impact: ImpactAnalysis, val execution: TestExecution)

/** Owns the common snapshot lifetime; execution follows static analysis in the same after checkout. */
class AnalyzeService(private val progress: (String) -> Unit = {}) {
    fun analyze(comparison: GitComparison, runTests: Boolean = true): AnalyzeResult = IsolatedSnapshots.use(comparison) { before, after ->
        val impact = ImpactAnalyzer().analyzeSnapshots(comparison, before, after)
        val execution = if (runTests) TargetTestRunner(progress = progress).run(after, comparison)
        else TestExecution(comparison.after, TestRunStatus.SKIPPED, notes = listOf("Explicit --skip-tests: no target build or test command executed."))
        AnalyzeResult(impact, execution)
    }
}
