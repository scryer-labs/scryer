package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.*
import java.nio.file.Files
import java.nio.file.Path
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.*

class ReflectionAnalysisTest {
    @TempDir lateinit var root: Path
    private fun graph(source: String): CallGraph {
        Files.writeString(root.resolve("Example.java"), source.trimIndent())
        return CallGraphCollector().collect(root)
    }
    private fun targets(graph: CallGraph, caller: String) = graph.edges.filter { it.kind == CallKind.REFLECTION && it.caller.name == caller }.map { it.callee }.toSet()

    @Test fun `class literals local arrays constants and static helper summaries resolve exact overloads`() {
        val graph = graph("""
            class Names { static final String OWNER = "Example"; static String owner() { return OWNER; } }
            class Example {
                static final String NAME = "target";
                public String target(String text, int[] numbers) { return text; }
                public String target(int number) { return "wrong overload"; }
                Object run() throws Exception {
                    Class<?> owner = Class.forName(Names.owner());
                    Class<?>[] parameters = new Class<?>[] { String.class, int[].class };
                    java.lang.reflect.Method target = owner.getMethod(NAME, parameters);
                    return target.invoke(this, "ok", new int[0]);
                }
                Object literal() throws Exception { return Example.class.getMethod("target", int.class).invoke(this, 1); }
            }
        """)
        assertEquals(setOf(SymbolId("Example", "target", "(Ljava/lang/String;[I)Ljava/lang/String;")), targets(graph, "run"))
        assertEquals(setOf(SymbolId("Example", "target", "(I)Ljava/lang/String;")), targets(graph, "literal"))
    }

    @Test fun `public and declared constructors use exact parameters and constructors are never inherited`() {
        val graph = graph("""
            class Base { public Base(int value) {} }
            class Example extends Base {
                public Example(String text) { super(1); }
                private Example(int value) { super(value); }
                Object construct() throws Exception {
                    Class<?>[] parameters = {String.class};
                    return Example.class.getConstructor(parameters).newInstance("ok");
                }
                Object declared() throws Exception {
                    java.lang.reflect.Constructor<?> constructor = Example.class.getDeclaredConstructor(int.class);
                    constructor.setAccessible(true);
                    return constructor.newInstance(2);
                }
                Object unavailable() throws Exception { return Example.class.getConstructor(int.class).newInstance(2); }
            }
        """)
        assertEquals(setOf(SymbolId("Example", "<init>", "(Ljava/lang/String;)V")), targets(graph, "construct"))
        assertEquals(setOf(SymbolId("Example", "<init>", "(I)V")), targets(graph, "declared"))
        assertTrue(targets(graph, "unavailable").isEmpty())
        assertTrue(graph.boundaries.any { it.caller?.name == "unavailable" && it.expression.contains("newInstance") })
    }

    @Test fun `mutated escaped arrays dynamic names and virtual helpers stay boundaries`() {
        val graph = graph("""
            class Mutator { Mutator(Class<?>[] values) { values[0] = String.class; } }
            class Example {
                public int target(int value) { return value; }
                String name() { return "target"; }
                static void mutate(Class<?>[] values) { values[0] = String.class; }
                Object changed() throws Exception {
                    Class<?>[] types = {int.class}; types[0] = String.class;
                    return Example.class.getMethod("target", types).invoke(this, 1);
                }
                Object escaped() throws Exception {
                    Class<?>[] types = {int.class}; mutate(types);
                    return Example.class.getMethod("target", types).invoke(this, 1);
                }
                Object constructorEscape() throws Exception {
                    Class<?>[] types = {int.class}; new Mutator(types);
                    return Example.class.getMethod("target", types).invoke(this, 1);
                }
                Object dynamic(String name) throws Exception { return Example.class.getMethod(name, int.class).invoke(this, 1); }
                Object virtual() throws Exception { return Example.class.getMethod(name(), int.class).invoke(this, 1); }
            }
        """)
        assertTrue(graph.edges.none { it.kind == CallKind.REFLECTION })
        for (caller in listOf("changed", "escaped", "constructorEscape", "dynamic", "virtual")) {
            assertTrue(graph.boundaries.any { it.caller?.name == caller && it.expression.contains("invoke") })
        }
    }
}
