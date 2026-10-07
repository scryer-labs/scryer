package com.edwardnoaland.scryer

import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.nio.file.Path
import kotlin.io.path.createDirectories
import kotlin.io.path.writeText
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith

class RepositoryScannerTest {
    @TempDir lateinit var root: Path

    @Test fun `reads Gradle distribution version without executing build`() {
        root.resolve("build.gradle").writeText("throw new RuntimeException('must not execute')")
        val wrapper = root.resolve("gradle/wrapper").createDirectories()
        wrapper.resolve("gradle-wrapper.properties").writeText("distributionUrl=https\\://services.gradle.org/distributions/gradle-4.10.3-bin.zip")
        assertEquals(BuildFacts("Gradle", "4.10.3", "build.gradle"), RepositoryScanner().scan(root).builds.single())
    }

    @Test fun `missing wrapper is unknown rather than installed tool version`() {
        root.resolve("pom.xml").writeText("<project/>")
        assertEquals(null, RepositoryScanner().scan(root).builds.single().version)
    }

    @Test fun `reports both build definitions instead of guessing`() {
        root.resolve("build.gradle.kts").writeText("")
        root.resolve("pom.xml").writeText("<project/>")
        assertEquals(listOf("Gradle", "Maven"), RepositoryScanner().scan(root).builds.map { it.tool })
    }

    @Test fun `rejects directory without a recognized build`() {
        assertFailsWith<ScanException> { RepositoryScanner().scan(root) }
    }
}
