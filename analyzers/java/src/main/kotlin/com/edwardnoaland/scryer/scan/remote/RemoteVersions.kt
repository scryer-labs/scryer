package com.edwardnoaland.scryer.scan.remote

import com.edwardnoaland.scryer.scan.model.DependencyDeclaration
import com.edwardnoaland.scryer.scan.model.RemoteVersion
import com.edwardnoaland.scryer.scan.model.RemoteVersions
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.time.Duration
import java.time.Instant
import java.util.concurrent.Executors
import javax.xml.parsers.DocumentBuilderFactory
import org.apache.maven.artifact.versioning.ComparableVersion

internal data class ReleaseLookup(val release: String?, val url: String?, val error: String? = null)

/** Enrichment is explicit and never changes build declarations or selected versions. */
internal class RemoteVersionCollector(
    private val lookup: (String, String) -> ReleaseLookup = MavenCentralMetadata()::lookup,
    private val progress: (String) -> Unit = {},
) {
    fun collect(dependencies: List<DependencyDeclaration>, selected: (DependencyDeclaration) -> List<String>): RemoteVersions {
        val coordinates = dependencies.map { it.group to it.artifact }.distinct()
        progress("Querying Maven Central release metadata for ${coordinates.size} direct dependency coordinates")
        val pool = Executors.newFixedThreadPool(4)
        val releases = try {
            val requests = coordinates.associateWith { (group, artifact) -> pool.submit<ReleaseLookup> {
                runCatching { lookup(group, artifact) }.getOrElse { ReleaseLookup(null, null, it.message ?: "Lookup failed") }
            } }
            requests.mapValues { it.value.get() }
        } finally { pool.shutdownNow() }
        val rows = dependencies.flatMap { dependency ->
            val resolved = selected(dependency).distinct().sorted()
            val currentVersions = resolved.ifEmpty { listOf(dependency.version) }
            currentVersions.map { current ->
                val release = releases.getValue(dependency.group to dependency.artifact)
                RemoteVersion(dependency.module, dependency.configuration, dependency.notation, current,
                    if (resolved.isEmpty()) "declared" else "selected", release.release,
                    comparisonStatus(current, release), release.url, release.error)
            }
        }.distinct()
        return RemoteVersions(Instant.now().toString(), "Maven Central metadata <release>; not compatibility advice", rows)
    }
}

internal fun comparisonStatus(current: String?, lookup: ReleaseLookup): String {
    if (lookup.error != null || lookup.release == null) return "UNAVAILABLE"
    if (current == null || !Regex("[A-Za-z0-9][A-Za-z0-9._-]*").matches(current) || current.lowercase() in setOf("latest.release", "latest.integration", "release", "latest")) return "UNKNOWN_CURRENT"
    val comparison = ComparableVersion(current).compareTo(ComparableVersion(lookup.release))
    return when { comparison < 0 -> "UPDATE_AVAILABLE"; comparison == 0 -> "CURRENT"; else -> "CURRENT_AHEAD" }
}

internal class MavenCentralMetadata(private val baseUrl: String = "https://repo.maven.apache.org/maven2") {
    private val client = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(5)).build()
    fun lookup(group: String, artifact: String): ReleaseLookup {
        if (!Regex("[A-Za-z0-9_-]+(?:\\.[A-Za-z0-9_-]+)*").matches(group) || !Regex("[A-Za-z0-9_][A-Za-z0-9_.-]*").matches(artifact)) {
            return ReleaseLookup(null, null, "Not a literal Maven coordinate; not queried")
        }
        val url = "$baseUrl/${group.replace('.', '/')}/$artifact/maven-metadata.xml"
        return try {
            val request = HttpRequest.newBuilder(URI.create(url)).timeout(Duration.ofSeconds(10)).header("User-Agent", "Scryer/0.1").GET().build()
            val response = client.send(request, HttpResponse.BodyHandlers.ofByteArray())
            if (response.statusCode() != 200) return ReleaseLookup(null, url, "HTTP ${response.statusCode()}; unavailable from Maven Central")
            require(response.body().size <= 512 * 1024) { "Metadata exceeds 512 KiB" }
            val factory = DocumentBuilderFactory.newInstance().apply {
                setFeature("http://apache.org/xml/features/disallow-doctype-decl", true)
                setFeature("http://xml.org/sax/features/external-general-entities", false)
                setFeature("http://xml.org/sax/features/external-parameter-entities", false)
                setAttribute(javax.xml.XMLConstants.ACCESS_EXTERNAL_DTD, "")
                setAttribute(javax.xml.XMLConstants.ACCESS_EXTERNAL_SCHEMA, "")
            }
            val document = factory.newDocumentBuilder().parse(response.body().inputStream())
            val release = document.getElementsByTagName("release").item(0)?.textContent?.trim()?.takeIf { it.isNotEmpty() }
            if (release == null) ReleaseLookup(null, url, "Metadata has no release version") else ReleaseLookup(release, url)
        } catch (exception: Exception) {
            if (exception is InterruptedException) Thread.currentThread().interrupt()
            ReleaseLookup(null, url, "Metadata lookup failed: ${exception.javaClass.simpleName}")
        }
    }
}
