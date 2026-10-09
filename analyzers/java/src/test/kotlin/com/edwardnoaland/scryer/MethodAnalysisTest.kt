package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.GitComparer
import com.edwardnoaland.scryer.analyze.IsolatedSnapshots
import com.edwardnoaland.scryer.analyze.MethodAnalyzer
import com.edwardnoaland.scryer.analyze.MethodChangeKind
import java.nio.file.Files
import java.nio.file.Path
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class MethodAnalysisTest {
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
        Files.writeString(root.resolve("Order.java"), source)
        git("add", ".")
        git("commit", "-m", "fixture")
        return git("rev-parse", "HEAD")
    }

    @Test fun `finds changed overload constructor nested method and added deleted declarations`() {
        initialize()
        val before = commit("""
            package example;
            class Order {
                Order() { int value = 1; }
                int total(int value) { return value; }
                int total(String value) { return 0; }
                void removed() { }
                static class Nested { int size() { return 1; } }
            }
        """.trimIndent())
        val after = commit("""
            package example;
            class Order {
                Order() { int value = 2; }
                int total(int value) { return value + 1; }
                int total(String value) { return 0; }
                void added() { }
                static class Nested { int size() { return 2; } }
            }
        """.trimIndent())
        Files.writeString(root.resolve("Order.java"), "dirty working tree")
        val status = git("status", "--porcelain")
        val worktrees = git("worktree", "list", "--porcelain")
        val result = MethodAnalyzer().analyze(GitComparer().compare(root, before, after))
        assertEquals(5, result.methods.size)
        val bySignature = result.methods.associateBy { (it.after ?: it.before)!!.signature }
        assertEquals(MethodChangeKind.MODIFIED, bySignature["example.Order#<init>()"]?.kind)
        assertEquals(MethodChangeKind.MODIFIED, bySignature["example.Order#total(int)"]?.kind)
        assertEquals(MethodChangeKind.MODIFIED, bySignature["example.Order.Nested#size()"]?.kind)
        assertEquals(MethodChangeKind.ADDED, bySignature["example.Order#added()"]?.kind)
        assertEquals(MethodChangeKind.DELETED, bySignature["example.Order#removed()"]?.kind)
        assertFalse(bySignature.containsKey("example.Order#total(String)"))
        assertEquals(4, bySignature["example.Order#total(int)"]?.after?.lines?.start)
        assertEquals(status, git("status", "--porcelain"))
        assertEquals(worktrees, git("worktree", "list", "--porcelain"))
    }

    @Test fun `ignores formatting comments and pure rename but exposes context limitation`() {
        initialize()
        val before = commit("class Order { int value() { return 1; } }")
        val after = commit("// comment\nclass Order {\n int value() {\n return 1;\n }\n}\n")
        val analysis = MethodAnalyzer().analyze(GitComparer().compare(root, before, after))
        assertTrue(analysis.methods.isEmpty())
        assertTrue(analysis.notes.isNotEmpty())
        Files.move(root.resolve("Order.java"), root.resolve("Renamed.java"))
        git("add", ".")
        git("commit", "-m", "rename")
        assertTrue(MethodAnalyzer().analyze(GitComparer().compare(root, after, "HEAD")).methods.isEmpty())
    }

    @Test fun `handles complete file addition deletion and signature change`() {
        initialize()
        val before = commit("class Order { int value(int input) { return input; } }")
        val after = commit("class Order { int value(long input) { return 1; } }")
        val comparison = GitComparer().compare(root, before, after)
        assertEquals(setOf(MethodChangeKind.ADDED, MethodChangeKind.DELETED),
            MethodAnalyzer().analyze(comparison).methods.map { it.kind }.toSet())
        Files.delete(root.resolve("Order.java"))
        git("add", ".")
        git("commit", "-m", "delete")
        assertEquals(MethodChangeKind.DELETED,
            MethodAnalyzer().analyze(GitComparer().compare(root, after, "HEAD")).methods.single().kind)
        assertEquals(MethodChangeKind.ADDED,
            MethodAnalyzer().analyze(GitComparer().compare(root, "HEAD", after)).methods.single().kind)
    }

    @Test fun `field only changes remain explicit context gaps and declaration changes are detected`() {
        initialize()
        val before = commit("class Order { int field = 1; int value() { return field; } }")
        val fieldChange = commit("class Order { int field = 2; int value() { return field; } }")
        val context = MethodAnalyzer().analyze(GitComparer().compare(root, before, fieldChange))
        assertTrue(context.methods.isEmpty())
        assertTrue(context.notes.single().contains("context"))
        val declarationChange = commit("class Order { int field = 2; @Deprecated long value() { return field; } }")
        assertEquals(MethodChangeKind.MODIFIED,
            MethodAnalyzer().analyze(GitComparer().compare(root, fieldChange, declarationChange)).methods.single().kind)
    }

    @Test fun `cleans isolation workspace on success and failure`() {
        initialize()
        val ref = commit("class Order {}")
        val comparison = GitComparer().compare(root, ref, ref)
        lateinit var temporary: Path
        IsolatedSnapshots.use(comparison) { before, after ->
            temporary = before.parent
            assertTrue(Files.exists(before.resolve("Order.java")))
            assertTrue(Files.exists(after.resolve("Order.java")))
        }
        assertFalse(Files.exists(temporary))
        assertFailsWith<IllegalStateException> {
            IsolatedSnapshots.use(comparison) { before, _ ->
                temporary = before.parent
                error("intentional failure")
            }
        }
        assertFalse(Files.exists(temporary))
    }

    @Test fun `parse errors and anonymous method identity are explicit failures`() {
        initialize()
        val before = commit("class Order {}")
        val broken = commit("class Order { void broken( }")
        val parseError = assertFailsWith<IllegalStateException> {
            MethodAnalyzer().analyze(GitComparer().compare(root, before, broken))
        }
        assertTrue(parseError.message!!.contains("Cannot parse"))
        val anonymous = commit("class Order { Runnable r = new Runnable() { public void run() {} }; }")
        val identityError = assertFailsWith<IllegalStateException> {
            MethodAnalyzer().analyze(GitComparer().compare(root, before, anonymous))
        }
        assertTrue(identityError.message!!.contains("Anonymous class"))
    }
}
