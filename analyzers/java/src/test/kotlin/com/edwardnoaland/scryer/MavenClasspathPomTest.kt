package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.analyze.model.writeExternalClasspathPom
import java.nio.file.Path
import javax.xml.parsers.DocumentBuilderFactory
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import kotlin.test.*

class MavenClasspathPomTest {
    @TempDir lateinit var root: Path
    @Test fun `reactor coordinates are excluded but unrelated external artifacts and repositories survive`() {
        val factory = DocumentBuilderFactory.newInstance()
        val project = factory.newDocumentBuilder().parse("""
            <project><modelVersion>4.0.0</modelVersion><groupId>fixture</groupId><artifactId>app</artifactId><version>1</version>
            <dependencies>
            <dependency><groupId>fixture</groupId><artifactId>domain</artifactId><version>1</version></dependency>
            <dependency><groupId>external</groupId><artifactId>domain</artifactId><version>2</version><scope>test</scope></dependency>
            </dependencies><repositories><repository><id>private</id><url>https://example.invalid</url></repository></repositories>
            <build><plugins><plugin><artifactId>custom-plugin</artifactId></plugin></plugins></build></project>
        """.trimIndent().byteInputStream()).documentElement
        val output = root.resolve("classpath.pom.xml")
        writeExternalClasspathPom(project, output, setOf("fixture" to "domain"))
        val result = factory.newDocumentBuilder().parse(output.toFile())
        assertEquals(1, result.getElementsByTagName("dependency").length)
        assertEquals("external", (result.getElementsByTagName("dependency").item(0) as org.w3c.dom.Element).getElementsByTagName("groupId").item(0).textContent)
        assertEquals("test", result.getElementsByTagName("scope").item(0).textContent)
        assertEquals(1, result.getElementsByTagName("repository").length)
        assertEquals(0, result.getElementsByTagName("build").length)
    }
}
