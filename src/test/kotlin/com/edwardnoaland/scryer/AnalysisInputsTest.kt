package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.*
import com.edwardnoaland.scryer.analyze.model.*
import java.nio.file.Files
import java.nio.file.Path
import javax.tools.ToolProvider
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.*

class AnalysisInputsTest {
    @TempDir lateinit var root: Path

    private fun source(path: String, text: String): Path {
        val file = root.resolve(path)
        Files.createDirectories(file.parent)
        Files.writeString(file, text)
        return file
    }

    @Test fun `module classpath resolves external parameter types and configured custom source roles`() {
        val dependency = source("dependency/External.java", "package dep; public class External {}")
        val classes = Files.createDirectories(root.resolve("external-classes"))
        assertEquals(0, ToolProvider.getSystemJavaCompiler().run(null, null, null, "-d", classes.toString(), dependency.toString()))
        source("application/custom/Entry.java", "package app; class Entry { int changed(dep.External value) { return 1; } int caller(dep.External value) { return changed(value); } }")
        val module = AnalysisModule(":application", root.resolve("application"),
            listOf(AnalysisSourceRoot(root.resolve("application/custom"), SourceRole.PRODUCTION)), listOf(classes))
        val graph = CallGraphCollector().collect(root, AnalysisInputs(listOf(module)))
        assertEquals("(Ldep/External;)I", graph.symbols.single { it.id.name == "changed" }.id.descriptor)
        assertTrue(graph.symbols.all { it.role == SourceRole.PRODUCTION && it.module == ":application" })
        assertTrue(graph.edges.any { it.caller.name == "caller" && it.callee.name == "changed" })
        assertTrue(graph.symbols.none { it.id.owner == "dep.External" })
    }

    @Test fun `reactor sources link callers across modules without requiring compiled reactor artifacts`() {
        source("library/java/Library.java", "package lib; public class Library { public int changed() { return 1; } }")
        source("application/java/Entry.java", "package app; class Entry { int run(lib.Library lib) { return lib.changed(); } }")
        val library = AnalysisModule(":library", root.resolve("library"), listOf(AnalysisSourceRoot(root.resolve("library/java"), SourceRole.PRODUCTION)), emptyList())
        val application = AnalysisModule(":application", root.resolve("application"), listOf(AnalysisSourceRoot(root.resolve("application/java"), SourceRole.PRODUCTION)), emptyList(), listOf(":library"))
        val graph = CallGraphCollector().collect(root, AnalysisInputs(listOf(library, application)))
        assertTrue(graph.edges.any { it.caller.owner == "app.Entry" && it.callee.owner == "lib.Library" })
        assertEquals(1, graph.symbols.count { it.id.name == "changed" })
    }

    @Test fun `same binary identities in independent modules remain explicitly ambiguous`() {
        val modules = listOf("one", "two").map { name ->
            source("$name/java/Example.java", "class Example { int changed() { return 1; } }")
            AnalysisModule(name, root.resolve(name), listOf(AnalysisSourceRoot(root.resolve("$name/java"), SourceRole.PRODUCTION)), emptyList())
        }
        val graph = CallGraphCollector().collect(root, AnalysisInputs(modules))
        assertTrue(graph.symbols.isEmpty())
        assertTrue(graph.notes.any { it.contains("across modules excluded") })
    }

    @Test fun `gradle input adapter preserves source roles dependencies and resolution failures`() {
        val output = root.resolve("model.json")
        Files.writeString(output, """{"modules":[{"id":":","directory":"$root","sources":[{"name":"main","roots":["$root/custom"]},{"name":"test","roots":["$root/specs"]},{"name":"other","roots":["$root/other"]}],"classpath":[],"dependencies":[":lib"],"notes":["test compileClasspath unavailable"]}]}""")
        val inputs = AnalysisInputCollector().readGradle(output, root)
        assertEquals(listOf(SourceRole.PRODUCTION, SourceRole.TEST, SourceRole.UNKNOWN), inputs.modules.single().sources.map { it.role })
        assertEquals(listOf(":lib"), inputs.modules.single().dependencies)
        assertEquals(listOf("test compileClasspath unavailable"), inputs.modules.single().notes)
    }
}
