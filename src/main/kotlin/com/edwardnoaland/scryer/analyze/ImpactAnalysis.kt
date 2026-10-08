package com.edwardnoaland.scryer.analyze

import com.edwardnoaland.scryer.analyze.model.AnalysisInputs
import java.nio.file.Path

data class SnapshotImpact(val graph: CallGraph, val changed: Set<SymbolId>, val affected: Set<SymbolId>, val unmatched: List<String>, val indirect: Set<SymbolId>, val indirectReasons: Map<SymbolId, CallEdge>)
data class ImpactAnalysis(val methods: MethodAnalysis, val before: SnapshotImpact, val after: SnapshotImpact)

class ImpactAnalyzer {
    fun analyze(comparison: GitComparison): ImpactAnalysis = IsolatedSnapshots.use(comparison) { beforeRoot, afterRoot ->
        analyzeSnapshots(comparison, beforeRoot, afterRoot)
    }

    internal fun analyzeSnapshots(
        comparison: GitComparison,
        beforeRoot: Path,
        afterRoot: Path,
        beforeInputs: AnalysisInputs = AnalysisInputs(),
        afterInputs: AnalysisInputs = AnalysisInputs(),
    ): ImpactAnalysis {
        val methods = MethodAnalyzer().analyzeSnapshots(comparison, beforeRoot, afterRoot)
        val collector = CallGraphCollector()
        return ImpactAnalysis(methods,
            impact(collector.collect(beforeRoot, beforeInputs), methods.methods, before = true),
            impact(collector.collect(afterRoot, afterInputs), methods.methods, before = false))
    }

    private fun impact(graph: CallGraph, changes: List<MethodChange>, before: Boolean): SnapshotImpact {
        val changed = linkedSetOf<SymbolId>()
        val unmatched = mutableListOf<String>()
        for (change in changes) {
            val method = (if (before) change.before else change.after) ?: continue
            val path = if (before) change.beforePath else change.afterPath
            val matches = graph.symbols.filter { it.path == path && it.sourceSignature == method.signature }
            if (matches.size == 1) changed += matches.single().id else unmatched += "$path: ${method.signature}"
        }
        val affected = changed.toMutableSet()
        val callers = graph.edges.groupBy { it.callee }
        val pending = ArrayDeque(changed)
        while (pending.isNotEmpty()) {
            callers[pending.removeFirst()].orEmpty().forEach { edge ->
                if (affected.add(edge.caller)) pending.addLast(edge.caller)
            }
        }
        val expanded = affected.toMutableSet()
        val reasons = linkedMapOf<SymbolId, CallEdge>()
        val callees = graph.edges.groupBy { it.caller }
        pending.addAll(affected)
        while (pending.isNotEmpty()) {
            callees[pending.removeFirst()].orEmpty().forEach { edge ->
                if (expanded.add(edge.callee)) {
                    reasons[edge.callee] = edge
                    pending.addLast(edge.callee)
                }
            }
        }
        return SnapshotImpact(graph, changed, affected, unmatched, expanded - affected, reasons)
    }
}
