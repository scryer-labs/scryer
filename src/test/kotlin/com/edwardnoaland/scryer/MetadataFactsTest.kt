package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.scan.inspect.RepositoryScanner
import com.edwardnoaland.scryer.scan.inspect.text
import java.nio.file.Path
import kotlin.io.path.*
import kotlin.test.*
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir

class MetadataFactsTest {
    @TempDir lateinit var root: Path
    private fun write(path: String, text: String) { root.resolve(path).parent.createDirectories(); root.resolve(path).writeText(text) }

    @Test fun `Gradle modules include dependencies and literal projectDir remaps`() {
        write("settings.gradle.kts", "include(\":api\", \":shared\")\nproject(\":shared\").projectDir = file(\"libs/shared\")")
        write("build.gradle.kts", "")
        write("api/build.gradle.kts", "dependencies { implementation(\"example:api:2\") }")
        write("libs/shared/build.gradle.kts", "dependencies { api(\"example:shared:3\") }")
        val facts = RepositoryScanner().scan(root)
        assertEquals(3, facts.modules.size)
        assertEquals(setOf(":api", ":shared"), facts.dependencies.map { it.module }.toSet())
    }

    @Test fun `Maven reactor modules are expanded and outside modules are not read`() {
        write("pom.xml", "<project><modules><module>api</module><module>../outside</module></modules></project>")
        write("api/pom.xml", "<project><dependencies><dependency><groupId>example</groupId><artifactId>api</artifactId><version>1</version></dependency></dependencies></project>")
        val facts = RepositoryScanner().scan(root)
        assertEquals(2, facts.modules.size)
        assertEquals("api", facts.dependencies.single().module)
        assertTrue(facts.notes.any { "missing" in it || "External" in it })
    }

    @Test fun `source metadata identifies legacy edges without claiming test execution`() {
        write("build.gradle", "sourceCompatibility = 1.8\ntargetCompatibility = 1.8\napply plugin: 'jacoco'")
        write("src/main/java/example/App.java", "package example; import javax.servlet.Servlet; class App { void f() { Class.forName(\"example.Hidden\"); } }")
        write("src/test/java/example/AppTest.java", "import org.junit.Test; import org.junit.Ignore; class AppTest { @Ignore @Test public void test() {} }")
        write("src/test/java/example/AppIntegrationTest.java", "import org.junit.Test; class AppIntegrationTest {}")
        write(".tooling/ignored.java", "jakarta.persistence.Fake")
        write("build/generated/test.java", "jakarta.persistence.Generated")
        val facts = RepositoryScanner().scan(root)
        assertEquals(listOf("8"), facts.language.sourceVersions)
        assertEquals(1, facts.testing.testSourceFiles)
        assertEquals(1, facts.testing.integrationSourceFiles)
        assertEquals(1, facts.testing.disabledAnnotationSignals)
        assertTrue(facts.testing.jacocoConfigured)
        assertTrue(facts.signals.first { it.name == "Reflection" }.locations.isNotEmpty())
        assertTrue(facts.signals.first { it.name == "jakarta usage" }.locations.isEmpty())
        assertContains(facts.testing.classification, "not executed")
        assertEquals(listOf("build/generated"), facts.sources.generatedRoots)
    }

    @Test fun `local managed dependency retains original missing version and source`() {
        write("pom.xml", """<project><dependencyManagement><dependencies><dependency><groupId>g</groupId><artifactId>a</artifactId><version>2</version></dependency></dependencies></dependencyManagement><dependencies><dependency><groupId>g</groupId><artifactId>a</artifactId></dependency></dependencies></project>""")
        val dep = RepositoryScanner().scan(root).dependencies.first()
        assertNull(dep.declaredVersion)
        assertEquals("2", dep.version)
        assertEquals("pom.xml dependencyManagement", dep.managementSource)
    }

    @Test fun `Maven compiler processor paths are distinct from runtime dependencies`() {
        write("pom.xml", """<project><build><plugins><plugin><artifactId>maven-compiler-plugin</artifactId><configuration><annotationProcessorPaths><path><groupId>org.mapstruct</groupId><artifactId>mapstruct-processor</artifactId><version>1.5.5.Final</version></path></annotationProcessorPaths></configuration></plugin></plugins></build></project>""")
        val facts = RepositoryScanner().scan(root)
        assertTrue(facts.dependencies.isEmpty())
        assertEquals("org.mapstruct:mapstruct-processor", facts.compileTooling.annotationProcessors.single().notation)
        assertContains(facts.compileTooling.codeGenerationSignals, "mapstruct")
    }
}
