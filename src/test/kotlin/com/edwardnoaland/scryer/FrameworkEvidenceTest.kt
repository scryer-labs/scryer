package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.scan.inspect.RepositoryScanner
import com.edwardnoaland.scryer.scan.model.ConfigurationGraph
import com.edwardnoaland.scryer.scan.model.GraphNode
import com.edwardnoaland.scryer.scan.model.ResolutionFacts
import com.edwardnoaland.scryer.scan.withResolution
import java.nio.file.Path
import kotlin.io.path.createDirectories
import kotlin.io.path.writeText
import kotlin.test.assertEquals
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir

class FrameworkEvidenceTest {
    @TempDir lateinit var root: Path

    private fun jupiterRepository() = run {
        root.resolve("build.gradle").writeText("plugins { id 'java' }")
        val tests = root.resolve("src/test/java").createDirectories()
        tests.resolve("ExampleTest.java").writeText("import org.junit.jupiter.api.Test; class ExampleTest { @Test void works() {} }")
        RepositoryScanner().scan(root)
    }

    @Test fun `Jupiter imports identify a framework without inventing its major version`() {
        assertEquals(listOf("JUnit Jupiter"), jupiterRepository().testing.frameworks)
    }

    @Test fun `resolved Jupiter six is not mislabeled as JUnit five`() {
        val graph = ConfigurationGraph(
            module = ":",
            configuration = "testRuntimeClasspath",
            root = "project :",
            nodes = listOf(GraphNode("jupiter", "org.junit.jupiter", "junit-jupiter-api", "6.1.3")),
            edges = emptyList(),
        )
        val resolution = ResolutionFacts("complete", configurations = listOf(graph))
        assertEquals(listOf("JUnit Jupiter 6"), withResolution(jupiterRepository(), resolution).testing.frameworks)
    }
}
