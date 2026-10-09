package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.output.MarkdownReport
import com.edwardnoaland.scryer.scan.model.*
import java.nio.file.Path
import org.junit.jupiter.api.Test
import kotlin.test.*

class RemoteMarkdownTest {
    @Test fun `Markdown preserves failed and unknown lookups selected provenance and escapes table content`() {
        val rows = listOf(RemoteVersion(":api", "compile", "example:lib", "1.0", "selected", "2.0", "UPDATE_AVAILABLE", "https://example.test/metadata"),
            RemoteVersion(":api", "test", "example:missing", null, "declared", null, "UNAVAILABLE", null, "failure | <tag>\nnext"))
        val facts = RepositoryFacts(Path.of("."), emptyList(), remoteVersions = RemoteVersions("timestamp", "Maven Central release", rows))
        val markdown = MarkdownReport().render(facts)
        assertContains(markdown, "Remote dependency releases")
        assertContains(markdown, "timestamp")
        assertContains(markdown, "selected")
        assertContains(markdown, "UPDATE_AVAILABLE")
        assertContains(markdown, "unknown")
        assertContains(markdown, "UNAVAILABLE")
        assertContains(markdown, "failure \\| &lt;tag&gt;<br>next")
        assertFalse(markdown.contains('\u001b'))
        assertFalse(MarkdownReport().render(facts.copy(remoteVersions = null)).contains("Remote dependency releases"))
    }
}
