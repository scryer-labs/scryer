package com.edwardnoaland.scryer.analyze.match

import com.edwardnoaland.scryer.analyze.*
import com.edwardnoaland.scryer.analyze.evidence.*
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus

/** Matches method identities and module output provenance, without inventing path/test attribution. */
class ImpactEvidenceMatcher {
    fun match(impact: ImpactAnalysis, evidence: ExecutionEvidence): ImpactEvidenceReport {
        val after = impact.after
        val scope = after.affected + after.indirect
        val production = after.graph.symbols.filter { it.id in scope && it.role == SourceRole.PRODUCTION }
        val validRun = evidence.afterSha == impact.methods.comparison.after &&
            evidence.commandStatus in setOf(TestRunStatus.SUCCEEDED, TestRunStatus.FAILED)
        val validEvidence = if (validRun) evidence else evidence.copy(tests = emptyList(), coverage = emptyList())
        val datasets = validEvidence.coverage
        val results = if (datasets.isEmpty()) listOf(DatasetImpactEvidence(null, production.map { location ->
            record(location, after, MethodEvidenceStatus.UNKNOWN, "No valid coverage dataset for requested after SHA", validEvidence)
        })) else datasets.map { dataset ->
            DatasetImpactEvidence(dataset.artifact, production.map { location -> assess(location, after, dataset, validEvidence) })
        }
        val notes = mutableListOf(
            "Coverage is aggregate: method hits do not prove a particular caller-to-callee path or a specific test exercised it.",
            "Direct/indirect test routes are static candidates; JUnit records do not establish test-to-method execution attribution.",
            "Each dataset is assessed separately. Method execution percentage excludes UNKNOWN and is not a safety/confidence or path coverage score.",
            "NOT_EXECUTED applies only to methods with instructions in class-ID-matched execution data; missing class records remain UNKNOWN.",
        )
        if (after.graph.symbols.any { it.id in scope && it.role == SourceRole.UNKNOWN }) notes += "Unknown-role scope symbols excluded from production metrics; source roles require review."
        if (evidence.commandStatus != TestRunStatus.SUCCEEDED) notes += "After command ${evidence.commandStatus}; any available hits are execution observations, not a passing build/test result."
        if (after.graph.boundaries.any { it.caller == null || it.caller in scope }) notes += "Unresolved/external call boundaries remain; the analyzed scope is partial."
        return ImpactEvidenceReport(impact.methods.comparison.after, results,
            impact.before.changed.filter { id -> after.graph.symbols.none { it.id == id } }, after.unmatched, notes)
    }

    private fun assess(location: SymbolLocation, impact: SnapshotImpact, dataset: CoverageDataset, evidence: ExecutionEvidence): ImpactMethodEvidence {
        val owners = dataset.classes.filter { it.name == location.id.owner }
        val candidates = owners.filter { owner ->
            val outputModule = owner.compiledFile?.let(::outputModule)
            outputModule != null && outputModule == sourceModule(location)
        }
        if (candidates.size != 1) return record(location, impact, MethodEvidenceStatus.UNKNOWN,
            if (owners.isEmpty()) "No class coverage record" else "Missing/ambiguous module output provenance", evidence)
        val owner = candidates.single()
        if (owner.match != ClassDataMatch.MATCHED) return record(location, impact, MethodEvidenceStatus.UNKNOWN,
            "Class data ${owner.match}; bytecode-matched execution record required", evidence)
        val method = owner.methods.singleOrNull { it.symbol == location.id }
            ?: return record(location, impact, MethodEvidenceStatus.UNKNOWN, "No unique binary method record", evidence)
        val instructions = method.instructions
        val status = when {
            instructions.covered + instructions.missed == 0 -> MethodEvidenceStatus.UNKNOWN
            instructions.covered == 0 -> MethodEvidenceStatus.NOT_EXECUTED
            instructions.missed > 0 || method.branches.missed > 0 -> MethodEvidenceStatus.PARTIALLY_EXECUTED
            else -> MethodEvidenceStatus.EXECUTED
        }
        val reason = when (status) {
            MethodEvidenceStatus.UNKNOWN -> "No executable instructions (for example abstract declaration)"
            MethodEvidenceStatus.NOT_EXECUTED -> "Matched class record, no method instruction hits in this dataset"
            MethodEvidenceStatus.PARTIALLY_EXECUTED -> "Instruction hits with missed instructions/branches; full path protection not established"
            MethodEvidenceStatus.EXECUTED -> "Instruction hits; caller/path and individual-test attribution unknown"
        }
        return record(location, impact, status, reason, evidence).copy(instructions = instructions, branches = method.branches)
    }

    private fun record(location: SymbolLocation, impact: SnapshotImpact, status: MethodEvidenceStatus, reason: String, evidence: ExecutionEvidence): ImpactMethodEvidence {
        val kind = when (location.id) {
            in impact.changed -> ImpactKind.CHANGED
            in impact.affected -> ImpactKind.CALLER
            else -> ImpactKind.POTENTIAL_INDIRECT
        }
        return ImpactMethodEvidence(location, kind, status, reason, routes = routes(location.id, impact.graph, evidence))
    }

    private fun routes(target: SymbolId, graph: CallGraph, evidence: ExecutionEvidence): List<TestRoute> {
        val reverse = graph.edges.groupBy { it.callee }
        val callers = linkedSetOf<SymbolId>()
        val pending = ArrayDeque(listOf(target))
        while (pending.isNotEmpty()) reverse[pending.removeFirst()].orEmpty().forEach { edge ->
            if (callers.add(edge.caller)) pending.addLast(edge.caller)
        }
        val passed = evidence.tests.flatMap { it.cases }.filter { it.status == TestCaseStatus.PASSED }.groupingBy { it.className }.eachCount()
        return graph.symbols.filter { it.role == SourceRole.TEST && it.id in callers }.map { test ->
            val direct = reverse[target].orEmpty().any { it.caller == test.id }
            TestRoute(test.id, if (direct) TestRouteKind.DIRECT_CANDIDATE else TestRouteKind.INDIRECT_CANDIDATE, passed[test.id.owner] ?: 0)
        }.distinct()
    }

    private fun sourceModule(location: SymbolLocation): String? = location.moduleDirectory ?: run {
        val path = location.path.replace('\\', '/')
        when {
            path.startsWith("src/") -> ""
            "/src/" in path -> path.substringBefore("/src/")
            else -> null
        }
    }

    private fun outputModule(path: String): String? {
        val normalized = path.replace('\\', '/')
        val roots = listOf("build/classes/java/main/", "build/classes/main/", "target/classes/")
        for (root in roots) {
            if (normalized.startsWith(root)) return ""
            if ("/$root" in normalized) return normalized.substringBefore("/$root")
        }
        return null
    }
}
