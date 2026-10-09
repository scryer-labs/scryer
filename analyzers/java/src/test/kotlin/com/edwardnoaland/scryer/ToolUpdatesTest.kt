package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.scan.model.BuildFacts
import com.edwardnoaland.scryer.scan.model.LanguageFacts
import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.remote.*
import java.nio.file.Path
import java.util.concurrent.ConcurrentHashMap
import org.junit.jupiter.api.Test
import kotlin.test.*

class ToolUpdatesTest {
    private fun facts() = RepositoryFacts(Path.of("."), listOf(
        BuildFacts("Gradle", "6.5.1", "build.gradle", true),
        BuildFacts("Maven", "3.9.3", "pom.xml", true)),
        language = LanguageFacts(sourceVersions = listOf("11")))

    @Test fun `queries stable build tools and Java feature and patch channels without using host Java`() {
        val urls = ConcurrentHashMap<String, Int>()
        val collector = ToolUpdateCollector(fetch = { url ->
            urls.merge(url, 1, Int::plus)
            when {
                url == ToolUpdateCollector.GRADLE -> """{"version":"9.1.0","snapshot":false}"""
                url == ToolUpdateCollector.MAVEN -> "<metadata><versioning><versions><version>4.0.0-rc-5</version><version>3.9.9</version><version>3.9.11</version><version>4.0.0-SNAPSHOT</version></versions></versioning></metadata>"
                url == ToolUpdateCollector.ADOPTIUM -> """{"most_recent_lts":25}"""
                "/25/" in url -> """[{"version":{"openjdk_version":"25.0.1+8-LTS"}}]"""
                "/11/" in url -> """[{"version":{"openjdk_version":"11.0.29+7"}}]"""
                else -> error("Unexpected URL")
            }.toByteArray()
        })
        val result = collector.collect(facts())
        assertEquals("3.9.11", result.tools.single { it.tool == "Maven" }.latest)
        assertEquals("UPDATE_AVAILABLE", result.tools.single { it.tool == "Gradle" }.status)
        val java = result.tools.filter { it.tool == "Java" }
        assertEquals(listOf("11", "11"), java.map { it.current })
        assertEquals(listOf("UPDATE_AVAILABLE", "UNKNOWN_CURRENT"), java.map { it.status })
        assertTrue(result.notes.any { "dependency conflicts" in it })
        assertTrue(urls.values.all { it == 1 })
    }

    @Test fun `matching latest Java LTS still cannot prove patch currency and deduplicates feature lookup`() {
        var assets = 0
        val result = ToolUpdateCollector(fetch = { url ->
            if (url == ToolUpdateCollector.ADOPTIUM) """{"most_recent_lts":25}""".toByteArray()
            else { assets++; """[{"version":{"openjdk_version":"25.0.1+8-LTS"}}]""".toByteArray() }
        }).collect(RepositoryFacts(Path.of("."), emptyList(), language = LanguageFacts(
            sourceVersions = listOf("11"), toolchainVersions = listOf("25"))))
        assertEquals(1, assets)
        assertEquals(listOf("CURRENT", "UNKNOWN_CURRENT"), result.tools.map { it.status })
        assertTrue(result.tools.all { it.current == "25" && it.currentSource == "declared Java toolchain" })
    }

    @Test fun `remote failures remain unavailable and unknown declarations do not borrow runtime versions`() {
        val result = ToolUpdateCollector(fetch = { error("offline") }).collect(facts().copy(language = LanguageFacts()))
        assertTrue(result.tools.all { it.status == "UNAVAILABLE" && it.latest == null })
        assertNull(result.tools.single { it.tool == "Java" }.current)
        assertTrue(result.tools.all { it.note?.contains("offline") == true })
        assertEquals(8, javaFeature("1.8"))
        assertEquals(21, javaFeature("21"))
        assertNull(javaFeature("\${java.version}"))
        assertFails { gradleRelease("""{"version":"10.0-rc-1"}""".toByteArray()) }
        assertFails { mavenRelease("<!DOCTYPE x SYSTEM 'file:///etc/passwd'><metadata/>".toByteArray()) }
    }
}
