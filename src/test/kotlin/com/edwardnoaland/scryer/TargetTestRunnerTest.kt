package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.GitComparer
import com.edwardnoaland.scryer.analyze.IsolatedSnapshots
import com.edwardnoaland.scryer.analyze.execute.*
import java.nio.file.Files
import java.nio.file.Path
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.*

class TargetTestRunnerTest {
    @TempDir lateinit var temporary: Path
    private val javaHome get() = Path.of(System.getProperty("java.home"))
    private fun environment() = System.getenv().toMutableMap().apply {
        remove("SCRYER_JAVA_HOME")
        put("SCRYER_CACHE_HOME", temporary.resolve("cache").toString())
    }
    private fun git(root: Path, vararg args: String): String {
        val process = ProcessBuilder(listOf("git", "-C", root.toString()) + args).redirectErrorStream(true).start()
        val text = process.inputStream.bufferedReader().readText()
        check(process.waitFor() == 0) { text }
        return text.trim()
    }
    private fun repository(): Path {
        val root = Files.createDirectory(temporary.resolve("repo"))
        git(root, "init")
        git(root, "config", "user.name", "Fixture")
        git(root, "config", "user.email", "fixture@example.com")
        return root
    }
    private fun commit(root: Path, command: String, wrapper: String = "gradlew"): String {
        Files.writeString(root.resolve(wrapper), "#!/bin/sh\n$command\n")
        Files.writeString(root.resolve("build.gradle"), "// synthetic execution fixture\n")
        git(root, "add", ".")
        git(root, "commit", "-m", "fixture")
        return git(root, "rev-parse", "HEAD")
    }

    @Test fun `executes after wrapper preserves original dirty state and retains logs after cleanup`() {
        val root = repository()
        val before = commit(root, "printf 'before\\n'")
        val after = commit(root, "printf 'after\\n'; printf '%s\\n' \"\$JAVA_HOME\"; touch executed")
        Files.writeString(root.resolve("build.gradle"), "dirty original source")
        val status = git(root, "status", "--porcelain")
        val worktrees = git(root, "worktree", "list", "--porcelain")
        val comparison = GitComparer().compare(root, before, after)
        lateinit var checkout: Path
        val result = IsolatedSnapshots.use(comparison) { _, snapshot ->
            checkout = snapshot
            TargetTestRunner(environment()).run(snapshot, comparison)
        }
        assertEquals(TestRunStatus.SUCCEEDED, result.status)
        assertEquals(after, result.afterSha)
        assertEquals(0, result.exitCode)
        assertEquals(listOf("clean", "test"), result.command.takeLast(2))
        assertContains(Files.readString(result.log), "after")
        assertFalse(Files.exists(checkout))
        assertFalse(Files.exists(root.resolve("executed")))
        assertEquals(status, git(root, "status", "--porcelain"))
        assertEquals(worktrees, git(root, "worktree", "list", "--porcelain"))
    }

    @Test fun `records nonzero exit without treating it as missing execution`() {
        val root = repository()
        val ref = commit(root, "printf 'test failure\\n'; exit 7")
        val comparison = GitComparer().compare(root, ref, ref)
        val result = IsolatedSnapshots.use(comparison) { _, snapshot -> TargetTestRunner(environment()).run(snapshot, comparison) }
        assertEquals(TestRunStatus.FAILED, result.status)
        assertEquals(7, result.exitCode)
        assertContains(Files.readString(result.log), "test failure")
    }

    @Test fun `timeout stops process and preserves partial log`() {
        val root = repository()
        val ref = commit(root, "printf 'starting\\n'; /bin/sleep 30")
        val comparison = GitComparer().compare(root, ref, ref)
        val env = environment().apply { put("SCRYER_TEST_TIMEOUT_SECONDS", "1") }
        val result = IsolatedSnapshots.use(comparison) { _, snapshot -> TargetTestRunner(env).run(snapshot, comparison) }
        assertEquals(TestRunStatus.TIMED_OUT, result.status)
        assertNull(result.exitCode)
        assertTrue(result.durationMillis < 10_000)
        assertContains(Files.readString(result.log), "starting")
    }

