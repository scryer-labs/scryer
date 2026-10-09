package com.edwardnoaland.scryer.scan.remote

import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.model.ToolUpdate
import com.edwardnoaland.scryer.scan.model.ToolUpdates
import com.edwardnoaland.scryer.serialization.jsonMapper
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.time.Duration
import java.time.Instant
import java.util.concurrent.Executors
import javax.xml.XMLConstants
import javax.xml.parsers.DocumentBuilderFactory
import org.apache.maven.artifact.versioning.ComparableVersion

/** Repository declarations stay distinct from the host runtime and vendor-specific JDK patches. */
internal class ToolUpdateCollector(
    private val fetch: (String) -> ByteArray = ToolReleaseHttp()::fetch,
    private val progress: (String) -> Unit = {},
) {
    fun collect(facts: RepositoryFacts): ToolUpdates {
        progress("Querying Java, Gradle and Maven release metadata")
        val jobs = listOf<() -> List<ToolUpdate>>({ javaUpdates(facts) }) + facts.builds
            .filter { it.tool.lowercase() in setOf("gradle", "maven") }
            .distinctBy { it.tool to it.version }.map { build -> {
                val url = if (build.tool.equals("gradle", true)) GRADLE else MAVEN
                val release = lookup(url) { bytes ->
                    if (build.tool.equals("gradle", true)) gradleRelease(bytes) else mavenRelease(bytes)
                }
                listOf(ToolUpdate(build.tool, "latest stable", build.version,
                    if (build.wrapperPresent) {
                        if (build.tool.equals("gradle", true)) "gradle/wrapper/gradle-wrapper.properties"
                        else ".mvn/wrapper/maven-wrapper.properties"
                    } else "${build.definition}; no Wrapper version observed",
                    release.release, comparisonStatus(build.version, release), release.url, release.error))
            } }
        val pool = Executors.newFixedThreadPool(3)
        val rows = try {
            jobs.map { pool.submit<List<ToolUpdate>> { it() } }.map { it.get() }.flatten()
        } finally { pool.shutdownNow() }
        return ToolUpdates(Instant.now().toString(), rows, listOf(
            "Available releases are version information, not upgrade recommendations. Separately validate Java/build-tool versions, framework/plugin compatibility and dependency conflicts before upgrading.",
            "Java current values are repository-declared feature levels, not installed JDK versions. Same-series patches use Eclipse Temurin as a reference; vendor/platform availability may differ.",
        ))
    }

    private fun javaUpdates(facts: RepositoryFacts): List<ToolUpdate> {
        val language = facts.language
        val (versions, source) = when {
            language.toolchainVersions.isNotEmpty() -> language.toolchainVersions to "declared Java toolchain"
            language.sourceVersions.isNotEmpty() -> language.sourceVersions to "declared Java source level"
            else -> language.targetVersions to "declared Java target level"
        }
        val features = versions.mapNotNull(::javaFeature).distinct().sorted()
        val available = lookup(ADOPTIUM) { bytes ->
            jsonMapper.readTree(bytes)["most_recent_lts"]?.asInt()?.takeIf { it > 0 }?.toString()
                ?: error("Missing latest LTS feature")
        }
        val latestLts = available.release?.toIntOrNull()
        // Cache per-feature lookup: current series can also be the latest LTS.
        val releases = mutableMapOf<Int, ReleaseLookup>()
        fun release(feature: Int): ReleaseLookup = releases.getOrPut(feature) {
            val url = "https://api.adoptium.net/v3/assets/latest/$feature/hotspot?architecture=x64&image_type=jdk&os=linux&vendor=eclipse"
            lookup(url) { bytes ->
                jsonMapper.readTree(bytes).firstOrNull()?.get("version")?.get("openjdk_version")?.asText()
                    ?.removeSuffix("-LTS")?.takeIf { it.matches(Regex("[0-9][0-9.+_]*")) }
                    ?: error("Missing stable Temurin JDK release")
            }
        }
        val latest = latestLts?.let(::release) ?: available
        val currents = features.map { it.toString() }.ifEmpty { listOf(null) }
        val ltsRows = currents.map { current ->
            // Compare feature levels only; matching majors do not establish patch currency.
            val status = if (latest.error != null) "UNAVAILABLE" else comparisonStatus(current, available)
            ToolUpdate("Java", "latest LTS (Temurin reference)", current, source,
                latest.release, status, latest.url,
                latest.error ?: "Comparison is of Java feature levels only; current JDK patch/vendor is unknown.")
        }
        val patchRows = features.map { feature ->
            val latestPatch = release(feature)
            ToolUpdate("Java", "Java $feature latest patch (Temurin reference)", feature.toString(), source,
                latestPatch.release, if (latestPatch.error == null) "UNKNOWN_CURRENT" else "UNAVAILABLE", latestPatch.url,
                latestPatch.error ?: "Current JDK patch version is not declared; patch update availability cannot be determined.")
        }
        return ltsRows + patchRows
    }

    private fun lookup(url: String, parse: (ByteArray) -> String): ReleaseLookup = try {
        ReleaseLookup(parse(fetch(url)), url)
    } catch (exception: Exception) {
        if (exception is InterruptedException) Thread.currentThread().interrupt()
        ReleaseLookup(null, url, "Release lookup failed: ${exception.message ?: exception.javaClass.simpleName}")
    }

    companion object {
        const val GRADLE = "https://services.gradle.org/versions/current"
        const val MAVEN = "https://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/maven-metadata.xml"
        const val ADOPTIUM = "https://api.adoptium.net/v3/info/available_releases"
    }
}

