package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.*
import java.nio.file.Files
import java.nio.file.Path
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.*

class ImpactAnalysisTest {
    @TempDir lateinit var root: Path

    private fun git(vararg args: String): String {
        val process = ProcessBuilder(listOf("git", "-C", root.toString()) + args).redirectErrorStream(true).start()
        val text = process.inputStream.bufferedReader().readText()
        check(process.waitFor() == 0) { text }
        return text.trim()
    }
    private fun initialize() {
        git("init")
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.com")
    }
    private fun commit(source: String): String {
        Files.writeString(root.resolve("Example.java"), source)
        git("add", ".")
        git("commit", "-m", "fixture")
        return git("rev-parse", "HEAD")
    }

    @Test fun `resolves overloads constructors generics nested owners and transitive recursive callers`() {
        initialize()
        val source = """
            package example;
            import java.util.List;
            class Example {
              Example() { }
              int leaf(int x) { return x; }
              int leaf(String x) { return 0; }
              int middle() { return leaf(1); }
              int entry() { return middle(); }
              int other() { return leaf("one"); }
              int recursive(int x) { return x == 0 ? leaf(0) : recursive(x - 1); }
              Example factory() { return new Example(); }
              <T> T generic(T value, List<String> values, int[] ints) { return value; }
              static class Nested { int size() { return 1; } }
            }
        """.trimIndent()
        val before = commit(source)
        val after = commit(source.replace("return x;", "return x + 1;"))
        val result = ImpactAnalyzer().analyze(GitComparer().compare(root, before, after))
        val impact = result.after
        assertEquals(SymbolId("example.Example", "leaf", "(I)I"), impact.changed.single())
        assertEquals(setOf("leaf", "middle", "entry", "recursive"), impact.affected.map { it.name }.toSet())
        assertTrue(impact.graph.edges.any { it.caller.name == "factory" && it.callee.name == "<init>" })
        assertTrue(impact.graph.symbols.any { it.id.owner == "example.Example\$Nested" })
        assertEquals("(Ljava/lang/Object;Ljava/util/List;[I)Ljava/lang/Object;",
            impact.graph.symbols.single { it.id.name == "generic" }.id.descriptor)
        assertEquals(result.before.changed, result.after.changed)
    }

    @Test fun `includes possible interface overrides but keeps super and static calls direct`() {
        initialize()
        val source = """
            interface Service { int total(); }
            class Base implements Service { public int total() { return 1; } static int helper() { return 1; } }
            class Child extends Base { public int total() { return 2; } int parent() { return super.total(); } }
            class Client { int run(Service service) { return service.total(); } int staticCall() { return Base.helper(); } }
        """.trimIndent()
        val before = commit(source)
        val after = commit(source.replace("return 2;", "return 3;"))
        val result = ImpactAnalyzer().analyze(GitComparer().compare(root, before, after)).after
        assertTrue(result.affected.any { it.owner == "Client" && it.name == "run" })
        assertFalse(result.affected.any { it.name == "parent" })
        assertTrue(result.graph.edges.any { it.callee.owner == "Child" && it.kind == CallKind.POSSIBLE_DISPATCH })
        assertTrue(result.graph.edges.filter { it.caller.name == "staticCall" }.all { it.kind == CallKind.DIRECT })
    }

    @Test fun `preserves before callers when methods are removed`() {
        initialize()
        val before = commit("class Example { int leaf() { return 1; } int caller() { return leaf(); } }")
        val after = commit("class Example { int caller() { return 0; } }")
        val result = ImpactAnalyzer().analyze(GitComparer().compare(root, before, after))
        assertTrue(result.before.changed.any { it.name == "leaf" })
        assertTrue(result.before.affected.any { it.name == "caller" })
        assertFalse(result.after.graph.symbols.any { it.id.name == "leaf" })
    }

    @Test fun `missing dependencies remain boundaries while local calls can resolve`() {
        initialize()
        val before = commit("class Example { int leaf() { return 1; } int caller() { Missing.run(); return leaf(); } }")
        val after = commit("class Example { int leaf() { return 2; } int caller() { Missing.run(); return leaf(); } }")
        val result = ImpactAnalyzer().analyze(GitComparer().compare(root, before, after)).after
        assertTrue(result.affected.any { it.name == "caller" })
        assertTrue(result.graph.boundaries.any { it.reason == "Unresolved call" && it.caller?.name == "caller" })
        assertTrue(result.graph.notes.any { it.contains("attribution reported") })
    }

    @Test fun `method references are potential calls and initializer calls remain boundaries`() {
        initialize()
        val before = commit("class Example { int leaf() { return 1; } java.util.function.IntSupplier reference() { return this::leaf; } int field = leaf(); }")
        val after = commit("class Example { int leaf() { return 2; } java.util.function.IntSupplier reference() { return this::leaf; } int field = leaf(); }")
        val result = ImpactAnalyzer().analyze(GitComparer().compare(root, before, after)).after
        assertTrue(result.graph.edges.any { it.kind == CallKind.METHOD_REFERENCE })
        assertTrue(result.affected.any { it.name == "reference" })
        assertTrue(result.graph.boundaries.any { it.caller == null && it.reason.contains("Initializer") })
    }
    @Test fun `duplicate module identities are excluded instead of merged`() {
        Files.createDirectories(root.resolve("module-a"))
        Files.createDirectories(root.resolve("module-b"))
        for (module in listOf("module-a", "module-b")) {
            Files.writeString(root.resolve("$module/Example.java"), "class Example { int leaf() { return 1; } }")
        }
        val graph = CallGraphCollector().collect(root)
        assertTrue(graph.symbols.isEmpty())
        assertTrue(graph.notes.any { it.contains("duplicate source symbol") })
    }

