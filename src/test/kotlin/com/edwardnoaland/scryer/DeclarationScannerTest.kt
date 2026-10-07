package com.edwardnoaland.scryer

import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.nio.file.Path
import kotlin.io.path.createDirectories
import kotlin.io.path.writeText
import kotlin.test.*

class DeclarationScannerTest {
    @TempDir lateinit var root: Path

    private fun gradle(text: String, filename: String = "build.gradle"): RepositoryFacts {
        root.resolve(filename).writeText(text)
        return RepositoryScanner().scan(root)
    }

    private fun maven(text: String): RepositoryFacts {
        root.resolve("pom.xml").writeText(text)
        return RepositoryScanner().scan(root)
    }

    @Test fun `legacy fixture declarations retain configurations and missing versions`() {
        val facts = gradle("""
            plugins { id 'org.springframework.boot' version '2.1.18.RELEASE' }
            dependencies {
                compile 'org.springframework.boot:spring-boot-starter-web'
                compile 'commons-lang:commons-lang:2.6'
                compile 'com.google.guava:guava:20.0'
                testCompile 'org.springframework.boot:spring-boot-starter-test'
            }
        """.trimIndent())
        assertEquals(listOf(null, "2.6", "20.0", null), facts.dependencies.map { it.version })
        assertEquals(listOf("compile", "compile", "compile", "testCompile"), facts.dependencies.map { it.configuration })
        assertEquals("2.1.18.RELEASE", facts.plugins.single().version)
    }

    @Test fun `comments and coordinate-like strings outside dependencies are not dependencies`() {
        val facts = gradle("""
            // dependencies { implementation 'fake:comment:1' }
            /* dependencies { implementation 'fake:block:1' } */
            def description = 'fake:description:1'
            dependencies {
                implementation 'real:artifact:2' // trailing comment
            }
        """.trimIndent())
        assertEquals(listOf("real:artifact"), facts.dependencies.map { it.notation })
    }

    @Test fun `reads Kotlin declarations multiline platforms and dollar properties`() {
        root.resolve("gradle.properties").writeText("libVersion=3.2\n")
        val facts = gradle("""
            val bomVersion = "1.0"
            dependencies {
                implementation(
                    "example:library:${'$'}{libVersion}"
                )
                implementation(platform("example:bom:${'$'}bomVersion"))
            }
        """.trimIndent(), "build.gradle.kts")
        assertEquals(listOf("3.2", "1.0"), facts.dependencies.map { it.version })
        assertEquals("platform", facts.dependencies.last().kind)
    }

    @Test fun `does not lose unresolved version expressions`() {
        val facts = gradle("dependencies { implementation 'example:library:${'$'}{unknown}' }")
        assertEquals("${'$'}{unknown}", facts.dependencies.single().version)
        assertContains(facts.dependencies.single().versionDisplay, "unresolved")
    }

    @Test fun `supports legacy map notation`() {
        val facts = gradle("dependencies { compile group: 'commons-lang', name: 'commons-lang', version: '2.6' }")
        assertEquals("commons-lang:commons-lang", facts.dependencies.single().notation)
        assertEquals("2.6", facts.dependencies.single().version)
    }

    @Test fun `dependency exclusions do not become declarations`() {
        val facts = gradle("""
            dependencies {
                implementation('example:library:1') {
                    exclude group: 'excluded', module: 'other'
                }
            }
        """.trimIndent())
        assertEquals("example:library", facts.dependencies.single().notation)
        assertTrue(facts.notes.isEmpty())
    }

    @Test fun `map variable versions are resolved or retained as unknown expressions`() {
        val facts = gradle("""
            def knownVersion = '3'
            dependencies {
                compile group: 'example', name: 'known', version: knownVersion
                compile group: 'example', name: 'unknown', version: unknownVersion
            }
        """.trimIndent())
        assertEquals("3", facts.dependencies.first().version)
        assertContains(facts.dependencies.last().versionDisplay, "unresolved expression")
    }

    @Test fun `map exclusions do not overwrite the dependency group`() {
        val facts = gradle("""
            dependencies {
                compile(group: 'example', name: 'library', version: '2') {
                    exclude group: 'other', module: 'excluded'
                }
            }
        """.trimIndent())
        assertEquals("example:library", facts.dependencies.single().notation)
    }