    @Test fun `invalidates results when build mutates tracked source`() {
        val root = repository()
        val ref = commit(root, "printf 'mutated' > build.gradle")
        val comparison = GitComparer().compare(root, ref, ref)
        val result = IsolatedSnapshots.use(comparison) { _, snapshot -> TargetTestRunner(environment()).run(snapshot, comparison) }
        assertEquals(TestRunStatus.SNAPSHOT_CHANGED, result.status)
        assertEquals(0, result.exitCode)
    }

    @Test fun `Maven uses verify while ambiguous or absent wrappers are unavailable`() {
        val root = Files.createDirectory(temporary.resolve("project"))
        val env = environment()
        val planner = TestRunPlanner(env)
        val cache = temporary.resolve("cache")
        val run = temporary.resolve("run")
        assertFailsWith<IllegalStateException> { planner.plan(root, root, cache, run) }
        Files.writeString(root.resolve("mvnw"), "#!/bin/sh\n")
        assertEquals(listOf("clean", "verify"), planner.plan(root, root, cache, run).command.takeLast(2))
        Files.writeString(root.resolve("gradlew"), "#!/bin/sh\n")
        assertFailsWith<IllegalStateException> { planner.plan(root, root, cache, run) }
    }

    @Test fun `old Gradle uses fixture local Java8 or explicit override rather than Scryer runtime`() {
        val root = Files.createDirectory(temporary.resolve("project"))
        Files.writeString(root.resolve("gradlew"), "#!/bin/sh\n")
        val wrapper = root.resolve("gradle/wrapper")
        Files.createDirectories(wrapper)
        Files.writeString(wrapper.resolve("gradle-wrapper.properties"), "distributionUrl=https\\://example.invalid/gradle-4.10.3-bin.zip\n")
        val planner = TestRunPlanner(environment())
        assertFailsWith<IllegalStateException> { planner.plan(root, root, temporary, temporary) }
        val local = root.resolve(".tooling/mise/data/installs/java/fixture-java8")
        Files.createDirectories(local.resolve("bin"))
        Files.writeString(local.resolve("bin/java"), "synthetic fixture marker")
        Files.writeString(local.resolve("release"), "JAVA_VERSION=\"1.8.0_472\"")
        assertEquals(local.toRealPath(), planner.plan(root, root, temporary, temporary).javaHome)
        val override = environment().apply { put("SCRYER_JAVA_HOME", javaHome.toString()) }
        assertEquals(javaHome, TestRunPlanner(override).plan(root, root, temporary, temporary).javaHome)
    }
    @Test fun `executes Maven verify and detects missing wrapper before launching`() {
        val root = repository()
        val ref = commit(root, "printf '%s\\n' \"\$@\"", wrapper = "mvnw")
        val comparison = GitComparer().compare(root, ref, ref)
        val result = IsolatedSnapshots.use(comparison) { _, snapshot -> TargetTestRunner(environment()).run(snapshot, comparison) }
        assertEquals(TestRunStatus.SUCCEEDED, result.status)
        assertContains(Files.readString(result.log), "verify")
        val missing = Files.createDirectory(temporary.resolve("missing"))
        val unavailable = TargetTestRunner(environment()).run(missing, comparison)
        assertEquals(TestRunStatus.UNAVAILABLE, unavailable.status)
        assertNull(unavailable.exitCode)
    }

    @Test fun `invalid timeout configuration is reported instead of silently defaulted`() {
        val root = repository()
        val ref = commit(root, "printf 'should not run'")
        val comparison = GitComparer().compare(root, ref, ref)
        val env = environment().apply { put("SCRYER_TEST_TIMEOUT_SECONDS", "invalid") }
        val result = TargetTestRunner(env).run(root, comparison)
        assertEquals(TestRunStatus.UNAVAILABLE, result.status)
        assertNull(result.log)
        assertNull(result.exitCode)
    }

    @Test fun `index-only mutation also invalidates snapshot evidence`() {
        val root = repository()
        val ref = commit(root, "printf 'staged' > build.gradle; git add build.gradle; printf '// synthetic execution fixture\\n' > build.gradle")
        val comparison = GitComparer().compare(root, ref, ref)
        val result = IsolatedSnapshots.use(comparison) { _, snapshot -> TargetTestRunner(environment()).run(snapshot, comparison) }
        assertEquals(TestRunStatus.SNAPSHOT_CHANGED, result.status)
    }

}
