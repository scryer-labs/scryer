package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.evidence.*
import com.edwardnoaland.scryer.analyze.execute.TestExecution
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus
import org.jacoco.core.data.ExecutionData
import org.jacoco.core.data.ExecutionDataWriter
import org.jacoco.core.instr.Instrumenter
import org.objectweb.asm.Opcodes
import java.nio.file.Files
import java.nio.file.Path
import javax.tools.ToolProvider
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.*

/** Real probe acquisition without requiring an agent in the Scryer test JVM. */
object EvidenceProbes {
    val data = mutableMapOf<Long, ExecutionData>()
    @JvmStatic fun probes(id: Long, name: String, count: Int): BooleanArray =
        data.getOrPut(id) { ExecutionData(id, name, count) }.probes
}

class ExecutionEvidenceTest {
    @TempDir lateinit var temporary: Path
    private fun root(): Path = Files.createDirectories(temporary.resolve("project"))
    private fun execution(status: TestRunStatus = TestRunStatus.SUCCEEDED): TestExecution {
        val log = temporary.resolve("run/test.log")
        Files.createDirectories(log.parent)
        Files.writeString(log, "execution fixture")
        return TestExecution("after-sha", status, log = log)
    }
    private fun write(root: Path, relative: String, content: String): Path {
        val path = root.resolve(relative)
        Files.createDirectories(path.parent)
        Files.writeString(path, content)
        return path
    }

    @Test fun `retains fresh JUnit records including failures errors skips and parameter names`() {
        val root = root()
        val before = ArtifactInventory.capture(root)
        write(root, "module/target/surefire-reports/TEST-Suite.xml", """
            <testsuites><testsuite><testcase classname="Suite" name="passed[1]" time="0.2"/>
            <testcase classname="Suite" name="fails"><failure/></testcase>
            <testcase classname="Suite" name="error"><error/></testcase>
            <testcase classname="Suite" name="skip"><skipped/></testcase></testsuite></testsuites>
        """.trimIndent())
        write(root, "src/test/resources/target/surefire-reports/TEST-Static.xml", "<testsuite><testcase name=\"fake\"/></testsuite>")
        val evidence = EvidenceCollector().collect(root, execution(TestRunStatus.FAILED), before)
        assertEquals("after-sha", evidence.afterSha)
        assertEquals(TestCaseStatus.entries.toSet(), evidence.tests.single().cases.map { it.status }.toSet())
        assertEquals("passed[1]", evidence.tests.single().cases.first().name)
        assertEquals(0.2, evidence.tests.single().cases.first().seconds)
        val artifact = evidence.tests.single().artifact
        assertContains(Files.readString(evidence.manifest), "after-sha")
        assertEquals(1, evidence.artifacts.size)
        root.toFile().deleteRecursively()
        assertTrue(Files.exists(artifact.retained))
        assertEquals(artifact.sha256, ArtifactInventory.hash(artifact.retained))
        assertTrue(evidence.notes.any { it.contains("partial/failed") })
    }

    @Test fun `unchanged reports are rejected and malformed fresh files remain available for inspection`() {
        val root = root()
        write(root, "build/test-results/test/TEST-Old.xml", "<testsuite><testcase name=\"old\"/></testsuite>")
        val before = ArtifactInventory.capture(root)
        write(root, "build/test-results/test/TEST-Broken.xml", "not XML")
        val evidence = EvidenceCollector().collect(root, execution(), before)
        assertTrue(evidence.tests.isEmpty())
        assertTrue(evidence.coverage.isEmpty())
        assertTrue(evidence.notes.any { it.contains("pre-existing") })
        assertTrue(evidence.notes.any { it.contains("Cannot parse") })
        assertTrue(Files.exists(temporary.resolve("run/evidence/build/test-results/test/TEST-Broken.xml")))
    }

    @Test fun `XML coverage permits JaCoCo doctype without fetching external resources`() {
        val root = root()
        val before = ArtifactInventory.capture(root)
        write(root, "target/site/jacoco/jacoco.xml", """
            <!DOCTYPE report PUBLIC "-//JACOCO//DTD Report 1.1//EN" "file:///nonexistent/report.dtd">
            <report name="coverage"><package name="example"><class name="example/Example">
            <method name="run" desc="()V" line="7"><counter type="INSTRUCTION" missed="1" covered="2"/></method>
            </class></package></report>
        """.trimIndent())
        val evidence = EvidenceCollector().collect(root, execution(), before)
        val klass = evidence.coverage.single().classes.single()
        assertEquals(ClassDataMatch.XML_UNVERIFIED, klass.match)
        assertEquals("example.Example", klass.name)
        assertEquals(2, klass.methods.single().instructions.covered)
        assertEquals(7, klass.methods.single().firstLine)
    }

    @Test fun `untrusted execution cannot produce authoritative coverage`() {
        val root = root()
        write(root, "build/test-results/test/TEST-Fake.xml", "<testsuite><testcase name=\"fake\"/></testsuite>")
        for (status in listOf(TestRunStatus.SKIPPED, TestRunStatus.TIMED_OUT, TestRunStatus.UNAVAILABLE, TestRunStatus.SNAPSHOT_CHANGED)) {
            val evidence = EvidenceCollector().collect(root, execution(status), emptyMap())
            assertTrue(evidence.tests.isEmpty())
            assertTrue(evidence.coverage.isEmpty())
            assertContains(evidence.notes.single(), status.name)
        }
    }

