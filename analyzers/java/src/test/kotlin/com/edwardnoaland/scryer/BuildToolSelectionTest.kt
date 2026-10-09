package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.AnalyzeService
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus
import com.edwardnoaland.scryer.analyze.GitComparison
import com.edwardnoaland.scryer.analyze.model.AnalysisInputCollector
import com.edwardnoaland.scryer.analyze.execute.TestRunPlanner
import com.edwardnoaland.scryer.cli.runCli
import com.edwardnoaland.scryer.repository.BuildTool
import com.edwardnoaland.scryer.scan.ScanService
import java.io.ByteArrayOutputStream
import java.io.PrintStream
import java.nio.file.Files
import java.nio.file.Path
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.*

class BuildToolSelectionTest {
    @TempDir lateinit var root: Path

    private fun dualBuild() {
        Files.writeString(root.resolve("pom.xml"), """<project><modelVersion>4.0.0</modelVersion><groupId>fixture</groupId><artifactId>app</artifactId><version>1</version><dependencies><dependency><groupId>maven.only</groupId><artifactId>dep</artifactId><version>1</version></dependency></dependencies></project>""")
        Files.writeString(root.resolve("build.gradle"), "dependencies { implementation 'gradle.only:dep:2' }")
        Files.writeString(root.resolve("mvnw"), "#!/bin/sh\nexit 0\n")
        Files.writeString(root.resolve("gradlew"), "#!/bin/sh\nexit 0\n")
    }

    @Test fun `selection inspects only selected declarations while retaining both build facts`() {
        dualBuild()
        for ((tool, group) in listOf(BuildTool.MAVEN to "maven.only", BuildTool.GRADLE to "gradle.only")) {
            val facts = ScanService().scan(root, staticOnly = true, buildTool = tool)
            assertEquals(setOf("Maven", "Gradle"), facts.builds.map { it.tool }.toSet())
            assertEquals(tool.flag, facts.selectedBuildTool)
            assertEquals(setOf(group), facts.dependencies.map { it.group }.toSet())
            assertTrue(facts.modules.all { (it.definition == "pom.xml") == (tool == BuildTool.MAVEN) })
        }
    }

    @Test fun `ambiguous default remains unavailable and missing selected definitions fail clearly`() {
        dualBuild()
        assertContains(ScanService().scan(root).resolution.notes.joinToString(), "--build-tool")
        Files.delete(root.resolve("pom.xml"))
        assertFailsWith<IllegalArgumentException> { ScanService().scan(root, staticOnly = true, buildTool = BuildTool.MAVEN) }
    }

    @Test fun `selected wrappers drive test plan and custom command replaces only lifecycle`() {
        dualBuild()
        val planner = TestRunPlanner(mapOf("SCRYER_JAVA_HOME" to System.getProperty("java.home")))
        val maven = planner.plan(root, root, root.resolve("cache"), root.resolve("run"), buildTool = BuildTool.MAVEN)
        val gradle = planner.plan(root, root, root.resolve("cache"), root.resolve("run"), buildTool = BuildTool.GRADLE)
        assertEquals("./mvnw", maven.command[1])
        assertEquals(listOf("clean", "verify"), maven.command.takeLast(2))
        assertEquals("./gradlew", gradle.command[1])
        assertEquals(listOf("clean", "test"), gradle.command.takeLast(2))
        assertEquals(listOf("sh", "-c", "./custom-tests"), planner.plan(root, root, root, root, "./custom-tests", BuildTool.MAVEN).command)
        Files.delete(root.resolve("mvnw"))
        assertFailsWith<IllegalStateException> { planner.plan(root, root, root, root, buildTool = BuildTool.MAVEN) }
    }

    @Test fun `invalid missing and duplicate CLI selections are rejected before repository operations`() {
        for (command in listOf("scan", "analyze")) {
            val prefix = if (command == "scan") arrayOf("scan", root.toString()) else arrayOf("analyze", "--before", "missing", "--after", "missing")
            for (flags in listOf(arrayOf("--build-tool"), arrayOf("--build-tool", "unknown"), arrayOf("--build-tool", "maven", "--build-tool", "gradle"))) {
                val out = ByteArrayOutputStream()
                val err = ByteArrayOutputStream()
                assertEquals(2, runCli(prefix + flags, PrintStream(out), PrintStream(err)))
                assertEquals("", out.toString())
            }
        }
    }

    @Test fun `explicit Maven model never invokes Gradle in dual wrapper snapshots`() {
        dualBuild()
        Files.writeString(root.resolve("gradlew"), "#!/bin/sh\necho wrong > gradle-invoked\nexit 99\n")
        Files.writeString(root.resolve("mvnw"), """
            #!/bin/sh
            for argument in "${'$'}@"; do
                case "${'$'}argument" in
                    -Doutput=*) printf '%s' '<project><modelVersion>4.0.0</modelVersion><groupId>fixture</groupId><artifactId>app</artifactId><version>1</version></project>' > "${'$'}{argument#-Doutput=}" ;;
                    -Dmdep.outputFile=*) printf '' > "${'$'}{argument#-Dmdep.outputFile=}" ;;
                esac
            done
        """.trimIndent())
        fun git(vararg args: String): String {
            val process = ProcessBuilder(listOf("git", "-C", root.toString()) + args).redirectErrorStream(true).start()
            val output = process.inputStream.bufferedReader().readText()
            check(process.waitFor() == 0) { output }
            return output.trim()
        }
        git("init")
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.com")
        git("add", ".")
        git("commit", "-m", "fixture")
        val sha = git("rev-parse", "HEAD")
        val comparison = GitComparison(root, sha, sha, emptyList())
        val inputs = AnalysisInputCollector().collect(root, comparison, sha, BuildTool.MAVEN)
        assertEquals(1, inputs.modules.size)
        assertTrue(inputs.notes.any { "Build model tool: maven (explicit" in it })
        assertFalse(Files.exists(root.resolve("gradle-invoked")))
        val result = AnalyzeService().analyze(comparison, buildTool = BuildTool.MAVEN)
        assertEquals(TestRunStatus.SUCCEEDED, result.execution.status)
        assertEquals("./mvnw", result.execution.command[1])
        assertTrue(result.impact.before.graph.notes.any { "Build model tool: maven (explicit" in it })
        assertTrue(result.impact.after.graph.notes.any { "Build model tool: maven (explicit" in it })
    }
}
