package com.edwardnoaland.scryer.analyze.execute

import com.edwardnoaland.scryer.analyze.GitComparison
import com.edwardnoaland.scryer.analyze.GitProcess
import java.nio.file.Files
import java.nio.file.Path
import java.util.concurrent.TimeUnit

internal class TargetTestRunner(
    private val environment: Map<String, String> = System.getenv(),
    private val progress: (String) -> Unit = {},
) {
    fun run(root: Path, comparison: GitComparison): TestExecution {
        val cache = Path.of(environment["SCRYER_CACHE_HOME"] ?: "${System.getProperty("java.io.tmpdir")}/scryer-cache").toAbsolutePath().normalize()
        val configuredTimeout = environment["SCRYER_TEST_TIMEOUT_SECONDS"]
        val timeout = configuredTimeout?.toLongOrNull() ?: if (configuredTimeout == null) 600L else 0L
        if (timeout <= 0) return TestExecution(comparison.after, TestRunStatus.UNAVAILABLE, notes = listOf("SCRYER_TEST_TIMEOUT_SECONDS must be a positive integer"))
        val run = try {
            Files.createDirectories(cache.resolve("runs"))
            Files.createTempDirectory(cache.resolve("runs"), "analyze-")
        } catch (exception: Exception) {
            return TestExecution(comparison.after, TestRunStatus.UNAVAILABLE, notes = listOf("Cannot create test log directory: ${exception.message}"))
        }
        val plan = try {
            TestRunPlanner(environment).plan(root, comparison.repository, cache, run)
        } catch (exception: Exception) {
            return TestExecution(comparison.after, TestRunStatus.UNAVAILABLE, notes = listOf(exception.message ?: "Cannot select target test command"))
        }
        val execution = execute(root, plan, run.resolve("test.log"), comparison.after, timeout)
        val intact = runCatching {
            GitProcess.run(root, "rev-parse", "HEAD").trim() == comparison.after &&
                GitProcess.run(root, "diff", "--name-only", "HEAD", "--").isBlank() &&
                GitProcess.run(root, "diff", "--cached", "--name-only", "HEAD", "--").isBlank()
        }.getOrDefault(false)
        return if (intact) execution else execution.copy(status = TestRunStatus.SNAPSHOT_CHANGED,
            notes = execution.notes + "The command changed tracked source/index or HEAD; results are not valid evidence for the requested after SHA.")
    }

    internal fun execute(root: Path, plan: TestRunPlan, log: Path, afterSha: String, timeout: Long, label: String = "after tests"): TestExecution {
        val started = System.nanoTime()
        val notes = listOf("Conventional Wrapper lifecycle only; custom test tasks/commands are not discovered yet.",
            "Command exit status is not per-test execution, build packaging or coverage evidence. Fresh reports/JaCoCo data are collected separately; no test-to-method attribution is inferred from command status.")
        var process: Process? = null
        try {
            val builder = ProcessBuilder(plan.command).directory(root.toFile()).redirectErrorStream(true).redirectOutput(log.toFile())
            builder.environment().putAll(plan.environment)
            // Target classpaths belong to the build, rather than the inherited shell.
            builder.environment().remove("CLASSPATH")
            process = builder.start()
            progress("Running $label: ${plan.command.joinToString(" ")}; JAVA_HOME=${plan.javaHome}")
            var nextProgress = 15L
            while (!process.waitFor(1, TimeUnit.SECONDS)) {
                val elapsed = TimeUnit.NANOSECONDS.toSeconds(System.nanoTime() - started)
                if (elapsed >= timeout) {
                    stop(process)
                    return TestExecution(afterSha, TestRunStatus.TIMED_OUT, plan.command, plan.javaHome.toString(),
                        durationMillis = elapsedMillis(started), log = log, notes = notes + "Timed out after ${timeout}s")
                }
                if (elapsed >= nextProgress) {
                    progress("$label still running (${elapsed}s); log: $log")
                    nextProgress += 15
                }
            }
            val exit = process.exitValue()
            return TestExecution(afterSha, if (exit == 0) TestRunStatus.SUCCEEDED else TestRunStatus.FAILED,
                plan.command, plan.javaHome.toString(), exit, elapsedMillis(started), log, notes)
        } catch (exception: Exception) {
            if (exception is InterruptedException) Thread.currentThread().interrupt()
            return TestExecution(afterSha, TestRunStatus.UNAVAILABLE, plan.command, plan.javaHome.toString(),
                durationMillis = elapsedMillis(started), log = log, notes = notes + "Cannot complete test command: ${exception.message}")
        } finally {
            process?.takeIf { it.isAlive }?.let(::stop)
        }
    }

    private fun stop(process: Process) {
        runCatching { process.toHandle().descendants().use { children -> children.forEach { it.destroyForcibly() } } }
        process.destroyForcibly()
        runCatching { process.waitFor(5, TimeUnit.SECONDS) }
    }

    private fun elapsedMillis(started: Long) = TimeUnit.NANOSECONDS.toMillis(System.nanoTime() - started)
}
