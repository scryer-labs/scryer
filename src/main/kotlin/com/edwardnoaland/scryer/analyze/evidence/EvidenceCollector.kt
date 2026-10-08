package com.edwardnoaland.scryer.analyze.evidence

import com.edwardnoaland.scryer.serialization.jsonMapper
import com.edwardnoaland.scryer.analyze.execute.TestExecution
import com.edwardnoaland.scryer.analyze.execute.TestRunStatus
import java.nio.file.Files
import java.nio.file.Path

internal class EvidenceCollector {
    fun collect(root: Path, execution: TestExecution, before: Map<Path, ArtifactStamp>): ExecutionEvidence {
        if (execution.status !in setOf(TestRunStatus.SUCCEEDED, TestRunStatus.FAILED)) {
            return ExecutionEvidence(execution.afterSha, execution.status, notes = listOf("Evidence unavailable for ${execution.status}; no coverage inferred."))
        }
        val log = execution.log ?: return ExecutionEvidence(execution.afterSha, execution.status, notes = listOf("No run artifact directory available."))
        val current = ArtifactInventory.capture(root)
        val fresh = current.filter { (path, stamp) -> before[path] != stamp }
        val notes = mutableListOf("Artifact discovery uses conventional output paths only; instrumentation is not installed or configured by Scryer.")
        val stale = current.size - fresh.size
        if (stale > 0) notes += "$stale unchanged pre-existing artifacts ignored."
        val destination = log.parent.resolve("evidence")
        Files.createDirectories(destination)
        val retained = mutableMapOf<Path, EvidenceArtifact>()
        for ((path, stamp) in fresh) {
            val relative = root.relativize(path)
            val target = destination.resolve(relative)
            try {
                Files.createDirectories(target.parent)
                Files.copy(path, target)
                check(ArtifactInventory.hash(target) == stamp.hash) { "Artifact changed during acquisition" }
                retained[path] = EvidenceArtifact(relative.toString(), target, stamp.hash)
            } catch (exception: Exception) {
                notes += "Cannot retain $relative: ${exception.message}"
            }
        }
        val classes = fresh.filterValues { it.kind == ArtifactKind.CLASS }.keys.mapNotNull { retained[it]?.retained }
        val tests = mutableListOf<TestReport>()
        val coverage = mutableListOf<CoverageDataset>()
        for ((path, stamp) in fresh) {
            val artifact = retained[path] ?: continue
            try {
                when (stamp.kind) {
                    ArtifactKind.TEST_XML -> tests += EvidenceXml.tests(artifact)
                    ArtifactKind.JACOCO_XML -> coverage += EvidenceXml.coverage(artifact)
                    ArtifactKind.JACOCO_EXEC -> coverage += JacocoExecReader.read(artifact, classes, destination)
                    ArtifactKind.CLASS -> Unit
                }
            } catch (exception: Exception) {
                notes += "Cannot parse ${artifact.source}: ${exception.message} (raw artifact retained)."
            }
        }
        if (tests.isEmpty()) notes += "No fresh usable test XML reports found; test execution counts are unknown."
        if (coverage.isEmpty()) notes += "No fresh usable JaCoCo data found; coverage is unavailable, not zero."
        if (execution.status == TestRunStatus.FAILED) notes += "Command failed: reports describe partial/failed execution only."
        val result = ExecutionEvidence(execution.afterSha, execution.status, tests, coverage, notes,
            artifacts = retained.values.toList(), manifest = destination.resolve("manifest.json"))
        return try {
            jsonMapper.writerWithDefaultPrettyPrinter().writeValue(result.manifest!!.toFile(), result)
            result
        } catch (exception: Exception) {
            result.copy(manifest = null, notes = result.notes + "Cannot write artifact manifest: ${exception.message}")
        }
    }
}