internal fun javaFeature(version: String): Int? {
    val match = Regex("^(?:1\\.)?(\\d+)(?:[._+].*)?$").matchEntire(version.trim()) ?: return null
    return match.groupValues[1].toIntOrNull()?.takeIf { it in 8..99 }
}

internal fun gradleRelease(bytes: ByteArray): String {
    val node = jsonMapper.readTree(bytes)
    val version = node["version"]?.asText().orEmpty()
    require(version.matches(Regex("\\d+\\.\\d+(?:\\.\\d+)?")) && node["snapshot"]?.asBoolean() != true) { "Not a stable Gradle release" }
    return version
}

internal fun mavenRelease(bytes: ByteArray): String {
    val factory = DocumentBuilderFactory.newInstance().apply {
        setFeature("http://apache.org/xml/features/disallow-doctype-decl", true)
        setFeature("http://xml.org/sax/features/external-general-entities", false)
        setFeature("http://xml.org/sax/features/external-parameter-entities", false)
        setAttribute(XMLConstants.ACCESS_EXTERNAL_DTD, "")
        setAttribute(XMLConstants.ACCESS_EXTERNAL_SCHEMA, "")
    }
    val versions = factory.newDocumentBuilder().parse(bytes.inputStream()).getElementsByTagName("version")
    return (0 until versions.length).map { versions.item(it).textContent.trim() }
        .filter { it.matches(Regex("\\d+\\.\\d+\\.\\d+")) }
        .maxWithOrNull(compareBy { ComparableVersion(it) }) ?: error("No stable Maven release")
}

internal class ToolReleaseHttp {
    private val client = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(5))
        .followRedirects(HttpClient.Redirect.NORMAL).build()
    fun fetch(url: String): ByteArray {
        val request = HttpRequest.newBuilder(URI.create(url)).timeout(Duration.ofSeconds(10))
            .header("User-Agent", "Scryer/0.1").GET().build()
        val response = client.send(request, HttpResponse.BodyHandlers.ofByteArray())
        require(response.statusCode() == 200) { "HTTP ${response.statusCode()}" }
        require(response.body().size <= 1024 * 1024) { "Release metadata exceeds 1 MiB" }
        return response.body()
    }
}
