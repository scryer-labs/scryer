package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.analyze.CallEdge
import com.edwardnoaland.scryer.analyze.SnapshotImpact
import com.edwardnoaland.scryer.analyze.SymbolId
import java.io.PrintStream

/** Expand each node once; references retain every edge without infinite recursive paths. */
internal fun renderImpactGraph(impact: SnapshotImpact, out: PrintStream) {
    val scope = impact.affected + impact.indirect
    if (scope.isEmpty()) {
        out.println("  (no resolved changed symbols)")
        return
    }
    val locations = impact.graph.symbols.associateBy { it.id }
    val nodes = scope.sortedBy { it.toString() }
    val numbers = nodes.withIndex().associate { it.value to it.index + 1 }
    val edges = impact.graph.edges.filter { it.caller in scope && it.callee in scope }
    val outgoing = edges.groupBy { it.caller }
    val boundaries = impact.graph.boundaries.filter { it.caller in scope }.groupBy { it.caller }
    val targets = edges.map { it.callee }.toSet()
    val roots = nodes.filter { it !in targets }
    val visited = mutableSetOf<SymbolId>()
    val active = mutableSetOf<SymbolId>()
    data class Visit(val node: SymbolId, val indent: String, val edge: CallEdge? = null, val exit: Boolean = false)
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
            val category = when (node) {
                in impact.changed -> "CHANGED"
                in impact.affected -> "CALLER"
                else -> "POTENTIAL INDIRECT"
            }
            val location = locations[node]
            val name = location?.sourceSignature ?: node.toString()
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
            outgoing[node].orEmpty().sortedWith(compareBy<CallEdge> { it.callee.toString() }.thenBy { it.kind.name })
                .asReversed().forEach { pending.addLast(Visit(it.callee, childIndent, it)) }
        }
    }
}
