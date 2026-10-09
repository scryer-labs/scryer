package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.analyze.report.ReportEdge
import com.edwardnoaland.scryer.analyze.report.ReportSnapshot
import com.edwardnoaland.scryer.analyze.report.snapshotReport
import com.edwardnoaland.scryer.analyze.SnapshotImpact
import java.io.PrintStream

/** Expand each node once; references retain every edge without infinite recursive paths. */
internal fun renderImpactGraph(impact: SnapshotImpact, out: PrintStream) {
    renderImpactGraph(snapshotReport(impact), out)
}

internal fun renderImpactGraph(impact: ReportSnapshot, out: PrintStream) {
    if (impact.nodes.isEmpty()) {
        out.println("  (no resolved changed symbols)")
        return
    }
    val locations = impact.nodes.associateBy { it.id }
    val nodes = locations.keys.sorted()
    val numbers = nodes.withIndex().associate { it.value to it.index + 1 }
    val edges = impact.edges
    val outgoing = edges.groupBy { it.caller }
    val boundaries = impact.boundaries.filter { it.caller != null }.groupBy { it.caller }
    val targets = edges.map { it.callee }.toSet()
    val roots = nodes.filter { it !in targets }
    val visited = mutableSetOf<String>()
    val active = mutableSetOf<String>()
    data class Visit(val node: String, val indent: String, val edge: ReportEdge? = null, val exit: Boolean = false)
    val pending = ArrayDeque<Visit>()

    out.println("  Arrows: caller -> callee; # references share a node; CYCLE closes a recursive edge")
    for (root in roots + nodes) {
        if (root in visited) continue
        pending.addLast(Visit(root, "  "))
        while (pending.isNotEmpty()) {
            val visit = pending.removeLast()
            if (visit.exit) { active.remove(visit.node); continue }
            val node = visit.node
            val prefix = visit.edge?.let { "|-- ${it.kind} -> " }.orEmpty()
            val location = locations[node]
            val category = location?.impact?.replace('_', ' ') ?: "UNKNOWN"
            val name = location?.signature ?: node
            val reference = when {
                node in active -> " [CYCLE -> #${numbers[node]}]"
                node in visited -> " [see #${numbers[node]}]"
                else -> ""
            }
            out.println("${visit.indent}$prefix#${numbers[node]} $name [$category] [${location?.role ?: "UNKNOWN"}]$reference")
            if (!visited.add(node)) continue
            active.add(node)
            val childIndent = visit.indent + "|   "
            val boundaryCount = boundaries[node].orEmpty().size
            if (boundaryCount > 0) {
                out.println("${childIndent}|-- ? BOUNDARY $boundaryCount site(s); details below")
            }
            pending.addLast(Visit(node, visit.indent, exit = true))
            outgoing[node].orEmpty().sortedWith(compareBy<ReportEdge> { it.callee }.thenBy { it.kind })
                .asReversed().forEach { pending.addLast(Visit(it.callee, childIndent, it)) }
        }
    }
}
