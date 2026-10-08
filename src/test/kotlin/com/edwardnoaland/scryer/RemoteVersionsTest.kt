package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.scan.remote.*
import com.edwardnoaland.scryer.scan.model.DependencyDeclaration
import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import org.junit.jupiter.api.Test
import kotlin.test.*

class RemoteVersionsTest {
    @Test fun `queries each coordinate once and preserves selected versions and unknown declarations`() {
        var calls = 0
        val collector = RemoteVersionCollector(lookup = { _, _ -> calls++; ReleaseLookup("2.0", "https://example.test/metadata") })
        val dependencies = listOf(DependencyDeclaration("compile", "example:lib", "1.+", "build.gradle"),
            DependencyDeclaration("test", "example:lib", null, "build.gradle"))
        val result = collector.collect(dependencies) { if (it.configuration == "compile") listOf("1.0", "3.0") else emptyList() }
        assertEquals(1, calls)
        assertEquals(listOf("UPDATE_AVAILABLE", "CURRENT_AHEAD", "UNKNOWN_CURRENT"), result.dependencies.map { it.status })
        assertEquals(listOf("selected", "selected", "declared"), result.dependencies.map { it.currentSource })
        assertEquals("CURRENT", comparisonStatus("2.0.0", ReleaseLookup("2.0", null)))
        assertEquals("UNKNOWN_CURRENT", comparisonStatus("\${version}", ReleaseLookup("2.0", null)))
    }

    @Test fun `HTTP metadata handles release missing artifacts and rejects external XML entities`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/example/good/maven-metadata.xml") { request ->
            val body = "<metadata><versioning><latest>9-SNAPSHOT</latest><release>2.0</release></versioning></metadata>".toByteArray()
            request.sendResponseHeaders(200, body.size.toLong()); request.responseBody.use { it.write(body) }
        }
        server.createContext("/example/unsafe/maven-metadata.xml") { request ->
            val body = "<!DOCTYPE metadata SYSTEM 'file:///etc/passwd'><metadata/>".toByteArray()
            request.sendResponseHeaders(200, body.size.toLong()); request.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val metadata = MavenCentralMetadata("http://127.0.0.1:${server.address.port}")
            assertEquals("2.0", metadata.lookup("example", "good").release)
            assertContains(metadata.lookup("example", "missing").error!!, "HTTP 404")
            assertNull(metadata.lookup("example", "unsafe").release)
            assertNull(metadata.lookup("../invalid", "good").url)
        } finally { server.stop(0) }
    }
}
