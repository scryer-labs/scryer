package com.edwardnoaland.scryer

import java.nio.file.Files
import java.nio.file.Path
import java.util.Properties
import kotlin.io.path.isDirectory
import kotlin.io.path.isRegularFile

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
    val compileTooling: CompileToolingFacts = CompileToolingFacts(),
)

class ScanException(message: String) : RuntimeException(message)

class RepositoryScanner {
    fun scan(path: Path): RepositoryFacts {
        val root = path.toAbsolutePath().normalize()
        if (!root.isDirectory()) throw ScanException("Not a directory: $root")
        val builds = mutableListOf<BuildFacts>()
        for (file in listOf("build.gradle", "build.gradle.kts")) {
            if (root.resolve(file).isRegularFile()) {
                builds += BuildFacts("Gradle", wrapperVersion(root.resolve("gradle/wrapper/gradle-wrapper.properties"),
                    Regex("gradle-(.+)-(?:bin|all)\\.zip")), file, root.resolve("gradlew").isRegularFile())
            }
        }
        if (builds.isEmpty()) {
            val settings = listOf("settings.gradle", "settings.gradle.kts").firstOrNull { root.resolve(it).isRegularFile() }
            if (settings != null) builds += BuildFacts("Gradle", wrapperVersion(root.resolve("gradle/wrapper/gradle-wrapper.properties"),
                Regex("gradle-(.+)-(?:bin|all)\\.zip")), settings, root.resolve("gradlew").isRegularFile())
        }
        if (root.resolve("pom.xml").isRegularFile()) {
            builds += BuildFacts("Maven", wrapperVersion(root.resolve(".mvn/wrapper/maven-wrapper.properties"),
                Regex("apache-maven-(.+)-bin\\.(?:zip|tar\\.gz)")), "pom.xml", root.resolve("mvnw").isRegularFile())
        }
        if (builds.isEmpty()) throw ScanException("No build.gradle, build.gradle.kts or pom.xml found in $root")
        val (modules, moduleNotes) = discoverModules(root, builds)
        val declarations = modules.map { module ->
            val directory = root.resolve(module.directory)
            val metadata = if (module.definition == "pom.xml") MavenDeclarations.read(directory)
                else if (module.definition.startsWith("build.gradle")) GradleDeclarations.read(directory, module.definition)
                else Declarations()
            metadata.copy(dependencies = metadata.dependencies.map { it.copy(module = module.id,
                source = if (module.directory == ".") it.source else "${module.directory}/${it.source}") })
        }
        val dependencies = declarations.flatMap { it.dependencies }
        val plugins = declarations.flatMap { it.plugins }
        val info = inspectMetadata(root, modules, dependencies, plugins)
        val parentPoms = modules.filter { it.definition == "pom.xml" }.mapNotNull { module ->
            readPom(root.resolve(module.directory).resolve("pom.xml")).child("parent")?.let {
                ParentPomFact(module.id, it.text("groupId"), it.text("artifactId"), it.text("version"))
            }
        }
        val processors = dependencies.filter { "annotationProcessor" in it.configuration || "kapt" in it.configuration }.toMutableList()
        for (module in modules.filter { it.definition == "pom.xml" }) {
            val pom = readPom(root.resolve(module.directory).resolve("pom.xml"))
            val paths = pom.getElementsByTagNameNS("*", "annotationProcessorPaths")
            for (i in 0 until paths.length) {
                val element = paths.item(i) as org.w3c.dom.Element
                for (path in element.children("path")) processors += DependencyDeclaration("annotationProcessor", "${path.text("groupId")}:${path.text("artifactId")}", path.text("version"), "${module.directory}/pom.xml", module = module.id)
            }
        }
        val buildText = modules.map { root.resolve(it.directory).resolve(it.definition) }.filter { it.isRegularFile() }.joinToString("\n") { it.toFile().readText() }
        val generators = listOf("mapstruct", "querydsl", "protobuf", "openapi", "lombok").filter { it in buildText.lowercase() }
        return RepositoryFacts(root, builds, dependencies, plugins,
            declarations.flatMap { it.notes }.filterNot { "root declarations only" in it } + moduleNotes,
            modules, info.language, info.sources, info.testing, info.signals, info.verification,
            info.repositories, info.customBuildFiles, info.frameworks, parentPoms = parentPoms,
            compileTooling = CompileToolingFacts(processors, generators, dependencies.any { "lombok" in it.notation } || info.signals.any { it.name == "Lombok" && it.locations.isNotEmpty() }))

    }

    private fun wrapperVersion(file: Path, pattern: Regex): String? {
        if (!file.isRegularFile()) return null
        val properties = Properties()
        Files.newInputStream(file).use(properties::load)
        val url = properties.getProperty("distributionUrl") ?: return null
        val filename = url.substringBefore('?').substringBefore('#').substringAfterLast('/')
        return pattern.matchEntire(filename)?.groupValues?.get(1)
    }
}
