package com.edwardnoaland.scryer.analyze.model

import com.edwardnoaland.scryer.analyze.SourceRole
import java.nio.file.Path

data class AnalysisSourceRoot(val path: Path, val role: SourceRole)
data class AnalysisModule(
    val id: String,
    val directory: Path,
    val sources: List<AnalysisSourceRoot>,
    val classpath: List<Path>,
    val dependencies: List<String> = emptyList(),
    val notes: List<String> = emptyList(),
)
data class AnalysisInputs(val modules: List<AnalysisModule> = emptyList(), val notes: List<String> = emptyList())
