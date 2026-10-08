package com.edwardnoaland.scryer.analyze.evidence

import com.edwardnoaland.scryer.analyze.SymbolId
import org.jacoco.core.analysis.Analyzer
import org.jacoco.core.analysis.ICounter
import org.jacoco.core.tools.ExecFileLoader
import java.nio.file.Files
import java.nio.file.Path

internal object JacocoExecReader {
    fun read(artifact: EvidenceArtifact, classes: List<Path>, root: Path): CoverageDataset {
        val loader = ExecFileLoader()
        loader.load(artifact.retained.toFile())
        val coverage = mutableListOf<ClassCoverage>()
        val notes = mutableListOf("Execution data is aggregate/unknown test scope; session IDs are not test IDs. Separate artifacts are not merged.")
        for (file in classes) {
            try {
                val analyzer = Analyzer(loader.executionDataStore) { data ->
                    val match = when {
                        data.isNoMatch -> ClassDataMatch.CLASS_ID_MISMATCH
                        loader.executionDataStore.get(data.id) != null -> ClassDataMatch.MATCHED
                        else -> ClassDataMatch.NO_EXECUTION_RECORD
                    }
                    val methods = data.methods.map { method ->
                        MethodCoverage(SymbolId(data.name.replace('/', '.'), method.name, method.desc),
                            method.firstLine.takeIf { it >= 0 }, counter(method.instructionCounter), counter(method.branchCounter), counter(method.lineCounter))
                    }
                    coverage += ClassCoverage(data.name.replace('/', '.'), root.relativize(file).toString(), match, methods, java.lang.Long.toUnsignedString(data.id, 16))
                }
                analyzer.analyzeClass(Files.readAllBytes(file), file.toString())
            } catch (exception: Exception) {
                notes += "Cannot analyze ${root.relativize(file)}: ${exception.message}"
            }
        }
        if (classes.isEmpty()) notes += "No fresh compiled production classes were found in conventional output roots."
        notes += "No execution record may mean never loaded or instrumentation exclusion; it does not establish a test-to-method relationship."
        return CoverageDataset(artifact, "JACOCO_EXEC", coverage, notes)
    }
    private fun counter(counter: ICounter) = CoverageCounter(counter.missedCount, counter.coveredCount)
}
