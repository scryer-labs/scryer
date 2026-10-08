package com.edwardnoaland.scryer.scan.model

import java.nio.file.Path

data class BuildFacts(val tool: String, val version: String?, val definition: String, val wrapperPresent: Boolean = false)
data class RepositoryFacts(
    val root: Path,
    val builds: List<BuildFacts>,
    val dependencies: List<DependencyDeclaration> = emptyList(),
    val plugins: List<PluginDeclaration> = emptyList(),
    val notes: List<String> = emptyList(),
    val modules: List<ModuleFacts> = emptyList(),
    val language: LanguageFacts = LanguageFacts(),
    val sources: SourceFacts = SourceFacts(),
    val testing: TestingFacts = TestingFacts(),
    val signals: List<SignalFact> = emptyList(),
    val verification: VerificationFacts = VerificationFacts(),
    val repositories: List<String> = emptyList(),
    val customBuildFiles: List<String> = emptyList(),
    val frameworks: List<String> = emptyList(),
    val resolution: ResolutionFacts = ResolutionFacts(),
    val schemaVersion: Int = 1,
    val parentPoms: List<ParentPomFact> = emptyList(),
    val remoteVersions: com.edwardnoaland.scryer.scan.model.RemoteVersions? = null,
    val compileTooling: CompileToolingFacts = CompileToolingFacts(),
)