    private fun compile(root: Path, number: Int): Path {
        val source = write(root, "ProbeExample.java", "public class ProbeExample { public static int hit() { return $number; } public static int never() { return 0; } }")
        val destination = root.resolve("build/classes/java/main")
        Files.createDirectories(destination)
        assertEquals(0, ToolProvider.getSystemJavaCompiler().run(null, null, null, "-g", "-d", destination.toString(), source.toString()))
        return destination.resolve("ProbeExample.class")
    }
    private fun recordExecution(classFile: Path, destination: Path) {
        EvidenceProbes.data.clear()
        val instrumenter = Instrumenter { id, name, count, visitor ->
            visitor.visitLdcInsn(id)
            visitor.visitLdcInsn(name)
            visitor.visitLdcInsn(count)
            visitor.visitMethodInsn(Opcodes.INVOKESTATIC, "com/edwardnoaland/scryer/EvidenceProbes", "probes", "(JLjava/lang/String;I)[Z", false)
            4
        }
        val bytes = instrumenter.instrument(Files.readAllBytes(classFile), "ProbeExample")
        val loader = object : ClassLoader(EvidenceProbes::class.java.classLoader) {
            fun define(): Class<*> = defineClass("ProbeExample", bytes, 0, bytes.size)
        }
        loader.define().getMethod("hit").invoke(null)
        Files.createDirectories(destination.parent)
        Files.newOutputStream(destination).use { stream ->
            val writer = ExecutionDataWriter(stream)
            EvidenceProbes.data.values.forEach(writer::visitClassExecution)
        }
    }

    @Test fun `real exec probes identify executed and untouched methods without per-test attribution`() {
        val root = root()
        val before = ArtifactInventory.capture(root)
        val file = compile(root, 1)
        recordExecution(file, root.resolve("build/jacoco/test.exec"))
        val evidence = EvidenceCollector().collect(root, execution(), before)
        val klass = evidence.coverage.single().classes.single()
        assertEquals(ClassDataMatch.MATCHED, klass.match)
        assertTrue(klass.methods.single { it.symbol.name == "hit" }.instructions.covered > 0)
        assertEquals(0, klass.methods.single { it.symbol.name == "never" }.instructions.covered)
        assertTrue(Files.exists(temporary.resolve("run/evidence/build/classes/java/main/ProbeExample.class")))
    }

    @Test fun `class ID mismatch is explicit and separate datasets are not merged`() {
        val root = root()
        val before = ArtifactInventory.capture(root)
        val file = compile(root, 1)
        recordExecution(file, root.resolve("build/jacoco/first.exec"))
        compile(root, 2)
        recordExecution(file, root.resolve("build/jacoco/second.exec"))
        val evidence = EvidenceCollector().collect(root, execution(), before)
        assertEquals(2, evidence.coverage.size)
        assertEquals(setOf(ClassDataMatch.MATCHED, ClassDataMatch.CLASS_ID_MISMATCH), evidence.coverage.map { it.classes.single().match }.toSet())
    }
    @Test fun `external entities never become report data and malformed exec is retained`() {
        val root = root()
        val before = ArtifactInventory.capture(root)
        val secret = write(root, "secret.txt", "PRIVATE_EXTERNAL_CONTENT")
        write(root, "build/test-results/test/TEST-Entity.xml", """
            <!DOCTYPE testsuite [<!ENTITY external SYSTEM "${secret.toUri()}">]>
            <testsuite><testcase classname="Suite" name="safe"><system-out>&external;</system-out></testcase></testsuite>
        """.trimIndent())
        write(root, "build/jacoco/broken.exec", "not execution data")
        val evidence = EvidenceCollector().collect(root, execution(), before)
        assertEquals("safe", evidence.tests.single().cases.single().name)
        assertFalse(evidence.toString().contains("PRIVATE_EXTERNAL_CONTENT"))
        assertTrue(evidence.coverage.isEmpty())
        assertTrue(evidence.notes.any { it.contains("broken.exec") })
        assertTrue(Files.exists(temporary.resolve("run/evidence/build/jacoco/broken.exec")))
    }

    @Test fun `service collects before worktree cleanup and missing coverage is explicit`() {
        val root = root()
        fun git(vararg args: String): String {
            val process = ProcessBuilder(listOf("git", "-C", root.toString()) + args).redirectErrorStream(true).start()
            val text = process.inputStream.bufferedReader().readText()
            check(process.waitFor() == 0) { text }
            return text.trim()
        }
        git("init")
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.com")
        write(root, "gradlew", """
            #!/bin/sh
            mkdir -p build/test-results/test
            printf '<testsuite><testcase classname="Suite" name="ran"/></testsuite>' > build/test-results/test/TEST-Suite.xml
        """.trimIndent())
        git("add", ".")
        git("commit", "-m", "fixture")
        val comparison = com.edwardnoaland.scryer.analyze.GitComparer().compare(root, "HEAD", "HEAD")
        val result = com.edwardnoaland.scryer.analyze.AnalyzeService().analyze(comparison)
        assertEquals(comparison.after, result.evidence.afterSha)
        assertEquals("ran", result.evidence.tests.single().cases.single().name)
        assertTrue(Files.exists(result.evidence.tests.single().artifact.retained))
        assertContains(Files.readString(result.evidence.manifest), comparison.after)
        assertTrue(result.evidence.coverage.isEmpty())
        assertTrue(result.evidence.notes.any { it.contains("unavailable, not zero") })
        assertFalse(Files.exists(root.resolve("build")))
        result.execution.log!!.parent.toFile().deleteRecursively()
    }

}