    @Test fun `Maven management with a different classifier does not invent dependency version`() {
        val facts = maven("""
            <project>
              <dependencyManagement><dependencies><dependency>
                <groupId>example</groupId><artifactId>library</artifactId><version>7</version><classifier>tests</classifier>
              </dependency></dependencies></dependencyManagement>
              <dependencies><dependency><groupId>example</groupId><artifactId>library</artifactId></dependency></dependencies>
            </project>
        """.trimIndent())
        assertNull(facts.dependencies.first().version)
    }

    @Test fun `catalog and dynamic declarations are explicitly flagged`() {
        val facts = gradle("""
            dependencies {
                implementation(libs.guava)
                implementation(computeCoordinate())
                implementation(project(":shared"))
            }
        """.trimIndent(), "build.gradle.kts")
        assertTrue(facts.dependencies.isEmpty())
        assertEquals(3, facts.notes.size)
        assertTrue(facts.notes.any { "libs.guava" in it })
    }

    @Test fun `root only scope is explicit for multi module Gradle builds`() {
        root.resolve("settings.gradle").writeText("include ':api'")
        val facts = gradle("")
        assertTrue(facts.notes.any { "Module directory missing" in it })
    }

    @Test fun `Maven reads namespace properties local management and scope`() {
        val facts = maven("""
            <project xmlns="http://maven.apache.org/POM/4.0.0">
              <modelVersion>4.0.0</modelVersion>
              <properties><lib.version>2.6</lib.version></properties>
              <dependencyManagement><dependencies><dependency>
                <groupId>example</groupId><artifactId>managed</artifactId><version>7</version>
              </dependency></dependencies></dependencyManagement>
              <dependencies>
                <dependency><groupId>commons-lang</groupId><artifactId>commons-lang</artifactId><version>${'$'}{lib.version}</version></dependency>
                <dependency><groupId>example</groupId><artifactId>managed</artifactId><scope>test</scope></dependency>
              </dependencies>
            </project>
        """.trimIndent())
        assertEquals(listOf("2.6", "7", "7"), facts.dependencies.map { it.version })
        assertEquals("test", facts.dependencies[1].configuration)
        assertEquals("managed declaration", facts.dependencies.last().kind)
    }

    @Test fun `Maven parent and BOM versions are not guessed`() {
        val facts = maven("""
            <project>
              <parent><groupId>org.springframework.boot</groupId><artifactId>spring-boot-starter-parent</artifactId><version>2.1.18.RELEASE</version></parent>
              <dependencyManagement><dependencies><dependency><groupId>example</groupId><artifactId>bom</artifactId><version>1</version><scope>import</scope><type>pom</type></dependency></dependencies></dependencyManagement>
              <dependencies><dependency><groupId>org.springframework.boot</groupId><artifactId>spring-boot-starter-web</artifactId></dependency></dependencies>
            </project>
        """.trimIndent())
        assertNull(facts.dependencies.first().version)
        assertTrue(facts.notes.any { "Parent POM:" in it })
    }

    @Test fun `Maven wrapper reports distribution not wrapper plugin version`() {
        root.resolve(".mvn/wrapper").createDirectories().resolve("maven-wrapper.properties")
            .writeText("wrapperVersion=3.3.2\ndistributionUrl=https\\://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/3.9.9/apache-maven-3.9.9-bin.zip\n")
        assertEquals("3.9.9", maven("<project/>").builds.single().version)
    }

    @Test fun `Maven external entity declarations are rejected`() {
        assertFailsWith<ScanException> {
            maven("""<!DOCTYPE project [<!ENTITY leaked SYSTEM "file:///etc/passwd">]><project><name>&leaked;</name></project>""")
        }
    }

    @Test fun `malformed POM is an error not an empty successful scan`() {
        assertFailsWith<ScanException> { maven("<project>") }
    }

    @Test fun `property cycles terminate without inventing a version`() {
        assertContains(expandProperties("${'$'}{a}", mapOf("a" to "${'$'}{b}", "b" to "${'$'}{a}")), "${'$'}")
    }
}