    @Test fun `unresolvable changed signature is explicitly unmatched`() {
        initialize()
        val before = commit("class Example { int leaf(Missing value) { return 1; } }")
        val after = commit("class Example { int leaf(Missing value) { return 2; } }")
        val result = ImpactAnalyzer().analyze(GitComparer().compare(root, before, after)).after
        assertTrue(result.changed.isEmpty())
        assertEquals(1, result.unmatched.size)
        assertTrue(result.graph.boundaries.any { it.reason == "Unresolved declaration signature" })
    }

    @Test fun `expands sibling branches transitively without promoting them to callers`() {
        initialize()
        val source = "class Example { int c() { return 1; } int b() { e(); return c(); } int a() { d(); return b(); } void d() { f(); } void e() {} void f() { d(); } void unrelated() {} }"
        val before = commit(source)
        val after = commit(source.replace("return 1;", "return 2;"))
        val impact = ImpactAnalyzer().analyze(GitComparer().compare(root, before, after)).after
        assertEquals(setOf("a", "b", "c"), impact.affected.map { it.name }.toSet())
        assertEquals(setOf("d", "e", "f"), impact.indirect.map { it.name }.toSet())
        assertEquals("a", impact.indirectReasons.entries.single { it.key.name == "d" }.value.caller.name)
        assertFalse((impact.affected + impact.indirect).any { it.name == "unrelated" })
    }

    @Test fun `classifies conventional roots without guessing from test class names`() {
        assertEquals(SourceRole.PRODUCTION, sourceRole("module/src/main/java/SomeTest.java"))
        assertEquals(SourceRole.TEST, sourceRole("module/src/test/java/Example.java"))
        assertEquals(SourceRole.TEST, sourceRole("src/integrationTest/java/Example.java"))
        assertEquals(SourceRole.UNKNOWN, sourceRole("custom/ExampleTest.java"))
        assertEquals(SourceRole.TEST, sourceRole("module\\src\\test\\java\\Example.java"))
    }

    @Test fun `literal reflection resolves overloads and inherited public methods`() {
        initialize()
        val source = """
            class Base { public int target(int value) { return 1; } public int target(String value) { return 0; } }
            class Child extends Base { }
            class Example {
              Object invoke() throws Exception {
                Class<?> type = Class.forName("Child");
                java.lang.reflect.Method method = type.getMethod("target", int.class);
                return method.invoke(new Child(), 1);
              }
            }
        """.trimIndent()
        val before = commit(source)
        val after = commit(source.replace("return 1;", "return 2;"))
        val impact = ImpactAnalyzer().analyze(GitComparer().compare(root, before, after)).after
        val edge = impact.graph.edges.single { it.kind == CallKind.REFLECTION }
        assertEquals(SymbolId("Base", "target", "(I)I"), edge.callee)
        assertEquals("invoke", edge.caller.name)
        assertTrue(impact.affected.any { it.name == "invoke" })
    }

    @Test fun `dynamic reflection and reassigned method variables stay boundaries`() {
        initialize()
        val source = """
            class Example {
              public int target() { return 1; }
              Object dynamic(String name) throws Exception {
                Class<?> type = Class.forName(name);
                java.lang.reflect.Method method = type.getMethod("target");
                return method.invoke(this);
              }
              Object reassigned(java.lang.reflect.Method other) throws Exception {
                Class<?> type = Class.forName("Example");
                java.lang.reflect.Method method = type.getMethod("target");
                method = other;
                return method.invoke(this);
              }
              Object branched(boolean flag) throws Exception {
                Class<?> type = Class.forName("Example");
                java.lang.reflect.Method method = type.getMethod("target");
                if (flag) method = null;
                return method.invoke(this);
              }
            }
        """.trimIndent()
        val before = commit(source)
        val after = commit(source.replace("return 1;", "return 2;"))
        val graph = ImpactAnalyzer().analyze(GitComparer().compare(root, before, after)).after.graph
        assertTrue(graph.edges.none { it.kind == CallKind.REFLECTION })
        assertTrue(graph.boundaries.any { it.expression.contains("method.invoke") })
    }

    @Test fun `declared reflection supports private methods but public lookup does not`() {
        initialize()
        val source = """
            class Example {
              private int target() { return 1; }
              Object declared() throws Exception {
                try {
                  Class<?> type = Class.forName("Example");
                  java.lang.reflect.Method method = type.getDeclaredMethod("target");
                  method.setAccessible(true);
                  return method.invoke(this);
                } catch (Exception e) { throw e; }
              }
              Object publicLookup() throws Exception {
                Class<?> type = Class.forName("Example");
                java.lang.reflect.Method method = type.getMethod("target");
                return method.invoke(this);
              }
            }
        """.trimIndent()
        val before = commit(source)
        val after = commit(source.replace("return 1;", "return 2;"))
        val graph = ImpactAnalyzer().analyze(GitComparer().compare(root, before, after)).after.graph
        assertEquals("declared", graph.edges.single { it.kind == CallKind.REFLECTION }.caller.name)
        assertTrue(graph.boundaries.any { it.caller?.name == "publicLookup" && it.expression.contains("method.invoke") })
    }

}
