package com.edwardnoaland.scryer.worker

import com.edwardnoaland.scryer.analyze.AnalyzeService
import com.edwardnoaland.scryer.analyze.GitComparer
import com.edwardnoaland.scryer.analyze.GitProcess
import com.edwardnoaland.scryer.analyze.report.buildAnalyzeReport
import com.edwardnoaland.scryer.repository.BuildTool
import com.edwardnoaland.scryer.scan.ScanService
import com.edwardnoaland.scryer.scan.directDependencies
import com.edwardnoaland.scryer.scan.remote.ToolUpdateCollector
import com.edwardnoaland.scryer.scan.remote.RemoteVersionCollector
import com.edwardnoaland.scryer.scan.remote.scopedSelectedVersions
import com.edwardnoaland.scryer.scan.resolve.DependencyResolver
import com.edwardnoaland.scryer.serialization.jsonMapper
import com.fasterxml.jackson.databind.JsonNode
import com.fasterxml.jackson.databind.node.ObjectNode
import java.nio.file.Path

/** One request/response per process. Stdout is protocol JSON only; diagnostics go to stderr. */
fun main() {
    var request: JsonNode? = null
    val response = linkedMapOf<String, Any?>("protocolVersion" to 1, "stack" to "java")
    try {
        val bytes = System.`in`.readNBytes(1_048_577)
        require(bytes.size <= 1_048_576) { "Request exceeds 1 MiB" }
        request = jsonMapper.readTree(bytes)
        response["requestId"] = request.requiredText("requestId")
        response["operation"] = request.requiredText("operation")
        require(request["protocolVersion"]?.asInt() == 1 && request["stack"]?.asText() == "java") { "Unsupported protocol version or stack" }
        val progress: (String) -> Unit = { System.err.println("scryer: $it") }
        response["report"] = when (request.requiredText("operation")) {
            "scan" -> {
                val input = requireNotNull(request["scan"]) { "Missing scan request" }
                val facts = ScanService(resolver = DependencyResolver(progress = progress)).scan(
                    Path.of(input.requiredText("repository")), input["static"]?.asBoolean() ?: false, input.buildTool())
                val enriched = if (input["remoteList"]?.asBoolean() == true) facts.copy(remoteVersions = RemoteVersionCollector(progress = progress)
                    .collect(directDependencies(facts)) { scopedSelectedVersions(facts, it) },
                    toolUpdates = ToolUpdateCollector(progress = progress).collect(facts)) else facts
                // Public contracts use filesystem paths, not Jackson's default Path URI representation.
                jsonMapper.valueToTree<ObjectNode>(enriched).put("root", enriched.root.toString())
            }
            "analyze" -> {
                val input = requireNotNull(request["analyze"]) { "Missing analyze request" }
                val comparison = GitComparer().compare(Path.of(input["comparisonRepository"]?.asText() ?: input.requiredText("repository")), input.requiredText("before"), input.requiredText("after"))
                    .copy(repository = Path.of(input.requiredText("repository")).toRealPath())
                val before = Path.of(input.requiredText("beforeRoot")).toRealPath()
                val after = Path.of(input.requiredText("afterRoot")).toRealPath()
                require(GitProcess.run(before, "rev-parse", "HEAD").trim() == comparison.before &&
                    GitProcess.run(after, "rev-parse", "HEAD").trim() == comparison.after) { "Snapshot SHA mismatch" }
                val result = AnalyzeService(progress).analyzeSnapshots(comparison, before, after,
                    runTests = !(input["skipTests"]?.asBoolean() ?: false),
                    testCommand = input["testCommand"]?.asText()?.takeIf { it.isNotBlank() }, buildTool = input.buildTool())
                buildAnalyzeReport(result)
            }
            else -> error("Unsupported operation")
        }
        response["error"] = null
    } catch (exception: Exception) {
        exception.printStackTrace(System.err)
        response["requestId"] = request?.get("requestId")?.asText().orEmpty()
        response["operation"] = request?.get("operation")?.asText().orEmpty()
        response["report"] = null
        response["error"] = mapOf("code" to "JAVA_ANALYZER_ERROR", "message" to (exception.message ?: exception.javaClass.simpleName))
    }
    System.out.println(jsonMapper.writeValueAsString(response))
}

private fun JsonNode.requiredText(name: String): String {
    val value = get(name)
    require(value != null && value.isTextual && value.asText().isNotBlank()) { "Missing/non-string $name" }
    return value.asText()
}
private fun JsonNode.buildTool(): BuildTool? = get("buildTool")?.asText()?.takeIf { it.isNotBlank() }?.let {
    requireNotNull(BuildTool.parse(it)) { "Unsupported buildTool: $it" }
}
