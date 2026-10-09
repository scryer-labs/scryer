package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.GitComparer
import com.edwardnoaland.scryer.analyze.LineRange
import java.nio.file.Files
import java.nio.file.Path
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue

class GitComparisonTest {
    @TempDir lateinit var root: Path

    private fun git(vararg args: String): String {
        val process = ProcessBuilder(listOf("git", "-C", root.toString()) + args).redirectErrorStream(true).start()
        val text = process.inputStream.bufferedReader().readText()
        check(process.waitFor() == 0) { text }
        return text.trim()
    }

    private fun write(name: String, text: String) { Files.writeString(root.resolve(name), text) }
    private fun commit(): String {
        git("add", ".")
        git("commit", "-m", "fixture")
        return git("rev-parse", "HEAD")
    }

    @Test fun `compares snapshots with rename addition deletion and precise line ranges`() {
        git("init")
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.com")
        write("Changed.java", "one\ntwo\nthree\n")
        write("Old name.java", "unchanged\n")
        write("Deleted.java", "deleted\n")
        val before = commit()
        git("tag", "baseline")
        write("Changed.java", "one\nupdated\nthree\n")
        Files.move(root.resolve("Old name.java"), root.resolve("New name.java"))
        Files.delete(root.resolve("Deleted.java"))
        write("Added.java", "new\n")
        val after = commit()
        write("Changed.java", "uncommitted changes must be ignored\n")
        val result = GitComparer().compare(root, "HEAD~1", "HEAD")
        assertEquals(before, result.before)
        assertEquals(after, result.after)
        assertEquals(4, result.files.size)
        val changed = result.files.single { it.afterPath == "Changed.java" }.lines.single()
        assertEquals(LineRange(2, 1), changed.before)
        assertEquals(LineRange(2, 1), changed.after)
        val added = result.files.single { it.status == "A" }
        assertEquals(null, added.beforePath)
        assertEquals(LineRange(0, 0), added.lines.single().before)
        val deleted = result.files.single { it.status == "D" }
        assertEquals(null, deleted.afterPath)
        assertEquals(LineRange(0, 0), deleted.lines.single().after)
        val renamed = result.files.single { it.status.startsWith("R") }
        assertEquals("Old name.java", renamed.beforePath)
        assertEquals("New name.java", renamed.afterPath)
        assertTrue(renamed.lines.isEmpty())
        assertEquals(result.files, GitComparer().compare(root, "baseline", after).files)
        assertTrue(GitComparer().compare(root, after, after).files.isEmpty())
        assertFailsWith<IllegalStateException> { GitComparer().compare(root, "missing-ref", after) }
    }

    @Test fun `rejects a directory outside a Git repository`() {
        assertFailsWith<IllegalStateException> { GitComparer().compare(root, "HEAD", "HEAD") }
    }
    @Test fun `non UTF8 diff bodies preserve ASCII hunk ranges`() {
        git("init")
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.com")
        Files.write(root.resolve("legacy.txt"), byteArrayOf(0xe9.toByte(), 10))
        val before = commit()
        Files.write(root.resolve("legacy.txt"), byteArrayOf(0xe9.toByte(), 33, 10))
        val after = commit()
        val file = GitComparer().compare(root, before, after).files.single()
        assertEquals(LineRange(1, 1), file.lines.single().after)
    }

}
