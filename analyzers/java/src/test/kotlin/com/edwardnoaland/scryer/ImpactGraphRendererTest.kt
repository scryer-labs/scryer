package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.*
import com.edwardnoaland.scryer.cli.output.renderImpactGraph
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import org.junit.jupiter.api.Test
import kotlin.test.*

class ImpactGraphRendererTest {
    @Test fun `shows complete branching shared nodes cycles roles and boundaries`() {
        val nodes = listOf("A", "B", "C", "D", "E").associateWith { SymbolId("Example", it, "()V") }
        fun edge(a: String, b: String, kind: CallKind = CallKind.DIRECT) = CallEdge(nodes.getValue(a), nodes.getValue(b), kind)
        val edges = listOf(edge("A", "B"), edge("B", "C"), edge("A", "D"), edge("B", "E"), edge("D", "E"), edge("E", "D", CallKind.REFLECTION))
        val graph = CallGraph(nodes.values.map { SymbolLocation(it, "src/main/java/Example.java", 1, "Example#${it.name}()") }, edges,
            listOf(CallBoundary(nodes.getValue("C"), "Example.java", 1, "external()", "Unresolved call")), emptyList())
        val affected = setOf(nodes.getValue("A"), nodes.getValue("B"), nodes.getValue("C"))
        val impact = SnapshotImpact(graph, setOf(nodes.getValue("C")), affected, emptyList(),
            setOf(nodes.getValue("D"), nodes.getValue("E")), emptyMap())
        val buffer = ByteArrayOutputStream()
        renderImpactGraph(impact, PrintStream(buffer))
        val output = buffer.toString(Charsets.UTF_8)
        assertEquals(edges.size, Regex("\\|-- (DIRECT|REFLECTION) ->").findAll(output).count())
        assertContains(output, "Example#C() [CHANGED] [PRODUCTION]")
        assertContains(output, "Example#A() [CALLER]")
        assertContains(output, "[POTENTIAL INDIRECT]")
        assertContains(output, "[CYCLE -> #")
        assertContains(output, "[see #")
        assertContains(output, "? BOUNDARY 1 site(s)")
    }

    @Test fun `renders rootless recursive components and isolated changes`() {
        val a = SymbolId("Example", "a", "()V")
        val b = SymbolId("Example", "b", "()V")
        val isolated = SymbolId("Example", "isolated", "()V")
        val graph = CallGraph(emptyList(), listOf(CallEdge(a,b,CallKind.DIRECT), CallEdge(b,a,CallKind.DIRECT)), emptyList(), emptyList())
        val buffer = ByteArrayOutputStream()
        renderImpactGraph(SnapshotImpact(graph, setOf(a, isolated), setOf(a,b,isolated), emptyList(), emptySet(), emptyMap()), PrintStream(buffer))
        val output = buffer.toString(Charsets.UTF_8)
        assertContains(output, "[CYCLE -> #")
        assertContains(output, "isolated()V [CHANGED] [UNKNOWN]")
        assertEquals(2, Regex("\\|-- DIRECT ->").findAll(output).count())
    }
}
